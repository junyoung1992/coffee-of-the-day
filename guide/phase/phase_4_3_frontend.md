# Phase 4-3 Frontend: 인증 UI

## 무엇을 만들었나

백엔드가 JWT + httpOnly 쿠키 방식으로 전환됨에 따라, 프론트엔드도 기존 `X-User-Id` 헤더 방식을 제거하고 쿠키 기반 인증을 도입했다.

- `src/api/client.ts` — `X-User-Id` 헤더 제거, `credentials: 'include'`, 자동 토큰 갱신 인터셉터
- `src/types/auth.ts` — `User`, `LoginRequest`, `RegisterRequest` 타입
- `src/api/auth.ts` — `login`, `register`, `logout`, `refresh`, `getMe`
- `src/hooks/useAuth.ts` — TanStack Query 기반 인증 상태 관리
- `src/pages/LoginPage.tsx`, `RegisterPage.tsx` — 로그인/회원가입 폼
- `src/router.tsx` — Protected Route 추가
- `src/components/Layout.tsx` — 로그아웃 버튼 및 사용자 이름 표시
- E2E 테스트 — 회원가입 후 기존 happy-path 실행

---

## Spring과의 비교

| Spring Security | 이 프로젝트 |
|---|---|
| `SecurityContextHolder` | TanStack Query의 `AUTH_KEY` 캐시 |
| `SecurityFilterChain` (인증 필터) | `ProtectedRoute` 컴포넌트 |
| `@AuthenticationPrincipal` | `useCurrentUser()` 훅 |
| `session.invalidate()` | `queryClient.clear()` |
| Thymeleaf 로그인 폼 | `LoginPage.tsx` / `RegisterPage.tsx` |

---

## 인증 상태 관리: TanStack Query

인증 상태를 `useState`나 Context로 관리하는 대신, TanStack Query 캐시를 사용한다.

```typescript
export const AUTH_KEY = ['auth', 'me'] as const

export function useCurrentUser() {
  return useQuery({
    queryKey: AUTH_KEY,
    queryFn: getMe,       // GET /api/v1/auth/me
    retry: false,
    staleTime: 5 * 60_000,
  })
}
```

**이 방식의 장점:**
- 로그인/로그아웃 후 `queryClient.setQueryData(AUTH_KEY, user)` / `queryClient.clear()`로 즉시 UI를 업데이트할 수 있다
- 인증 상태가 캐시에 있으면 페이지 이동 시 불필요한 `/me` 요청이 발생하지 않는다 (`staleTime: 5분`)
- `retry: false` — 401 응답에 재시도하면 의미 없으므로 비활성화

### 로그인 후 캐시 즉시 채우기

```typescript
onSuccess: (user) => {
  // /me 네트워크 요청 없이 서버가 이미 반환한 user를 캐시에 직접 저장한다
  queryClient.setQueryData(AUTH_KEY, user)
  navigate('/')
}
```

---

## Protected Route 패턴

```tsx
function ProtectedRoute() {
  const { data: user, isLoading, isError } = useCurrentUser()

  if (isLoading) return null          // 깜빡임 방지
  if (isError || !user) return <Navigate to="/login" replace />
  return <Outlet />                   // 자식 라우트 렌더링
}
```

`<Outlet />`은 Spring MVC의 `FilterChain.doFilter()`와 유사 — 검사를 통과하면 다음 계층(실제 페이지)으로 요청을 넘긴다.

---

## client.ts: 자동 토큰 갱신 인터셉터

axios의 `interceptors.response`와 동일한 패턴을 fetch 위에 직접 구현했다.

```typescript
async function requestInternal<T>(path, init, isRetry): Promise<T> {
  const res = await doFetch(path, init)

  if (res.status === 401 && !isRetry && !path.startsWith('/auth/')) {
    // 1. 리프레시 토큰으로 새 액세스 토큰 발급 시도
    const refreshRes = await doFetch('/auth/refresh', { method: 'POST' })
    if (refreshRes.ok) {
      return requestInternal(path, init, true)  // 2. 원래 요청 재시도 (isRetry=true)
    }
    throw new ApiError(401, 'UNAUTHORIZED', '세션이 만료되었습니다')
  }
  // ...
}
```

**isRetry 플래그의 역할:**
- 갱신 후 재시도에서도 401이 오면 무한 루프가 된다
- `isRetry=true`이면 401에서 갱신을 시도하지 않고 바로 에러를 던진다

**`path.startsWith('/auth/')` 조건:**
- `/auth/refresh`나 `/auth/login` 자체가 401을 반환할 때는 갱신을 시도하지 않는다
- 특히 `/auth/refresh`가 401이면 리프레시 토큰도 만료된 것이므로 로그인이 필요하다

---

## credentials: 'include'가 필요한 이유

```typescript
function doFetch(path, init) {
  return fetch(`${BASE_URL}${path}`, {
    ...init,
    credentials: 'include',  // 이게 없으면 쿠키가 cross-origin 요청에 포함되지 않는다
    headers: { 'Content-Type': 'application/json', ...init.headers },
  })
}
```

브라우저는 기본적으로 cross-origin 요청에 쿠키를 포함하지 않는다. 프론트엔드(`:5173`)에서 백엔드(`:8080`)로 요청할 때 쿠키가 포함되려면 `credentials: 'include'`가 필요하고, 이에 맞춰 백엔드도 `Access-Control-Allow-Credentials: true` + specific origin을 설정해야 한다.

---

## E2E 테스트 변경

기존 `POC_SEED_USER_ID`로 사용자를 자동 생성하는 방식을 제거하고, 테스트 시작 시 직접 회원가입하는 방식으로 전환했다.

```typescript
test.beforeEach(async ({ page }) => {
  await page.goto('/register')
  // ... 회원가입 폼 작성
  await page.getByRole('button', { name: '회원가입' }).click()
  await expect(page).toHaveURL('/')
})
```

매 실행마다 새 DB(`e2eDBPath = /tmp/coffee-...-${pid}.db`)를 사용하므로 항상 동일한 이메일로 가입이 가능하다.
