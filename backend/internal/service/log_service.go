package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/repository"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

var (
	ErrNotFound        = errors.New("log not found")
	ErrInvalidArgument = errors.New("invalid argument")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (e *ValidationError) Unwrap() error {
	return ErrInvalidArgument
}

type CreateLogRequest struct {
	RecordedAt string
	Companions []string
	LogType    domain.LogType
	Memo       *string
	Cafe       *domain.CafeDetail
	Brew       *domain.BrewDetail
}

type UpdateLogRequest struct {
	RecordedAt string
	Companions []string
	LogType    domain.LogType
	Memo       *string
	Cafe       *domain.CafeDetail
	Brew       *domain.BrewDetail
}

// defaultTimezone은 날짜 필터의 기본 타임존이다.
// 향후 사용자별 타임존을 지원하려면 ListLogsFilter.Timezone 필드에 값을 채우면 된다.
const defaultTimezone = "Asia/Seoul"

type ListLogsFilter struct {
	LogType  *domain.LogType
	DateFrom *string
	DateTo   *string
	Cursor   *string
	Limit    int
	// Timezone은 YYYY-MM-DD 날짜 필터를 UTC 경계로 변환할 때 사용하는 타임존이다.
	// 빈 문자열이면 defaultTimezone(Asia/Seoul)을 사용한다.
	Timezone string
}

type ListLogsResult struct {
	Items      []domain.CoffeeLogFull
	NextCursor *string
	HasNext    bool
}

type LogService interface {
	CreateLog(ctx context.Context, userID string, req CreateLogRequest) (domain.CoffeeLogFull, error)
	GetLog(ctx context.Context, userID, logID string) (domain.CoffeeLogFull, error)
	ListLogs(ctx context.Context, userID string, filter ListLogsFilter) (ListLogsResult, error)
	UpdateLog(ctx context.Context, userID, logID string, req UpdateLogRequest) (domain.CoffeeLogFull, error)
	DeleteLog(ctx context.Context, userID, logID string) error
}

type DefaultLogService struct {
	repo  repository.LogRepository
	now   func() time.Time
	newID func() (string, error)
}

func NewLogService(repo repository.LogRepository) *DefaultLogService {
	return &DefaultLogService{
		repo:  repo,
		now:   time.Now,
		newID: newUUID,
	}
}

func (s *DefaultLogService) CreateLog(ctx context.Context, userID string, req CreateLogRequest) (domain.CoffeeLogFull, error) {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return domain.CoffeeLogFull{}, err
	}

	normalizedReq, err := normalizeCreateRequest(req)
	if err != nil {
		return domain.CoffeeLogFull{}, err
	}

	id, err := s.newID()
	if err != nil {
		return domain.CoffeeLogFull{}, fmt.Errorf("create log: generate id: %w", err)
	}

	now := s.now().UTC().Format(time.RFC3339)
	log := domain.CoffeeLogFull{
		CoffeeLog: domain.CoffeeLog{
			ID:         id,
			UserID:     normalizedUserID,
			RecordedAt: normalizedReq.RecordedAt,
			Companions: normalizedReq.Companions,
			LogType:    normalizedReq.LogType,
			Memo:       normalizedReq.Memo,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		Cafe: normalizedReq.Cafe,
		Brew: normalizedReq.Brew,
	}

	if err := s.repo.CreateLog(ctx, log); err != nil {
		return domain.CoffeeLogFull{}, fmt.Errorf("create log: %w", err)
	}

	return log, nil
}

func (s *DefaultLogService) GetLog(ctx context.Context, userID, logID string) (domain.CoffeeLogFull, error) {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return domain.CoffeeLogFull{}, err
	}
	normalizedLogID, err := validateIdentifier("log_id", logID)
	if err != nil {
		return domain.CoffeeLogFull{}, err
	}

	log, err := s.repo.GetLogByID(ctx, normalizedLogID, normalizedUserID)
	if err != nil {
		return domain.CoffeeLogFull{}, mapRepositoryError("get log", err)
	}

	return log, nil
}

func (s *DefaultLogService) ListLogs(ctx context.Context, userID string, filter ListLogsFilter) (ListLogsResult, error) {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return ListLogsResult{}, err
	}

	repoFilter, limit, err := normalizeListFilter(filter)
	if err != nil {
		return ListLogsResult{}, err
	}

	repoFilter.Limit = limit + 1

	items, err := s.repo.ListLogs(ctx, normalizedUserID, repoFilter)
	if err != nil {
		return ListLogsResult{}, fmt.Errorf("list logs: %w", err)
	}
	if items == nil {
		items = []domain.CoffeeLogFull{}
	}

	result := ListLogsResult{
		Items: items,
	}

	if len(items) <= limit {
		return result, nil
	}

	result.HasNext = true
	result.Items = items[:limit]

	last := result.Items[len(result.Items)-1]
	nextCursor := repository.EncodeCursor(repository.Cursor{
		SortBy:    "recorded_at",
		Order:     "desc",
		SortValue: last.RecordedAt,
		ID:        last.ID,
	})
	result.NextCursor = &nextCursor

	return result, nil
}

func (s *DefaultLogService) UpdateLog(ctx context.Context, userID, logID string, req UpdateLogRequest) (domain.CoffeeLogFull, error) {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return domain.CoffeeLogFull{}, err
	}
	normalizedLogID, err := validateIdentifier("log_id", logID)
	if err != nil {
		return domain.CoffeeLogFull{}, err
	}

	existing, err := s.repo.GetLogByID(ctx, normalizedLogID, normalizedUserID)
	if err != nil {
		return domain.CoffeeLogFull{}, mapRepositoryError("update log", err)
	}

	normalizedReq, err := normalizeUpdateRequest(req, existing.LogType)
	if err != nil {
		return domain.CoffeeLogFull{}, err
	}

	updated := domain.CoffeeLogFull{
		CoffeeLog: domain.CoffeeLog{
			ID:         existing.ID,
			UserID:     existing.UserID,
			RecordedAt: normalizedReq.RecordedAt,
			Companions: normalizedReq.Companions,
			LogType:    existing.LogType,
			Memo:       normalizedReq.Memo,
			CreatedAt:  existing.CreatedAt,
			UpdatedAt:  s.now().UTC().Format(time.RFC3339),
		},
		Cafe: normalizedReq.Cafe,
		Brew: normalizedReq.Brew,
	}

	if err := s.repo.UpdateLog(ctx, updated); err != nil {
		return domain.CoffeeLogFull{}, mapRepositoryError("update log", err)
	}

	return updated, nil
}

func (s *DefaultLogService) DeleteLog(ctx context.Context, userID, logID string) error {
	normalizedUserID, err := validateIdentifier("user_id", userID)
	if err != nil {
		return err
	}
	normalizedLogID, err := validateIdentifier("log_id", logID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteLog(ctx, normalizedLogID, normalizedUserID); err != nil {
		return mapRepositoryError("delete log", err)
	}

	return nil
}

func normalizeCreateRequest(req CreateLogRequest) (CreateLogRequest, error) {
	logType, err := validateLogType("log_type", req.LogType)
	if err != nil {
		return CreateLogRequest{}, err
	}

	recordedAt, err := validateRecordedAt(req.RecordedAt)
	if err != nil {
		return CreateLogRequest{}, err
	}

	normalized := CreateLogRequest{
		RecordedAt: recordedAt,
		Companions: normalizeStringSlice(req.Companions),
		LogType:    logType,
		Memo:       normalizeOptionalString(req.Memo),
	}

	switch logType {
	case domain.LogTypeCafe:
		if req.Brew != nil {
			return CreateLogRequest{}, newValidationError("brew", "cafe 로그에는 brew 상세를 함께 보낼 수 없습니다")
		}
		detail, err := normalizeCafeDetail(req.Cafe)
		if err != nil {
			return CreateLogRequest{}, err
		}
		normalized.Cafe = detail
	case domain.LogTypeBrew:
		if req.Cafe != nil {
			return CreateLogRequest{}, newValidationError("cafe", "brew 로그에는 cafe 상세를 함께 보낼 수 없습니다")
		}
		detail, err := normalizeBrewDetail(req.Brew)
		if err != nil {
			return CreateLogRequest{}, err
		}
		normalized.Brew = detail
	}

	return normalized, nil
}

func normalizeUpdateRequest(req UpdateLogRequest, existingLogType domain.LogType) (UpdateLogRequest, error) {
	logType := existingLogType
	if req.LogType != "" {
		validatedLogType, err := validateLogType("log_type", req.LogType)
		if err != nil {
			return UpdateLogRequest{}, err
		}
		if validatedLogType != existingLogType {
			return UpdateLogRequest{}, newValidationError("log_type", "기존 로그 타입은 수정할 수 없습니다")
		}
		logType = validatedLogType
	}

	recordedAt, err := validateRecordedAt(req.RecordedAt)
	if err != nil {
		return UpdateLogRequest{}, err
	}

	normalized := UpdateLogRequest{
		RecordedAt: recordedAt,
		Companions: normalizeStringSlice(req.Companions),
		LogType:    logType,
		Memo:       normalizeOptionalString(req.Memo),
	}

	switch logType {
	case domain.LogTypeCafe:
		if req.Brew != nil {
			return UpdateLogRequest{}, newValidationError("brew", "cafe 로그에는 brew 상세를 함께 보낼 수 없습니다")
		}
		detail, err := normalizeCafeDetail(req.Cafe)
		if err != nil {
			return UpdateLogRequest{}, err
		}
		normalized.Cafe = detail
	case domain.LogTypeBrew:
		if req.Cafe != nil {
			return UpdateLogRequest{}, newValidationError("cafe", "brew 로그에는 cafe 상세를 함께 보낼 수 없습니다")
		}
		detail, err := normalizeBrewDetail(req.Brew)
		if err != nil {
			return UpdateLogRequest{}, err
		}
		normalized.Brew = detail
	}

	return normalized, nil
}

func normalizeListFilter(filter ListLogsFilter) (repository.ListFilter, int, error) {
	limit := filter.Limit
	if limit == 0 {
		limit = defaultListLimit
	}
	if limit < 0 {
		return repository.ListFilter{}, 0, newValidationError("limit", "0 이상이어야 합니다")
	}
	if limit > maxListLimit {
		return repository.ListFilter{}, 0, newValidationError("limit", fmt.Sprintf("%d 이하여야 합니다", maxListLimit))
	}
	if limit == 0 {
		return repository.ListFilter{}, 0, newValidationError("limit", "0보다 커야 합니다")
	}

	var (
		logTypeStr *string
		dateFrom   *string
		dateTo     *string
		cursor     *repository.Cursor
	)

	if filter.LogType != nil {
		logType, err := validateLogType("log_type", *filter.LogType)
		if err != nil {
			return repository.ListFilter{}, 0, err
		}
		s := string(logType)
		logTypeStr = &s
	}

	// 타임존 로드: YYYY-MM-DD 날짜 필터를 해당 타임존 기준 UTC 경계로 변환하기 위해 필요하다.
	// Timezone이 지정되지 않으면 앱 기본값(Asia/Seoul)을 사용한다.
	tz := filter.Timezone
	if tz == "" {
		tz = defaultTimezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return repository.ListFilter{}, 0, newValidationError("timezone", "지원하지 않는 타임존입니다")
	}

	var fromTime time.Time
	if filter.DateFrom != nil {
		validated, parsed, err := validateDateFilter("date_from", *filter.DateFrom, false, loc)
		if err != nil {
			return repository.ListFilter{}, 0, err
		}
		dateFrom = &validated
		fromTime = parsed
	}

	var toTime time.Time
	if filter.DateTo != nil {
		validated, parsed, err := validateDateFilter("date_to", *filter.DateTo, true, loc)
		if err != nil {
			return repository.ListFilter{}, 0, err
		}
		dateTo = &validated
		toTime = parsed
	}

	if !fromTime.IsZero() && !toTime.IsZero() && fromTime.After(toTime) {
		return repository.ListFilter{}, 0, newValidationError("date_from", "date_to보다 이후일 수 없습니다")
	}

	if filter.Cursor != nil {
		cursorValue := strings.TrimSpace(*filter.Cursor)
		if cursorValue == "" {
			return repository.ListFilter{}, 0, newValidationError("cursor", "빈 값일 수 없습니다")
		}

		decoded, err := repository.DecodeCursor(cursorValue)
		if err != nil {
			return repository.ListFilter{}, 0, newValidationError("cursor", "유효한 커서 형식이 아닙니다")
		}
		if decoded.SortBy != "recorded_at" {
			return repository.ListFilter{}, 0, newValidationError("cursor", "지원하지 않는 정렬 기준입니다")
		}
		if decoded.Order != "desc" {
			return repository.ListFilter{}, 0, newValidationError("cursor", "지원하지 않는 정렬 방향입니다")
		}
		if _, _, err := parseDateTime(decoded.SortValue); err != nil {
			return repository.ListFilter{}, 0, newValidationError("cursor", "커서의 기준 시각이 올바르지 않습니다")
		}

		cursor = &decoded
	}

	return repository.ListFilter{
		LogType:  logTypeStr,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Cursor:   cursor,
		Limit:    limit,
	}, limit, nil
}

func normalizeCafeDetail(detail *domain.CafeDetail) (*domain.CafeDetail, error) {
	if detail == nil {
		return nil, newValidationError("cafe", "cafe 로그에는 cafe 상세가 필요합니다")
	}

	cafeName, err := validateRequiredString("cafe.cafe_name", detail.CafeName)
	if err != nil {
		return nil, err
	}
	coffeeName, err := validateRequiredString("cafe.coffee_name", detail.CoffeeName)
	if err != nil {
		return nil, err
	}
	roastLevel, err := validateRoastLevel(detail.RoastLevel)
	if err != nil {
		return nil, err
	}
	if err := validateRating("cafe.rating", detail.Rating); err != nil {
		return nil, err
	}

	return &domain.CafeDetail{
		CafeName:    cafeName,
		Location:    normalizeOptionalString(detail.Location),
		CoffeeName:  coffeeName,
		BeanOrigin:  normalizeOptionalString(detail.BeanOrigin),
		BeanProcess: normalizeOptionalString(detail.BeanProcess),
		RoastLevel:  roastLevel,
		TastingTags: normalizeStringSlice(detail.TastingTags),
		TastingNote: normalizeOptionalString(detail.TastingNote),
		Impressions: normalizeOptionalString(detail.Impressions),
		Rating:      cloneFloat64(detail.Rating),
	}, nil
}

func normalizeBrewDetail(detail *domain.BrewDetail) (*domain.BrewDetail, error) {
	if detail == nil {
		return nil, newValidationError("brew", "brew 로그에는 brew 상세가 필요합니다")
	}

	beanName, err := validateRequiredString("brew.bean_name", detail.BeanName)
	if err != nil {
		return nil, err
	}
	roastLevel, err := validateRoastLevel(detail.RoastLevel)
	if err != nil {
		return nil, err
	}
	brewMethod, err := validateBrewMethod(detail.BrewMethod)
	if err != nil {
		return nil, err
	}
	roastDate, err := validateRoastDate(detail.RoastDate)
	if err != nil {
		return nil, err
	}
	if err := validatePositiveFloat("brew.coffee_amount_g", detail.CoffeeAmountG); err != nil {
		return nil, err
	}
	if err := validatePositiveFloat("brew.water_amount_ml", detail.WaterAmountMl); err != nil {
		return nil, err
	}
	if err := validatePositiveFloat("brew.water_temp_c", detail.WaterTempC); err != nil {
		return nil, err
	}
	if err := validatePositiveInt("brew.brew_time_sec", detail.BrewTimeSec); err != nil {
		return nil, err
	}
	if err := validateRating("brew.rating", detail.Rating); err != nil {
		return nil, err
	}

	return &domain.BrewDetail{
		BeanName:      beanName,
		BeanOrigin:    normalizeOptionalString(detail.BeanOrigin),
		BeanProcess:   normalizeOptionalString(detail.BeanProcess),
		RoastLevel:    roastLevel,
		RoastDate:     roastDate,
		TastingTags:   normalizeStringSlice(detail.TastingTags),
		TastingNote:   normalizeOptionalString(detail.TastingNote),
		BrewMethod:    brewMethod,
		BrewDevice:    normalizeOptionalString(detail.BrewDevice),
		CoffeeAmountG: cloneFloat64(detail.CoffeeAmountG),
		WaterAmountMl: cloneFloat64(detail.WaterAmountMl),
		WaterTempC:    cloneFloat64(detail.WaterTempC),
		BrewTimeSec:   cloneInt(detail.BrewTimeSec),
		GrindSize:     normalizeOptionalString(detail.GrindSize),
		BrewSteps:     normalizeStringSlice(detail.BrewSteps),
		Impressions:   normalizeOptionalString(detail.Impressions),
		Rating:        cloneFloat64(detail.Rating),
	}, nil
}

func validateIdentifier(field, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", newValidationError(field, "필수값입니다")
	}
	return trimmed, nil
}

func validateRequiredString(field, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", newValidationError(field, "필수값입니다")
	}
	return trimmed, nil
}

func validateLogType(field string, logType domain.LogType) (domain.LogType, error) {
	switch logType {
	case domain.LogTypeCafe, domain.LogTypeBrew:
		return logType, nil
	default:
		return "", newValidationError(field, "cafe 또는 brew만 허용됩니다")
	}
}

func validateRoastLevel(level *domain.RoastLevel) (*domain.RoastLevel, error) {
	if level == nil {
		return nil, nil
	}

	switch *level {
	case domain.RoastLight, domain.RoastMedium, domain.RoastDark:
		value := *level
		return &value, nil
	default:
		return nil, newValidationError("roast_level", "light, medium, dark만 허용됩니다")
	}
}

func validateBrewMethod(method domain.BrewMethod) (domain.BrewMethod, error) {
	switch method {
	case domain.BrewMethodPourOver,
		domain.BrewMethodImmersion,
		domain.BrewMethodAeropress,
		domain.BrewMethodEspresso,
		domain.BrewMethodMokaPot,
		domain.BrewMethodSiphon,
		domain.BrewMethodColdBrew,
		domain.BrewMethodOther:
		return method, nil
	default:
		return "", newValidationError("brew.brew_method", "지원하지 않는 추출 방식입니다")
	}
}

func validateRecordedAt(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", newValidationError("recorded_at", "필수값입니다")
	}
	parsed, _, err := parseDateTime(trimmed)
	if err != nil {
		return "", newValidationError("recorded_at", "RFC3339 datetime 형식이어야 합니다")
	}
	// SQLite 문자열 비교가 시각 순서와 일치하도록 UTC RFC3339Nano로 정규화한다.
	// +09:00 등 오프셋 포함 입력이 섞이면 정렬과 커서 페이지네이션이 깨지므로 저장 전에 변환해야 한다.
	return parsed.UTC().Format(time.RFC3339Nano), nil
}

// validateDateFilter는 날짜 필터 값을 검증하고 UTC RFC3339Nano 문자열로 정규화한다.
// YYYY-MM-DD 입력은 loc 타임존 기준 하루 경계(00:00:00 또는 23:59:59.999)를 UTC로 변환한다.
// RFC3339 입력은 UTC로 정규화하여 저장된 recorded_at(UTC)과 문자열 비교가 가능하게 한다.
func validateDateFilter(field, value string, endOfDay bool, loc *time.Location) (string, time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", time.Time{}, newValidationError(field, "빈 값일 수 없습니다")
	}

	// YYYY-MM-DD 형식인 경우 loc 타임존 기준으로 하루의 경계를 구한 뒤 UTC로 변환한다.
	// UTC 고정 대신 loc을 쓰는 이유: 한국 사용자가 2026-03-29를 입력하면 KST 기준
	// 00:00~23:59 범위를 의미하므로, 서울 자정(UTC 전날 15:00)부터 잡아야 기록이 누락되지 않는다.
	if d, err := time.Parse("2006-01-02", trimmed); err == nil {
		var normalized time.Time
		if endOfDay {
			normalized = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 999_000_000, loc)
		} else {
			normalized = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
		}
		utc := normalized.UTC()
		return utc.Format(time.RFC3339Nano), utc, nil
	}

	parsed, _, err := parseDateTime(trimmed)
	if err != nil {
		return "", time.Time{}, newValidationError(field, "RFC3339 datetime 또는 YYYY-MM-DD 형식이어야 합니다")
	}

	// 저장된 recorded_at이 UTC RFC3339Nano로 정규화되므로, 필터 경계값도 같은 포맷으로 맞춰야
	// 문자열 비교가 정확하게 동작한다.
	utc := parsed.UTC()
	return utc.Format(time.RFC3339Nano), utc, nil
}

func validateRoastDate(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return nil, newValidationError("brew.roast_date", "YYYY-MM-DD 형식이어야 합니다")
	}
	return &trimmed, nil
}

func validatePositiveFloat(field string, value *float64) error {
	if value == nil {
		return nil
	}
	if *value <= 0 {
		return newValidationError(field, "0보다 커야 합니다")
	}
	return nil
}

func validatePositiveInt(field string, value *int) error {
	if value == nil {
		return nil
	}
	if *value <= 0 {
		return newValidationError(field, "0보다 커야 합니다")
	}
	return nil
}

func validateRating(field string, value *float64) error {
	if value == nil {
		return nil
	}
	if *value < 0.5 || *value > 5.0 {
		return newValidationError(field, "0.5 이상 5.0 이하여야 합니다")
	}

	doubled := *value * 2
	if math.Abs(doubled-math.Round(doubled)) > 1e-9 {
		return newValidationError(field, "0.5 단위여야 합니다")
	}

	return nil
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	if normalized == nil {
		return []string{}
	}
	return normalized
}

func cloneFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

// parseDateTime은 RFC3339 형식만 허용한다.
// YYYY-MM-DD 형식은 validateDateFilter에서 정규화 후 처리한다.
func parseDateTime(value string) (time.Time, string, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, layout, nil
		}
	}
	return time.Time{}, "", fmt.Errorf("unsupported datetime format: %s", value)
}

func mapRepositoryError(_ string, err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func newValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

func newUUID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}

	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		raw[0:4],
		raw[4:6],
		raw[6:8],
		raw[8:10],
		raw[10:16],
	), nil
}
