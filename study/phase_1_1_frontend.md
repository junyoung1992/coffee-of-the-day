# Phase 1-1 Frontend 학습 문서

> Java/Spring 경험을 기반으로, TypeScript/React 프론트엔드 초기 설정이 왜 이렇게 구성됐는지 설명합니다.

---

## 1. Vite — 빌드 도구

**Spring에서의 대응**: Maven/Gradle (빌드 도구)

Vite는 프론트엔드의 빌드 도구입니다. TypeScript/JSX 코드를 브라우저가 이해할 수 있는 JavaScript로 변환하고, 개발 서버를 실행합니다.

```bash
npm run dev    # 개발 서버 시작 (Spring Boot DevTools처럼 핫 리로드 지원)
npm run build  # 프로덕션 빌드 (mvn package와 유사)
```

**왜 Vite인가?**
기존에는 Create React App(CRA)이 많이 쓰였지만, Vite가 훨씬 빠릅니다. 개발 서버 시작이 수초 이내이고, 파일 수정 시 반영도 즉각적입니다.

---

## 2. TypeScript

**Spring에서의 대응**: Java (정적 타입 언어)

JavaScript는 동적 타입 언어라 런타임에 타입 오류가 발생합니다. TypeScript는 JavaScript에 Java처럼 정적 타입을 추가한 언어입니다.

```typescript
// JavaScript — 런타임에 오류 발생 가능
function getLog(id) { ... }

// TypeScript — 컴파일 시점에 타입 검사
function getLog(id: string): Promise<CoffeeLog> { ... }
```

Spring에서 Java를 쓰는 이유와 같습니다. 컴파일 타임에 오류를 잡고, IDE 자동완성이 잘 됩니다.

---

## 3. React

**Spring에서의 대응**: Thymeleaf (단, React는 SPA)

Spring MVC + Thymeleaf는 서버가 HTML을 완성해서 보내는 방식(SSR)입니다. React는 브라우저에서 JavaScript가 직접 DOM을 구성하는 방식(CSR/SPA)입니다.

핵심 개념은 **컴포넌트**입니다. UI를 작은 단위로 쪼개서 재사용합니다.

```tsx
// 컴포넌트 = 화면의 한 조각을 담당하는 함수
function LogCard({ log }: { log: CoffeeLog }) {
  return (
    <div>
      <h2>{log.cafe?.cafe_name}</h2>
      <p>{log.recorded_at}</p>
    </div>
  )
}
```

JSX(`<div>`, `<h2>` 등)는 HTML처럼 보이지만 실제로는 JavaScript 함수 호출입니다. Vite가 빌드 시 변환합니다.

---

## 4. 패키지 관리 (`package.json`)

**Spring에서의 대응**: `pom.xml` / `build.gradle`

```json
{
  "dependencies": {
    "react": "^19.2.4",
    "@tanstack/react-query": "^5.95.2"
  },
  "devDependencies": {
    "typescript": "...",
    "vite": "..."
  }
}
```

- `dependencies`: 런타임에 필요한 라이브러리 (`<dependency>` scope=compile)
- `devDependencies`: 빌드/개발 시에만 필요 (`<dependency>` scope=test/provided)
- `npm install`: `mvn install`처럼 의존성을 `node_modules/`에 설치

---

## 5. 디렉토리 구조

**Spring에서의 대응**: 레이어드 아키텍처 패키지 구조

```
src/
├── pages/       # 라우트에 대응하는 화면 컴포넌트 (@Controller의 View와 유사)
├── components/  # 재사용 가능한 UI 조각 (공용 Thymeleaf fragment)
├── api/         # 백엔드 API 호출 함수 (FeignClient / RestTemplate)
├── types/       # TypeScript 타입 정의 (DTO/Entity 클래스)
└── hooks/       # 서버 상태 관리 로직 (TanStack Query 래핑)
```

---

## 6. 라우터 (`react-router-dom`)

**Spring에서의 대응**: `@RequestMapping` URL 매핑

```tsx
// router.tsx
export const router = createBrowserRouter([
  { path: '/',              element: <HomePage /> },
  { path: '/logs/new',      element: <LogFormPage /> },
  { path: '/logs/:id',      element: <LogDetailPage /> },
  { path: '/logs/:id/edit', element: <LogFormPage /> },
])
```

Spring MVC에서 URL → Controller 메서드를 매핑하듯, 여기서는 URL → 컴포넌트를 매핑합니다. `:id`는 Spring의 `@PathVariable`과 동일한 개념입니다.

---

## 7. TanStack Query

**Spring에서의 대응**: 딱 맞는 대응이 없음. 굳이 비유하면 캐시 레이어 + 비동기 상태 관리

프론트엔드에서 서버 데이터를 다룰 때의 공통 문제들을 해결합니다:
- API 호출 중 로딩 상태 처리
- 에러 처리
- 데이터 캐싱 (같은 데이터를 여러 컴포넌트에서 쓸 때 중복 요청 방지)
- 데이터 최신화 (stale-while-revalidate)

```tsx
// hooks/useLogs.ts
function useLog(id: string) {
  return useQuery({
    queryKey: ['logs', id],   // 캐시 키 (Redis key와 유사)
    queryFn: () => getLog(id) // 실제 API 호출
  })
}

// 컴포넌트에서 사용
function LogDetailPage() {
  const { data, isLoading, error } = useLog(id)
  if (isLoading) return <Spinner />
  if (error) return <ErrorMessage />
  return <div>{data.cafe?.cafe_name}</div>
}
```

**왜 Redux 대신 TanStack Query인가?**
Redux는 클라이언트 상태(UI 상태)와 서버 상태(API 데이터)를 모두 관리합니다. 이 프로젝트에서 서버 상태가 대부분이므로, 서버 상태 관리에 특화된 TanStack Query만으로 충분합니다. 훨씬 적은 boilerplate로 같은 결과를 얻습니다.

---

## 8. Tailwind CSS

**Spring에서의 대응**: 없음 (순수 CSS 유틸리티)

Tailwind는 미리 정의된 CSS 클래스 모음입니다. 별도 CSS 파일을 작성하지 않고 HTML/JSX에 클래스를 조합해서 스타일을 적용합니다.

```tsx
// 전통적인 CSS 방식
<div className="card">...</div>
// .card { padding: 16px; border-radius: 8px; background: white; }

// Tailwind 방식
<div className="p-4 rounded-lg bg-white shadow">...</div>
```

**Tailwind v4의 변화**: 기존 v3에서는 `tailwind.config.ts`와 `postcss.config.ts` 설정 파일이 필요했지만, v4부터는 Vite 플러그인(`@tailwindcss/vite`)만 추가하면 됩니다.

```ts
// vite.config.ts
import tailwindcss from '@tailwindcss/vite'
export default defineConfig({
  plugins: [react(), tailwindcss()],
})
```

```css
/* index.css — 이 한 줄로 끝 */
@import "tailwindcss";
```

---

## 9. `api/client.ts` — API 클라이언트

**Spring에서의 대응**: `RestTemplate` / `FeignClient`의 기반 설정

```typescript
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      'X-User-Id': userId,   // POC용 인증 헤더
      ...init.headers,
    },
  })

  if (!res.ok) {
    throw new ApiError(res.status, ...)
  }
  return res.json()
}
```

`fetch`는 브라우저 내장 HTTP 클라이언트입니다. 이 `request` 함수는 모든 API 호출에서 공통으로 필요한 것들(Base URL, 인증 헤더, 에러 처리)을 한 곳에서 처리합니다. FeignClient의 `@RequestHeader`, `@PathVariable` 설정을 한 곳에 모아둔 것과 같습니다.

`X-User-Id` 헤더는 POC 단계의 임시 인증 방식입니다. Phase 4에서 JWT Cookie로 교체됩니다.

---

## 10. `main.tsx` — 애플리케이션 진입점

**Spring에서의 대응**: `@SpringBootApplication` + `ApplicationContext` 구성

```tsx
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>  {/* TanStack Query 컨텍스트 */}
      <RouterProvider router={router} />         {/* 라우터 */}
    </QueryClientProvider>
  </StrictMode>,
)
```

Spring의 `ApplicationContext`가 Bean들을 감싸듯, React는 Provider로 전역 컨텍스트를 제공합니다. `QueryClientProvider`는 하위 모든 컴포넌트에서 TanStack Query를 사용할 수 있게 해줍니다.
