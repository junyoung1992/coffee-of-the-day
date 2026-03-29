package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// insertCafeLogWithTags는 주어진 companions와 tasting_tags로 cafe 로그를 삽입한다.
func insertCafeLogWithTags(t *testing.T, db interface{ Exec(string, ...any) (interface{}, error) }, logID, userID string, companions []string, tastingTags []string) {
	t.Helper()
}

// insertTestCafeLogs는 태그 및 동반자 suggestion 테스트를 위한 픽스처 데이터를 삽입한다.
func insertSuggestionFixtures(t *testing.T, db interface {
	Exec(string, ...any) (interface{ RowsAffected() (int64, error) }, error)
}) {
	t.Helper()
}

// ---------------------------------------------------------------------------
// GetTagSuggestions
// ---------------------------------------------------------------------------

func TestGetTagSuggestions_PrefixMatch(t *testing.T) {
	db := setupTestDB(t)
	now := "2026-01-01T00:00:00Z"

	// user-1의 cafe 로그에 다양한 tasting_tags를 삽입한다.
	_, err := db.Exec(
		`INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"log-1", testUserID, now, "[]", "cafe", now, now,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO cafe_logs (log_id, cafe_name, coffee_name, tasting_tags)
		 VALUES (?, ?, ?, ?)`,
		"log-1", "카페A", "에티오피아", `["초콜릿", "체리", "다크초콜릿"]`,
	)
	require.NoError(t, err)

	repo := NewSQLiteSuggestionRepository(db)
	got, err := repo.GetTagSuggestions(context.Background(), testUserID, "초")
	require.NoError(t, err)

	// prefix "초"는 "초콜릿"만 매칭해야 한다. "다크초콜릿"은 포함되면 안 된다.
	assert.Equal(t, []string{"초콜릿"}, got)
}

func TestGetTagSuggestions_CaseInsensitive(t *testing.T) {
	db := setupTestDB(t)
	now := "2026-01-01T00:00:00Z"

	_, err := db.Exec(
		`INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"log-1", testUserID, now, "[]", "cafe", now, now,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO cafe_logs (log_id, cafe_name, coffee_name, tasting_tags)
		 VALUES (?, ?, ?, ?)`,
		"log-1", "카페A", "에티오피아", `["Floral", "Fruity"]`,
	)
	require.NoError(t, err)

	repo := NewSQLiteSuggestionRepository(db)
	got, err := repo.GetTagSuggestions(context.Background(), testUserID, "fl")
	require.NoError(t, err)

	// 소문자 쿼리로 대문자 시작 태그를 조회할 수 있어야 한다.
	assert.Equal(t, []string{"Floral"}, got)
}

func TestGetTagSuggestions_EmptyQ_ReturnsAll(t *testing.T) {
	db := setupTestDB(t)
	now := "2026-01-01T00:00:00Z"

	_, err := db.Exec(
		`INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"log-1", testUserID, now, "[]", "cafe", now, now,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO cafe_logs (log_id, cafe_name, coffee_name, tasting_tags)
		 VALUES (?, ?, ?, ?)`,
		"log-1", "카페A", "에티오피아", `["초콜릿", "체리"]`,
	)
	require.NoError(t, err)

	repo := NewSQLiteSuggestionRepository(db)
	got, err := repo.GetTagSuggestions(context.Background(), testUserID, "")
	require.NoError(t, err)

	// 빈 검색어는 전체 태그를 반환해야 한다.
	assert.Len(t, got, 2)
}

func TestGetTagSuggestions_FrequencyOrder(t *testing.T) {
	db := setupTestDB(t)
	now := "2026-01-01T00:00:00Z"

	// "체리"가 2회, "초콜릿"이 1회 등장하도록 삽입한다.
	for i, tags := range []string{`["체리", "초콜릿"]`, `["체리"]`} {
		logID := "log-" + string(rune('1'+i))
		_, err := db.Exec(
			`INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			logID, testUserID, now, "[]", "cafe", now, now,
		)
		require.NoError(t, err)
		_, err = db.Exec(
			`INSERT INTO cafe_logs (log_id, cafe_name, coffee_name, tasting_tags)
			 VALUES (?, ?, ?, ?)`,
			logID, "카페A", "에티오피아", tags,
		)
		require.NoError(t, err)
	}

	repo := NewSQLiteSuggestionRepository(db)
	got, err := repo.GetTagSuggestions(context.Background(), testUserID, "")
	require.NoError(t, err)

	// 빈도 내림차순이므로 "체리"가 먼저 와야 한다.
	require.Len(t, got, 2)
	assert.Equal(t, "체리", got[0])
}

func TestGetTagSuggestions_OtherUserData_NotReturned(t *testing.T) {
	db := setupTestDB(t)
	now := "2026-01-01T00:00:00Z"

	// otherUserID의 로그에만 태그를 삽입한다.
	_, err := db.Exec(
		`INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"log-other", otherUserID, now, "[]", "cafe", now, now,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO cafe_logs (log_id, cafe_name, coffee_name, tasting_tags)
		 VALUES (?, ?, ?, ?)`,
		"log-other", "카페B", "케냐", `["초콜릿"]`,
	)
	require.NoError(t, err)

	repo := NewSQLiteSuggestionRepository(db)
	got, err := repo.GetTagSuggestions(context.Background(), testUserID, "")
	require.NoError(t, err)

	// 다른 유저의 태그는 반환되지 않아야 한다.
	assert.Empty(t, got)
}

func TestGetTagSuggestions_BrewLog_Included(t *testing.T) {
	db := setupTestDB(t)
	now := "2026-01-01T00:00:00Z"

	// brew 타입 로그에 태그를 삽입한다.
	_, err := db.Exec(
		`INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"log-brew", testUserID, now, "[]", "brew", now, now,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO brew_logs (log_id, bean_name, brew_method, tasting_tags)
		 VALUES (?, ?, ?, ?)`,
		"log-brew", "케냐 AA", "pour_over", `["플로럴", "밝은 산미"]`,
	)
	require.NoError(t, err)

	repo := NewSQLiteSuggestionRepository(db)
	got, err := repo.GetTagSuggestions(context.Background(), testUserID, "플")
	require.NoError(t, err)

	// brew 로그의 태그도 자동완성에 포함되어야 한다.
	assert.Equal(t, []string{"플로럴"}, got)
}

// ---------------------------------------------------------------------------
// GetCompanionSuggestions
// ---------------------------------------------------------------------------

func TestGetCompanionSuggestions_PrefixMatch(t *testing.T) {
	db := setupTestDB(t)
	now := "2026-01-01T00:00:00Z"

	_, err := db.Exec(
		`INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"log-1", testUserID, now, `["지수", "지훈", "민준"]`, "cafe", now, now,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO cafe_logs (log_id, cafe_name, coffee_name) VALUES (?, ?, ?)`,
		"log-1", "카페A", "에티오피아",
	)
	require.NoError(t, err)

	repo := NewSQLiteSuggestionRepository(db)
	got, err := repo.GetCompanionSuggestions(context.Background(), testUserID, "지")
	require.NoError(t, err)

	// prefix "지"는 "지수", "지훈"만 매칭해야 한다. "민준"은 포함되면 안 된다.
	assert.Len(t, got, 2)
	assert.Contains(t, got, "지수")
	assert.Contains(t, got, "지훈")
}

func TestGetCompanionSuggestions_EmptyQ_ReturnsAll(t *testing.T) {
	db := setupTestDB(t)
	now := "2026-01-01T00:00:00Z"

	_, err := db.Exec(
		`INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"log-1", testUserID, now, `["지수", "민준"]`, "cafe", now, now,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO cafe_logs (log_id, cafe_name, coffee_name) VALUES (?, ?, ?)`,
		"log-1", "카페A", "에티오피아",
	)
	require.NoError(t, err)

	repo := NewSQLiteSuggestionRepository(db)
	got, err := repo.GetCompanionSuggestions(context.Background(), testUserID, "")
	require.NoError(t, err)

	assert.Len(t, got, 2)
}

func TestGetCompanionSuggestions_OtherUserData_NotReturned(t *testing.T) {
	db := setupTestDB(t)
	now := "2026-01-01T00:00:00Z"

	_, err := db.Exec(
		`INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"log-other", otherUserID, now, `["지수"]`, "cafe", now, now,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO cafe_logs (log_id, cafe_name, coffee_name) VALUES (?, ?, ?)`,
		"log-other", "카페B", "케냐",
	)
	require.NoError(t, err)

	repo := NewSQLiteSuggestionRepository(db)
	got, err := repo.GetCompanionSuggestions(context.Background(), testUserID, "")
	require.NoError(t, err)

	assert.Empty(t, got)
}
