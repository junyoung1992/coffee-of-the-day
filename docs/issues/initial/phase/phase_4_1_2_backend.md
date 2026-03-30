# Phase 4-1·4-2 Backend: 인증 (DB 마이그레이션 + JWT 인증 API)

## 무엇을 만들었나

POC 단계에서 사용하던 `X-User-Id` 헤더 기반 인증을 **JWT + httpOnly 쿠키** 방식으로 교체했다.

- `005` 마이그레이션: `users` 테이블에 `email`, `password_hash` 컬럼 추가
- `POST /api/v1/auth/register` — bcrypt 해싱 후 사용자 생성, 토큰 발급
- `POST /api/v1/auth/login` — 비밀번호 검증, 토큰 발급
- `POST /api/v1/auth/refresh` — 리프레시 토큰으로 새 토큰 쌍 발급
- `POST /api/v1/auth/logout` — 쿠키 만료 처리
- 기존 `UserIDMiddleware` → `JWTMiddleware`로 교체

---

## Spring과의 비교

| Spring Security | 이 프로젝트 |
|---|---|
| `SecurityFilterChain` | chi 라우터 그룹에 `JWTMiddleware` 적용 |
| `UsernamePasswordAuthenticationFilter` | `AuthHandler.Login()` |
| `OncePerRequestFilter` (JWT 검증) | `JWTMiddleware(jwtSecret)` |
| `BCryptPasswordEncoder` | `golang.org/x/crypto/bcrypt` |
| `@RestController` with `@PostMapping("/auth/login")` | `AuthHandler` + chi route |

---

## 왜 httpOnly 쿠키인가

JWT를 어디에 저장할지는 항상 tradeoff다:

| 방식 | XSS 취약 | CSRF 취약 |
|---|---|---|
| `localStorage` | O (직접 탈취 가능) | X |
| `httpOnly` 쿠키 | X (JS 접근 불가) | O (SameSite로 완화) |

`httpOnly + SameSite=Strict` 조합:
- `httpOnly`: JavaScript에서 `document.cookie`로 접근 불가 → XSS로 토큰 탈취 차단
- `SameSite=Strict`: 외부 사이트에서 유발된 요청에 쿠키 미전송 → CSRF 차단

---

## Access Token + Refresh Token 분리 이유

Access token만 쓰면 탈취 시 유효기간 동안 계속 사용 가능하다. 두 토큰을 분리하면:

- **Access token** (15분): 짧은 수명 → 탈취돼도 피해 최소화
- **Refresh token** (7일): 긴 수명, httpOnly 쿠키로만 전달 → 자동 갱신 UX 유지

### token_type 필드의 역할

두 토큰 모두 같은 비밀키로 서명되므로, `token_type` 클레임 없이는 리프레시 토큰을 액세스 토큰 자리에 쓰는 **토큰 혼용 공격(token confusion attack)**이 가능하다.

```go
// 미들웨어에서 token_type 검증
if claims.TokenType != "access" {
    return "", fmt.Errorf("expected access token, got %q", claims.TokenType)
}
```

테스트 `TestJWTMiddleware_WithRefreshTokenRejected`와 `TestAuthService_Refresh_AccessTokenRejected`가 이 보안 속성을 명시적으로 검증한다.

---

## bcrypt: 왜 단순 해시가 아닌가

`SHA256(password)`는 같은 비밀번호에 항상 같은 해시를 반환 → **Rainbow table 공격** 가능.

bcrypt는:
1. 내부에 랜덤 salt를 자동 포함 → 같은 비밀번호도 매번 다른 해시
2. cost factor로 연산 속도를 의도적으로 늦춤 → brute-force 비용 증가
3. `CompareHashAndPassword`는 timing-safe 비교를 수행

```go
// 항상 서비스 에러로 래핑: 사용자 존재 여부를 외부에 노출하지 않는다
if errors.Is(err, repository.ErrUserNotFound) {
    return domain.AuthUser{}, AuthTokens{}, ErrInvalidCredentials
}
```

---

## 레이어 구조

```
handler/auth_handler.go   ← HTTP 요청 파싱, 쿠키 설정
service/auth_service.go   ← 비밀번호 검증, JWT 생성/파싱
repository/user_repository.go ← DB CRUD (sqlc 사용)
db/migrations/005_*.sql   ← 스키마 변경
```

`UserRecord` 타입이 repository 패키지에 정의된 이유: `password_hash`는 bcrypt 검증 목적으로만 서비스 레이어에서 사용되고 외부로 나가지 않아야 한다. 도메인 타입 `AuthUser`에는 포함하지 않아 민감 정보 누출을 방지한다.

---

## SQLite ALTER TABLE 제약

SQLite는 `ALTER TABLE ADD COLUMN NOT NULL` 을 default 없이 지원하지 않는다. 기존 행(POC 사용자)은 `email = NULL`, `password_hash = NULL`로 남는다. 서비스 레이어에서 `PasswordHash == nil` 체크로 이 케이스를 처리한다.

```go
// email/password가 없는 POC 시드 사용자는 새 인증 방식으로 로그인 불가
if rec.PasswordHash == nil {
    return domain.AuthUser{}, AuthTokens{}, ErrInvalidCredentials
}
```

---

## CORS 변경사항

쿠키를 cross-origin 요청에서 전송하려면 `Access-Control-Allow-Credentials: true`가 필요하다. `*` 와일드카드 origin과는 함께 사용할 수 없어, 이미 specific origin을 허용하고 있던 기존 CORS 설정이 그대로 호환된다.

프론트엔드에서는 fetch 요청 시 `credentials: 'include'`를 추가해야 한다 (Phase 4-3에서 처리).
