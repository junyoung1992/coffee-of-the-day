package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"coffee-of-the-day/backend/internal/db"
)

var (
	// ErrUserNotFound는 이메일 또는 ID로 사용자를 찾지 못했을 때 반환된다.
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailTaken은 이미 등록된 이메일로 가입을 시도할 때 반환된다.
	ErrEmailTaken = errors.New("email already taken")
)

// UserRecord는 UserRepository가 반환하는 사용자 레코드다.
// PasswordHash는 서비스 계층에서 bcrypt 검증 목적으로만 사용한다.
// TokenVersion은 리프레시 토큰 무효화에 사용한다 — 로그아웃 시 증가시킨다.
type UserRecord struct {
	ID           string
	Username     string
	DisplayName  string
	Email        *string
	PasswordHash *string
	CreatedAt    string
	TokenVersion int64
}

// CreateUserParams는 사용자 생성 시 필요한 파라미터다.
type CreateUserParams struct {
	ID           string
	Username     string
	DisplayName  string
	Email        string
	PasswordHash string
}

// UserRepository는 사용자 영속성 인터페이스를 정의한다.
type UserRepository interface {
	CreateUser(ctx context.Context, params CreateUserParams) (UserRecord, error)
	GetUserByEmail(ctx context.Context, email string) (UserRecord, error)
	GetUserByID(ctx context.Context, id string) (UserRecord, error)
	IncrementTokenVersion(ctx context.Context, id string) error
}

// SQLiteUserRepository는 SQLite 기반 UserRepository 구현체다.
type SQLiteUserRepository struct {
	queries *db.Queries
}

// NewSQLiteUserRepository는 SQLiteUserRepository를 생성한다.
func NewSQLiteUserRepository(sqlDB *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{queries: db.New(sqlDB)}
}

func (r *SQLiteUserRepository) CreateUser(ctx context.Context, params CreateUserParams) (UserRecord, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	user, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		ID:           params.ID,
		Username:     params.Username,
		DisplayName:  params.DisplayName,
		Email:        &params.Email,
		PasswordHash: &params.PasswordHash,
		CreatedAt:    now,
	})
	if err != nil {
		// SQLite UNIQUE 제약 위반: 이미 사용 중인 이메일
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return UserRecord{}, ErrEmailTaken
		}
		return UserRecord{}, fmt.Errorf("create user: %w", err)
	}
	return dbUserToRecord(user), nil
}

func (r *SQLiteUserRepository) GetUserByEmail(ctx context.Context, email string) (UserRecord, error) {
	// sqlc는 nullable TEXT 컬럼을 *string으로 생성하므로 포인터로 전달한다
	user, err := r.queries.GetUserByEmail(ctx, &email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserRecord{}, ErrUserNotFound
		}
		return UserRecord{}, fmt.Errorf("get user by email: %w", err)
	}
	return dbUserToRecord(user), nil
}

func (r *SQLiteUserRepository) GetUserByID(ctx context.Context, id string) (UserRecord, error) {
	user, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserRecord{}, ErrUserNotFound
		}
		return UserRecord{}, fmt.Errorf("get user by id: %w", err)
	}
	return dbUserToRecord(user), nil
}

func (r *SQLiteUserRepository) IncrementTokenVersion(ctx context.Context, id string) error {
	return r.queries.IncrementTokenVersion(ctx, id)
}

func dbUserToRecord(u db.User) UserRecord {
	return UserRecord{
		ID:           u.ID,
		Username:     u.Username,
		DisplayName:  u.DisplayName,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
		TokenVersion: u.TokenVersion,
	}
}
