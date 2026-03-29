package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// SuggestionRepository는 자동완성 집계 데이터를 조회하는 인터페이스다.
type SuggestionRepository interface {
	// GetTagSuggestions는 해당 유저의 과거 tasting_tags를 빈도순으로 반환한다.
	GetTagSuggestions(ctx context.Context, userID, q string) ([]string, error)
	// GetCompanionSuggestions는 해당 유저의 과거 companions를 빈도순으로 반환한다.
	GetCompanionSuggestions(ctx context.Context, userID, q string) ([]string, error)
}

// SQLiteSuggestionRepository는 SQLite 기반 SuggestionRepository 구현체다.
// json_each() 가상 테이블을 사용하는 쿼리는 sqlc가 지원하지 않아 raw SQL로 실행한다.
type SQLiteSuggestionRepository struct {
	db *sql.DB
}

// NewSQLiteSuggestionRepository는 SQLiteSuggestionRepository를 생성한다.
func NewSQLiteSuggestionRepository(db *sql.DB) *SQLiteSuggestionRepository {
	return &SQLiteSuggestionRepository{db: db}
}

const tagSuggestionsQuery = `
WITH all_tags AS (
    SELECT j.value AS tag
    FROM cafe_logs cl
    JOIN coffee_logs l ON l.id = cl.log_id
    JOIN json_each(cl.tasting_tags) j
    WHERE l.user_id = ?
    UNION ALL
    SELECT j.value AS tag
    FROM brew_logs bl
    JOIN coffee_logs l ON l.id = bl.log_id
    JOIN json_each(bl.tasting_tags) j
    WHERE l.user_id = ?
)
SELECT tag, COUNT(*) AS cnt
FROM all_tags
WHERE ? = '' OR LOWER(tag) LIKE LOWER(?) || '%'
GROUP BY tag
ORDER BY cnt DESC, tag ASC
LIMIT 10
`

func (r *SQLiteSuggestionRepository) GetTagSuggestions(ctx context.Context, userID, q string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, tagSuggestionsQuery, userID, userID, q, q)
	if err != nil {
		return nil, fmt.Errorf("get tag suggestions: %w", err)
	}
	defer rows.Close()

	return scanSuggestions(rows)
}

const companionSuggestionsQuery = `
SELECT j.value AS companion, COUNT(*) AS cnt
FROM coffee_logs l
JOIN json_each(l.companions) j
WHERE l.user_id = ?
  AND (? = '' OR LOWER(j.value) LIKE LOWER(?) || '%')
GROUP BY companion
ORDER BY cnt DESC, companion ASC
LIMIT 10
`

func (r *SQLiteSuggestionRepository) GetCompanionSuggestions(ctx context.Context, userID, q string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, companionSuggestionsQuery, userID, q, q)
	if err != nil {
		return nil, fmt.Errorf("get companion suggestions: %w", err)
	}
	defer rows.Close()

	return scanSuggestions(rows)
}

// scanSuggestions는 (value TEXT, cnt INTEGER) 형식의 rows를 문자열 슬라이스로 변환한다.
func scanSuggestions(rows *sql.Rows) ([]string, error) {
	var suggestions []string
	for rows.Next() {
		var value string
		var cnt int64
		if err := rows.Scan(&value, &cnt); err != nil {
			return nil, fmt.Errorf("scan suggestion row: %w", err)
		}
		suggestions = append(suggestions, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate suggestion rows: %w", err)
	}
	if suggestions == nil {
		return []string{}, nil
	}
	return suggestions, nil
}
