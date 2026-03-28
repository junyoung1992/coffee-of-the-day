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

React 18의 주요 특징 중 이 프로젝트와 직접 관련 있는 것:
- `Suspense` + TanStack Query 조합으로 로딩 상태를 선언적으로 처리 가능
- 컴포넌트 단위 개발로 카페 폼 / 브루 폼을 독립적으로 구성하기 용이

---

## 스타일: Tailwind CSS

**결정**: Tailwind CSS v3 사용.

POC에서 스타일링에 쓰는 시간을 최소화하기 위한 선택입니다.

**왜 Tailwind인가**

- CSS 파일을 별도로 관리하지 않아도 됨 — 컴포넌트에 클래스만 쓰면 됨
- 디자인 토큰(색상, 간격, 폰트 크기)이 미리 정의되어 있어 일관된 UI가 자연스럽게 만들어짐
- POC에서 빠른 프로토타이핑에 적합

**트레이드오프**
- JSX 마크업이 클래스 이름으로 길어질 수 있음 → 반복되는 패턴은 컴포넌트로 추출해서 해결
- 커스텀 디자인 시스템이 필요해지면 `tailwind.config`에서 토큰을 정의해 확장 가능

---

## 서버 상태 관리: TanStack Query (React Query)

**결정**: TanStack Query v5 사용.

이 프로젝트에서 상태는 크게 두 종류입니다.

1. **서버 상태**: 커피 기록 목록, 기록 상세 — API에서 가져오고, 캐시하고, 무효화해야 함
2. **클라이언트 상태**: 폼 입력값, 필터 탭 선택 — 브라우저 내에서만 존재

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

---

## 라우팅: React Router v6

**결정**: React Router v6 사용.

현재 React 생태계에서 가장 표준적인 라우터입니다. v6의 `createBrowserRouter` API를 사용하면 라우트 단위로 `loader` / `action`을 정의할 수 있어 나중에 TanStack Query와 prefetch를 연동하기도 용이합니다.

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
├── types/          # 도메인 타입 정의
└── hooks/          # TanStack Query를 래핑한 커스텀 훅
```

**왜 `api/`와 `hooks/`를 분리하는가**

- `api/`: 순수 함수. HTTP 요청을 보내고 응답을 반환. React에 의존하지 않음.
- `hooks/`: TanStack Query 훅. React 컴포넌트에서 사용. 캐싱·에러·로딩 상태 포함.

이렇게 분리하면 `api/`를 테스트할 때 React 환경이 필요 없고, 나중에 TanStack Query를 다른 라이브러리로 교체하더라도 `api/`는 손대지 않아도 됩니다.

---

## 타입 설계: Discriminated Union

백엔드 API 응답과 1:1로 대응하는 TypeScript 타입을 정의합니다.

```typescript
// types/log.ts

type LogType = 'cafe' | 'brew'

interface CoffeeLogBase {
  id: string
  userId: string
  recordedAt: string
  companions: string[]
  logType: LogType
  memo?: string
  createdAt: string
  updatedAt: string
}

interface CafeLogFull extends CoffeeLogBase {
  logType: 'cafe'
  cafe: CafeDetail
}

interface BrewLogFull extends CoffeeLogBase {
  logType: 'brew'
  brew: BrewDetail
}

type CoffeeLogFull = CafeLogFull | BrewLogFull
```

이 구조에서 TypeScript 컴파일러는 `log.logType === 'cafe'`로 좁히면 `log.cafe`가 존재함을 보장합니다. 런타임 오류 없이 타입 안전하게 서브 타입별 UI를 렌더링할 수 있습니다.

```tsx
// 타입 가드 없이 안전하게 분기
if (log.logType === 'cafe') {
  return <CafeLogDetail cafe={log.cafe} />  // log.cafe 타입이 CafeDetail로 좁혀짐
} else {
  return <BrewLogDetail brew={log.brew} />  // log.brew 타입이 BrewDetail로 좁혀짐
}
```

---

*Last updated: 2026-03-28*
