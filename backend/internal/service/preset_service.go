package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/repository"
)

// CreatePresetRequest는 프리셋 생성 요청 파라미터이다.
type CreatePresetRequest struct {
	Name    string
	LogType domain.LogType
	Cafe    *domain.CafePresetDetail
	Brew    *domain.BrewPresetDetail
}

// UpdatePresetRequest는 프리셋 수정 요청 파라미터이다.
type UpdatePresetRequest struct {
	Name string
	Cafe *domain.CafePresetDetail
	Brew *domain.BrewPresetDetail
}

// PresetService는 프리셋 비즈니스 로직 인터페이스이다.
type PresetService interface {
	CreatePreset(ctx context.Context, userID string, req CreatePresetRequest) (domain.PresetFull, error)
	GetPreset(ctx context.Context, userID, presetID string) (domain.PresetFull, error)
	ListPresets(ctx context.Context, userID string) ([]domain.PresetFull, error)
	UpdatePreset(ctx context.Context, userID, presetID string, req UpdatePresetRequest) (domain.PresetFull, error)
	DeletePreset(ctx context.Context, userID, presetID string) error
	UsePreset(ctx context.Context, userID, presetID string) error
}

// DefaultPresetService는 PresetService의 기본 구현체이다.
type DefaultPresetService struct {
	repo  repository.PresetRepository
	now   func() time.Time
	newID func() (string, error)
}

// NewPresetService는 새 DefaultPresetService를 생성한다.
func NewPresetService(repo repository.PresetRepository) *DefaultPresetService {
	return &DefaultPresetService{
		repo:  repo,
		now:   time.Now,
		newID: newUUID,
	}
}

func (s *DefaultPresetService) CreatePreset(ctx context.Context, userID string, req CreatePresetRequest) (domain.PresetFull, error) {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return domain.PresetFull{}, err
	}

	normalizedReq, err := normalizeCreatePresetRequest(req)
	if err != nil {
		return domain.PresetFull{}, err
	}

	id, err := s.newID()
	if err != nil {
		return domain.PresetFull{}, fmt.Errorf("create preset: generate id: %w", err)
	}

	now := s.now().UTC().Format(time.RFC3339)
	preset := domain.PresetFull{
		Preset: domain.Preset{
			ID:        id,
			UserID:    normalizedUserID,
			Name:      normalizedReq.Name,
			LogType:   normalizedReq.LogType,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Cafe: normalizedReq.Cafe,
		Brew: normalizedReq.Brew,
	}

	if err := s.repo.CreatePreset(ctx, preset); err != nil {
		return domain.PresetFull{}, fmt.Errorf("create preset: %w", err)
	}

	return preset, nil
}

func (s *DefaultPresetService) GetPreset(ctx context.Context, userID, presetID string) (domain.PresetFull, error) {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return domain.PresetFull{}, err
	}
	normalizedPresetID, err := validateIdentifier("preset_id", presetID)
	if err != nil {
		return domain.PresetFull{}, err
	}

	preset, err := s.repo.GetPresetByID(ctx, normalizedPresetID, normalizedUserID)
	if err != nil {
		return domain.PresetFull{}, mapRepositoryError("get preset", err)
	}
	return preset, nil
}

func (s *DefaultPresetService) ListPresets(ctx context.Context, userID string) ([]domain.PresetFull, error) {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.ListPresets(ctx, normalizedUserID)
	if err != nil {
		return nil, fmt.Errorf("list presets: %w", err)
	}
	if items == nil {
		items = []domain.PresetFull{}
	}
	return items, nil
}

func (s *DefaultPresetService) UpdatePreset(ctx context.Context, userID, presetID string, req UpdatePresetRequest) (domain.PresetFull, error) {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return domain.PresetFull{}, err
	}
	normalizedPresetID, err := validateIdentifier("preset_id", presetID)
	if err != nil {
		return domain.PresetFull{}, err
	}

	// 기존 프리셋 조회 (소유권 확인 + log_type 확인)
	existing, err := s.repo.GetPresetByID(ctx, normalizedPresetID, normalizedUserID)
	if err != nil {
		return domain.PresetFull{}, mapRepositoryError("update preset", err)
	}

	normalizedReq, err := normalizeUpdatePresetRequest(existing.LogType, req)
	if err != nil {
		return domain.PresetFull{}, err
	}

	now := s.now().UTC().Format(time.RFC3339)
	existing.Name = normalizedReq.Name
	existing.Cafe = normalizedReq.Cafe
	existing.Brew = normalizedReq.Brew
	existing.UpdatedAt = now

	if err := s.repo.UpdatePreset(ctx, existing); err != nil {
		return domain.PresetFull{}, fmt.Errorf("update preset: %w", err)
	}
	return existing, nil
}

func (s *DefaultPresetService) DeletePreset(ctx context.Context, userID, presetID string) error {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return err
	}
	normalizedPresetID, err := validateIdentifier("preset_id", presetID)
	if err != nil {
		return err
	}

	if err := s.repo.DeletePreset(ctx, normalizedPresetID, normalizedUserID); err != nil {
		return mapRepositoryError("delete preset", err)
	}
	return nil
}

func (s *DefaultPresetService) UsePreset(ctx context.Context, userID, presetID string) error {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return err
	}
	normalizedPresetID, err := validateIdentifier("preset_id", presetID)
	if err != nil {
		return err
	}

	// 프리셋 존재 및 소유권 확인
	if _, err := s.repo.GetPresetByID(ctx, normalizedPresetID, normalizedUserID); err != nil {
		return mapRepositoryError("use preset", err)
	}

	now := s.now().UTC().Format(time.RFC3339)
	if err := s.repo.UpdateLastUsedAt(ctx, normalizedPresetID, normalizedUserID, now); err != nil {
		return fmt.Errorf("use preset: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 검증/정규화 함수
// ---------------------------------------------------------------------------

func normalizeCreatePresetRequest(req CreatePresetRequest) (CreatePresetRequest, error) {
	name, err := validateRequiredString("name", req.Name)
	if err != nil {
		return CreatePresetRequest{}, err
	}
	req.Name = name

	logType, err := validateLogType("log_type", req.LogType)
	if err != nil {
		return CreatePresetRequest{}, err
	}
	req.LogType = logType

	switch logType {
	case domain.LogTypeCafe:
		if req.Cafe == nil {
			return CreatePresetRequest{}, newValidationError("cafe", "카페 프리셋에는 cafe 데이터가 필요합니다")
		}
		normalized, err := normalizeCafePresetDetail(req.Cafe)
		if err != nil {
			return CreatePresetRequest{}, err
		}
		req.Cafe = normalized
		req.Brew = nil
	case domain.LogTypeBrew:
		if req.Brew == nil {
			return CreatePresetRequest{}, newValidationError("brew", "홈브루 프리셋에는 brew 데이터가 필요합니다")
		}
		normalized, err := normalizeBrewPresetDetail(req.Brew)
		if err != nil {
			return CreatePresetRequest{}, err
		}
		req.Brew = normalized
		req.Cafe = nil
	}

	return req, nil
}

func normalizeUpdatePresetRequest(logType domain.LogType, req UpdatePresetRequest) (UpdatePresetRequest, error) {
	name, err := validateRequiredString("name", req.Name)
	if err != nil {
		return UpdatePresetRequest{}, err
	}
	req.Name = name

	switch logType {
	case domain.LogTypeCafe:
		if req.Cafe == nil {
			return UpdatePresetRequest{}, newValidationError("cafe", "카페 프리셋에는 cafe 데이터가 필요합니다")
		}
		normalized, err := normalizeCafePresetDetail(req.Cafe)
		if err != nil {
			return UpdatePresetRequest{}, err
		}
		req.Cafe = normalized
		req.Brew = nil
	case domain.LogTypeBrew:
		if req.Brew == nil {
			return UpdatePresetRequest{}, newValidationError("brew", "홈브루 프리셋에는 brew 데이터가 필요합니다")
		}
		normalized, err := normalizeBrewPresetDetail(req.Brew)
		if err != nil {
			return UpdatePresetRequest{}, err
		}
		req.Brew = normalized
		req.Cafe = nil
	}

	return req, nil
}

func normalizeCafePresetDetail(c *domain.CafePresetDetail) (*domain.CafePresetDetail, error) {
	cafeName, err := validateRequiredString("cafe.cafe_name", c.CafeName)
	if err != nil {
		return nil, err
	}
	coffeeName, err := validateRequiredString("cafe.coffee_name", c.CoffeeName)
	if err != nil {
		return nil, err
	}
	return &domain.CafePresetDetail{
		CafeName:    cafeName,
		CoffeeName:  coffeeName,
		TastingTags: normalizeStringSlice(c.TastingTags),
	}, nil
}

func normalizeBrewPresetDetail(b *domain.BrewPresetDetail) (*domain.BrewPresetDetail, error) {
	beanName, err := validateRequiredString("brew.bean_name", b.BeanName)
	if err != nil {
		return nil, err
	}
	brewMethod, err := validateBrewMethod(b.BrewMethod)
	if err != nil {
		return nil, err
	}

	var recipeDetail *string
	if b.RecipeDetail != nil {
		trimmed := strings.TrimSpace(*b.RecipeDetail)
		if trimmed != "" {
			recipeDetail = &trimmed
		}
	}

	return &domain.BrewPresetDetail{
		BeanName:     beanName,
		BrewMethod:   brewMethod,
		RecipeDetail: recipeDetail,
		BrewSteps:    normalizeStringSlice(b.BrewSteps),
	}, nil
}
