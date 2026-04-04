package handler

import (
	"encoding/json"
	"net/http"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/service"

	"github.com/go-chi/chi/v5"
)

// ---------------------------------------------------------------------------
// JSON request/response 타입
// ---------------------------------------------------------------------------

type cafePresetDetailJSON struct {
	CafeName    string   `json:"cafe_name"`
	CoffeeName  string   `json:"coffee_name"`
	TastingTags []string `json:"tasting_tags"`
}

type brewPresetDetailJSON struct {
	BeanName     string   `json:"bean_name"`
	BrewMethod   string   `json:"brew_method"`
	RecipeDetail *string  `json:"recipe_detail,omitempty"`
	BrewSteps    []string `json:"brew_steps"`
}

type presetResponse struct {
	ID         string                `json:"id"`
	UserID     string                `json:"user_id"`
	Name       string                `json:"name"`
	LogType    string                `json:"log_type"`
	LastUsedAt *string               `json:"last_used_at"`
	CreatedAt  string                `json:"created_at"`
	UpdatedAt  string                `json:"updated_at"`
	Cafe       *cafePresetDetailJSON `json:"cafe,omitempty"`
	Brew       *brewPresetDetailJSON `json:"brew,omitempty"`
}

type createPresetRequest struct {
	Name    string                `json:"name"`
	LogType string                `json:"log_type"`
	Cafe    *cafePresetDetailJSON `json:"cafe"`
	Brew    *brewPresetDetailJSON `json:"brew"`
}

type updatePresetRequest struct {
	Name string                `json:"name"`
	Cafe *cafePresetDetailJSON `json:"cafe"`
	Brew *brewPresetDetailJSON `json:"brew"`
}

type listPresetsResponse struct {
	Items []presetResponse `json:"items"`
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// PresetHandler는 프리셋 API 엔드포인트를 처리한다.
type PresetHandler struct {
	svc service.PresetService
}

// NewPresetHandler는 새 PresetHandler를 생성한다.
func NewPresetHandler(svc service.PresetService) *PresetHandler {
	return &PresetHandler{svc: svc}
}

// CreatePreset handles POST /api/v1/presets
func (h *PresetHandler) CreatePreset(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req createPresetRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "요청 본문을 파싱할 수 없습니다")
		return
	}

	preset, err := h.svc.CreatePreset(r.Context(), userID, service.CreatePresetRequest{
		Name:    req.Name,
		LogType: domain.LogType(req.LogType),
		Cafe:    cafePresetJSONToDomain(req.Cafe),
		Brew:    brewPresetJSONToDomain(req.Brew),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, presetToResponse(preset))
}

// ListPresets handles GET /api/v1/presets
func (h *PresetHandler) ListPresets(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	items, err := h.svc.ListPresets(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := make([]presetResponse, len(items))
	for i, item := range items {
		resp[i] = presetToResponse(item)
	}

	writeJSON(w, http.StatusOK, listPresetsResponse{Items: resp})
}

// GetPreset handles GET /api/v1/presets/{id}
func (h *PresetHandler) GetPreset(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	presetID := chi.URLParam(r, "id")

	preset, err := h.svc.GetPreset(r.Context(), userID, presetID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, presetToResponse(preset))
}

// UpdatePreset handles PUT /api/v1/presets/{id}
func (h *PresetHandler) UpdatePreset(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	presetID := chi.URLParam(r, "id")

	var req updatePresetRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "요청 본문을 파���할 수 없습니다")
		return
	}

	preset, err := h.svc.UpdatePreset(r.Context(), userID, presetID, service.UpdatePresetRequest{
		Name: req.Name,
		Cafe: cafePresetJSONToDomain(req.Cafe),
		Brew: brewPresetJSONToDomain(req.Brew),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, presetToResponse(preset))
}

// DeletePreset handles DELETE /api/v1/presets/{id}
func (h *PresetHandler) DeletePreset(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	presetID := chi.URLParam(r, "id")

	if err := h.svc.DeletePreset(r.Context(), userID, presetID); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UsePreset handles POST /api/v1/presets/{id}/use
func (h *PresetHandler) UsePreset(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	presetID := chi.URLParam(r, "id")

	if err := h.svc.UsePreset(r.Context(), userID, presetID); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Domain ↔ JSON 변환
// ---------------------------------------------------------------------------

func presetToResponse(p domain.PresetFull) presetResponse {
	resp := presetResponse{
		ID:         p.ID,
		UserID:     p.UserID,
		Name:       p.Name,
		LogType:    string(p.LogType),
		LastUsedAt: p.LastUsedAt,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
	if p.Cafe != nil {
		resp.Cafe = cafePresetDomainToJSON(p.Cafe)
	}
	if p.Brew != nil {
		resp.Brew = brewPresetDomainToJSON(p.Brew)
	}
	return resp
}

func cafePresetDomainToJSON(c *domain.CafePresetDetail) *cafePresetDetailJSON {
	if c == nil {
		return nil
	}
	j := &cafePresetDetailJSON{
		CafeName:    c.CafeName,
		CoffeeName:  c.CoffeeName,
		TastingTags: c.TastingTags,
	}
	if j.TastingTags == nil {
		j.TastingTags = []string{}
	}
	return j
}

func brewPresetDomainToJSON(b *domain.BrewPresetDetail) *brewPresetDetailJSON {
	if b == nil {
		return nil
	}
	j := &brewPresetDetailJSON{
		BeanName:     b.BeanName,
		BrewMethod:   string(b.BrewMethod),
		RecipeDetail: b.RecipeDetail,
		BrewSteps:    b.BrewSteps,
	}
	if j.BrewSteps == nil {
		j.BrewSteps = []string{}
	}
	return j
}

func cafePresetJSONToDomain(j *cafePresetDetailJSON) *domain.CafePresetDetail {
	if j == nil {
		return nil
	}
	return &domain.CafePresetDetail{
		CafeName:    j.CafeName,
		CoffeeName:  j.CoffeeName,
		TastingTags: j.TastingTags,
	}
}

func brewPresetJSONToDomain(j *brewPresetDetailJSON) *domain.BrewPresetDetail {
	if j == nil {
		return nil
	}
	return &domain.BrewPresetDetail{
		BeanName:     j.BeanName,
		BrewMethod:   domain.BrewMethod(j.BrewMethod),
		RecipeDetail: j.RecipeDetail,
		BrewSteps:    j.BrewSteps,
	}
}

