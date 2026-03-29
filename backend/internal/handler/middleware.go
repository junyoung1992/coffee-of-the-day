package handler

import (
	"context"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "userID"

// UserIDMiddleware는 X-User-Id 헤더에서 사용자 ID를 읽어 context에 저장한다.
// POC 단계에서는 JWT 없이 이 헤더로 사용자를 식별한다.
// Phase 4에서 JWT 미들웨어로 교체 예정.
func UserIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-Id")
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "X-User-Id 헤더가 필요합니다")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CORSMiddleware는 프론트엔드 개발 서버(localhost:5173)의 cross-origin 요청을 허용한다.
// OPTIONS preflight 요청은 204로 즉시 응답한다.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:5173" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-Id")
		}

		// preflight 요청은 여기서 종료한다 — UserIDMiddleware까지 도달하면 안 됨
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getUserID는 context에서 사용자 ID를 꺼낸다.
// UserIDMiddleware를 통과한 요청에서만 유효한 값을 반환한다.
func getUserID(r *http.Request) string {
	if id, ok := r.Context().Value(userIDKey).(string); ok {
		return id
	}
	return ""
}
