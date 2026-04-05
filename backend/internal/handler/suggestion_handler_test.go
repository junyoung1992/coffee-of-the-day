package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"coffee-of-the-day/backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSuggestionService는 테스트에서 SuggestionService를 대체하는 스텁이다.
type stubSuggestionService struct {
	tagsFunc       func(ctx context.Context, userID, q string) ([]string, error)
	companionsFunc func(ctx context.Context, userID, q string) ([]string, error)
}

func (s *stubSuggestionService) GetTagSuggestions(ctx context.Context, userID, q string) ([]string, error) {
	return s.tagsFunc(ctx, userID, q)
}

func (s *stubSuggestionService) GetCompanionSuggestions(ctx context.Context, userID, q string) ([]string, error) {
	return s.companionsFunc(ctx, userID, q)
}

// ---------------------------------------------------------------------------
// GetTagSuggestions
// ---------------------------------------------------------------------------

func TestGetTagSuggestions_Success(t *testing.T) {
	svc := &stubSuggestionService{
		tagsFunc: func(_ context.Context, userID, q string) ([]string, error) {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, "초", q)
			return []string{"초콜릿", "초록사과"}, nil
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/tags?q=초", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetTagSuggestions(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, []string{"초콜릿", "초록사과"}, resp.Suggestions)
}

func TestGetTagSuggestions_EmptyQ_ReturnsEmptyArray(t *testing.T) {
	svc := &stubSuggestionService{
		tagsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			t.Fatal("빈 q일 때 서비스가 호출되면 안 된다")
			return nil, nil
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/tags?q=", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetTagSuggestions(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Suggestions)
	assert.Empty(t, resp.Suggestions)
}

func TestGetTagSuggestions_WhitespaceOnlyQ_ReturnsEmptyArray(t *testing.T) {
	svc := &stubSuggestionService{
		tagsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			t.Fatal("공백만 있는 q일 때 서비스가 호출되면 안 된다")
			return nil, nil
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/tags?q=%20%20", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetTagSuggestions(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Suggestions)
	assert.Empty(t, resp.Suggestions)
}

func TestGetTagSuggestions_MissingQ_ReturnsEmptyArray(t *testing.T) {
	svc := &stubSuggestionService{
		tagsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			t.Fatal("q 파라미터 누락 시 서비스가 호출되면 안 된다")
			return nil, nil
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/tags", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetTagSuggestions(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Suggestions)
	assert.Empty(t, resp.Suggestions)
}

func TestGetTagSuggestions_NoResults_ReturnsEmptyArray(t *testing.T) {
	svc := &stubSuggestionService{
		tagsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return []string{}, nil
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/tags?q=없는태그", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetTagSuggestions(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// null 대신 빈 배열로 직렬화되어야 한다.
	assert.NotNil(t, resp.Suggestions)
	assert.Empty(t, resp.Suggestions)
}

func TestGetTagSuggestions_ValidationError_ReturnsBadRequest(t *testing.T) {
	svc := &stubSuggestionService{
		tagsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return nil, &service.ValidationError{Field: "q", Message: "검색어는 100자 이하여야 합니다"}
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/tags?q=x", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetTagSuggestions(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp errorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Field)
	assert.Equal(t, "q", *resp.Field)
}

func TestGetTagSuggestions_ServiceError_ReturnsInternalServerError(t *testing.T) {
	svc := &stubSuggestionService{
		tagsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return nil, errors.New("db connection lost")
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/tags?q=초", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetTagSuggestions(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// GetCompanionSuggestions
// ---------------------------------------------------------------------------

func TestGetCompanionSuggestions_Success(t *testing.T) {
	svc := &stubSuggestionService{
		companionsFunc: func(_ context.Context, userID, q string) ([]string, error) {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, "지", q)
			return []string{"지수", "지훈"}, nil
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/companions?q=지", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetCompanionSuggestions(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, []string{"지수", "지훈"}, resp.Suggestions)
}

func TestGetCompanionSuggestions_WhitespaceOnlyQ_ReturnsEmptyArray(t *testing.T) {
	svc := &stubSuggestionService{
		companionsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			t.Fatal("공백만 있는 q일 때 서비스가 호출되면 안 된다")
			return nil, nil
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/companions?q=%20%20", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetCompanionSuggestions(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Suggestions)
	assert.Empty(t, resp.Suggestions)
}

func TestGetCompanionSuggestions_EmptyQ_ReturnsEmptyArray(t *testing.T) {
	svc := &stubSuggestionService{
		companionsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			t.Fatal("빈 q일 때 서비스가 호출되면 안 된다")
			return nil, nil
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/companions?q=", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetCompanionSuggestions(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Suggestions)
	assert.Empty(t, resp.Suggestions)
}

func TestGetCompanionSuggestions_MissingQ_ReturnsEmptyArray(t *testing.T) {
	svc := &stubSuggestionService{
		companionsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			t.Fatal("q 파라미터 누락 시 서비스가 호출되면 안 된다")
			return nil, nil
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/companions", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetCompanionSuggestions(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp suggestionsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Suggestions)
	assert.Empty(t, resp.Suggestions)
}

func TestGetCompanionSuggestions_ValidationError_ReturnsBadRequest(t *testing.T) {
	svc := &stubSuggestionService{
		companionsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return nil, &service.ValidationError{Field: "user_id", Message: "user_id는 필수입니다"}
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/companions?q=지", nil)
	r = withUserID(r, "")
	w := httptest.NewRecorder()

	h.GetCompanionSuggestions(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetCompanionSuggestions_ServiceError_ReturnsInternalServerError(t *testing.T) {
	svc := &stubSuggestionService{
		companionsFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return nil, errors.New("unexpected db error")
		},
	}
	h := NewSuggestionHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/companions?q=지", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.GetCompanionSuggestions(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
