package service

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// mockSuggestionRepo는 테스트용 SuggestionRepository 모의 구현체다.
type mockSuggestionRepo struct {
	tags       []string
	companions []string
	err        error
	// 마지막 호출 인자를 기록해 검증에 활용한다.
	lastUserID string
	lastQ      string
}

func (m *mockSuggestionRepo) GetTagSuggestions(_ context.Context, userID, q string) ([]string, error) {
	m.lastUserID = userID
	m.lastQ = q
	return m.tags, m.err
}

func (m *mockSuggestionRepo) GetCompanionSuggestions(_ context.Context, userID, q string) ([]string, error) {
	m.lastUserID = userID
	m.lastQ = q
	return m.companions, m.err
}

func TestGetTagSuggestions_NormalFlow(t *testing.T) {
	repo := &mockSuggestionRepo{
		tags: []string{"초콜릿", "체리", "플로럴"},
	}
	svc := NewSuggestionService(repo)

	got, err := svc.GetTagSuggestions(context.Background(), "user-1", "초")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 suggestions, got %d", len(got))
	}
	// 서비스가 쿼리를 그대로 repo에 전달했는지 검증한다.
	if repo.lastQ != "초" {
		t.Errorf("expected q=초, got %q", repo.lastQ)
	}
}

func TestGetTagSuggestions_EmptyQ_ReturnsAll(t *testing.T) {
	repo := &mockSuggestionRepo{
		tags: []string{"초콜릿", "과일향"},
	}
	svc := NewSuggestionService(repo)

	got, err := svc.GetTagSuggestions(context.Background(), "user-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	// 빈 q는 그대로 repo에 전달한다.
	if repo.lastQ != "" {
		t.Errorf("expected empty q, got %q", repo.lastQ)
	}
}

func TestGetTagSuggestions_QWithWhitespace_IsTrimmed(t *testing.T) {
	repo := &mockSuggestionRepo{tags: []string{"초콜릿"}}
	svc := NewSuggestionService(repo)

	_, err := svc.GetTagSuggestions(context.Background(), "user-1", "  초콜릿  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastQ != "초콜릿" {
		t.Errorf("expected trimmed q, got %q", repo.lastQ)
	}
}

func TestGetTagSuggestions_QTooLong_ReturnsValidationError(t *testing.T) {
	repo := &mockSuggestionRepo{}
	svc := NewSuggestionService(repo)

	longQ := strings.Repeat("a", 101)
	_, err := svc.GetTagSuggestions(context.Background(), "user-1", longQ)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetTagSuggestions_EmptyUserID_ReturnsValidationError(t *testing.T) {
	repo := &mockSuggestionRepo{}
	svc := NewSuggestionService(repo)

	_, err := svc.GetTagSuggestions(context.Background(), "", "초콜릿")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetTagSuggestions_RepoError_IsWrapped(t *testing.T) {
	repoErr := errors.New("db connection lost")
	repo := &mockSuggestionRepo{err: repoErr}
	svc := NewSuggestionService(repo)

	_, err := svc.GetTagSuggestions(context.Background(), "user-1", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// repo 오류가 적절히 래핑되어 전파되어야 한다.
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error to be wrapped, got %v", err)
	}
}

func TestGetCompanionSuggestions_NormalFlow(t *testing.T) {
	repo := &mockSuggestionRepo{
		companions: []string{"지수", "민준"},
	}
	svc := NewSuggestionService(repo)

	got, err := svc.GetCompanionSuggestions(context.Background(), "user-1", "지")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 suggestions, got %d", len(got))
	}
	if repo.lastQ != "지" {
		t.Errorf("expected q=지, got %q", repo.lastQ)
	}
}

func TestGetCompanionSuggestions_EmptyUserID_ReturnsValidationError(t *testing.T) {
	repo := &mockSuggestionRepo{}
	svc := NewSuggestionService(repo)

	_, err := svc.GetCompanionSuggestions(context.Background(), "   ", "지수")
	if err == nil {
		t.Fatal("expected validation error for blank user_id, got nil")
	}
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
