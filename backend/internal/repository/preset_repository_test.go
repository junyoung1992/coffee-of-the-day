package repository

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"coffee-of-the-day/backend/internal/domain"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupPresetTestDB는 프리셋 테이블까지 포함한 in-memory SQLite DB를 생성한다.
func setupPresetTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	migrationsDir := filepath.Join("..", "..", "db", "migrations")
	for _, name := range []string{
		"001_create_users.up.sql",
		"002_create_coffee_logs.up.sql",
		"003_create_cafe_logs.up.sql",
		"004_create_brew_logs.up.sql",
		"007_create_presets.up.sql",
	} {
		raw, err := os.ReadFile(filepath.Join(migrationsDir, name))
		require.NoError(t, err, "reading migration %s", name)
		_, err = db.Exec(string(raw))
		require.NoError(t, err, "running migration %s", name)
	}

	// 테스트 사용자 삽입
	_, err = db.Exec(
		"INSERT INTO users (id, username, display_name, created_at) VALUES (?, ?, ?, ?)",
		testUserID, "testuser", "Test User", "2026-01-01T00:00:00Z",
	)
	require.NoError(t, err)
	_, err = db.Exec(
		"INSERT INTO users (id, username, display_name, created_at) VALUES (?, ?, ?, ?)",
		otherUserID, "otheruser", "Other User", "2026-01-01T00:00:00Z",
	)
	require.NoError(t, err)

	return db
}

func newCafePreset(id, name string) domain.PresetFull {
	return domain.PresetFull{
		Preset: domain.Preset{
			ID:        id,
			UserID:    testUserID,
			Name:      name,
			LogType:   domain.LogTypeCafe,
			CreatedAt: "2026-04-01T00:00:00Z",
			UpdatedAt: "2026-04-01T00:00:00Z",
		},
		Cafe: &domain.CafePresetDetail{
			CafeName:    "블루보틀",
			CoffeeName:  "싱글 오리진",
			TastingTags: []string{"fruity", "floral"},
		},
	}
}

func newBrewPreset(id, name string) domain.PresetFull {
	return domain.PresetFull{
		Preset: domain.Preset{
			ID:        id,
			UserID:    testUserID,
			Name:      name,
			LogType:   domain.LogTypeBrew,
			CreatedAt: "2026-04-01T00:00:00Z",
			UpdatedAt: "2026-04-01T00:00:00Z",
		},
		Brew: &domain.BrewPresetDetail{
			BeanName:     "에티오피아 예가체프",
			BrewMethod:   domain.BrewMethodPourOver,
			RecipeDetail: ptrStr("V60 레시피"),
			BrewSteps:    []string{"뜸들이기 30초", "1차 주수", "2차 주수"},
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPresetRepository_CreateAndGet(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	t.Run("cafe preset", func(t *testing.T) {
		p := newCafePreset("preset-cafe-1", "출근길 아메리카노")
		require.NoError(t, repo.CreatePreset(ctx, p))

		got, err := repo.GetPresetByID(ctx, "preset-cafe-1", testUserID)
		require.NoError(t, err)

		assert.Equal(t, "출근길 아메리카노", got.Name)
		assert.Equal(t, domain.LogTypeCafe, got.LogType)
		require.NotNil(t, got.Cafe)
		assert.Equal(t, "블루보틀", got.Cafe.CafeName)
		assert.Equal(t, "싱글 오리진", got.Cafe.CoffeeName)
		assert.Equal(t, []string{"fruity", "floral"}, got.Cafe.TastingTags)
		assert.Nil(t, got.Brew)
	})

	t.Run("brew preset", func(t *testing.T) {
		p := newBrewPreset("preset-brew-1", "주말 핸드드립")
		require.NoError(t, repo.CreatePreset(ctx, p))

		got, err := repo.GetPresetByID(ctx, "preset-brew-1", testUserID)
		require.NoError(t, err)

		assert.Equal(t, "주말 핸드드립", got.Name)
		assert.Equal(t, domain.LogTypeBrew, got.LogType)
		require.NotNil(t, got.Brew)
		assert.Equal(t, "에티오피아 예가체프", got.Brew.BeanName)
		assert.Equal(t, domain.BrewMethodPourOver, got.Brew.BrewMethod)
		assert.Equal(t, "V60 레시피", *got.Brew.RecipeDetail)
		assert.Equal(t, []string{"뜸들이기 30초", "1차 주수", "2차 주수"}, got.Brew.BrewSteps)
		assert.Nil(t, got.Cafe)
	})
}

func TestPresetRepository_GetNotFound(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	_, err := repo.GetPresetByID(ctx, "nonexistent", testUserID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPresetRepository_GetOtherUserDenied(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	p := newCafePreset("preset-1", "내 프리셋")
	require.NoError(t, repo.CreatePreset(ctx, p))

	// 다른 사용자가 접근 시도
	_, err := repo.GetPresetByID(ctx, "preset-1", otherUserID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPresetRepository_ListPresets(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	// 빈 목록
	items, err := repo.ListPresets(ctx, testUserID)
	require.NoError(t, err)
	assert.Empty(t, items)

	// 프리셋 3개 생성 (cafe 2개, brew 1개)
	p1 := newCafePreset("p1", "프리셋1")
	p1.CreatedAt = "2026-04-01T00:00:00Z"
	p2 := newBrewPreset("p2", "프리셋2")
	p2.CreatedAt = "2026-04-02T00:00:00Z"
	p3 := newCafePreset("p3", "프리셋3")
	p3.CreatedAt = "2026-04-03T00:00:00Z"

	require.NoError(t, repo.CreatePreset(ctx, p1))
	require.NoError(t, repo.CreatePreset(ctx, p2))
	require.NoError(t, repo.CreatePreset(ctx, p3))

	items, err = repo.ListPresets(ctx, testUserID)
	require.NoError(t, err)
	assert.Len(t, items, 3)

	// 상세 데이터가 모두 채워져 있는지 확인
	for _, item := range items {
		switch item.LogType {
		case domain.LogTypeCafe:
			require.NotNil(t, item.Cafe)
		case domain.LogTypeBrew:
			require.NotNil(t, item.Brew)
		}
	}
}

func TestPresetRepository_ListPresets_SortOrder(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	// last_used_at이 있는 프리셋이 NULL보다 먼저 와야 한다
	p1 := newCafePreset("p-old", "오래된 프리셋")
	p1.CreatedAt = "2026-04-01T00:00:00Z"
	// last_used_at = nil

	p2 := newCafePreset("p-used", "사용된 프리셋")
	p2.CreatedAt = "2026-04-02T00:00:00Z"
	// last_used_at = nil (생성 시)

	p3 := newBrewPreset("p-new", "새 프리셋")
	p3.CreatedAt = "2026-04-03T00:00:00Z"
	// last_used_at = nil

	require.NoError(t, repo.CreatePreset(ctx, p1))
	require.NoError(t, repo.CreatePreset(ctx, p2))
	require.NoError(t, repo.CreatePreset(ctx, p3))

	// p-used에 last_used_at 설정
	require.NoError(t, repo.UpdateLastUsedAt(ctx, "p-used", testUserID, "2026-04-04T10:00:00Z"))

	items, err := repo.ListPresets(ctx, testUserID)
	require.NoError(t, err)
	require.Len(t, items, 3)

	// 첫 번째: last_used_at이 있는 p-used
	assert.Equal(t, "p-used", items[0].ID)
	// 나머지: last_used_at이 NULL → created_at DESC
	assert.Equal(t, "p-new", items[1].ID)
	assert.Equal(t, "p-old", items[2].ID)
}

func TestPresetRepository_Update(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	p := newCafePreset("preset-upd", "원래 이름")
	require.NoError(t, repo.CreatePreset(ctx, p))

	// 이름과 상세 필드 수정
	p.Name = "바뀐 이름"
	p.Cafe.CafeName = "스타벅스"
	p.Cafe.TastingTags = []string{"nutty"}
	p.UpdatedAt = "2026-04-02T00:00:00Z"

	require.NoError(t, repo.UpdatePreset(ctx, p))

	got, err := repo.GetPresetByID(ctx, "preset-upd", testUserID)
	require.NoError(t, err)
	assert.Equal(t, "바뀐 이름", got.Name)
	assert.Equal(t, "스타벅스", got.Cafe.CafeName)
	assert.Equal(t, []string{"nutty"}, got.Cafe.TastingTags)
	assert.Equal(t, "2026-04-02T00:00:00Z", got.UpdatedAt)
}

func TestPresetRepository_Delete(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	p := newCafePreset("preset-del", "삭제 대상")
	require.NoError(t, repo.CreatePreset(ctx, p))

	require.NoError(t, repo.DeletePreset(ctx, "preset-del", testUserID))

	// 삭제 후 조회 실패
	_, err := repo.GetPresetByID(ctx, "preset-del", testUserID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPresetRepository_DeleteNotFound(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	err := repo.DeletePreset(ctx, "nonexistent", testUserID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPresetRepository_DeleteOtherUserDenied(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	p := newCafePreset("preset-other", "남의 프리셋")
	require.NoError(t, repo.CreatePreset(ctx, p))

	err := repo.DeletePreset(ctx, "preset-other", otherUserID)
	assert.ErrorIs(t, err, ErrNotFound)

	// 원래 소유자로는 여전히 접근 가능
	_, err = repo.GetPresetByID(ctx, "preset-other", testUserID)
	assert.NoError(t, err)
}

func TestPresetRepository_UpdateLastUsedAt(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	p := newCafePreset("preset-use", "사용 테스트")
	require.NoError(t, repo.CreatePreset(ctx, p))

	usedAt := "2026-04-04T12:00:00Z"
	require.NoError(t, repo.UpdateLastUsedAt(ctx, "preset-use", testUserID, usedAt))

	got, err := repo.GetPresetByID(ctx, "preset-use", testUserID)
	require.NoError(t, err)
	require.NotNil(t, got.LastUsedAt)
	assert.Equal(t, usedAt, *got.LastUsedAt)
}

func TestPresetRepository_UpdateNotFound(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	p := newCafePreset("nonexistent", "없는 프리셋")
	err := repo.UpdatePreset(ctx, p)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPresetRepository_UpdateOtherUserDenied(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	p := newCafePreset("preset-upd-deny", "내 프리셋")
	require.NoError(t, repo.CreatePreset(ctx, p))

	// 다른 사용자로 수정 시도
	p.UserID = otherUserID
	err := repo.UpdatePreset(ctx, p)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPresetRepository_UpdateLastUsedAtNotFound(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	err := repo.UpdateLastUsedAt(ctx, "nonexistent", testUserID, "2026-04-04T12:00:00Z")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestPresetRepository_CascadeDelete(t *testing.T) {
	sqlDB := setupPresetTestDB(t)
	repo := NewSQLitePresetRepository(sqlDB)
	ctx := context.Background()

	p := newBrewPreset("preset-cascade", "CASCADE 테스트")
	require.NoError(t, repo.CreatePreset(ctx, p))

	// presets 삭제 시 brew_presets도 CASCADE로 삭제되는지 확인
	require.NoError(t, repo.DeletePreset(ctx, "preset-cascade", testUserID))

	var count int
	err := sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM brew_presets WHERE preset_id = ?", "preset-cascade").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
