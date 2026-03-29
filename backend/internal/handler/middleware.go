package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "userID"

var allowedOrigins = map[string]struct{}{
	"http://localhost:5173": {},
	"http://127.0.0.1:4173": {},
}

// jwtClaims는 미들웨어에서 액세스 토큰을 검증할 때 사용하는 JWT 페이로드 구조다.
type jwtClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTMiddleware는 액세스 토큰 쿠키를 검증하고 userID를 context에 주입한다.
// jwtSecret은 토큰 서명에 사용한 것과 동일한 값이어야 한다.
func JWTMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	secretBytes := []byte(jwtSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("access_token")
			if err != nil {
				writeError(w, http.StatusUnauthorized, "인증이 필요합니다")
				return
			}

			userID, err := parseAccessToken(cookie.Value, secretBytes)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "유효하지 않은 토큰입니다")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// parseAccessToken은 JWT 액세스 토큰을 파싱하고 userID(Subject)를 반환한다.
func parseAccessToken(tokenStr string, secret []byte) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid claims")
	}

	// 리프레시 토큰이 액세스 토큰 자리에 오는 토큰 혼용 공격을 차단한다
	if claims.TokenType != "access" {
		return "", fmt.Errorf("expected access token, got %q", claims.TokenType)
	}

	return claims.Subject, nil
}

// CORSMiddleware는 프론트엔드 개발 서버의 cross-origin 요청을 허용한다.
// Access-Control-Allow-Credentials: true — 쿠키 기반 인증을 위해 필요하다.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowedOrigins[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			// 쿠키 전송을 허용하려면 반드시 specific origin과 함께 설정해야 한다
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// preflight 요청은 여기서 종료한다 — JWTMiddleware까지 도달하면 안 됨
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getUserID는 context에서 사용자 ID를 꺼낸다.
// JWTMiddleware를 통과한 요청에서만 유효한 값을 반환한다.
func getUserID(r *http.Request) string {
	if id, ok := r.Context().Value(userIDKey).(string); ok {
		return id
	}
	return ""
}
