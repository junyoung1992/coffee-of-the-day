package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupUserTestDB는 006 마이그레이션(token_version 컬럼)까지 적용된 DB를 반환한다.
func setupUserTestDB(t *testing.T) *SQLiteUserRepository {
	t.Helper()
	db := setupTestDB(t)

	for _, file := range []string{
		"005_add_auth_to_users.up.sql",
		"006_add_token_version_to_users.up.sql",
	} {
		migration, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", file))
		require.NoError(t, err)
		_, err = db.Exec(string(migration))
		require.NoError(t, err)
	}

	return NewSQLiteUserRepository(db)
}

// --- CreateUser 테스트 ---

func TestUserRepository_CreateUser_Success(t *testing.T) {
	repo := setupUserTestDB(t)

	rec, err := repo.CreateUser(context.Background(), CreateUserParams{
		ID:           "new-user-1",
		Username:     "newuser",
		DisplayName:  "New User",
		Email:        "new@example.com",
		PasswordHash: "hashed-password",
	})

	require.NoError(t, err)
	assert.Equal(t, "new-user-1", rec.ID)
	assert.Equal(t, "newuser", rec.Username)
	require.NotNil(t, rec.Email)
	assert.Equal(t, "new@example.com", *rec.Email)
	require.NotNil(t, rec.PasswordHash)
	assert.Equal(t, "hashed-password", *rec.PasswordHash)
	assert.NotEmpty(t, rec.CreatedAt)
}

func TestUserRepository_CreateUser_DuplicateEmail_ReturnsErrEmailTaken(t *testing.T) {
	repo := setupUserTestDB(t)

	params := CreateUserParams{
		ID:           "user-a",
		Username:     "usera",
		DisplayName:  "User A",
		Email:        "dup@example.com",
		PasswordHash: "hash",
	}
	_, err := repo.CreateUser(context.Background(), params)
	require.NoError(t, err)

	// 동일 이메일로 두 번째 생성 시도
	params.ID = "user-b"
	params.Username = "userb"
	_, err = repo.CreateUser(context.Background(), params)

	assert.ErrorIs(t, err, ErrEmailTaken)
}

func TestUserRepository_CreateUser_DuplicateUsername_Fails(t *testing.T) {
	repo := setupUserTestDB(t)

	// 기존 testUserID("user-1")는 username="testuser"로 setupTestDB에서 생성됨
	_, err := repo.CreateUser(context.Background(), CreateUserParams{
		ID:           "user-x",
		Username:     "testuser", // 중복 username
		DisplayName:  "X",
		Email:        "x@example.com",
		PasswordHash: "hash",
	})

	assert.Error(t, err)
}

// --- GetUserByEmail 테스트 ---

func TestUserRepository_GetUserByEmail_Success(t *testing.T) {
	repo := setupUserTestDB(t)

	_, err := repo.CreateUser(context.Background(), CreateUserParams{
		ID:           "email-user",
		Username:     "emailuser",
		DisplayName:  "Email User",
		Email:        "find@example.com",
		PasswordHash: "hash",
	})
	require.NoError(t, err)

	rec, err := repo.GetUserByEmail(context.Background(), "find@example.com")

	require.NoError(t, err)
	assert.Equal(t, "email-user", rec.ID)
	require.NotNil(t, rec.Email)
	assert.Equal(t, "find@example.com", *rec.Email)
}

func TestUserRepository_GetUserByEmail_NotFound_ReturnsErrUserNotFound(t *testing.T) {
	repo := setupUserTestDB(t)

	_, err := repo.GetUserByEmail(context.Background(), "nobody@example.com")

	assert.ErrorIs(t, err, ErrUserNotFound)
}

// --- GetUserByID 테스트 ---

func TestUserRepository_GetUserByID_Success(t *testing.T) {
	repo := setupUserTestDB(t)

	_, err := repo.CreateUser(context.Background(), CreateUserParams{
		ID:           "id-user",
		Username:     "iduser",
		DisplayName:  "ID User",
		Email:        "id@example.com",
		PasswordHash: "hash",
	})
	require.NoError(t, err)

	rec, err := repo.GetUserByID(context.Background(), "id-user")

	require.NoError(t, err)
	assert.Equal(t, "id-user", rec.ID)
}

func TestUserRepository_GetUserByID_NotFound_ReturnsErrUserNotFound(t *testing.T) {
	repo := setupUserTestDB(t)

	_, err := repo.GetUserByID(context.Background(), "nonexistent-id")

	assert.ErrorIs(t, err, ErrUserNotFound)
}

// POC 시드 사용자(email=NULL)는 GetUserByEmail로 조회되지 않아야 한다
func TestUserRepository_GetUserByEmail_NullEmailUser_NotReturned(t *testing.T) {
	repo := setupUserTestDB(t)

	// setupTestDB에서 생성된 testUserID("user-1")는 email=NULL이다
	_, err := repo.GetUserByEmail(context.Background(), "")

	assert.ErrorIs(t, err, ErrUserNotFound)
}
