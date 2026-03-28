package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"coffee-of-the-day/backend/internal/db"
	"coffee-of-the-day/backend/internal/domain"
)

// ErrNotFound is returned when a requested log does not exist or does not
// belong to the requesting user.
var ErrNotFound = errors.New("log not found")

// ListFilter holds optional filters for listing coffee logs.
type ListFilter struct {
	LogType  *string
	DateFrom *string
	DateTo   *string
	Cursor   *Cursor
	Limit    int
}

// LogRepository defines the persistence interface for coffee logs.
type LogRepository interface {
	CreateLog(ctx context.Context, log domain.CoffeeLogFull) error
	GetLogByID(ctx context.Context, logID, userID string) (domain.CoffeeLogFull, error)
	ListLogs(ctx context.Context, userID string, filter ListFilter) ([]domain.CoffeeLogFull, error)
	UpdateLog(ctx context.Context, log domain.CoffeeLogFull) error
	DeleteLog(ctx context.Context, logID, userID string) error
}

// SQLiteLogRepository implements LogRepository backed by SQLite.
type SQLiteLogRepository struct {
	sqlDB   *sql.DB
	queries *db.Queries
}

// NewSQLiteLogRepository creates a new SQLiteLogRepository.
func NewSQLiteLogRepository(sqlDB *sql.DB) *SQLiteLogRepository {
	return &SQLiteLogRepository{
		sqlDB:   sqlDB,
		queries: db.New(sqlDB),
	}
}

func (r *SQLiteLogRepository) CreateLog(ctx context.Context, log domain.CoffeeLogFull) error {
	tx, err := r.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("create log: begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	err = qtx.InsertLog(ctx, db.InsertLogParams{
		ID:         log.ID,
		UserID:     log.UserID,
		RecordedAt: log.RecordedAt,
		Companions: domain.StringsToJSON(log.Companions),
		LogType:    string(log.LogType),
		Memo:       log.Memo,
		CreatedAt:  log.CreatedAt,
		UpdatedAt:  log.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("create log: insert log: %w", err)
	}

	// 로그 타입에 따라 상세 테이블(cafe_logs 또는 brew_logs)에 데이터 삽입
	switch log.LogType {
	case domain.LogTypeCafe:
		c := log.Cafe
		err = qtx.InsertCafeLog(ctx, db.InsertCafeLogParams{
			LogID:       log.ID,
			CafeName:    c.CafeName,
			Location:    c.Location,
			CoffeeName:  c.CoffeeName,
			BeanOrigin:  c.BeanOrigin,
			BeanProcess: c.BeanProcess,
			RoastLevel:  roastToStr(c.RoastLevel),
			TastingTags: domain.StringsToJSON(c.TastingTags),
			TastingNote: c.TastingNote,
			Impressions: c.Impressions,
			Rating:      c.Rating,
		})
	case domain.LogTypeBrew:
		b := log.Brew
		err = qtx.InsertBrewLog(ctx, db.InsertBrewLogParams{
			LogID:         log.ID,
			BeanName:      b.BeanName,
			BeanOrigin:    b.BeanOrigin,
			BeanProcess:   b.BeanProcess,
			RoastLevel:    roastToStr(b.RoastLevel),
			RoastDate:     b.RoastDate,
			TastingTags:   domain.StringsToJSON(b.TastingTags),
			TastingNote:   b.TastingNote,
			BrewMethod:    string(b.BrewMethod),
			BrewDevice:    b.BrewDevice,
			CoffeeAmountG: b.CoffeeAmountG,
			WaterAmountMl: b.WaterAmountMl,
			WaterTempC:    b.WaterTempC,
			BrewTimeSec:   intToInt64(b.BrewTimeSec),
			GrindSize:     b.GrindSize,
			BrewSteps:     domain.StringsToJSON(b.BrewSteps),
			Impressions:   b.Impressions,
			Rating:        b.Rating,
		})
	}
	if err != nil {
		return fmt.Errorf("create log: insert detail: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("create log: commit: %w", err)
	}
	return nil
}

func (r *SQLiteLogRepository) GetLogByID(ctx context.Context, logID, userID string) (domain.CoffeeLogFull, error) {
	row, err := r.queries.GetLogByID(ctx, db.GetLogByIDParams{ID: logID, UserID: userID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.CoffeeLogFull{}, ErrNotFound
		}
		return domain.CoffeeLogFull{}, fmt.Errorf("get log: %w", err)
	}

	f := coffeeLogToFull(row)

	if err := r.loadDetail(ctx, &f); err != nil {
		return domain.CoffeeLogFull{}, fmt.Errorf("get log: load detail: %w", err)
	}

	return f, nil
}

func (r *SQLiteLogRepository) ListLogs(ctx context.Context, userID string, filter ListFilter) ([]domain.CoffeeLogFull, error) {
	query := `SELECT id, user_id, recorded_at, companions, log_type, memo, created_at, updated_at
		FROM coffee_logs WHERE user_id = ?`
	args := []any{userID}

	if filter.LogType != nil {
		query += ` AND log_type = ?`
		args = append(args, *filter.LogType)
	}
	if filter.DateFrom != nil {
		query += ` AND recorded_at >= ?`
		args = append(args, *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query += ` AND recorded_at <= ?`
		args = append(args, *filter.DateTo)
	}
	// 커서 기반 페이지네이션: recorded_at DESC, id DESC 정렬에서
	// 이전 페이지의 마지막 항목 이후 데이터만 조회
	if filter.Cursor != nil {
		query += ` AND (recorded_at < ? OR (recorded_at = ? AND id < ?))`
		args = append(args, filter.Cursor.SortValue, filter.Cursor.SortValue, filter.Cursor.ID)
	}

	query += ` ORDER BY recorded_at DESC, id DESC LIMIT ?`
	args = append(args, filter.Limit)

	rows, err := r.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list logs: query: %w", err)
	}
	defer rows.Close()

	var items []domain.CoffeeLogFull
	for rows.Next() {
		var f domain.CoffeeLogFull
		var companions, logType string
		if err := rows.Scan(&f.ID, &f.UserID, &f.RecordedAt, &companions, &logType, &f.Memo, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("list logs: scan: %w", err)
		}
		f.Companions = domain.JSONToStrings(companions)
		f.LogType = domain.LogType(logType)
		items = append(items, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list logs: rows: %w", err)
	}

	for i := range items {
		if err := r.loadDetail(ctx, &items[i]); err != nil {
			return nil, fmt.Errorf("list logs: load detail: %w", err)
		}
	}

	return items, nil
}

func (r *SQLiteLogRepository) UpdateLog(ctx context.Context, log domain.CoffeeLogFull) error {
	tx, err := r.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("update log: begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	err = qtx.UpdateLog(ctx, db.UpdateLogParams{
		RecordedAt: log.RecordedAt,
		Companions: domain.StringsToJSON(log.Companions),
		Memo:       log.Memo,
		UpdatedAt:  log.UpdatedAt,
		ID:         log.ID,
		UserID:     log.UserID,
	})
	if err != nil {
		return fmt.Errorf("update log: update common: %w", err)
	}

	// 로그 타입에 따라 상세 테이블(cafe_logs 또는 brew_logs) 업데이트
	switch log.LogType {
	case domain.LogTypeCafe:
		c := log.Cafe
		err = qtx.UpdateCafeLog(ctx, db.UpdateCafeLogParams{
			CafeName:    c.CafeName,
			Location:    c.Location,
			CoffeeName:  c.CoffeeName,
			BeanOrigin:  c.BeanOrigin,
			BeanProcess: c.BeanProcess,
			RoastLevel:  roastToStr(c.RoastLevel),
			TastingTags: domain.StringsToJSON(c.TastingTags),
			TastingNote: c.TastingNote,
			Impressions: c.Impressions,
			Rating:      c.Rating,
			LogID:       log.ID,
		})
	case domain.LogTypeBrew:
		b := log.Brew
		err = qtx.UpdateBrewLog(ctx, db.UpdateBrewLogParams{
			BeanName:      b.BeanName,
			BeanOrigin:    b.BeanOrigin,
			BeanProcess:   b.BeanProcess,
			RoastLevel:    roastToStr(b.RoastLevel),
			RoastDate:     b.RoastDate,
			TastingTags:   domain.StringsToJSON(b.TastingTags),
			TastingNote:   b.TastingNote,
			BrewMethod:    string(b.BrewMethod),
			BrewDevice:    b.BrewDevice,
			CoffeeAmountG: b.CoffeeAmountG,
			WaterAmountMl: b.WaterAmountMl,
			WaterTempC:    b.WaterTempC,
			BrewTimeSec:   intToInt64(b.BrewTimeSec),
			GrindSize:     b.GrindSize,
			BrewSteps:     domain.StringsToJSON(b.BrewSteps),
			Impressions:   b.Impressions,
			Rating:        b.Rating,
			LogID:         log.ID,
		})
	}
	if err != nil {
		return fmt.Errorf("update log: update detail: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("update log: commit: %w", err)
	}
	return nil
}

func (r *SQLiteLogRepository) DeleteLog(ctx context.Context, logID, userID string) error {
	res, err := r.sqlDB.ExecContext(ctx, "DELETE FROM coffee_logs WHERE id = ? AND user_id = ?", logID, userID)
	if err != nil {
		return fmt.Errorf("delete log: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete log: rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// loadDetail은 로그 타입에 따라 상세 테이블에서 데이터를 조회하여 채운다.
// 상세 데이터가 없는 경우(ErrNoRows)는 정상으로 처리한다 —
// 공통 로그는 존재하지만 상세가 아직 없는 상태를 허용하기 위함.
func (r *SQLiteLogRepository) loadDetail(ctx context.Context, f *domain.CoffeeLogFull) error {
	switch f.LogType {
	case domain.LogTypeCafe:
		row, err := r.queries.GetCafeLogByLogID(ctx, f.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		f.Cafe = &domain.CafeDetail{
			CafeName:    row.CafeName,
			Location:    row.Location,
			CoffeeName:  row.CoffeeName,
			BeanOrigin:  row.BeanOrigin,
			BeanProcess: row.BeanProcess,
			RoastLevel:  strToRoast(row.RoastLevel),
			TastingTags: domain.JSONToStrings(row.TastingTags),
			TastingNote: row.TastingNote,
			Impressions: row.Impressions,
			Rating:      row.Rating,
		}
	case domain.LogTypeBrew:
		row, err := r.queries.GetBrewLogByLogID(ctx, f.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		f.Brew = &domain.BrewDetail{
			BeanName:      row.BeanName,
			BeanOrigin:    row.BeanOrigin,
			BeanProcess:   row.BeanProcess,
			RoastLevel:    strToRoast(row.RoastLevel),
			RoastDate:     row.RoastDate,
			TastingTags:   domain.JSONToStrings(row.TastingTags),
			TastingNote:   row.TastingNote,
			BrewMethod:    domain.BrewMethod(row.BrewMethod),
			BrewDevice:    row.BrewDevice,
			CoffeeAmountG: row.CoffeeAmountG,
			WaterAmountMl: row.WaterAmountMl,
			WaterTempC:    row.WaterTempC,
			BrewTimeSec:   int64ToInt(row.BrewTimeSec),
			GrindSize:     row.GrindSize,
			BrewSteps:     domain.JSONToStrings(row.BrewSteps),
			Impressions:   row.Impressions,
			Rating:        row.Rating,
		}
	}
	return nil
}

func coffeeLogToFull(row db.CoffeeLog) domain.CoffeeLogFull {
	return domain.CoffeeLogFull{
		CoffeeLog: domain.CoffeeLog{
			ID:         row.ID,
			UserID:     row.UserID,
			RecordedAt: row.RecordedAt,
			Companions: domain.JSONToStrings(row.Companions),
			LogType:    domain.LogType(row.LogType),
			Memo:       row.Memo,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		},
	}
}

func roastToStr(r *domain.RoastLevel) *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}

func strToRoast(s *string) *domain.RoastLevel {
	if s == nil {
		return nil
	}
	r := domain.RoastLevel(*s)
	return &r
}

func intToInt64(i *int) *int64 {
	if i == nil {
		return nil
	}
	v := int64(*i)
	return &v
}

func int64ToInt(i *int64) *int {
	if i == nil {
		return nil
	}
	v := int(*i)
	return &v
}
