package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/service"
)

// ---------------------------------------------------------------------------
// Stub
// ---------------------------------------------------------------------------

type stubPresetService struct {
	createFunc func(ctx context.Context, userID string, req service.CreatePresetRequest) (domain.PresetFull, error)
	getFunc    func(ctx context.Context, userID, presetID string) (domain.PresetFull, error)
	listFunc   func(ctx context.Context, userID string) ([]domain.PresetFull, error)
	updateFunc func(ctx context.Context, userID, presetID string, req service.UpdatePresetRequest) (domain.PresetFull, error)
	deleteFunc func(ctx context.Context, userID, presetID string) error
	useFunc    func(ctx context.Context, userID, presetID string) error
}

func (s *stubPresetService) CreatePreset(ctx context.Context, userID string, req service.CreatePresetRequest) (domain.PresetFull, error) {
	return s.createFunc(ctx, userID, req)
}
func (s *stubPresetService) GetPreset(ctx context.Context, userID, presetID string) (domain.PresetFull, error) {
	return s.getFunc(ctx, userID, presetID)
}
func (s *stubPresetService) ListPresets(ctx context.Context, userID string) ([]domain.PresetFull, error) {
	return s.listFunc(ctx, userID)
}
func (s *stubPresetService) UpdatePreset(ctx context.Context, userID, presetID string, req service.UpdatePresetRequest) (domain.PresetFull, error) {
	return s.updateFunc(ctx, userID, presetID, req)
}
func (s *stubPresetService) DeletePreset(ctx context.Context, userID, presetID string) error {
	return s.deleteFunc(ctx, userID, presetID)
}
func (s *stubPresetService) UsePreset(ctx context.Context, userID, presetID string) error {
	return s.useFunc(ctx, userID, presetID)
}

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

func sampleCafePreset() domain.PresetFull {
	return domain.PresetFull{
		Preset: domain.Preset{
			ID:        "preset-1",
			UserID:    "user-1",
			Name:      "출근길 아메리카노",
			LogType:   domain.LogTypeCafe,
			CreatedAt: "2026-04-01T00:00:00Z",
			UpdatedAt: "2026-04-01T00:00:00Z",
		},
		Cafe: &domain.CafePresetDetail{
			CafeName:    "블루보틀",
			CoffeeName:  "싱글 오리진",
			TastingTags: []string{"fruity"},
		},
	}
}

// ---------------------------------------------------------------------------
// CreatePreset tests
// ---------------------------------------------------------------------------

func TestCreatePreset_Success(t *testing.T) {
	svc := &stubPresetService{
		createFunc: func(_ context.Context, userID string, req service.CreatePresetRequest) (domain.PresetFull, error) {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, domain.LogTypeCafe, req.LogType)
			return sampleCafePreset(), nil
		},
	}
	h := NewPresetHandler(svc)

	body := `{
		"name": "출근길 아메리카노",
		"log_type": "cafe",
		"cafe": {
			"cafe_name": "블루보틀",
			"coffee_name": "싱글 오리진",
			"tasting_tags": ["fruity"]
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presets", bytes.NewBufferString(body))
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()

	h.CreatePreset(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp presetResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "preset-1", resp.ID)
	assert.Equal(t, "cafe", resp.LogType)
	require.NotNil(t, resp.Cafe)
	assert.Equal(t, "블루보틀", resp.Cafe.CafeName)
}

func TestCreatePreset_InvalidJSON(t *testing.T) {
	svc := &stubPresetService{}
	h := NewPresetHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/presets", bytes.NewBufferString("{invalid"))
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()

	h.CreatePreset(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePreset_ValidationError(t *testing.T) {
	svc := &stubPresetService{
		createFunc: func(_ context.Context, _ string, _ service.CreatePresetRequest) (domain.PresetFull, error) {
			return domain.PresetFull{}, &service.ValidationError{Field: "name", Message: "필수값입니다"}
		},
	}
	h := NewPresetHandler(svc)

	body := `{"name": "", "log_type": "cafe"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/presets", bytes.NewBufferString(body))
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()

	h.CreatePreset(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// ListPresets tests
// ---------------------------------------------------------------------------

func TestListPresets_Success(t *testing.T) {
	svc := &stubPresetService{
		listFunc: func(_ context.Context, userID string) ([]domain.PresetFull, error) {
			return []domain.PresetFull{sampleCafePreset()}, nil
		},
	}
	h := NewPresetHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/presets", nil)
	req = withUserID(req, "user-1")
	w := httptest.NewRecorder()

	h.ListPresets(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp listPresetsResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "preset-1", resp.Items[0].ID)
}

// ---------------------------------------------------------------------------
// GetPreset tests
// ---------------------------------------------------------------------------

func TestGetPreset_Success(t *testing.T) {
	svc := &stubPresetService{
		getFunc: func(_ context.Context, userID, presetID string) (domain.PresetFull, error) {
			assert.Equal(t, "preset-1", presetID)
			return sampleCafePreset(), nil
		},
	}
	h := NewPresetHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/presets/preset-1", nil)
	req = withUserID(req, "user-1")
	req = withChiParam(req, "id", "preset-1")
	w := httptest.NewRecorder()

	h.GetPreset(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp presetResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "preset-1", resp.ID)
}

func TestGetPreset_NotFound(t *testing.T) {
	svc := &stubPresetService{
		getFunc: func(_ context.Context, _, _ string) (domain.PresetFull, error) {
			return domain.PresetFull{}, service.ErrNotFound
		},
	}
	h := NewPresetHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/presets/nonexistent", nil)
	req = withUserID(req, "user-1")
	req = withChiParam(req, "id", "nonexistent")
	w := httptest.NewRecorder()

	h.GetPreset(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// UpdatePreset tests
// ---------------------------------------------------------------------------

func TestUpdatePreset_Success(t *testing.T) {
	updated := sampleCafePreset()
	updated.Name = "바뀐 이름"

	svc := &stubPresetService{
		updateFunc: func(_ context.Context, userID, presetID string, req service.UpdatePresetRequest) (domain.PresetFull, error) {
			assert.Equal(t, "preset-1", presetID)
			return updated, nil
		},
	}
	h := NewPresetHandler(svc)

	body := `{
		"name": "바뀐 이름",
		"cafe": {
			"cafe_name": "블루보틀",
			"coffee_name": "싱글 오리진",
			"tasting_tags": ["fruity"]
		}
	}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/presets/preset-1", bytes.NewBufferString(body))
	req = withUserID(req, "user-1")
	req = withChiParam(req, "id", "preset-1")
	w := httptest.NewRecorder()

	h.UpdatePreset(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp presetResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "바뀐 이름", resp.Name)
}

// ---------------------------------------------------------------------------
// DeletePreset tests
// ---------------------------------------------------------------------------

func TestDeletePreset_Success(t *testing.T) {
	svc := &stubPresetService{
		deleteFunc: func(_ context.Context, userID, presetID string) error {
			assert.Equal(t, "preset-1", presetID)
			return nil
		},
	}
	h := NewPresetHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/presets/preset-1", nil)
	req = withUserID(req, "user-1")
	req = withChiParam(req, "id", "preset-1")
	w := httptest.NewRecorder()

	h.DeletePreset(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeletePreset_NotFound(t *testing.T) {
	svc := &stubPresetService{
		deleteFunc: func(_ context.Context, _, _ string) error {
			return service.ErrNotFound
		},
	}
	h := NewPresetHandler(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/presets/nonexistent", nil)
	req = withUserID(req, "user-1")
	req = withChiParam(req, "id", "nonexistent")
	w := httptest.NewRecorder()

	h.DeletePreset(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// UsePreset tests
// ---------------------------------------------------------------------------

func TestUsePreset_Success(t *testing.T) {
	svc := &stubPresetService{
		useFunc: func(_ context.Context, userID, presetID string) error {
			assert.Equal(t, "preset-1", presetID)
			return nil
		},
	}
	h := NewPresetHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/presets/preset-1/use", nil)
	req = withUserID(req, "user-1")
	req = withChiParam(req, "id", "preset-1")
	w := httptest.NewRecorder()

	h.UsePreset(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestUsePreset_NotFound(t *testing.T) {
	svc := &stubPresetService{
		useFunc: func(_ context.Context, _, _ string) error {
			return service.ErrNotFound
		},
	}
	h := NewPresetHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/presets/nonexistent/use", nil)
	req = withUserID(req, "user-1")
	req = withChiParam(req, "id", "nonexistent")
	w := httptest.NewRecorder()

	h.UsePreset(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
