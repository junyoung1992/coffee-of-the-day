package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/service"
)

// stubAuthService는 테스트에서 AuthService를 대체하는 스텁이다.
type stubAuthService struct {
	registerFunc func(ctx context.Context, req domain.RegisterRequest) (domain.AuthUser, service.AuthTokens, error)
	loginFunc    func(ctx context.Context, req domain.LoginRequest) (domain.AuthUser, service.AuthTokens, error)
	refreshFunc  func(ctx context.Context, refreshToken string) (service.AuthTokens, error)
	getUserFunc  func(ctx context.Context, userID string) (domain.AuthUser, error)
}

func (s *stubAuthService) Register(ctx context.Context, req domain.RegisterRequest) (domain.AuthUser, service.AuthTokens, error) {
	return s.registerFunc(ctx, req)
}
func (s *stubAuthService) Login(ctx context.Context, req domain.LoginRequest) (domain.AuthUser, service.AuthTokens, error) {
	return s.loginFunc(ctx, req)
}
func (s *stubAuthService) Refresh(ctx context.Context, refreshToken string) (service.AuthTokens, error) {
	return s.refreshFunc(ctx, refreshToken)
}
func (s *stubAuthService) GetUser(ctx context.Context, userID string) (domain.AuthUser, error) {
	return s.getUserFunc(ctx, userID)
}

var sampleAuthUser = domain.AuthUser{
	ID:          "user-1",
	Email:       "test@example.com",
	Username:    "testuser",
	DisplayName: "Test User",
}

var sampleTokens = service.AuthTokens{
	AccessToken:  "access.token.here",
	RefreshToken: "refresh.token.here",
}

// --- Register 테스트 ---

func TestAuthHandler_Register_Success(t *testing.T) {
	svc := &stubAuthService{
		registerFunc: func(_ context.Context, req domain.RegisterRequest) (domain.AuthUser, service.AuthTokens, error) {
			assert.Equal(t, "test@example.com", req.Email)
			assert.Equal(t, "testuser", req.Username)
			return sampleAuthUser, sampleTokens, nil
		},
	}
	h := NewAuthHandler(svc, false)

	body, _ := json.Marshal(map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"username": "testuser",
	})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Register(w, r)

	assert.Equal(t, http.StatusCreated, w.Code)

	// 응답 본문에 사용자 정보가 포함된다
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "user-1", resp["id"])
	assert.Equal(t, "test@example.com", resp["email"])

	// httpOnly 쿠키 두 개가 설정된다
	cookies := w.Result().Cookies()
	cookieNames := make(map[string]bool)
	for _, c := range cookies {
		cookieNames[c.Name] = true
		assert.True(t, c.HttpOnly, "쿠키 %s는 httpOnly여야 한다", c.Name)
	}
	assert.True(t, cookieNames["access_token"])
	assert.True(t, cookieNames["refresh_token"])
}

func TestAuthHandler_Register_EmailTaken(t *testing.T) {
	svc := &stubAuthService{
		registerFunc: func(_ context.Context, _ domain.RegisterRequest) (domain.AuthUser, service.AuthTokens, error) {
			return domain.AuthUser{}, service.AuthTokens{}, service.ErrEmailTaken
		},
	}
	h := NewAuthHandler(svc, false)

	body, _ := json.Marshal(map[string]string{"email": "taken@example.com", "password": "pw12345678", "username": "u"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Register(w, r)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAuthHandler_Register_ValidationError(t *testing.T) {
	svc := &stubAuthService{
		registerFunc: func(_ context.Context, _ domain.RegisterRequest) (domain.AuthUser, service.AuthTokens, error) {
			return domain.AuthUser{}, service.AuthTokens{}, &service.ValidationError{Field: "email", Message: "올바른 이메일 형식이 아닙니다"}
		},
	}
	h := NewAuthHandler(svc, false)

	body, _ := json.Marshal(map[string]string{"email": "bad", "password": "pw12345678", "username": "u"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Register(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "email", resp["field"])
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	h := NewAuthHandler(&stubAuthService{}, false)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte("not-json")))
	w := httptest.NewRecorder()

	h.Register(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Login 테스트 ---

func TestAuthHandler_Login_Success(t *testing.T) {
	svc := &stubAuthService{
		loginFunc: func(_ context.Context, req domain.LoginRequest) (domain.AuthUser, service.AuthTokens, error) {
			assert.Equal(t, "test@example.com", req.Email)
			return sampleAuthUser, sampleTokens, nil
		},
	}
	h := NewAuthHandler(svc, false)

	body, _ := json.Marshal(map[string]string{"email": "test@example.com", "password": "password123"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 2)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	svc := &stubAuthService{
		loginFunc: func(_ context.Context, _ domain.LoginRequest) (domain.AuthUser, service.AuthTokens, error) {
			return domain.AuthUser{}, service.AuthTokens{}, service.ErrInvalidCredentials
		},
	}
	h := NewAuthHandler(svc, false)

	body, _ := json.Marshal(map[string]string{"email": "test@example.com", "password": "wrong"})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.Login(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Refresh 테스트 ---

func TestAuthHandler_Refresh_Success(t *testing.T) {
	svc := &stubAuthService{
		refreshFunc: func(_ context.Context, token string) (service.AuthTokens, error) {
			assert.Equal(t, "valid-refresh-token", token)
			return sampleTokens, nil
		},
	}
	h := NewAuthHandler(svc, false)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	r.AddCookie(&http.Cookie{Name: "refresh_token", Value: "valid-refresh-token"})
	w := httptest.NewRecorder()

	h.Refresh(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 2)
}

func TestAuthHandler_Refresh_MissingCookie(t *testing.T) {
	h := NewAuthHandler(&stubAuthService{}, false)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	w := httptest.NewRecorder()

	h.Refresh(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	svc := &stubAuthService{
		refreshFunc: func(_ context.Context, _ string) (service.AuthTokens, error) {
			return service.AuthTokens{}, service.ErrInvalidToken
		},
	}
	h := NewAuthHandler(svc, false)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	r.AddCookie(&http.Cookie{Name: "refresh_token", Value: "expired-token"})
	w := httptest.NewRecorder()

	h.Refresh(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Me 테스트 ---

func TestAuthHandler_Me_Success(t *testing.T) {
	svc := &stubAuthService{
		getUserFunc: func(_ context.Context, userID string) (domain.AuthUser, error) {
			assert.Equal(t, "user-1", userID)
			return sampleAuthUser, nil
		},
	}
	h := NewAuthHandler(svc, false)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	h.Me(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "user-1", resp["id"])
}

// --- Logout 테스트 ---

func TestAuthHandler_Logout_ClearsCookies(t *testing.T) {
	h := NewAuthHandler(&stubAuthService{}, false)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	h.Logout(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	// 쿠키를 만료(MaxAge=-1)시켜 삭제한다
	cookies := w.Result().Cookies()
	cookieMaxAge := make(map[string]int)
	for _, c := range cookies {
		cookieMaxAge[c.Name] = c.MaxAge
	}
	assert.Equal(t, -1, cookieMaxAge["access_token"])
	assert.Equal(t, -1, cookieMaxAge["refresh_token"])
}
