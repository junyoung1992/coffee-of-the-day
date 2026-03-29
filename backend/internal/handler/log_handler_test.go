package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/service"
)

// stubLogService는 테스트에서 LogService를 대체하는 스텁이다.
type stubLogService struct {
	createFunc func(ctx context.Context, userID string, req service.CreateLogRequest) (domain.CoffeeLogFull, error)
	getFunc    func(ctx context.Context, userID, logID string) (domain.CoffeeLogFull, error)
	listFunc   func(ctx context.Context, userID string, filter service.ListLogsFilter) (service.ListLogsResult, error)
	updateFunc func(ctx context.Context, userID, logID string, req service.UpdateLogRequest) (domain.CoffeeLogFull, error)
	deleteFunc func(ctx context.Context, userID, logID string) error
}

func (s *stubLogService) CreateLog(ctx context.Context, userID string, req service.CreateLogRequest) (domain.CoffeeLogFull, error) {
	return s.createFunc(ctx, userID, req)
}
func (s *stubLogService) GetLog(ctx context.Context, userID, logID string) (domain.CoffeeLogFull, error) {
	return s.getFunc(ctx, userID, logID)
}
func (s *stubLogService) ListLogs(ctx context.Context, userID string, filter service.ListLogsFilter) (service.ListLogsResult, error) {
	return s.listFunc(ctx, userID, filter)
}
func (s *stubLogService) UpdateLog(ctx context.Context, userID, logID string, req service.UpdateLogRequest) (domain.CoffeeLogFull, error) {
	return s.updateFunc(ctx, userID, logID, req)
}
func (s *stubLogService) DeleteLog(ctx context.Context, userID, logID string) error {
	return s.deleteFunc(ctx, userID, logID)
}

// withUserID는 테스트 요청에 userID context를 주입하는 헬퍼이다.
func withUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), userIDKey, userID)
	return r.WithContext(ctx)
}

// withChiParam은 chi URL 파라미터를 테스트 요청에 주입하는 헬퍼이다.
func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func sampleCafeLog() domain.CoffeeLogFull {
	cafeName := "블루보틀"
	coffeeName := "싱글 오리진"
	rating := 4.5
	return domain.CoffeeLogFull{
		CoffeeLog: domain.CoffeeLog{
			ID:         "log-1",
			UserID:     "user-1",
			RecordedAt: "2026-03-29T10:00:00Z",
			Companions: []string{},
			LogType:    domain.LogTypeCafe,
			CreatedAt:  "2026-03-29T10:00:00Z",
			UpdatedAt:  "2026-03-29T10:00:00Z",
		},
		Cafe: &domain.CafeDetail{
			CafeName:    cafeName,
			CoffeeName:  coffeeName,
			TastingTags: []string{"초콜릿", "체리"},
			Rating:      &rating,
		},
	}
}

// --- CreateLog 테스트 ---

func TestCreateLog_Success(t *testing.T) {
	svc := &stubLogService{
		createFunc: func(_ context.Context, userID string, req service.CreateLogRequest) (domain.CoffeeLogFull, error) {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, domain.LogTypeCafe, req.LogType)
			return sampleCafeLog(), nil
		},
	}
	h := NewLogHandler(svc)

	body := `{
		"recorded_at": "2026-03-29T10:00:00Z",
		"log_type": "cafe",
		"companions": [],
		"cafe": {
			"cafe_name": "블루보틀",
			"coffee_name": "싱글 오리진",
			"tasting_tags": ["초콜릿", "체리"],
			"rating": 4.5
		}
	}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/logs", bytes.NewBufferString(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.CreateLog(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp coffeeLogResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "log-1", resp.ID)
	assert.Equal(t, "cafe", resp.LogType)
	require.NotNil(t, resp.Cafe)
	assert.Equal(t, "블루보틀", resp.Cafe.CafeName)
}

func TestCreateLog_InvalidJSON(t *testing.T) {
	h := NewLogHandler(&stubLogService{})

	r := httptest.NewRequest(http.MethodPost, "/api/v1/logs", bytes.NewBufferString("not-json"))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.CreateLog(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateLog_ValidationError(t *testing.T) {
	svc := &stubLogService{
		createFunc: func(_ context.Context, userID string, req service.CreateLogRequest) (domain.CoffeeLogFull, error) {
			return domain.CoffeeLogFull{}, &service.ValidationError{Field: "log_type", Message: "cafe 또는 brew만 허용됩니다"}
		},
	}
	h := NewLogHandler(svc)

	body := `{"recorded_at":"2026-03-29","log_type":"invalid","companions":[]}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/logs", bytes.NewBufferString(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.CreateLog(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- ListLogs 테스트 ---

func TestListLogs_Success(t *testing.T) {
	svc := &stubLogService{
		listFunc: func(_ context.Context, userID string, filter service.ListLogsFilter) (service.ListLogsResult, error) {
			assert.Equal(t, "user-1", userID)
			return service.ListLogsResult{
				Items:   []domain.CoffeeLogFull{sampleCafeLog()},
				HasNext: false,
			}, nil
		},
	}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.ListLogs(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp listLogsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Items, 1)
	assert.False(t, resp.HasNext)
}

func TestListLogs_WithFilter(t *testing.T) {
	svc := &stubLogService{
		listFunc: func(_ context.Context, userID string, filter service.ListLogsFilter) (service.ListLogsResult, error) {
			require.NotNil(t, filter.LogType)
			assert.Equal(t, domain.LogTypeCafe, *filter.LogType)
			assert.Equal(t, 10, filter.Limit)
			return service.ListLogsResult{Items: []domain.CoffeeLogFull{}}, nil
		},
	}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs?log_type=cafe&limit=10", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.ListLogs(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListLogs_InvalidLimit(t *testing.T) {
	h := NewLogHandler(&stubLogService{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs?limit=abc", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.ListLogs(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- GetLog 테스트 ---

func TestGetLog_Success(t *testing.T) {
	svc := &stubLogService{
		getFunc: func(_ context.Context, userID, logID string) (domain.CoffeeLogFull, error) {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, "log-1", logID)
			return sampleCafeLog(), nil
		},
	}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/log-1", nil)
	r = withUserID(r, "user-1")
	r = withChiParam(r, "id", "log-1")
	w := httptest.NewRecorder()

	h.GetLog(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp coffeeLogResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "log-1", resp.ID)
}

func TestGetLog_NotFound(t *testing.T) {
	svc := &stubLogService{
		getFunc: func(_ context.Context, userID, logID string) (domain.CoffeeLogFull, error) {
			return domain.CoffeeLogFull{}, service.ErrNotFound
		},
	}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs/missing", nil)
	r = withUserID(r, "user-1")
	r = withChiParam(r, "id", "missing")
	w := httptest.NewRecorder()

	h.GetLog(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- UpdateLog 테스트 ---

func TestUpdateLog_Success(t *testing.T) {
	updated := sampleCafeLog()
	updated.RecordedAt = "2026-03-28T09:00:00Z"

	svc := &stubLogService{
		updateFunc: func(_ context.Context, userID, logID string, req service.UpdateLogRequest) (domain.CoffeeLogFull, error) {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, "log-1", logID)
			return updated, nil
		},
	}
	h := NewLogHandler(svc)

	body := `{
		"recorded_at": "2026-03-28T09:00:00Z",
		"log_type": "cafe",
		"companions": [],
		"cafe": {
			"cafe_name": "블루보틀",
			"coffee_name": "싱글 오리진",
			"tasting_tags": []
		}
	}`
	r := httptest.NewRequest(http.MethodPut, "/api/v1/logs/log-1", bytes.NewBufferString(body))
	r = withUserID(r, "user-1")
	r = withChiParam(r, "id", "log-1")
	w := httptest.NewRecorder()

	h.UpdateLog(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp coffeeLogResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "2026-03-28T09:00:00Z", resp.RecordedAt)
}

func TestUpdateLog_NotFound(t *testing.T) {
	svc := &stubLogService{
		updateFunc: func(_ context.Context, userID, logID string, req service.UpdateLogRequest) (domain.CoffeeLogFull, error) {
			return domain.CoffeeLogFull{}, service.ErrNotFound
		},
	}
	h := NewLogHandler(svc)

	body := `{"recorded_at":"2026-03-28","log_type":"cafe","companions":[],"cafe":{"cafe_name":"X","coffee_name":"Y","tasting_tags":[]}}`
	r := httptest.NewRequest(http.MethodPut, "/api/v1/logs/missing", bytes.NewBufferString(body))
	r = withUserID(r, "user-1")
	r = withChiParam(r, "id", "missing")
	w := httptest.NewRecorder()

	h.UpdateLog(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- DeleteLog 테스트 ---

func TestDeleteLog_Success(t *testing.T) {
	svc := &stubLogService{
		deleteFunc: func(_ context.Context, userID, logID string) error {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, "log-1", logID)
			return nil
		},
	}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodDelete, "/api/v1/logs/log-1", nil)
	r = withUserID(r, "user-1")
	r = withChiParam(r, "id", "log-1")
	w := httptest.NewRecorder()

	h.DeleteLog(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteLog_NotFound(t *testing.T) {
	svc := &stubLogService{
		deleteFunc: func(_ context.Context, userID, logID string) error {
			return service.ErrNotFound
		},
	}
	h := NewLogHandler(svc)

	r := httptest.NewRequest(http.MethodDelete, "/api/v1/logs/missing", nil)
	r = withUserID(r, "user-1")
	r = withChiParam(r, "id", "missing")
	w := httptest.NewRecorder()

	h.DeleteLog(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- UserIDMiddleware 테스트 ---

func TestUserIDMiddleware_MissingHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	UserIDMiddleware(next).ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserIDMiddleware_WithHeader(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "user-1", getUserID(r))
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-User-Id", "user-1")
	w := httptest.NewRecorder()

	UserIDMiddleware(next).ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- CORSMiddleware 테스트 ---

func TestCORSMiddleware_Options(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("OPTIONS preflight should not reach next handler")
	})

	r := httptest.NewRequest(http.MethodOptions, "/api/v1/logs", nil)
	r.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	CORSMiddleware(next).ServeHTTP(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_UnknownOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	r.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()

	CORSMiddleware(next).ServeHTTP(w, r)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- 응답 형식 테스트 ---

func TestLogResponse_CompanionsNeverNull(t *testing.T) {
	// companions가 nil일 때도 JSON에서 [] 로 직렬화되어야 한다
	log := domain.CoffeeLogFull{
		CoffeeLog: domain.CoffeeLog{
			ID:         "log-1",
			UserID:     "user-1",
			RecordedAt: "2026-03-29",
			Companions: nil, // nil 슬라이스
			LogType:    domain.LogTypeCafe,
			CreatedAt:  "2026-03-29",
			UpdatedAt:  "2026-03-29",
		},
		Cafe: &domain.CafeDetail{
			CafeName:    "카페",
			CoffeeName:  "아메리카노",
			TastingTags: []string{},
		},
	}

	resp := logToResponse(log)
	assert.NotNil(t, resp.Companions)
	assert.Empty(t, resp.Companions)

	data, err := json.Marshal(resp)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"companions":[]`)
}
