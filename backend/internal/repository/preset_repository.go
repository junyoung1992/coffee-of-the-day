package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"coffee-of-the-day/backend/internal/db"
	"coffee-of-the-day/backend/internal/domain"
)

// PresetRepository는 프리셋의 영속성 인터페이스를 정의한다.
type PresetRepository interface {
	CreatePreset(ctx context.Context, preset domain.PresetFull) error
	GetPresetByID(ctx context.Context, presetID, userID string) (domain.PresetFull, error)
	ListPresets(ctx context.Context, userID string) ([]domain.PresetFull, error)
	UpdatePreset(ctx context.Context, preset domain.PresetFull) error
	DeletePreset(ctx context.Context, presetID, userID string) error
	UpdateLastUsedAt(ctx context.Context, presetID, userID string, usedAt string) error
}

// SQLitePresetRepository는 SQLite 기반 PresetRepository 구현체이다.
type SQLitePresetRepository struct {
	sqlDB   *sql.DB
	queries *db.Queries
}

// NewSQLitePresetRepository는 새 SQLitePresetRepository를 생성한다.
func NewSQLitePresetRepository(sqlDB *sql.DB) *SQLitePresetRepository {
	return &SQLitePresetRepository{
		sqlDB:   sqlDB,
		queries: db.New(sqlDB),
	}
}

func (r *SQLitePresetRepository) CreatePreset(ctx context.Context, preset domain.PresetFull) error {
	tx, err := r.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("create preset: begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	err = qtx.InsertPreset(ctx, db.InsertPresetParams{
		ID:         preset.ID,
		UserID:     preset.UserID,
		Name:       preset.Name,
		LogType:    string(preset.LogType),
		LastUsedAt: preset.LastUsedAt,
		CreatedAt:  preset.CreatedAt,
		UpdatedAt:  preset.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("create preset: insert preset: %w", err)
	}

	// 프리셋 타입에 따라 서브 테이블에 데이터 삽입
	switch preset.LogType {
	case domain.LogTypeCafe:
		c := preset.Cafe
		err = qtx.InsertCafePreset(ctx, db.InsertCafePresetParams{
			PresetID:    preset.ID,
			CafeName:    c.CafeName,
			CoffeeName:  c.CoffeeName,
			TastingTags: domain.StringsToJSON(c.TastingTags),
		})
	case domain.LogTypeBrew:
		b := preset.Brew
		err = qtx.InsertBrewPreset(ctx, db.InsertBrewPresetParams{
			PresetID:     preset.ID,
			BeanName:     b.BeanName,
			BrewMethod:   string(b.BrewMethod),
			RecipeDetail: b.RecipeDetail,
			BrewSteps:    domain.StringsToJSON(b.BrewSteps),
		})
	default:
		return fmt.Errorf("create preset: unsupported log_type: %s", preset.LogType)
	}
	if err != nil {
		return fmt.Errorf("create preset: insert detail: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("create preset: commit: %w", err)
	}
	return nil
}

func (r *SQLitePresetRepository) GetPresetByID(ctx context.Context, presetID, userID string) (domain.PresetFull, error) {
	row, err := r.queries.GetPresetByID(ctx, db.GetPresetByIDParams{ID: presetID, UserID: userID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.PresetFull{}, ErrNotFound
		}
		return domain.PresetFull{}, fmt.Errorf("get preset: %w", err)
	}

	f := presetToFull(row)
	if err := r.loadDetail(ctx, &f); err != nil {
		return domain.PresetFull{}, fmt.Errorf("get preset: load detail: %w", err)
	}
	return f, nil
}

func (r *SQLitePresetRepository) ListPresets(ctx context.Context, userID string) ([]domain.PresetFull, error) {
	rows, err := r.queries.ListPresetsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list presets: %w", err)
	}

	items := make([]domain.PresetFull, len(rows))
	for i, row := range rows {
		items[i] = presetToFull(row)
	}

	if len(items) == 0 {
		return items, nil
	}

	// N+1 방지: 타입별 ID를 분리하여 배치 조회
	var cafeIDs, brewIDs []string
	for _, item := range items {
		switch item.LogType {
		case domain.LogTypeCafe:
			cafeIDs = append(cafeIDs, item.ID)
		case domain.LogTypeBrew:
			brewIDs = append(brewIDs, item.ID)
		}
	}

	cafeMap, err := r.batchLoadCafePresets(ctx, cafeIDs)
	if err != nil {
		return nil, fmt.Errorf("list presets: batch load cafe: %w", err)
	}
	brewMap, err := r.batchLoadBrewPresets(ctx, brewIDs)
	if err != nil {
		return nil, fmt.Errorf("list presets: batch load brew: %w", err)
	}

	for i := range items {
		switch items[i].LogType {
		case domain.LogTypeCafe:
			detail, ok := cafeMap[items[i].ID]
			if !ok {
				return nil, fmt.Errorf("list presets: cafe detail missing for preset %s: %w", items[i].ID, ErrNotFound)
			}
			items[i].Cafe = detail
		case domain.LogTypeBrew:
			detail, ok := brewMap[items[i].ID]
			if !ok {
				return nil, fmt.Errorf("list presets: brew detail missing for preset %s: %w", items[i].ID, ErrNotFound)
			}
			items[i].Brew = detail
		}
	}

	return items, nil
}

func (r *SQLitePresetRepository) UpdatePreset(ctx context.Context, preset domain.PresetFull) error {
	tx, err := r.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("update preset: begin tx: %w", err)
	}
	defer tx.Rollback()

	// 공통 테이블 업데이트 + rows affected 확인
	res, err := tx.ExecContext(ctx,
		"UPDATE presets SET name = ?, updated_at = ? WHERE id = ? AND user_id = ?",
		preset.Name, preset.UpdatedAt, preset.ID, preset.UserID,
	)
	if err != nil {
		return fmt.Errorf("update preset: update common: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update preset: rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}

	qtx := r.queries.WithTx(tx)

	// 프리셋 타입에 따라 서브 테이블 업데이트
	switch preset.LogType {
	case domain.LogTypeCafe:
		c := preset.Cafe
		err = qtx.UpdateCafePreset(ctx, db.UpdateCafePresetParams{
			CafeName:    c.CafeName,
			CoffeeName:  c.CoffeeName,
			TastingTags: domain.StringsToJSON(c.TastingTags),
			PresetID:    preset.ID,
		})
	case domain.LogTypeBrew:
		b := preset.Brew
		err = qtx.UpdateBrewPreset(ctx, db.UpdateBrewPresetParams{
			BeanName:     b.BeanName,
			BrewMethod:   string(b.BrewMethod),
			RecipeDetail: b.RecipeDetail,
			BrewSteps:    domain.StringsToJSON(b.BrewSteps),
			PresetID:     preset.ID,
		})
	default:
		return fmt.Errorf("update preset: unsupported log_type: %s", preset.LogType)
	}
	if err != nil {
		return fmt.Errorf("update preset: update detail: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("update preset: commit: %w", err)
	}
	return nil
}

func (r *SQLitePresetRepository) DeletePreset(ctx context.Context, presetID, userID string) error {
	res, err := r.sqlDB.ExecContext(ctx, "DELETE FROM presets WHERE id = ? AND user_id = ?", presetID, userID)
	if err != nil {
		return fmt.Errorf("delete preset: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete preset: rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLitePresetRepository) UpdateLastUsedAt(ctx context.Context, presetID, userID string, usedAt string) error {
	res, err := r.sqlDB.ExecContext(ctx,
		"UPDATE presets SET last_used_at = ?, updated_at = ? WHERE id = ? AND user_id = ?",
		usedAt, usedAt, presetID, userID,
	)
	if err != nil {
		return fmt.Errorf("update last used at: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update last used at: rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// loadDetail은 프리셋 타입에 따라 서브 테이블에서 상세 데이터를 조회한다.
func (r *SQLitePresetRepository) loadDetail(ctx context.Context, f *domain.PresetFull) error {
	switch f.LogType {
	case domain.LogTypeCafe:
		row, err := r.queries.GetCafePresetByPresetID(ctx, f.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("cafe detail missing for preset %s: %w", f.ID, ErrNotFound)
			}
			return err
		}
		f.Cafe = &domain.CafePresetDetail{
			CafeName:    row.CafeName,
			CoffeeName:  row.CoffeeName,
			TastingTags: domain.JSONToStrings(row.TastingTags),
		}
	case domain.LogTypeBrew:
		row, err := r.queries.GetBrewPresetByPresetID(ctx, f.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("brew detail missing for preset %s: %w", f.ID, ErrNotFound)
			}
			return err
		}
		f.Brew = &domain.BrewPresetDetail{
			BeanName:     row.BeanName,
			BrewMethod:   domain.BrewMethod(row.BrewMethod),
			RecipeDetail: row.RecipeDetail,
			BrewSteps:    domain.JSONToStrings(row.BrewSteps),
		}
	}
	return nil
}

func (r *SQLitePresetRepository) batchLoadCafePresets(ctx context.Context, ids []string) (map[string]*domain.CafePresetDetail, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf(
		`SELECT preset_id, cafe_name, coffee_name, tasting_tags FROM cafe_presets WHERE preset_id IN (%s)`,
		placeholders,
	)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := r.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]*domain.CafePresetDetail, len(ids))
	for rows.Next() {
		var presetID, cafeName, coffeeName, tastingTags string
		if err := rows.Scan(&presetID, &cafeName, &coffeeName, &tastingTags); err != nil {
			return nil, err
		}
		result[presetID] = &domain.CafePresetDetail{
			CafeName:    cafeName,
			CoffeeName:  coffeeName,
			TastingTags: domain.JSONToStrings(tastingTags),
		}
	}
	return result, rows.Err()
}

func (r *SQLitePresetRepository) batchLoadBrewPresets(ctx context.Context, ids []string) (map[string]*domain.BrewPresetDetail, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	query := fmt.Sprintf(
		`SELECT preset_id, bean_name, brew_method, recipe_detail, brew_steps FROM brew_presets WHERE preset_id IN (%s)`,
		placeholders,
	)
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := r.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]*domain.BrewPresetDetail, len(ids))
	for rows.Next() {
		var presetID, beanName, brewMethod, brewSteps string
		var recipeDetail *string
		if err := rows.Scan(&presetID, &beanName, &brewMethod, &recipeDetail, &brewSteps); err != nil {
			return nil, err
		}
		result[presetID] = &domain.BrewPresetDetail{
			BeanName:     beanName,
			BrewMethod:   domain.BrewMethod(brewMethod),
			RecipeDetail: recipeDetail,
			BrewSteps:    domain.JSONToStrings(brewSteps),
		}
	}
	return result, rows.Err()
}

func presetToFull(row db.Preset) domain.PresetFull {
	return domain.PresetFull{
		Preset: domain.Preset{
			ID:         row.ID,
			UserID:     row.UserID,
			Name:       row.Name,
			LogType:    domain.LogType(row.LogType),
			LastUsedAt: row.LastUsedAt,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		},
	}
}
