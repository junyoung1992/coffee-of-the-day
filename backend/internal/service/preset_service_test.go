package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Stub repository
// ---------------------------------------------------------------------------

type stubPresetRepository struct {
	createCalls        []domain.PresetFull
	getCalls           []struct{ presetID, userID string }
	listCalls          []string
	updateCalls        []domain.PresetFull
	deleteCalls        []struct{ presetID, userID string }
	updateLastUsedCalls []struct{ presetID, userID, usedAt string }

	createFunc        func(ctx context.Context, preset domain.PresetFull) error
	getFunc           func(ctx context.Context, presetID, userID string) (domain.PresetFull, error)
	listFunc          func(ctx context.Context, userID string) ([]domain.PresetFull, error)
	updateFunc        func(ctx context.Context, preset domain.PresetFull) error
	deleteFunc        func(ctx context.Context, presetID, userID string) error
	updateLastUsedFunc func(ctx context.Context, presetID, userID, usedAt string) error
}

func (s *stubPresetRepository) CreatePreset(ctx context.Context, preset domain.PresetFull) error {
	s.createCalls = append(s.createCalls, preset)
	if s.createFunc != nil {
		return s.createFunc(ctx, preset)
	}
	return nil
}

func (s *stubPresetRepository) GetPresetByID(ctx context.Context, presetID, userID string) (domain.PresetFull, error) {
	s.getCalls = append(s.getCalls, struct{ presetID, userID string }{presetID, userID})
	if s.getFunc != nil {
		return s.getFunc(ctx, presetID, userID)
	}
	return domain.PresetFull{}, nil
}

func (s *stubPresetRepository) ListPresets(ctx context.Context, userID string) ([]domain.PresetFull, error) {
	s.listCalls = append(s.listCalls, userID)
	if s.listFunc != nil {
		return s.listFunc(ctx, userID)
	}
	return nil, nil
}

func (s *stubPresetRepository) UpdatePreset(ctx context.Context, preset domain.PresetFull) error {
	s.updateCalls = append(s.updateCalls, preset)
	if s.updateFunc != nil {
		return s.updateFunc(ctx, preset)
	}
	return nil
}

func (s *stubPresetRepository) DeletePreset(ctx context.Context, presetID, userID string) error {
	s.deleteCalls = append(s.deleteCalls, struct{ presetID, userID string }{presetID, userID})
	if s.deleteFunc != nil {
		return s.deleteFunc(ctx, presetID, userID)
	}
	return nil
}

func (s *stubPresetRepository) UpdateLastUsedAt(ctx context.Context, presetID, userID, usedAt string) error {
	s.updateLastUsedCalls = append(s.updateLastUsedCalls, struct{ presetID, userID, usedAt string }{presetID, userID, usedAt})
	if s.updateLastUsedFunc != nil {
		return s.updateLastUsedFunc(ctx, presetID, userID, usedAt)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestPresetService(repo repository.PresetRepository) *DefaultPresetService {
	return &DefaultPresetService{
		repo: repo,
		now: func() time.Time {
			return time.Date(2026, time.April, 4, 12, 0, 0, 0, time.UTC)
		},
		newID: func() (string, error) {
			return "generated-preset-id", nil
		},
	}
}

func validCafePresetRequest() CreatePresetRequest {
	return CreatePresetRequest{
		Name:    "출근길 아메리카노",
		LogType: domain.LogTypeCafe,
		Cafe: &domain.CafePresetDetail{
			CafeName:    "블루보틀",
			CoffeeName:  "싱글 오리진",
			TastingTags: []string{"fruity"},
		},
	}
}

func validBrewPresetRequest() CreatePresetRequest {
	return CreatePresetRequest{
		Name:    "주말 핸드드립",
		LogType: domain.LogTypeBrew,
		Brew: &domain.BrewPresetDetail{
			BeanName:   "에티오피아 예가체프",
			BrewMethod: domain.BrewMethodPourOver,
			BrewSteps:  []string{"뜸들이기"},
		},
	}
}

// ---------------------------------------------------------------------------
// CreatePreset tests
// ---------------------------------------------------------------------------

func TestCreatePreset_CafeHappyPath(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	preset, err := svc.CreatePreset(context.Background(), "user-1", validCafePresetRequest())
	require.NoError(t, err)

	assert.Equal(t, "generated-preset-id", preset.ID)
	assert.Equal(t, "user-1", preset.UserID)
	assert.Equal(t, "출근길 아메리카노", preset.Name)
	assert.Equal(t, domain.LogTypeCafe, preset.LogType)
	assert.Equal(t, "2026-04-04T12:00:00Z", preset.CreatedAt)
	require.NotNil(t, preset.Cafe)
	assert.Equal(t, "블루보틀", preset.Cafe.CafeName)
	assert.Nil(t, preset.Brew)
	require.Len(t, repo.createCalls, 1)
}

func TestCreatePreset_BrewHappyPath(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	preset, err := svc.CreatePreset(context.Background(), "user-1", validBrewPresetRequest())
	require.NoError(t, err)

	assert.Equal(t, domain.LogTypeBrew, preset.LogType)
	require.NotNil(t, preset.Brew)
	assert.Equal(t, "에티오피아 예가체프", preset.Brew.BeanName)
	assert.Equal(t, domain.BrewMethodPourOver, preset.Brew.BrewMethod)
	assert.Nil(t, preset.Cafe)
}

func TestCreatePreset_TrimsName(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	req := validCafePresetRequest()
	req.Name = "  공백 이름  "
	preset, err := svc.CreatePreset(context.Background(), "user-1", req)
	require.NoError(t, err)
	assert.Equal(t, "공백 이름", preset.Name)
}

func TestCreatePreset_EmptyNameFails(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	req := validCafePresetRequest()
	req.Name = "   "

	_, err := svc.CreatePreset(context.Background(), "user-1", req)
	require.Error(t, err)

	var ve *ValidationError
	require.True(t, errors.As(err, &ve))
	assert.Equal(t, "name", ve.Field)
}

func TestCreatePreset_InvalidLogType(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	req := validCafePresetRequest()
	req.LogType = "invalid"

	_, err := svc.CreatePreset(context.Background(), "user-1", req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidArgument))
}

func TestCreatePreset_CafeMissingDetail(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	req := CreatePresetRequest{
		Name:    "테스트",
		LogType: domain.LogTypeCafe,
		Cafe:    nil,
	}

	_, err := svc.CreatePreset(context.Background(), "user-1", req)
	require.Error(t, err)

	var ve *ValidationError
	require.True(t, errors.As(err, &ve))
	assert.Equal(t, "cafe", ve.Field)
}

func TestCreatePreset_BrewMissingDetail(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	req := CreatePresetRequest{
		Name:    "테스트",
		LogType: domain.LogTypeBrew,
		Brew:    nil,
	}

	_, err := svc.CreatePreset(context.Background(), "user-1", req)
	require.Error(t, err)

	var ve *ValidationError
	require.True(t, errors.As(err, &ve))
	assert.Equal(t, "brew", ve.Field)
}

func TestCreatePreset_BrewInvalidMethod(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	req := validBrewPresetRequest()
	req.Brew.BrewMethod = "invalid_method"

	_, err := svc.CreatePreset(context.Background(), "user-1", req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidArgument))
}

func TestCreatePreset_EmptyUserID(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	_, err := svc.CreatePreset(context.Background(), "", validCafePresetRequest())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidArgument))
}

// ---------------------------------------------------------------------------
// UpdatePreset tests
// ---------------------------------------------------------------------------

func TestUpdatePreset_HappyPath(t *testing.T) {
	existing := domain.PresetFull{
		Preset: domain.Preset{
			ID:      "preset-1",
			UserID:  "user-1",
			Name:    "원래 이름",
			LogType: domain.LogTypeCafe,
		},
		Cafe: &domain.CafePresetDetail{
			CafeName:    "블루보틀",
			CoffeeName:  "오리진",
			TastingTags: []string{},
		},
	}

	repo := &stubPresetRepository{
		getFunc: func(ctx context.Context, presetID, userID string) (domain.PresetFull, error) {
			return existing, nil
		},
	}
	svc := newTestPresetService(repo)

	updated, err := svc.UpdatePreset(context.Background(), "user-1", "preset-1", UpdatePresetRequest{
		Name: "바뀐 이름",
		Cafe: &domain.CafePresetDetail{
			CafeName:    "스타벅스",
			CoffeeName:  "아메리카노",
			TastingTags: []string{"bold"},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "바뀐 이름", updated.Name)
	assert.Equal(t, "스타벅스", updated.Cafe.CafeName)
	assert.Equal(t, "2026-04-04T12:00:00Z", updated.UpdatedAt)
	require.Len(t, repo.updateCalls, 1)
}

func TestUpdatePreset_NotFoundReturnsError(t *testing.T) {
	repo := &stubPresetRepository{
		getFunc: func(ctx context.Context, presetID, userID string) (domain.PresetFull, error) {
			return domain.PresetFull{}, repository.ErrNotFound
		},
	}
	svc := newTestPresetService(repo)

	_, err := svc.UpdatePreset(context.Background(), "user-1", "nonexistent", UpdatePresetRequest{
		Name: "테스트",
		Cafe: &domain.CafePresetDetail{CafeName: "a", CoffeeName: "b"},
	})
	assert.ErrorIs(t, err, ErrNotFound)
}

// ---------------------------------------------------------------------------
// UsePreset tests
// ---------------------------------------------------------------------------

func TestUsePreset_HappyPath(t *testing.T) {
	repo := &stubPresetRepository{
		getFunc: func(ctx context.Context, presetID, userID string) (domain.PresetFull, error) {
			return domain.PresetFull{}, nil
		},
	}
	svc := newTestPresetService(repo)

	err := svc.UsePreset(context.Background(), "user-1", "preset-1")
	require.NoError(t, err)

	require.Len(t, repo.updateLastUsedCalls, 1)
	assert.Equal(t, "preset-1", repo.updateLastUsedCalls[0].presetID)
	assert.Equal(t, "2026-04-04T12:00:00Z", repo.updateLastUsedCalls[0].usedAt)
}

func TestUsePreset_NotFound(t *testing.T) {
	repo := &stubPresetRepository{
		getFunc: func(ctx context.Context, presetID, userID string) (domain.PresetFull, error) {
			return domain.PresetFull{}, repository.ErrNotFound
		},
	}
	svc := newTestPresetService(repo)

	err := svc.UsePreset(context.Background(), "user-1", "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ---------------------------------------------------------------------------
// DeletePreset tests
// ---------------------------------------------------------------------------

func TestDeletePreset_HappyPath(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	err := svc.DeletePreset(context.Background(), "user-1", "preset-1")
	require.NoError(t, err)
	require.Len(t, repo.deleteCalls, 1)
	assert.Equal(t, "preset-1", repo.deleteCalls[0].presetID)
}

func TestDeletePreset_NotFound(t *testing.T) {
	repo := &stubPresetRepository{
		deleteFunc: func(ctx context.Context, presetID, userID string) error {
			return repository.ErrNotFound
		},
	}
	svc := newTestPresetService(repo)

	err := svc.DeletePreset(context.Background(), "user-1", "nonexistent")
	assert.ErrorIs(t, err, ErrNotFound)
}

// ---------------------------------------------------------------------------
// Normalization edge cases
// ---------------------------------------------------------------------------

func TestCreatePreset_NormalizesStringSlice(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	req := validCafePresetRequest()
	req.Cafe.TastingTags = []string{"  fruity  ", "", "  floral"}

	preset, err := svc.CreatePreset(context.Background(), "user-1", req)
	require.NoError(t, err)
	assert.Equal(t, []string{"fruity", "floral"}, preset.Cafe.TastingTags)
}

func TestCreatePreset_TrimsRecipeDetail(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	req := validBrewPresetRequest()
	detail := "  V60 레시피  "
	req.Brew.RecipeDetail = &detail

	preset, err := svc.CreatePreset(context.Background(), "user-1", req)
	require.NoError(t, err)
	require.NotNil(t, preset.Brew.RecipeDetail)
	assert.Equal(t, "V60 레시피", *preset.Brew.RecipeDetail)
}

func TestCreatePreset_EmptyRecipeDetailBecomesNil(t *testing.T) {
	repo := &stubPresetRepository{}
	svc := newTestPresetService(repo)

	req := validBrewPresetRequest()
	empty := "   "
	req.Brew.RecipeDetail = &empty

	preset, err := svc.CreatePreset(context.Background(), "user-1", req)
	require.NoError(t, err)
	assert.Nil(t, preset.Brew.RecipeDetail)
}
