# Phase 4 리팩토링 기록

## 개요

이 문서는 `review/code_review_phase_4.md`에 정리된 Codex 리뷰를 바탕으로 진행한 리팩토링의 내용과 판단 근거를 기록한다.

리팩토링 범위:

- **Backend**: JWT 설정 강제화, 리프레시 토큰 무효화, rate limiting, 이메일 정규화
- **Frontend**: refresh single-flight, lint 수정, OpenAPI 정합성
- **E2E 테스트**: `beforeEach` 구조 버그 수정

---

## 1. P1-01 — JWT_SECRET 강제 설정

### 리뷰 지적

운영 환경에서 `JWT_SECRET` 환경변수를 설정하지 않아도 서버가 정상 시작됐다. 이 경우 `"dev-secret-change-in-production"`이라는 예측 가능한 값으로 토큰을 서명하게 되어, 외부 공격자가 임의의 JWT를 위조할 수 있었다.

### 변경 내용

**`backend/config/config.go`**

```go
// 변경 전
func Load() Config {
    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        jwtSecret = "dev-secret-change-in-production"
    }
    return Config{ ... }
}

// 변경 후
func Load() (Config, error) {
    jwtSecret, err := loadJWTSecret(isProduction)
    if err != nil {
        return Config{}, err
    }
    return Config{ ... }, nil
}

func loadJWTSecret(isProduction bool) (string, error) {
    secret := os.Getenv("JWT_SECRET")
    if isProduction {
        if secret == "" {
            return "", errors.New("JWT_SECRET이 설정되지 않았습니다")
        }
        if len(secret) < minJWTSecretLen { // 32바이트
            return "", errors.New("JWT_SECRET이 너무 짧습니다")
        }
        return secret, nil
    }
    if secret != "" {
        return secret, nil
    }
    return "dev-secret-change-in-production-must-be-32b", nil
}
```

**`backend/cmd/server/main.go`**

```go
cfg, err := config.Load()
if err != nil {
    log.Fatalf("설정 오류: %v", err)
}
```

### 판단 근거

Go에서 함수가 실패할 수 있으면 `(T, error)`를 반환하는 것이 관용적이다. Spring Boot의 `@Value`에 `required = true`를 걸면 `ApplicationContext` 로딩 자체가 실패하는 것과 같은 원리다. "운영 환경에서 설정 실수가 발생했을 때 시스템이 조용히 동작하는 것"보다 "즉시 실패해서 배포 단계에서 문제를 드러내는 것"이 더 안전하다.

개발 환경에서 fallback을 허용한 이유는 개발자 경험을 위해서다. `GO_ENV=production`을 명시하지 않으면 dev secret이 사용된다.

### 추가된 테스트

`backend/config/config_test.go` 신규 작성:
- `production + 미설정` → 에러
- `production + 짧은 시크릿` → 에러
- `production + 32바이트 이상` → 성공
- `development + 미설정` → dev fallback 반환
- `development + 명시적 설정` → 해당 값 반환

---

## 2. P1-02 — Refresh Token Revocation (token_version)

### 리뷰 지적

리프레시 토큰이 서버에서 추적되지 않아 로그아웃 후에도 탈취된 토큰을 계속 사용할 수 있었다. `Logout` 핸들러는 클라이언트 쿠키만 만료시켰고 서버 상태는 아무것도 바뀌지 않았다.

### 변경 내용

**DB 마이그레이션** (`backend/db/migrations/006_add_token_version_to_users.up.sql`)

```sql
ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;
```

**SQL 쿼리** (`backend/db/queries/users.sql`)

```sql
-- name: IncrementTokenVersion :exec
UPDATE users SET token_version = token_version + 1 WHERE id = ?;
```

이후 `sqlc generate`로 `internal/db/` 코드를 재생성했다.

**JWT 클레임 구조** (`backend/internal/service/auth_service.go`)

```go
// 변경 전
type tokenClaims struct {
    TokenType string `json:"token_type"`
    jwt.RegisteredClaims
}

// 변경 후
type tokenClaims struct {
    TokenType    string `json:"token_type"`
    TokenVersion int64  `json:"token_version"`
    jwt.RegisteredClaims
}
```

**Refresh 검증 강화**

```go
// 변경 전: 사용자 존재 여부만 확인
userID, err := s.parseToken(refreshToken, "refresh")
if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
    return AuthTokens{}, ErrInvalidToken
}
return s.generateTokens(userID)

// 변경 후: token_version까지 비교
claims, err := s.parseTokenClaims(refreshToken, "refresh")
rec, err := s.repo.GetUserByID(ctx, claims.Subject)
if claims.TokenVersion != rec.TokenVersion {
    return AuthTokens{}, ErrInvalidToken
}
return s.generateTokens(rec.ID, rec.TokenVersion)
```

**Logout 서비스 메서드 신규 추가**

```go
func (s *DefaultAuthService) Logout(ctx context.Context, refreshToken string) error {
    claims, err := s.parseTokenClaims(refreshToken, "refresh")
    if err != nil {
        return nil // 토큰이 없거나 만료되어도 쿠키 만료는 핸들러가 처리한다
    }
    _ = s.repo.IncrementTokenVersion(ctx, claims.Subject)
    return nil
}
```

**Logout 핸들러 변경**

```go
// 변경 전: 쿠키만 만료
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    h.clearAuthCookies(w)
    w.WriteHeader(http.StatusNoContent)
}

// 변경 후: token_version 증가 후 쿠키 만료
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
    if cookie, err := r.Cookie("refresh_token"); err == nil {
        _ = h.svc.Logout(r.Context(), cookie.Value)
    }
    h.clearAuthCookies(w)
    w.WriteHeader(http.StatusNoContent)
}
```

### 판단 근거

token_version 방식은 "가장 단순하게 로그아웃 무효화를 구현하는 방법"이다. 로그아웃 한 번이 해당 사용자의 모든 리프레시 토큰을 일괄 무효화한다. 디바이스별 세션 관리가 필요하면 별도 세션 테이블 방식으로 확장할 수 있는데, 이 POC에서는 단일 사용자이므로 token_version으로 충분하다.

Spring Security 관점에서 보면, 기존 구현은 `TokenStore` 없이 JWT 서명 검증만 하는 stateless 방식이었다. `token_version`을 도입하면 토큰이 담고 있는 클레임을 DB와 대조하는 semi-stateful 방식이 된다.

`Logout` 서비스 메서드가 토큰 파싱 실패 시 에러를 반환하지 않는 이유는 쿠키 만료가 항상 실행되어야 하기 때문이다. 리프레시 토큰이 이미 만료되었거나 쿠키가 없는 상태에서도 로그아웃 자체는 성공으로 처리한다.

### 추가된 테스트

`backend/internal/service/auth_service_test.go`에 다음 케이스를 추가했다:
- 로그아웃 후 이전 리프레시 토큰으로 refresh 시도 → `ErrInvalidToken` 반환
- `Logout` 호출 시 `IncrementTokenVersion`이 정확한 userID로 호출되는지 확인
- 유효하지 않은 토큰으로 `Logout` 호출 시 에러 없이 반환되는지 확인

---

## 3. P2-01 — Auth 엔드포인트 Rate Limiting

### 리뷰 지적

`/auth/register`, `/auth/login` 등에 시도 횟수 제한이 없어 무차별 대입(brute-force), credential stuffing 시도를 서비스 레벨에서 막을 수 없었다.

### 변경 내용

**`go.mod`에 `github.com/go-chi/httprate` 추가**

**`backend/cmd/server/main.go`**

```go
r.Route("/auth", func(r chi.Router) {
    // IP 기준 1분에 20회 초과 시 429 반환
    r.Use(httprate.LimitByIP(20, 1*time.Minute))
    r.Post("/register", authHandler.Register)
    r.Post("/login", authHandler.Login)
    r.Post("/refresh", authHandler.Refresh)
    r.Post("/logout", authHandler.Logout)
})
```

### 판단 근거

`go-chi/httprate`는 chi와 동일한 에코시스템의 미들웨어로, chi 라우터 그룹에 `r.Use()`로 붙이는 방식이 Spring의 `OncePerRequestFilter`를 `FilterRegistrationBean`으로 특정 URL 패턴에만 등록하는 것과 구조가 같다.

임계치를 1분 20회로 설정한 이유는 정상 사용자의 UX를 해치지 않는 범위에서 자동화된 공격을 차단하기 위해서다. 로그아웃과 refresh도 같은 그룹에 포함시킨 것은 이 엔드포인트들도 자동화된 토큰 재사용 시도의 대상이 될 수 있기 때문이다.

---

## 4. P2-02 — 이메일 정규화

### 리뷰 지적

회원가입 검증에서 `TrimSpace`만 수행하고 정규화된 값을 저장하지 않아, `User@example.com`과 `user@example.com`이 서로 다른 계정으로 저장될 수 있었다. 또한 가입 시 대문자로 입력했을 경우 로그인 시 소문자로 입력하면 계정을 찾지 못하는 문제가 있었다.

### 변경 내용

**`backend/internal/service/auth_service.go`**

```go
// 정규화 함수 추가
func normalizeEmail(email string) string {
    return strings.ToLower(strings.TrimSpace(email))
}

// Register 진입 시 적용
func (s *DefaultAuthService) Register(ctx context.Context, req domain.RegisterRequest) (...) {
    req.Email = normalizeEmail(req.Email)
    // 이후 검증 및 저장
}

// Login 진입 시 동일하게 적용
func (s *DefaultAuthService) Login(ctx context.Context, req domain.LoginRequest) (...) {
    req.Email = normalizeEmail(req.Email)
    // 이후 조회
}
```

기존 `validateRegisterRequest` 내부의 `strings.TrimSpace(req.Email)` 검증 코드도 함께 정리했다. 정규화 이후에 검증이 실행되므로 검증 시점에는 이미 trim된 값이 들어온다.

### 판단 근거

정규화는 검증보다 먼저 실행해야 한다. 검증이 통과한 값을 정규화하면 "검증 기준과 저장 기준이 다른" 상황이 생길 수 있다. 예를 들어 `" user@example.com "`은 trim 후 유효한 이메일이지만, trim 없이 저장하면 같은 사람이 `"user@example.com"`으로 로그인할 때 다른 값으로 조회된다.

Register와 Login 양쪽에 동일한 `normalizeEmail` 함수를 적용한 이유는 정규화 규칙을 한 곳에서 관리해 불일치를 방지하기 위해서다.

### 추가된 테스트

- `Register` 호출 시 대문자/공백이 포함된 이메일이 소문자 정규화된 상태로 저장되는지 확인
- `Login` 호출 시 동일 정규화를 거쳐 조회하는지 확인

---

## 5. P3-01 — 프론트 Refresh Single-Flight

### 리뷰 지적

액세스 토큰이 만료된 상태에서 화면이 여러 API를 동시에 요청하면 각 요청이 독립적으로 `/auth/refresh`를 호출한다. token_version 기반 revocation이 도입된 뒤에는 "먼저 refresh에 성공한 요청이 token_version을 소비하면 나머지 동시 refresh 요청은 같은 버전의 토큰을 사용하므로 실패"하는 race condition으로 이어진다.

### 변경 내용

**`frontend/src/api/client.ts`**

```typescript
// 변경 전: 각 401 응답이 독립적으로 refresh를 호출
if (res.status === 401 && !isRetry && !path.startsWith('/auth/')) {
    const refreshRes = await doFetch('/auth/refresh', { method: 'POST' })
    if (refreshRes.ok) {
        return requestInternal(path, init, true)
    }
    throw new ApiError(401, 'UNAUTHORIZED', '세션이 만료되었습니다')
}

// 변경 후: 진행 중인 refresh Promise를 공유
let refreshPromise: Promise<void> | null = null

// ...requestInternal 내부
if (res.status === 401 && !isRetry && !path.startsWith('/auth/')) {
    if (!refreshPromise) {
        refreshPromise = doFetch('/auth/refresh', { method: 'POST' })
            .then((r) => {
                if (!r.ok) throw new ApiError(401, 'UNAUTHORIZED', '세션이 만료되었습니다')
            })
            .finally(() => {
                refreshPromise = null
            })
    }
    try {
        await refreshPromise
        return requestInternal(path, init, true)
    } catch {
        throw new ApiError(401, 'UNAUTHORIZED', '세션이 만료되었습니다')
    }
}
```

### 판단 근거

이 패턴은 Go의 `golang.org/x/sync/singleflight`와 동일한 개념이다. 처음 도달한 요청만 실제 작업을 수행하고, 이후 도달한 요청은 그 결과를 공유한다. JavaScript/TypeScript에서는 `Promise`가 이미 완료된 뒤에도 `await`할 수 있어 언어 자체가 이 패턴을 자연스럽게 지원한다.

`finally`에서 `refreshPromise = null`을 초기화하는 이유는 refresh 성공/실패 여부와 무관하게 다음 refresh 시도를 허용하기 위해서다. `then`이나 `catch`에서만 초기화하면 한 쪽 분기에서 초기화가 누락된다.

### 추가된 테스트

`frontend/src/api/client.test.ts` 신규 작성:
- 동시 401 발생 시 `/auth/refresh` 1회만 호출되는지 확인
- refresh 실패 시 모든 대기 요청이 `ApiError`를 던지는지 확인

테스트마다 모듈 캐시를 초기화(`vi.resetModules()`)해 `refreshPromise` 모듈 스코프 상태를 격리했다.

---

## 6. P3-02 — 프론트 Lint 수정 (ProtectedRoute 분리)

### 리뷰 지적

`frontend/src/router.tsx`에서 `ProtectedRoute` 컴포넌트와 `router` 객체를 같은 파일에서 export해 `react-refresh/only-export-components` ESLint 규칙을 위반했다. `npm run lint`가 실패하는 상태로 방치되어 이후 리팩토링에서 실제 회귀와 기존 노이즈를 구분하기 어려웠다.

### 변경 내용

`ProtectedRoute`를 `frontend/src/components/ProtectedRoute.tsx`로 분리하고, `router.tsx`에서 import해 사용하도록 변경했다.

```tsx
// frontend/src/components/ProtectedRoute.tsx (신규)
export default function ProtectedRoute() {
    const { data: user, isLoading, isError } = useCurrentUser()
    if (isLoading) return null
    if (isError || !user) return <Navigate to="/login" replace />
    return <Outlet />
}

// frontend/src/router.tsx (정리 후)
import ProtectedRoute from './components/ProtectedRoute'
export const router = createBrowserRouter([ ... ])
```

### 판단 근거

`react-refresh/only-export-components` 규칙은 React Fast Refresh(HMR)가 컴포넌트가 아닌 값과 같은 파일에 있으면 전체 모듈을 리로드해버리기 때문에 존재한다. 컴포넌트와 라우터 설정을 분리하는 것은 책임 분리 측면에서도 자연스럽다. Spring에서 `@Component`와 설정 `@Bean`을 같은 클래스에 두지 않는 관행과 유사하다.

---

## 7. P3-03 — OpenAPI 스펙 정합성

### 리뷰 지적

실제 서버는 `/auth/logout`을 인증 불필요 라우트로 등록했지만, `openapi.yml`의 전역 `cookieAuth` 보안 설정에서 `/auth/logout`에 `security: []` 예외를 선언하지 않아 스펙상으로는 로그아웃에 인증이 필요한 것처럼 읽혔다.

### 변경 내용

**`openapi.yml`**

```yaml
/api/v1/auth/logout:
    post:
        summary: 로그아웃
        tags: [auth]
        security: []   # 추가: 인증 불필요 — 실제 라우팅 정책과 일치
        responses:
            '204':
                description: 로그아웃 성공 (토큰 쿠키 만료 처리)
```

이후 `npm run generate`를 실행해 `frontend/src/types/schema.ts`를 갱신했다.

### 판단 근거

OpenAPI 스펙은 단순 문서가 아니라 `openapi-typescript`로 프론트엔드 타입을 자동 생성하는 소스다. 스펙과 실제 구현이 어긋나면 생성된 타입도 잘못되고, 이를 기반으로 만든 API 클라이언트나 테스트 도구가 실제 서버와 다르게 동작할 수 있다.

---

## 8. E2E 테스트 버그 수정

### 문제

`beforeEach`에서 매 테스트마다 동일한 이메일(`e2e@example.com`)로 회원가입을 시도했다. 테스트 스위트 내 DB를 공유하므로 두 번째 테스트의 `beforeEach`에서 이메일 중복 오류가 발생해 `/register`에 머물렀고, `toHaveURL('/')`이 실패했다.

### 변경 내용

**`frontend/e2e/log-happy-path.spec.ts`**

```typescript
// 변경 전: beforeEach에서 매번 회원가입
test.beforeEach(async ({ page }) => {
    await page.goto('/register')
    // 회원가입 폼 제출 → 두 번째 테스트부터 이메일 중복 오류 발생
})

// 변경 후: 회원가입(beforeAll) + 로그인(beforeEach) 분리
test.beforeAll(async ({ browser }) => {
    const page = await browser.newPage()
    await page.goto('/register')
    // 회원가입 1회 수행
    await page.close()
})

test.beforeEach(async ({ page }) => {
    await page.goto('/login')
    // 매 테스트 전 로그인
})
```

### 판단 근거

`beforeAll`은 "테스트 스위트 전체에서 한 번만 필요한 선행 조건"에 적합하고, `beforeEach`는 "각 테스트가 독립적인 상태에서 시작하기 위한 초기화"에 적합하다. 회원가입은 DB에 사용자가 존재하지 않을 때 한 번만 하면 되고, 로그인은 이전 테스트의 로그아웃 등으로 세션이 초기화될 수 있으므로 매 테스트 전에 수행한다.

JUnit의 `@BeforeAll` / `@BeforeEach`와 정확히 같은 역할 구분이다.

---

## 최종 검증

리팩토링 완료 후 아래 검증을 모두 통과했다.

| 검증 항목 | 결과 |
|----------|------|
| `cd backend && go test ./...` | 통과 |
| `cd frontend && npx vitest run` (59개) | 통과 |
| `cd frontend && npm run build` | 통과 |
| `cd frontend && npm run lint` | 통과 (clean) |
| `cd frontend && npm run test:e2e` (2개) | 통과 |
