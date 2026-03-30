# Frontend 아키텍처 결정 문서

> 이 코드베이스에서 작업할 때 알아야 하는 비자명한 규칙과 제약을 설명합니다.

---

## 디렉토리 구조와 분리 원칙

```
src/
├── pages/          # 라우트와 1:1 대응하는 페이지 컴포넌트
├── components/     # 여러 페이지에서 재사용되는 UI 컴포넌트
├── api/            # API 호출 함수 (순수 함수, React 무의존)
├── types/          # OpenAPI 생성 타입 + 파생 타입
├── hooks/          # TanStack Query를 래핑한 커스텀 훅
└── utils/          # 도메인 로직 유틸리티 (React 무의존, 테스트 용이)
```

`api/`와 `hooks/`를 분리하는 이유: `api/`는 React 없이 테스트 가능하고, 상태 관리 라이브러리를 교체해도 `api/`는 영향 없음.

---

## 상태 관리 전략

| 상태 유형 | 도구 | 예시 |
|---|---|---|
| 서버 상태 | TanStack Query v5 | 커피 기록 목록, 기록 상세, 자동완성 제안 |
| 클라이언트 상태 | `useState` / `useReducer` | 폼 입력값, 드롭다운 열림 여부 |
| URL 상태 | `useSearchParams` | 목록 필터 (새로고침/공유/뒤로가기 보존) |

별도 전역 상태 라이브러리(Redux, Zustand 등)를 사용하지 않는다.

**인증 상태도 TanStack Query 캐시로 관리한다.** 로그인 성공 시 `setQueryData`로 즉시 캐시 저장, 로그아웃 시 `queryClient.clear()`로 전체 캐시 제거.

→ `hooks/useAuth.ts`

---

## 타입 설계: OpenAPI → Discriminated Union

`openapi.yml`이 단일 소스 오브 트루스.

1. `openapi.yml` 수정
2. `npm run generate` → `src/types/schema.ts` 자동 생성
3. `src/types/*.ts`에서 프로젝트 친화적인 파생 타입 정의

OpenAPI 3.0 스키마만으로는 `log_type`에 따라 `cafe` 또는 `brew`가 반드시 존재한다는 제약을 표현하기 어렵다. 그래서 생성 타입 위에 TypeScript discriminated union을 한 겹 더 얹는다. `log.log_type === 'cafe'`로 좁히면 `log.cafe` 존재가 보장된다.

→ `types/log.ts`

---

## 라우팅: React Router v7

`RootLayout`이 모든 라우트를 감싸는 최상위 레이아웃 컴포넌트. `ScrollRestoration`(페이지 전환 시 스크롤 초기화)과 `ScrollToTop`(스크롤 최상단 이동 버튼)을 라우터 레벨에서 한 번만 선언한다. 개별 페이지가 스크롤 로직을 신경쓸 필요 없음.

`ProtectedRoute`로 인증이 필요한 라우트를 감싸는 구조. `ProtectedRoute`는 `react-refresh/only-export-components` 규칙 때문에 독립 파일로 분리되어 있다 (같은 파일에서 컴포넌트 + 라우터 설정 객체를 export하면 HMR이 전체 리로드).

→ `router.tsx`, `components/ProtectedRoute.tsx`, `components/ScrollToTop.tsx`

---

## Refresh Single-Flight

`api/client.ts`의 토큰 갱신 인터셉터는 single-flight 패턴을 사용한다. 여러 요청이 동시에 401을 받으면 모듈 스코프의 `refreshPromise`를 공유해 `/auth/refresh`를 한 번만 호출한다. Refresh rotation 도입 시 중복 요청은 race condition을 유발하므로 이 패턴이 필수적이다.

→ `api/client.ts`

---

## Tailwind CSS v4

Vite 플러그인 방식으로 적용. `tailwind.config.ts` / `postcss.config.ts` 없음.

→ `vite.config.ts`, `src/index.css`

---

## 날짜 유틸리티와 타임존

`toISOString()`은 UTC 기준이라 KST 자정~오전 사이에 날짜가 하루 밀린다. 이 문제를 방지하기 위해 모든 날짜 포매팅은 `getFullYear()` / `getMonth()` / `getDate()`로 로컬 타임존을 유지한다. 날짜 관련 유틸리티는 `utils/date.ts`에 모아두며, React 무의존이므로 단위 테스트가 용이하다.

→ `utils/date.ts`

---

## 환경 변수와 API 접근

모든 환경에서 상대 경로(`/api/v1`)를 기본으로 사용한다.

- **운영**: Go 바이너리가 SPA를 embed해 동일 origin 서빙 → CORS 불필요
- **로컬**: Vite proxy가 `/api` 요청을 Go 서버(`localhost:8080`)로 전달

→ `vite.config.ts`, `.env`

---

*Last updated: 2026-03-30*
