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

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const testUserID = "user-1"
const otherUserID = "user-2"

// setupTestDB creates an in-memory SQLite database, runs all migrations, and
// inserts test users. It returns the open *sql.DB.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	// Enable foreign keys.
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	// Run migrations in order.
	migrationsDir := filepath.Join("..", "..", "db", "migrations")
	for _, name := range []string{
		"001_create_users.up.sql",
		"002_create_coffee_logs.up.sql",
		"003_create_cafe_logs.up.sql",
		"004_create_brew_logs.up.sql",
	} {
		raw, err := os.ReadFile(filepath.Join(migrationsDir, name))
		require.NoError(t, err, "reading migration %s", name)
		_, err = db.Exec(string(raw))
		require.NoError(t, err, "running migration %s", name)
	}

	// Insert test users.
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

// ---------------------------------------------------------------------------
// Test data builders
// ---------------------------------------------------------------------------

func ptrStr(s string) *string                         { return &s }
func ptrFloat(f float64) *float64                     { return &f }
func ptrInt(i int) *int                               { return &i }
func ptrRoast(r domain.RoastLevel) *domain.RoastLevel { return &r }

func newCafeLog(id, recordedAt string) domain.CoffeeLogFull {
	return domain.CoffeeLogFull{
		CoffeeLog: domain.CoffeeLog{
			ID:         id,
			UserID:     testUserID,
			RecordedAt: recordedAt,
			Companions: []string{"Alice"},
			LogType:    domain.LogTypeCafe,
			Memo:       ptrStr("좋은 카페"),
			CreatedAt:  "2026-03-15T10:00:00Z",
			UpdatedAt:  "2026-03-15T10:00:00Z",
		},
		Cafe: &domain.CafeDetail{
			CafeName:    "블루보틀",
			Location:    ptrStr("성수, 서울"),
			CoffeeName:  "에티오피아 예가체프",
			BeanOrigin:  ptrStr("Ethiopia"),
			BeanProcess: ptrStr("washed"),
			RoastLevel:  ptrRoast(domain.RoastLight),
			TastingTags: []string{"fruity", "sweet"},
			TastingNote: ptrStr("밝은 산미와 블루베리 노트"),
			Impressions: ptrStr("부드럽고 균형잡힌 맛"),
			Rating:      ptrFloat(4.5),
		},
	}
}

func newBrewLog(id, recordedAt string) domain.CoffeeLogFull {
	return domain.CoffeeLogFull{
		CoffeeLog: domain.CoffeeLog{
			ID:         id,
			UserID:     testUserID,
			RecordedAt: recordedAt,
			Companions: []string{},
			LogType:    domain.LogTypeBrew,
			Memo:       nil,
			CreatedAt:  "2026-03-16T08:00:00Z",
			UpdatedAt:  "2026-03-16T08:00:00Z",
		},
		Brew: &domain.BrewDetail{
			BeanName:      "에티오피아 코체레",
			BeanOrigin:    ptrStr("Ethiopia"),
			BeanProcess:   ptrStr("natural"),
			RoastLevel:    ptrRoast(domain.RoastMedium),
			RoastDate:     ptrStr("2026-03-10"),
			TastingTags:   []string{"floral"},
			TastingNote:   ptrStr("자스민과 레몬 제스트"),
			BrewMethod:    domain.BrewMethodPourOver,
			BrewDevice:    ptrStr("Hario V60"),
			CoffeeAmountG: ptrFloat(18.0),
			WaterAmountMl: ptrFloat(300.0),
			WaterTempC:    ptrFloat(93.0),
			BrewTimeSec:   ptrInt(180),
			GrindSize:     ptrStr("medium-fine"),
			BrewSteps:     []string{"bloom 30s", "pour 200ml"},
			Impressions:   ptrStr("깔끔한 컵, 좋은 투명도"),
			Rating:        ptrFloat(4.0),
		},
	}
}

// ---------------------------------------------------------------------------
// 1. CreateLog + GetLogByID — cafe log round-trip
// ---------------------------------------------------------------------------

func TestCreateAndGetCafeLog_RoundTrip(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteLogRepository(db)
	ctx := context.Background()

	cafe := newCafeLog("log-cafe-001", "2026-03-15")

	err := repo.CreateLog(ctx, cafe)
	require.NoError(t, err)

	got, err := repo.GetLogByID(ctx, "log-cafe-001", testUserID)
	require.NoError(t, err)

	// Common fields.
	assert.Equal(t, cafe.ID, got.ID)
	assert.Equal(t, cafe.UserID, got.UserID)
	assert.Equal(t, cafe.RecordedAt, got.RecordedAt)
	assert.Equal(t, cafe.Companions, got.Companions)
	assert.Equal(t, cafe.LogType, got.LogType)
	assert.Equal(t, cafe.Memo, got.Memo)

	// Cafe detail.
	require.NotNil(t, got.Cafe)
	assert.Nil(t, got.Brew)
	assert.Equal(t, cafe.Cafe.CafeName, got.Cafe.CafeName)
	assert.Equal(t, cafe.Cafe.Location, got.Cafe.Location)
	assert.Equal(t, cafe.Cafe.CoffeeName, got.Cafe.CoffeeName)
	assert.Equal(t, cafe.Cafe.BeanOrigin, got.Cafe.BeanOrigin)
	assert.Equal(t, cafe.Cafe.BeanProcess, got.Cafe.BeanProcess)
	assert.Equal(t, cafe.Cafe.RoastLevel, got.Cafe.RoastLevel)
	assert.Equal(t, cafe.Cafe.TastingTags, got.Cafe.TastingTags)
	assert.Equal(t, cafe.Cafe.TastingNote, got.Cafe.TastingNote)
	assert.Equal(t, cafe.Cafe.Impressions, got.Cafe.Impressions)
	assert.Equal(t, cafe.Cafe.Rating, got.Cafe.Rating)
}

// ---------------------------------------------------------------------------
// 2. CreateLog + GetLogByID — brew log round-trip
// ---------------------------------------------------------------------------

func TestCreateAndGetBrewLog_RoundTrip(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteLogRepository(db)
	ctx := context.Background()

	brew := newBrewLog("log-brew-001", "2026-03-16")

	err := repo.CreateLog(ctx, brew)
	require.NoError(t, err)

	got, err := repo.GetLogByID(ctx, "log-brew-001", testUserID)
	require.NoError(t, err)

	// Common fields.
	assert.Equal(t, brew.ID, got.ID)
	assert.Equal(t, brew.UserID, got.UserID)
	assert.Equal(t, brew.LogType, got.LogType)

	// Brew detail.
	require.NotNil(t, got.Brew)
	assert.Nil(t, got.Cafe)
	assert.Equal(t, brew.Brew.BeanName, got.Brew.BeanName)
	assert.Equal(t, brew.Brew.BeanOrigin, got.Brew.BeanOrigin)
	assert.Equal(t, brew.Brew.BeanProcess, got.Brew.BeanProcess)
	assert.Equal(t, brew.Brew.RoastLevel, got.Brew.RoastLevel)
	assert.Equal(t, brew.Brew.RoastDate, got.Brew.RoastDate)
	assert.Equal(t, brew.Brew.TastingTags, got.Brew.TastingTags)
	assert.Equal(t, brew.Brew.TastingNote, got.Brew.TastingNote)
	assert.Equal(t, brew.Brew.BrewMethod, got.Brew.BrewMethod)
	assert.Equal(t, brew.Brew.BrewDevice, got.Brew.BrewDevice)
	assert.Equal(t, brew.Brew.CoffeeAmountG, got.Brew.CoffeeAmountG)
	assert.Equal(t, brew.Brew.WaterAmountMl, got.Brew.WaterAmountMl)
	assert.Equal(t, brew.Brew.WaterTempC, got.Brew.WaterTempC)
	assert.Equal(t, brew.Brew.BrewTimeSec, got.Brew.BrewTimeSec)
	assert.Equal(t, brew.Brew.GrindSize, got.Brew.GrindSize)
	assert.Equal(t, brew.Brew.BrewSteps, got.Brew.BrewSteps)
	assert.Equal(t, brew.Brew.Impressions, got.Brew.Impressions)
	assert.Equal(t, brew.Brew.Rating, got.Brew.Rating)
}

// ---------------------------------------------------------------------------
// 3. GetLogByID — non-existent ID returns ErrNotFound
// ---------------------------------------------------------------------------

func TestGetLogByID_NonExistentID_ReturnsErrNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteLogRepository(db)
	ctx := context.Background()

	_, err := repo.GetLogByID(ctx, "does-not-exist", testUserID)
	assert.ErrorIs(t, err, ErrNotFound)
}

// ---------------------------------------------------------------------------
// 4. GetLogByID — wrong userID returns ErrNotFound
// ---------------------------------------------------------------------------

func TestGetLogByID_WrongUserID_ReturnsErrNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteLogRepository(db)
	ctx := context.Background()

	cafe := newCafeLog("log-cafe-002", "2026-03-15")
	err := repo.CreateLog(ctx, cafe)
	require.NoError(t, err)

	_, err = repo.GetLogByID(ctx, "log-cafe-002", otherUserID)
	assert.ErrorIs(t, err, ErrNotFound)
}

// ---------------------------------------------------------------------------
// 5. ListLogs — no filter, returns all user logs in recorded_at DESC order
// ---------------------------------------------------------------------------

func TestListLogs_NoFilter_ReturnsAllUserLogsDescOrder(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteLogRepository(db)
	ctx := context.Background()

	cafe := newCafeLog("log-list-001", "2026-03-15")
	brew := newBrewLog("log-list-002", "2026-03-16")
	require.NoError(t, repo.CreateLog(ctx, cafe))
	require.NoError(t, repo.CreateLog(ctx, brew))

	result, err := repo.ListLogs(ctx, testUserID, ListFilter{Limit: 20})
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// recorded_at DESC: brew (03-16) should come before cafe (03-15).
	assert.Equal(t, "log-list-002", result[0].ID)
	assert.Equal(t, "log-list-001", result[1].ID)
}

// ---------------------------------------------------------------------------
// 6. ListLogs — log_type filter
// ---------------------------------------------------------------------------

func TestListLogs_FilterByLogType_ReturnsOnlyMatchingType(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteLogRepository(db)
	ctx := context.Background()

	cafe := newCafeLog("log-type-001", "2026-03-15")
	brew := newBrewLog("log-type-002", "2026-03-16")
	require.NoError(t, repo.CreateLog(ctx, cafe))
	require.NoError(t, repo.CreateLog(ctx, brew))

	cafeType := "cafe"
	result, err := repo.ListLogs(ctx, testUserID, ListFilter{LogType: &cafeType, Limit: 20})
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, domain.LogTypeCafe, result[0].LogType)
}

// ---------------------------------------------------------------------------
// 7. ListLogs — date range filter
// ---------------------------------------------------------------------------

func TestListLogs_FilterByDateRange_ReturnsLogsInRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteLogRepository(db)
	ctx := context.Background()

	log1 := newCafeLog("log-date-001", "2026-03-10")
	log2 := newCafeLog("log-date-002", "2026-03-15")
	log2.Cafe.CafeName = "Second Cafe"
	log3 := newCafeLog("log-date-003", "2026-03-20")
	log3.Cafe.CafeName = "Third Cafe"
	require.NoError(t, repo.CreateLog(ctx, log1))
	require.NoError(t, repo.CreateLog(ctx, log2))
	require.NoError(t, repo.CreateLog(ctx, log3))

	from := "2026-03-12"
	to := "2026-03-18"
	result, err := repo.ListLogs(ctx, testUserID, ListFilter{DateFrom: &from, DateTo: &to, Limit: 20})
	require.NoError(t, err)

	// Only log2 (2026-03-15) should be in range.
	assert.Len(t, result, 1)
	assert.Equal(t, "log-date-002", result[0].ID)
}

// ---------------------------------------------------------------------------
// 8. ListLogs — cursor-based pagination
// ---------------------------------------------------------------------------

func TestListLogs_CursorPagination_ReturnsCorrectPages(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteLogRepository(db)
	ctx := context.Background()

	// Create 5 logs with distinct recorded_at dates.
	dates := []string{"2026-03-21", "2026-03-22", "2026-03-23", "2026-03-24", "2026-03-25"}
	for i, date := range dates {
		id := "log-page-" + string(rune('A'+i))
		log := newCafeLog(id, date)
		log.Cafe.CafeName = "Cafe " + string(rune('A'+i))
		require.NoError(t, repo.CreateLog(ctx, log))
	}

	// First page: limit=2, no cursor.
	page1, err := repo.ListLogs(ctx, testUserID, ListFilter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page1, 2, "first page should have 2 items")

	// Build cursor from last item of first page for second page.
	lastItem := page1[len(page1)-1]
	cursor1 := &Cursor{
		SortBy:    "recorded_at",
		Order:     "desc",
		SortValue: lastItem.RecordedAt,
		ID:        lastItem.ID,
	}

	// Second page: limit=2, with cursor.
	page2, err := repo.ListLogs(ctx, testUserID, ListFilter{Cursor: cursor1, Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page2, 2, "second page should have 2 items")

	// Build cursor from last item of second page for third page.
	lastItem2 := page2[len(page2)-1]
	cursor2 := &Cursor{
		SortBy:    "recorded_at",
		Order:     "desc",
		SortValue: lastItem2.RecordedAt,
		ID:        lastItem2.ID,
	}

	// Third page: should have 1 remaining item.
	page3, err := repo.ListLogs(ctx, testUserID, ListFilter{Cursor: cursor2, Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page3, 1, "last page should have 1 item")

	// All IDs across pages should be unique (no duplicates, no gaps).
	seen := map[string]bool{}
	for _, page := range [][]domain.CoffeeLogFull{page1, page2, page3} {
		for _, item := range page {
			assert.False(t, seen[item.ID], "duplicate ID %s across pages", item.ID)
			seen[item.ID] = true
		}
	}
	assert.Len(t, seen, 5, "all 5 logs should appear across pages")
}

// ---------------------------------------------------------------------------
// 9. UpdateLog — updates common + sub-table fields
// ---------------------------------------------------------------------------

func TestUpdateLog_UpdatesCommonAndSubTableFields(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteLogRepository(db)
	ctx := context.Background()

	original := newCafeLog("log-update-001", "2026-03-15")
	require.NoError(t, repo.CreateLog(ctx, original))

	// Modify fields.
	updated := original
	updated.RecordedAt = "2026-03-16"
	updated.Companions = []string{"Charlie"}
	updated.Memo = ptrStr("Updated memo")
	updated.UpdatedAt = "2026-03-16T12:00:00Z"
	updated.Cafe.CafeName = "스타벅스"
	updated.Cafe.Rating = ptrFloat(3.0)
	updated.Cafe.TastingTags = []string{"bitter", "smoky"}

	err := repo.UpdateLog(ctx, updated)
	require.NoError(t, err)

	got, err := repo.GetLogByID(ctx, "log-update-001", testUserID)
	require.NoError(t, err)

	assert.Equal(t, "2026-03-16", got.RecordedAt)
	assert.Equal(t, []string{"Charlie"}, got.Companions)
	assert.Equal(t, ptrStr("Updated memo"), got.Memo)
	require.NotNil(t, got.Cafe)
	assert.Equal(t, "스타벅스", got.Cafe.CafeName)
	assert.Equal(t, ptrFloat(3.0), got.Cafe.Rating)
	assert.Equal(t, []string{"bitter", "smoky"}, got.Cafe.TastingTags)
}

// ---------------------------------------------------------------------------
// 10. DeleteLog — removes log and cascades to sub-table
// ---------------------------------------------------------------------------

func TestDeleteLog_RemovesLogAndCascadesToSubTable(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSQLiteLogRepository(db)
	ctx := context.Background()

	cafe := newCafeLog("log-delete-001", "2026-03-15")
	require.NoError(t, repo.CreateLog(ctx, cafe))

	// Verify it exists first.
	_, err := repo.GetLogByID(ctx, "log-delete-001", testUserID)
	require.NoError(t, err)

	// Delete.
	err = repo.DeleteLog(ctx, "log-delete-001", testUserID)
	require.NoError(t, err)

	// Verify the log is gone.
	_, err = repo.GetLogByID(ctx, "log-delete-001", testUserID)
	assert.ErrorIs(t, err, ErrNotFound)

	// Verify the cafe sub-table row is also gone (CASCADE).
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cafe_logs WHERE log_id = ?", "log-delete-001").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "cafe_logs row should be deleted by CASCADE")
}
