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
	createFunc           func(ctx context.Context, params repository.CreateUserParams) (repository.UserRecord, error)
	getByEmail           func(ctx context.Context, email string) (repository.UserRecord, error)
	getByID              func(ctx context.Context, id string) (repository.UserRecord, error)
	incrementTokenVersion func(ctx context.Context, id string) error
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
func (s *stubUserRepository) IncrementTokenVersion(ctx context.Context, id string) error {
	if s.incrementTokenVersion != nil {
		return s.incrementTokenVersion(ctx, id)
	}
	return nil
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
	const tokenVersion int64 = 0
	repo := &stubUserRepository{
		getByID: func(_ context.Context, id string) (repository.UserRecord, error) {
			return repository.UserRecord{ID: id, TokenVersion: tokenVersion}, nil
		},
	}

	svc := newTestAuthService(repo)
	original, err := svc.generateTokens("user-1", tokenVersion)
	require.NoError(t, err)

	renewed, err := svc.Refresh(context.Background(), original.RefreshToken)

	require.NoError(t, err)
	assert.NotEmpty(t, renewed.AccessToken)
	assert.NotEmpty(t, renewed.RefreshToken)
}

func TestAuthService_Refresh_AccessTokenRejected(t *testing.T) {
	svc := newTestAuthService(&stubUserRepository{})
	original, err := svc.generateTokens("user-1", 0)
	require.NoError(t, err)

	// 액세스 토큰을 리프레시 자리에 사용하면 거부되어야 한다
	_, err = svc.Refresh(context.Background(), original.AccessToken)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestAuthService_Refresh_RevokedAfterLogout(t *testing.T) {
	// 로그아웃 후 이전 리프레시 토큰으로 갱신을 시도하면 거부되어야 한다.
	// DB의 token_version이 증가했으므로 토큰 클레임과 불일치한다.
	const oldVersion int64 = 0
	const newVersion int64 = 1

	repo := &stubUserRepository{
		getByID: func(_ context.Context, id string) (repository.UserRecord, error) {
			// 로그아웃 후 DB에는 version=1이 저장된 상태
			return repository.UserRecord{ID: id, TokenVersion: newVersion}, nil
		},
	}

	svc := newTestAuthService(repo)
	// 로그아웃 전에 발급된 리프레시 토큰 (version=0)
	oldTokens, err := svc.generateTokens("user-1", oldVersion)
	require.NoError(t, err)

	_, err = svc.Refresh(context.Background(), oldTokens.RefreshToken)

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestAuthService_Logout_IncrementsTokenVersion(t *testing.T) {
	incrementCalled := false
	var incrementedID string

	repo := &stubUserRepository{
		getByID: func(_ context.Context, id string) (repository.UserRecord, error) {
			return repository.UserRecord{ID: id, TokenVersion: 0}, nil
		},
		incrementTokenVersion: func(_ context.Context, id string) error {
			incrementCalled = true
			incrementedID = id
			return nil
		},
	}

	svc := newTestAuthService(repo)
	tokens, err := svc.generateTokens("user-1", 0)
	require.NoError(t, err)

	err = svc.Logout(context.Background(), tokens.RefreshToken)

	require.NoError(t, err)
	assert.True(t, incrementCalled, "IncrementTokenVersion이 호출되어야 한다")
	assert.Equal(t, "user-1", incrementedID)
}

func TestAuthService_Logout_InvalidToken_NoError(t *testing.T) {
	// 리프레시 토큰이 없거나 만료되었더라도 Logout은 에러를 반환하지 않아야 한다.
	svc := newTestAuthService(&stubUserRepository{})
	err := svc.Logout(context.Background(), "invalid-token")
	assert.NoError(t, err)
}

func TestAuthService_Register_EmailNormalization(t *testing.T) {
	var storedEmail string
	repo := &stubUserRepository{
		createFunc: func(_ context.Context, params repository.CreateUserParams) (repository.UserRecord, error) {
			storedEmail = params.Email
			return repository.UserRecord{
				ID:          "new-id",
				Username:    params.Username,
				DisplayName: params.DisplayName,
				Email:       &params.Email,
			}, nil
		},
	}

	svc := newTestAuthService(repo)
	_, _, err := svc.Register(context.Background(), domain.RegisterRequest{
		Email:    "  User@Example.COM  ",
		Password: "password123",
		Username: "testuser",
	})

	require.NoError(t, err)
	assert.Equal(t, "user@example.com", storedEmail, "이메일은 소문자 정규화되어 저장되어야 한다")
}

func TestAuthService_Login_EmailNormalization(t *testing.T) {
	normalizedEmail := "user@example.com"
	var queriedEmail string

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	hashStr := string(hash)

	repo := &stubUserRepository{
		getByEmail: func(_ context.Context, email string) (repository.UserRecord, error) {
			queriedEmail = email
			return repository.UserRecord{
				ID:           "user-1",
				Email:        &normalizedEmail,
				PasswordHash: &hashStr,
			}, nil
		},
	}

	svc := newTestAuthService(repo)
	_, _, err := svc.Login(context.Background(), domain.LoginRequest{
		Email:    "  User@Example.COM  ",
		Password: "password123",
	})

	require.NoError(t, err)
	assert.Equal(t, "user@example.com", queriedEmail, "로그인 시 이메일은 정규화된 후 조회되어야 한다")
}
