package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/service"
)

// AuthHandler는 인증 관련 HTTP 요청을 처리한다.
type AuthHandler struct {
	svc    service.AuthService
	isProd bool
}

// NewAuthHandler는 AuthHandler를 생성한다.
func NewAuthHandler(svc service.AuthService, isProd bool) *AuthHandler {
	return &AuthHandler{svc: svc, isProd: isProd}
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authUserResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// Register는 POST /api/v1/auth/register를 처리한다.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "요청 본문을 파싱할 수 없습니다")
		return
	}

	user, tokens, err := h.svc.Register(r.Context(), domain.RegisterRequest{
		Email:       req.Email,
		Password:    req.Password,
		Username:    req.Username,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		writeAuthServiceError(w, err)
		return
	}

	h.setAuthCookies(w, tokens)
	writeJSON(w, http.StatusCreated, toAuthUserResponse(user))
}

// Login은 POST /api/v1/auth/login을 처리한다.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "요청 본문을 파싱할 수 없습니다")
		return
	}

	user, tokens, err := h.svc.Login(r.Context(), domain.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeAuthServiceError(w, err)
		return
	}

	h.setAuthCookies(w, tokens)
	writeJSON(w, http.StatusOK, toAuthUserResponse(user))
}

// Refresh는 POST /api/v1/auth/refresh를 처리한다.
// 리프레시 토큰 쿠키를 검증하고 새 토큰 쌍을 발급한다.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "리프레시 토큰이 없습니다")
		return
	}

	tokens, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	h.setAuthCookies(w, tokens)
	w.WriteHeader(http.StatusNoContent)
}

// Logout은 POST /api/v1/auth/logout을 처리한다.
// 쿠키를 만료시켜 클라이언트 측 토큰을 무효화한다.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// setAuthCookies는 액세스/리프레시 토큰을 httpOnly 쿠키로 설정한다.
// httpOnly: JavaScript 접근 차단 → XSS 공격으로 토큰 탈취 불가
// SameSite=Strict: 외부 사이트에서 요청 시 쿠키 미전송 → CSRF 공격 차단
func (h *AuthHandler) setAuthCookies(w http.ResponseWriter, tokens service.AuthTokens) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokens.AccessToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.isProd,
		Path:     "/",
		MaxAge:   15 * 60, // 15분
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   h.isProd,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7일
	})
}

func (h *AuthHandler) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "access_token", Value: "", MaxAge: -1, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "", MaxAge: -1, Path: "/"})
}

func toAuthUserResponse(user domain.AuthUser) authUserResponse {
	return authUserResponse{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
	}
}

func writeAuthServiceError(w http.ResponseWriter, err error) {
	var ve *service.ValidationError
	switch {
	case errors.Is(err, service.ErrEmailTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, service.ErrInvalidToken):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.As(err, &ve):
		field := ve.Field
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ve.Message, Field: &field})
	default:
		writeError(w, http.StatusInternalServerError, "내부 오류가 발생했습니다")
	}
}
