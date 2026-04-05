package service

import (
	"context"
	"fmt"
	"strings"

	"coffee-of-the-day/backend/internal/repository"
)

// SuggestionService는 태그·동반자 자동완성 제안을 반환하는 인터페이스다.
type SuggestionService interface {
	GetTagSuggestions(ctx context.Context, userID, q string) ([]string, error)
	GetCompanionSuggestions(ctx context.Context, userID, q string) ([]string, error)
}

// DefaultSuggestionService는 SuggestionService의 기본 구현체다.
type DefaultSuggestionService struct {
	repo repository.SuggestionRepository
}

// NewSuggestionService는 DefaultSuggestionService를 생성한다.
func NewSuggestionService(repo repository.SuggestionRepository) *DefaultSuggestionService {
	return &DefaultSuggestionService{repo: repo}
}

func (s *DefaultSuggestionService) GetTagSuggestions(ctx context.Context, userID, q string) ([]string, error) {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return nil, err
	}

	normalizedQ, err := normalizeQ(q)
	if err != nil {
		return nil, err
	}

	suggestions, err := s.repo.GetTagSuggestions(ctx, normalizedUserID, normalizedQ)
	if err != nil {
		return nil, fmt.Errorf("get tag suggestions: %w", err)
	}
	return suggestions, nil
}

func (s *DefaultSuggestionService) GetCompanionSuggestions(ctx context.Context, userID, q string) ([]string, error) {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return nil, err
	}

	normalizedQ, err := normalizeQ(q)
	if err != nil {
		return nil, err
	}

	suggestions, err := s.repo.GetCompanionSuggestions(ctx, normalizedUserID, normalizedQ)
	if err != nil {
		return nil, fmt.Errorf("get companion suggestions: %w", err)
	}
	return suggestions, nil
}

// normalizeQ는 검색어를 정규화한다. handler에서 빈 입력을 사전 차단하므로 빈 문자열은 도달하지 않는다.
func normalizeQ(q string) (string, error) {
	trimmed := strings.TrimSpace(q)
	if len(trimmed) > 100 {
		return "", newValidationError("q", "검색어는 100자 이하여야 합니다")
	}
	return trimmed, nil
}
