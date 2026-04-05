package handler

import (
	"errors"
	"net/http"
	"strings"

	"coffee-of-the-day/backend/internal/service"
)

type suggestionsResponse struct {
	Suggestions []string `json:"suggestions"`
}

// SuggestionHandler는 자동완성 관련 HTTP 요청을 처리한다.
type SuggestionHandler struct {
	svc service.SuggestionService
}

// NewSuggestionHandler는 SuggestionHandler를 생성한다.
func NewSuggestionHandler(svc service.SuggestionService) *SuggestionHandler {
	return &SuggestionHandler{svc: svc}
}

// GetTagSuggestions는 GET /api/v1/suggestions/tags?q= 를 처리한다.
func (h *SuggestionHandler) GetTagSuggestions(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	q := r.URL.Query().Get("q")

	// 빈 입력 또는 공백만 있는 경우 서비스 호출 없이 빈 배열 반환
	if len(strings.TrimSpace(q)) < 1 {
		writeJSON(w, http.StatusOK, suggestionsResponse{Suggestions: []string{}})
		return
	}

	suggestions, err := h.svc.GetTagSuggestions(r.Context(), userID, q)
	if err != nil {
		writeSuggestionServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, suggestionsResponse{Suggestions: suggestions})
}

// GetCompanionSuggestions는 GET /api/v1/suggestions/companions?q= 를 처리한다.
func (h *SuggestionHandler) GetCompanionSuggestions(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	q := r.URL.Query().Get("q")

	// 빈 입력 또는 공백만 있는 경우 서비스 호출 없이 빈 배열 반환
	if len(strings.TrimSpace(q)) < 1 {
		writeJSON(w, http.StatusOK, suggestionsResponse{Suggestions: []string{}})
		return
	}

	suggestions, err := h.svc.GetCompanionSuggestions(r.Context(), userID, q)
	if err != nil {
		writeSuggestionServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, suggestionsResponse{Suggestions: suggestions})
}

func writeSuggestionServiceError(w http.ResponseWriter, err error) {
	var ve *service.ValidationError
	switch {
	case errors.As(err, &ve):
		field := ve.Field
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ve.Message, Field: &field})
	default:
		writeError(w, http.StatusInternalServerError, "내부 오류가 발생했습니다")
	}
}
