package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/repository"
)

// stubUserRepository는 테스트에서 UserRepository를 대체하는 스텁이다.
type stubUserRepository struct {
	createFunc func(ctx context.Context, params repository.CreateUserParams) (repository.UserRecord, error)
	getByEmail func(ctx context.Context, email string) (repository.UserRecord, error)
	getByID    func(ctx context.Context, id string) (repository.UserRecord, error)
}

func (s *stubUserRepository) CreateUser(ctx context.Context, params repository.CreateUserParams) (repository.UserRecord, error) {
	return s.createFunc(ctx, params)
}
func (s *stubUserRepository) GetUserByEmail(ctx context.Context, email string) (repository.UserRecord, error) {
	return s.getByEmail(ctx, email)
}
func (s *stubUserRepository) GetUserByID(ctx context.Context, id string) (repository.UserRecord, error) {
	return s.getByID(ctx, id)
}

const testJWTSecret = "test-secret"

func newTestAuthService(repo repository.UserRepository) *DefaultAuthService {
	svc := NewAuthService(repo, testJWTSecret)
	svc.now = func() time.Time { return time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC) }
	return svc
}

func TestAuthService_Register_Success(t *testing.T) {
	email := "test@example.com"
	repo := &stubUserRepository{
		createFunc: func(_ context.Context, params repository.CreateUserParams) (repository.UserRecord, error) {
			return repository.UserRecord{
				ID:          "new-id",
				Username:    params.Username,
				DisplayName: params.DisplayName,
				Email:       &params.Email,
			}, nil
		},
	}

	svc := newTestAuthService(repo)
	user, tokens, err := svc.Register(context.Background(), domain.RegisterRequest{
		Email:    email,
		Password: "password123",
		Username: "testuser",
	})

	require.NoError(t, err)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, "testuser", user.Username)
	// display_name 미입력 시 username으로 대체된다
	assert.Equal(t, "testuser", user.DisplayName)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestAuthService_Register_EmailTaken(t *testing.T) {
	repo := &stubUserRepository{
		createFunc: func(_ context.Context, _ repository.CreateUserParams) (repository.UserRecord, error) {
			return repository.UserRecord{}, repository.ErrEmailTaken
		},
	}

	svc := newTestAuthService(repo)
	_, _, err := svc.Register(context.Background(), domain.RegisterRequest{
		Email:    "taken@example.com",
		Password: "password123",
		Username: "testuser",
	})

	assert.ErrorIs(t, err, ErrEmailTaken)
}

func TestAuthService_Register_ValidationErrors(t *testing.T) {
	svc := newTestAuthService(&stubUserRepository{})

	tests := []struct {
		name  string
		req   domain.RegisterRequest
		field string
	}{
		{"invalid email", domain.RegisterRequest{Email: "notanemail", Password: "password123", Username: "u"}, "email"},
		{"short password", domain.RegisterRequest{Email: "a@b.com", Password: "short", Username: "u"}, "password"},
		{"empty username", domain.RegisterRequest{Email: "a@b.com", Password: "password123", Username: ""}, "username"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := svc.Register(context.Background(), tc.req)
			var ve *ValidationError
			require.True(t, errors.As(err, &ve), "expected ValidationError")
			assert.Equal(t, tc.field, ve.Field)
		})
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	email := "test@example.com"
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	require.NoError(t, err)
	hashStr := string(hash)

	repo := &stubUserRepository{
		getByEmail: func(_ context.Context, _ string) (repository.UserRecord, error) {
			return repository.UserRecord{
				ID:           "user-1",
				Username:     "testuser",
				DisplayName:  "Test User",
				Email:        &email,
				PasswordHash: &hashStr,
			}, nil
		},
	}

	svc := newTestAuthService(repo)
	user, tokens, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    email,
		Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, "user-1", user.ID)
	assert.NotEmpty(t, tokens.AccessToken)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	email := "test@example.com"
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	hashStr := string(hash)

	repo := &stubUserRepository{
		getByEmail: func(_ context.Context, _ string) (repository.UserRecord, error) {
			return repository.UserRecord{
				ID:           "user-1",
				Email:        &email,
				PasswordHash: &hashStr,
			}, nil
		},
	}

	svc := newTestAuthService(repo)
	_, _, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    email,
		Password: "wrong-password",
	})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := &stubUserRepository{
		getByEmail: func(_ context.Context, _ string) (repository.UserRecord, error) {
			return repository.UserRecord{}, repository.ErrUserNotFound
		},
	}

	svc := newTestAuthService(repo)
	_, _, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    "nobody@example.com",
		Password: "password123",
	})

	// 사용자 존재 여부를 노출하지 않고 동일한 오류를 반환해야 한다
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Refresh_Success(t *testing.T) {
	repo := &stubUserRepository{
		getByID: func(_ context.Context, id string) (repository.UserRecord, error) {
			return repository.UserRecord{ID: id}, nil
		},
	}

	svc := newTestAuthService(repo)
	original, err := svc.generateTokens("user-1")
	require.NoError(t, err)

	renewed, err := svc.Refresh(context.Background(), original.RefreshToken)

	require.NoError(t, err)
	assert.NotEmpty(t, renewed.AccessToken)
	assert.NotEmpty(t, renewed.RefreshToken)
}

func TestAuthService_Refresh_AccessTokenRejected(t *testing.T) {
	svc := newTestAuthService(&stubUserRepository{})
	original, err := svc.generateTokens("user-1")
	require.NoError(t, err)

	// 액세스 토큰을 리프레시 자리에 사용하면 거부되어야 한다
	_, err = svc.Refresh(context.Background(), original.AccessToken)

	assert.ErrorIs(t, err, ErrInvalidToken)
}
