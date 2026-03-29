# Frontend 아키텍처 결정 문서

> 각 기술 선택의 이유와 트레이드오프를 설명합니다.

---

## 번들러: Vite

**결정**: Vite 사용 (Create React App 대신).

CRA(Create React App)는 2023년 이후 사실상 유지보수가 중단된 상태입니다. 현재 React 공식 문서도 Vite를 권장합니다.

| | CRA | **Vite** |
|--|-----|----------|
| 개발 서버 시작 | 전체 번들링 후 시작 (느림) | 네이티브 ESM, 즉시 시작 |
| HMR (코드 변경 반영) | 느림 | 거의 즉각적 |
| 유지보수 상태 | 사실상 중단 | 활발히 관리됨 |

---

## UI 프레임워크: React

**결정**: 사용자가 선택.

현재 프로젝트는 **React 19**를 사용합니다.

이 프로젝트와 직접 관련 있는 포인트:
- 컴포넌트 단위 개발로 카페 폼 / 브루 폼 / 자동완성 입력을 독립적으로 구성하기 용이
- `StrictMode` + 함수형 컴포넌트 기반으로 상태 흐름을 비교적 단순하게 유지 가능
- TanStack Query, React Router와 조합이 자연스러워 페이지 단위 상태 관리에 적합

---

## 스타일: Tailwind CSS

**결정**: Tailwind CSS v4 사용.

POC에서 스타일링에 쓰는 시간을 최소화하기 위한 선택입니다.

**왜 Tailwind인가**

- CSS 파일을 별도로 관리하지 않아도 됨 — 컴포넌트에 클래스만 쓰면 됨
- 디자인 토큰(색상, 간격, 폰트 크기)이 미리 정의되어 있어 일관된 UI가 자연스럽게 만들어짐
- POC에서 빠른 프로토타이핑에 적합

**트레이드오프**
- JSX 마크업이 클래스 이름으로 길어질 수 있음 → 반복되는 패턴은 컴포넌트로 추출해서 해결
- 커스텀 디자인 시스템이 필요해지면 CSS 변수나 Tailwind 설정 확장으로 대응 가능

**v4 기준 현재 적용 방식**

이 프로젝트는 예전 Tailwind v3 방식(`tailwind.config.ts`, `postcss.config.ts`) 대신, Vite 플러그인과 CSS import 방식으로 붙입니다.

```ts
// vite.config.ts
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
})
```

```css
/* src/index.css */
@import "tailwindcss";
```

즉, 지금 문서를 읽는 기준점은 "설정 파일을 많이 만지는 Tailwind"가 아니라 **Vite 플러그인 중심의 Tailwind v4**입니다.

---

## 서버 상태 관리: TanStack Query (React Query)

**결정**: TanStack Query v5 사용.

이 프로젝트에서 다루는 상태는 크게 세 종류입니다.

1. **서버 상태**: 커피 기록 목록, 기록 상세, 자동완성 제안처럼 API에서 가져오고 캐시해야 하는 값
2. **클라이언트 상태**: 폼 입력값, 드롭다운 열림 여부처럼 컴포넌트 내부에만 머무는 값
3. **URL 상태**: 목록 필터처럼 공유/북마크가 가능해야 하는 값

Redux나 Zustand 같은 전역 상태 관리 라이브러리는 주로 클라이언트 상태를 위한 도구입니다. 하지만 이 프로젝트의 주요 상태는 서버 상태이고, 서버 상태를 전역 스토어에 넣으면 캐시 무효화, 로딩/에러 처리, 재시도 로직을 직접 구현해야 합니다.

TanStack Query는 이 모든 것을 해결합니다.

```typescript
// 이게 전부. 캐싱, 로딩, 에러, 재시도가 자동으로 처리됨
const { data, isLoading, error } = useQuery({
  queryKey: ['logs', filters],
  queryFn: () => getLogs(filters),
})

// 기록 생성 후 목록 자동 갱신
const mutation = useMutation({
  mutationFn: createLog,
  onSuccess: () => queryClient.invalidateQueries({ queryKey: ['logs'] }),
})
```

**클라이언트 상태**(폼 입력, 탭 선택 등)는 `useState` / `useReducer`로 충분합니다. 별도 전역 상태 라이브러리를 추가하지 않습니다.

**URL 상태**는 `useSearchParams`로 관리합니다. Phase 2부터 목록 필터를 URL에 반영해 새로고침, 공유, 뒤로가기와 자연스럽게 연결되도록 설계했습니다.

Phase 3 자동완성도 같은 원칙을 따릅니다.

- API 호출: `useQuery`
- 목록 누적: `useInfiniteQuery`
- 조건부 요청: `enabled`
- 캐시 구분: `queryKey`

즉, 이 프로젝트에서 TanStack Query는 단순 fetch helper가 아니라 **서버 상태 캐시 계층**입니다.

---

## 라우팅: React Router v7

**결정**: React Router v7 사용.

현재 구현은 `createBrowserRouter` 기반의 단순 라우트 구성입니다.

```tsx
export const router = createBrowserRouter([
  { path: '/', element: <HomePage /> },
  { path: '/logs/new', element: <LogFormPage /> },
  { path: '/logs/:id', element: <LogDetailPage /> },
  { path: '/logs/:id/edit', element: <LogFormPage /> },
])
```

라우트 수가 적고, 각 화면의 데이터 로딩은 이미 TanStack Query 훅이 담당하고 있으므로 현재 단계에서는 `loader` / `action` 중심 구조보다 이 구성이 더 단순합니다.

```
/                    → HomePage (기록 목록)
/logs/new            → LogFormPage (신규 작성)
/logs/:id            → LogDetailPage (상세)
/logs/:id/edit       → LogFormPage (수정, 같은 컴포넌트 재사용)
```

`/logs/new`와 `/logs/:id/edit`을 같은 `LogFormPage` 컴포넌트로 처리합니다. `id` 파라미터 유무로 생성/수정을 구분합니다.

---

## 디렉토리 구조와 설계 원칙

```
src/
├── pages/          # 라우트와 1:1 대응하는 페이지 컴포넌트
├── components/     # 여러 페이지에서 재사용되는 UI 컴포넌트
├── api/            # API 호출 함수 (fetch 래핑)
├── types/          # OpenAPI 생성 타입 + 파생 타입
└── hooks/          # TanStack Query를 래핑한 커스텀 훅
```

**왜 `api/`와 `hooks/`를 분리하는가**

- `api/`: 순수 함수. HTTP 요청을 보내고 응답을 반환. React에 의존하지 않음.
- `hooks/`: TanStack Query 훅. React 컴포넌트에서 사용. 캐싱·에러·로딩 상태 포함.

이렇게 분리하면 `api/`를 테스트할 때 React 환경이 필요 없고, 나중에 TanStack Query를 다른 라이브러리로 교체하더라도 `api/`는 손대지 않아도 됩니다.

---

## 타입 설계: OpenAPI 생성 타입 + Discriminated Union

프론트 타입의 **단일 소스 오브 트루스는 `openapi.yml`** 입니다.

흐름은 다음과 같습니다.

1. `openapi.yml` 수정
2. `npm run generate`
3. `src/types/schema.ts` 자동 생성
4. `src/types/*.ts`에서 프로젝트 친화적인 alias / 파생 타입 정의

즉, 프론트 타입을 백엔드 Go 코드에서 손으로 옮기지 않습니다.

```typescript
// types/log.ts

export type CoffeeLogResponse = components['schemas']['CoffeeLogResponse']

export type CafeLogFull = Omit<CoffeeLogResponse, 'log_type' | 'cafe' | 'brew'> & {
  log_type: 'cafe'
  cafe: NonNullable<CoffeeLogResponse['cafe']>
  brew?: never
}

export type BrewLogFull = Omit<CoffeeLogResponse, 'log_type' | 'cafe' | 'brew'> & {
  log_type: 'brew'
  brew: NonNullable<CoffeeLogResponse['brew']>
  cafe?: never
}

export type CoffeeLogFull = CafeLogFull | BrewLogFull
```

OpenAPI 3.0 스키마만으로는 `log_type`에 따라 `cafe` 또는 `brew`가 반드시 존재한다는 제약을 프론트 타입에 완전히 표현하기 어렵습니다. 그래서 생성 타입 위에 TypeScript 전용 discriminated union을 한 겹 더 얹습니다.

이 구조에서 TypeScript 컴파일러는 `log.log_type === 'cafe'`로 좁히면 `log.cafe`가 존재함을 보장합니다. 런타임 오류 없이 타입 안전하게 서브 타입별 UI를 렌더링할 수 있습니다.

```tsx
// 타입 가드 없이 안전하게 분기
if (log.log_type === 'cafe') {
  return <CafeLogDetail cafe={log.cafe} />  // log.cafe 타입이 CafeDetail로 좁혀짐
} else {
  return <BrewLogDetail brew={log.brew} />  // log.brew 타입이 BrewDetail로 좁혀짐
}
```

이 방식의 장점:

- API 계약 변경 시 `openapi.yml`을 기준으로 일관되게 반영 가능
- 생성 타입과 화면용 파생 타입의 역할이 분리됨
- Phase 3처럼 `SuggestionsResponse`가 추가되어도 같은 워크플로우로 확장 가능

---

*Last updated: 2026-03-29 (React 19 / React Router v7 / Tailwind CSS v4 / OpenAPI 타입 흐름 반영)*
