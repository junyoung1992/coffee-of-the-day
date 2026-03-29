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

type stubLogRepository struct {
	createCalls []domain.CoffeeLogFull
	getCalls    []struct {
		logID  string
		userID string
	}
	listCalls   []repository.ListFilter
	updateCalls []domain.CoffeeLogFull
	deleteCalls []struct {
		logID  string
		userID string
	}

	createFunc func(ctx context.Context, log domain.CoffeeLogFull) error
	getFunc    func(ctx context.Context, logID, userID string) (domain.CoffeeLogFull, error)
	listFunc   func(ctx context.Context, userID string, filter repository.ListFilter) ([]domain.CoffeeLogFull, error)
	updateFunc func(ctx context.Context, log domain.CoffeeLogFull) error
	deleteFunc func(ctx context.Context, logID, userID string) error
}

func (s *stubLogRepository) CreateLog(ctx context.Context, log domain.CoffeeLogFull) error {
	s.createCalls = append(s.createCalls, log)
	if s.createFunc != nil {
		return s.createFunc(ctx, log)
	}
	return nil
}

func (s *stubLogRepository) GetLogByID(ctx context.Context, logID, userID string) (domain.CoffeeLogFull, error) {
	s.getCalls = append(s.getCalls, struct {
		logID  string
		userID string
	}{logID: logID, userID: userID})
	if s.getFunc != nil {
		return s.getFunc(ctx, logID, userID)
	}
	return domain.CoffeeLogFull{}, nil
}

func (s *stubLogRepository) ListLogs(ctx context.Context, userID string, filter repository.ListFilter) ([]domain.CoffeeLogFull, error) {
	s.listCalls = append(s.listCalls, filter)
	if s.listFunc != nil {
		return s.listFunc(ctx, userID, filter)
	}
	return nil, nil
}

func (s *stubLogRepository) UpdateLog(ctx context.Context, log domain.CoffeeLogFull) error {
	s.updateCalls = append(s.updateCalls, log)
	if s.updateFunc != nil {
		return s.updateFunc(ctx, log)
	}
	return nil
}

func (s *stubLogRepository) DeleteLog(ctx context.Context, logID, userID string) error {
	s.deleteCalls = append(s.deleteCalls, struct {
		logID  string
		userID string
	}{logID: logID, userID: userID})
	if s.deleteFunc != nil {
		return s.deleteFunc(ctx, logID, userID)
	}
	return nil
}

func TestCreateLog_ValidCafe_PersistsNormalizedLog(t *testing.T) {
	repo := &stubLogRepository{}
	svc := newTestService(repo)

	memo := "  여유로운 오후  "
	location := "  성수동  "
	origin := "  Ethiopia  "
	rating := 4.5
	req := CreateLogRequest{
		RecordedAt: " 2026-03-29T09:00:00Z ",
		Companions: []string{" Alice ", "", " Bob "},
		LogType:    domain.LogTypeCafe,
		Memo:       &memo,
		Cafe: &domain.CafeDetail{
			CafeName:    "  블루보틀  ",
			Location:    &location,
			CoffeeName:  "  플랫화이트  ",
			BeanOrigin:  &origin,
			TastingTags: []string{" sweet ", " ", " berry "},
			Rating:      &rating,
		},
	}

	got, err := svc.CreateLog(context.Background(), " user-1 ", req)
	require.NoError(t, err)

	require.Len(t, repo.createCalls, 1)
	saved := repo.createCalls[0]
	assert.Equal(t, got, saved)
	assert.Equal(t, "generated-id", got.ID)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, "2026-03-29T09:00:00Z", got.RecordedAt)
	assert.Equal(t, []string{"Alice", "Bob"}, got.Companions)
	assert.Equal(t, "2026-03-29T12:34:56Z", got.CreatedAt)
	assert.Equal(t, "2026-03-29T12:34:56Z", got.UpdatedAt)
	require.NotNil(t, got.Cafe)
	assert.Equal(t, "블루보틀", got.Cafe.CafeName)
	assert.Equal(t, "플랫화이트", got.Cafe.CoffeeName)
	assert.Equal(t, []string{"sweet", "berry"}, got.Cafe.TastingTags)
	require.NotNil(t, got.Memo)
	assert.Equal(t, "여유로운 오후", *got.Memo)
}

func TestCreateLog_InvalidDetailCombination_ReturnsValidationError(t *testing.T) {
	repo := &stubLogRepository{}
	svc := newTestService(repo)

	_, err := svc.CreateLog(context.Background(), "user-1", CreateLogRequest{
		RecordedAt: "2026-03-29T09:00:00Z",
		LogType:    domain.LogTypeBrew,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidArgument)
	assert.Equal(t, "brew: brew 로그에는 brew 상세가 필요합니다", err.Error())
	assert.Empty(t, repo.createCalls)
}

func TestGetLog_MapsRepositoryNotFound(t *testing.T) {
	repo := &stubLogRepository{
		getFunc: func(ctx context.Context, logID, userID string) (domain.CoffeeLogFull, error) {
			return domain.CoffeeLogFull{}, repository.ErrNotFound
		},
	}
	svc := newTestService(repo)

	_, err := svc.GetLog(context.Background(), "user-1", "log-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListLogs_CreatesNextCursorFromTrimmedPage(t *testing.T) {
	repo := &stubLogRepository{
		listFunc: func(ctx context.Context, userID string, filter repository.ListFilter) ([]domain.CoffeeLogFull, error) {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, 3, filter.Limit)
			return []domain.CoffeeLogFull{
				{CoffeeLog: domain.CoffeeLog{ID: "log-3", RecordedAt: "2026-03-30", LogType: domain.LogTypeCafe}},
				{CoffeeLog: domain.CoffeeLog{ID: "log-2", RecordedAt: "2026-03-29", LogType: domain.LogTypeCafe}},
				{CoffeeLog: domain.CoffeeLog{ID: "log-1", RecordedAt: "2026-03-28", LogType: domain.LogTypeCafe}},
			}, nil
		},
	}
	svc := newTestService(repo)

	result, err := svc.ListLogs(context.Background(), "user-1", ListLogsFilter{Limit: 2})
	require.NoError(t, err)

	assert.Len(t, result.Items, 2)
	assert.True(t, result.HasNext)
	require.NotNil(t, result.NextCursor)
	decoded, err := repository.DecodeCursor(*result.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "recorded_at", decoded.SortBy)
	assert.Equal(t, "desc", decoded.Order)
	assert.Equal(t, "2026-03-29", decoded.SortValue)
	assert.Equal(t, "log-2", decoded.ID)
}

func TestListLogs_InvalidCursor_ReturnsValidationError(t *testing.T) {
	repo := &stubLogRepository{}
	svc := newTestService(repo)
	invalidCursor := "not-base64"

	_, err := svc.ListLogs(context.Background(), "user-1", ListLogsFilter{
		Limit:  20,
		Cursor: &invalidCursor,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidArgument)
	assert.Empty(t, repo.listCalls)
}

func TestUpdateLog_UsesExistingOwnershipAndCreatedAt(t *testing.T) {
	existing := domain.CoffeeLogFull{
		CoffeeLog: domain.CoffeeLog{
			ID:         "log-1",
			UserID:     "user-1",
			RecordedAt: "2026-03-20",
			Companions: []string{"Alice"},
			LogType:    domain.LogTypeBrew,
			CreatedAt:  "2026-03-20T00:00:00Z",
			UpdatedAt:  "2026-03-20T00:00:00Z",
		},
		Brew: &domain.BrewDetail{
			BeanName:   "기존 원두",
			BrewMethod: domain.BrewMethodPourOver,
		},
	}

	repo := &stubLogRepository{
		getFunc: func(ctx context.Context, logID, userID string) (domain.CoffeeLogFull, error) {
			return existing, nil
		},
	}
	svc := newTestService(repo)

	beanOrigin := " Colombia "
	brewTime := 180
	req := UpdateLogRequest{
		RecordedAt: "2026-03-29T09:30:00Z",
		Companions: []string{" Charlie ", ""},
		Memo:       strPtr("  더 깔끔했다  "),
		Brew: &domain.BrewDetail{
			BeanName:      "  새 원두  ",
			BeanOrigin:    &beanOrigin,
			BrewMethod:    domain.BrewMethodImmersion,
			BrewTimeSec:   &brewTime,
			TastingTags:   []string{" floral "},
			BrewSteps:     []string{" bloom ", " pour "},
			CoffeeAmountG: floatPtr(18),
		},
	}

	got, err := svc.UpdateLog(context.Background(), "user-1", "log-1", req)
	require.NoError(t, err)

	require.Len(t, repo.updateCalls, 1)
	saved := repo.updateCalls[0]
	assert.Equal(t, got, saved)
	assert.Equal(t, "log-1", saved.ID)
	assert.Equal(t, "user-1", saved.UserID)
	assert.Equal(t, "2026-03-20T00:00:00Z", saved.CreatedAt)
	assert.Equal(t, "2026-03-29T12:34:56Z", saved.UpdatedAt)
	assert.Equal(t, domain.LogTypeBrew, saved.LogType)
	require.NotNil(t, saved.Brew)
	assert.Equal(t, "새 원두", saved.Brew.BeanName)
	assert.Equal(t, domain.BrewMethodImmersion, saved.Brew.BrewMethod)
	assert.Equal(t, []string{"Charlie"}, saved.Companions)
}

func TestUpdateLog_RejectsLogTypeChange(t *testing.T) {
	existing := domain.CoffeeLogFull{
		CoffeeLog: domain.CoffeeLog{
			ID:      "log-1",
			UserID:  "user-1",
			LogType: domain.LogTypeCafe,
		},
		Cafe: &domain.CafeDetail{
			CafeName:   "카페",
			CoffeeName: "아메리카노",
		},
	}

	repo := &stubLogRepository{
		getFunc: func(ctx context.Context, logID, userID string) (domain.CoffeeLogFull, error) {
			return existing, nil
		},
	}
	svc := newTestService(repo)

	_, err := svc.UpdateLog(context.Background(), "user-1", "log-1", UpdateLogRequest{
		RecordedAt: "2026-03-29T09:00:00Z",
		LogType:    domain.LogTypeBrew,
		Brew: &domain.BrewDetail{
			BeanName:   "원두",
			BrewMethod: domain.BrewMethodPourOver,
		},
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidArgument)
	assert.Empty(t, repo.updateCalls)
}

func TestDeleteLog_MapsRepositoryNotFound(t *testing.T) {
	repo := &stubLogRepository{
		deleteFunc: func(ctx context.Context, logID, userID string) error {
			return repository.ErrNotFound
		},
	}
	svc := newTestService(repo)

	err := svc.DeleteLog(context.Background(), "user-1", "log-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func newTestService(repo repository.LogRepository) *DefaultLogService {
	return &DefaultLogService{
		repo: repo,
		now: func() time.Time {
			return time.Date(2026, time.March, 29, 12, 34, 56, 0, time.UTC)
		},
		newID: func() (string, error) {
			return "generated-id", nil
		},
	}
}

func strPtr(value string) *string {
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

func TestValidationError_UnwrapsToInvalidArgument(t *testing.T) {
	err := newValidationError("field", "문제")
	assert.True(t, errors.Is(err, ErrInvalidArgument))
}
