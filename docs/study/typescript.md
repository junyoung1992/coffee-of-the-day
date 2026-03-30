# TypeScript 학습 가이드 — 프로젝트 코드 따라가기

> 대상 독자: Java/Spring 개발자
> 목표: Coffee of the Day 프론트 코드를 읽을 때 필요한 TypeScript, React, 관련 패턴을 빠르게 익힌다.

---

## 먼저 감 잡기: TypeScript를 Java와 어떻게 대응해서 보면 되나

| TypeScript | Java에서 비슷한 개념 |
|------|------|
| `interface` | interface / DTO 구조 정의 |
| `type` alias | 별칭 타입, union 타입 정의 |
| union type (`A \| B`) | sealed type / enum + subtype 조합 비슷 |
| discriminated union | `instanceof` 분기 가능한 다형적 DTO |
| generic (`<T>`) | Java generic |
| `null`, `undefined` | `null` + 값 없음 상태 |
| `async/await` | `CompletableFuture`를 더 직관적으로 쓰는 문법 느낌 |
| `import type` | 타입 전용 import |
| React hook | 상태/생명주기/참조 관리 API |

중요한 차이는 하나입니다.

**TypeScript는 "클래스 중심"보다 "값 + 함수 + 타입 조합"으로 작성하는 경우가 훨씬 많습니다.**

- 데이터 모양은 `type`, `interface`
- 동작은 함수
- 상태 관리는 hook

---

# Part 1: TypeScript 언어 기초

## `interface`와 `type` — 언제 무엇을 쓰나

### `interface`

객체 구조를 설명할 때 자주 씁니다.

```ts
export interface CursorPage<T> {
  items: T[]
  next_cursor: string | null
  has_next: boolean
}
```

Java로 보면 필드만 있는 DTO 정의입니다.

### `type`

조합형 타입을 만들 때 많이 씁니다. Java enum보다 가볍고 유연합니다.

```ts
export type FormLogType = 'cafe' | 'brew'
```

generated type에 프로젝트 전용 이름을 붙이는 용도로도 씁니다.

```ts
export type CreateLogInput = components['schemas']['CreateLogRequest']
```

---

## union type — "둘 중 하나"를 타입으로 표현

```ts
type FormLogType = 'cafe' | 'brew'
```

`string`이면 아무 문자열이나 들어갈 수 있지만, `'cafe' | 'brew'`로 좁혀두면 오타를 컴파일 단계에서 잡아줍니다.

---

## discriminated union — 이 프로젝트에서 가장 중요한 TypeScript 기능

```ts
export type CoffeeLogFull = CafeLogFull | BrewLogFull
```

`log_type`으로 구분합니다. `log.log_type === 'cafe'`로 좁히면 `log.cafe`가 존재함을 컴파일러가 보장합니다. Java의 `instanceof` + sealed interface와 비슷하지만, 값 검사만으로 narrowing이 가능합니다.

→ `types/log.ts`

---

## generic (`<T>`)

Java generic과 거의 같습니다.

```ts
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> { ... }
```

호출하는 쪽이 "이 API는 어떤 타입을 돌려줄지"를 타입으로 알려줍니다.

---

## optional chaining (`?.`)와 nullish coalescing (`??`)

```ts
data?.pages        // 왼쪽이 null/undefined면 에러 없이 undefined 반환
import.meta.env.VITE_API_BASE_URL ?? '/api/v1'  // null/undefined일 때만 기본값
```

`||`는 빈 문자열, 0, false도 fallback으로 취급하지만, `??`는 정말 `null`/`undefined`일 때만 fallback합니다. optional field 처리에서 이 차이가 중요합니다.

---

## `null`과 `undefined`

- `null`: 명시적으로 값이 없음
- `undefined`: 아직 할당되지 않았거나 존재하지 않음

이 프로젝트에서는 라이브러리 계약에 맞춰 구분합니다. 예: API 응답은 `next_cursor: string | null`이지만, TanStack Query는 "다음 페이지 없음"을 `undefined`로 해석합니다.

---

## 그 외 자주 쓰이는 문법

### `as const`

```ts
export const brewMethodOptions = [
  { label: 'Pour Over', value: 'pour_over' },
] as const
```

없으면 `value`가 `string`, 있으면 `'pour_over'` literal 타입으로 유지됩니다. 옵션 목록, query key, 타입 분기용 상수에 많이 쓰입니다.

### `async/await`

함수는 `Promise`를 반환하고, `await`는 Promise가 끝날 때까지 기다린 뒤 결과를 꺼냅니다. Java의 `CompletableFuture`보다 문법이 훨씬 직관적입니다.

### `import type`

런타임 값은 가져오지 않고 타입 정보만 import합니다. TypeScript는 build 단계에서 타입이 사라지므로, 타입 전용 import를 구분할 수 있습니다.

### 타입 단언 (`as ...`)

개발자가 "이 값은 이 타입으로 봐도 된다"고 컴파일러에게 알려주는 것. Java의 강제 캐스팅과 비슷하지만 런타임 변환이 아니라 타입 검사 보조입니다. 남용하면 타입 안전성이 약해지므로 필요한 경우에만 씁니다.

### 함수형 스타일

Java Stream API에 익숙하면 이해가 빠릅니다. `map`, `filter`, `flatMap`이 그대로 대응됩니다.

```ts
state.brew.brewSteps.map((step) => step.trim()).filter(Boolean)
```

---

# Part 2: React 패턴

## 핵심 hook 4개

### `useState`

컴포넌트 내부 상태를 저장합니다. 서버 세션이 아니라 **브라우저 화면 내부 상태**입니다.

```ts
const [form, setForm] = useState(() => createEmptyFormState())
```

### `useEffect`

외부 세계와 동기화할 때 사용합니다. 서버 데이터를 폼에 주입, DOM API 사용, observer 등록/해제 등.

### `useMemo`

비싼 계산 결과를 필요할 때만 다시 계산합니다.

### `useRef`

렌더 사이에 유지되는 mutable 값이나 DOM 참조를 저장합니다. `IntersectionObserver` 대상, hydration 여부 추적 등에 씁니다.

---

## 불변 상태 업데이트

React는 참조가 바뀌는지를 보고 다시 렌더링할지 판단합니다. 기존 객체를 직접 수정하면 상태 변경을 놓치기 쉬우므로, spread(`...`)로 새 객체/배열을 만들어 교체합니다.

```ts
setForm((prev) => ({ ...prev, logType }))
```

---

## controlled component와 state ownership

`TagInput`은 내부에서 태그 배열을 최종 소유하지 않습니다. 실제 태그 목록은 부모(`LogFormPage`)가 가지고, `TagInput`은 현재 입력 중인 문자열과 드롭다운 열림 상태만 가집니다.

이렇게 하면 폼 제출 시 부모 상태만 읽으면 되고, 재사용 컴포넌트가 특정 API 훅에 덜 묶입니다.

→ `components/TagInput.tsx`

---

## URL을 상태 저장소로 쓰기: `useSearchParams`

필터 상태를 컴포넌트 `useState`가 아니라 URL 쿼리 파라미터에 저장합니다. 새로고침해도 유지되고, 뒤로가기/앞으로가기와 자연스럽게 연결되며, URL 공유만으로 같은 필터 상태를 재현할 수 있습니다.

→ `pages/HomePage.tsx`

---

## `useId`와 ARIA

input과 suggestion list를 접근성 속성으로 연결합니다. DOM이 가까이 있다고 해서 스크린 리더가 관계를 자동 이해하지는 않으므로, `aria-controls`, `role="listbox"`, `role="option"`으로 의미를 연결합니다.

접근성 속성이 좋아질수록 E2E 테스트도 `getByRole` 기반으로 더 안정적이 됩니다.

→ `components/TagInput.tsx`

---

# Part 3: 서버 상태 관리 (TanStack Query)

## 상태 분류

| 상태 유형 | 도구 | 예시 |
|---|---|---|
| 서버 상태 | TanStack Query v5 | 커피 기록 목록, 상세, 자동완성 |
| 클라이언트 상태 | `useState` / `useReducer` | 폼 입력값, 드롭다운 열림 |
| URL 상태 | `useSearchParams` | 목록 필터 |

별도 전역 상태 라이브러리(Redux, Zustand 등)를 사용하지 않습니다.

---

## `useQuery`와 `useMutation`

```ts
const { data, isLoading, error } = useQuery({
  queryKey: ['logs', filters],
  queryFn: () => getLogs(filters),
})
```

기록 생성 후 `queryClient.invalidateQueries({ queryKey: ['logs'] })`로 목록을 자동 갱신합니다.

---

## `queryKey`는 캐시 식별자

단순한 옵션이 아니라, **요청 결과를 어떤 이름으로 캐시에 저장할지 정하는 키**입니다.

`['suggestions', 'tags', '초']`와 `['suggestions', 'tags', '초콜']`은 다른 캐시 엔트리입니다. 코드를 읽을 때 어떤 값이 키에 들어가는지, 어떤 값이 바뀌면 새 요청이 발생하는지를 함께 봐야 합니다.

---

## `enabled`로 조건부 요청

React에서 hook을 조건문 안에서 호출하면 안 됩니다. 대신 **hook 호출은 항상 유지하고, 내부 실행만 `enabled`로 제어**합니다.

```ts
return useQuery({
  queryKey: ['suggestions', type, q],
  queryFn: () => getSuggestions(type, q),
  enabled: q.length > 0,
})
```

"호출 위치의 안정성"이 React 규칙이고, "실행 여부"는 TanStack Query 옵션으로 푸는 방식입니다.

→ `hooks/useSuggestions.ts`

---

## `useInfiniteQuery` — 페이지를 배열로 쌓기

```ts
return useInfiniteQuery({
  queryKey: LOG_KEYS.list(params),
  queryFn: ({ pageParam }) => getLogs({ ...params, cursor: pageParam }),
  getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
})
```

- `data.pages`: 지금까지 받아온 페이지 배열
- `getNextPageParam`이 `undefined`를 반환하면 마지막 페이지

API 응답은 `null`, TanStack Query는 `undefined` — **경계마다 값 없음 표현이 다를 수 있다**는 점을 이해해야 합니다.

→ `hooks/useLogs.ts`

---

## Refresh Single-Flight — 중복 토큰 갱신 방지

`api/client.ts`의 토큰 갱신 인터셉터는 모듈 스코프의 `refreshPromise`를 공유합니다. 여러 요청이 동시에 401을 받으면, 각각이 `/auth/refresh`를 호출하는 대신 하나의 Promise를 함께 기다립니다.

```ts
let refreshPromise: Promise<void> | null = null

// 401 수신 시
if (!refreshPromise) {
  refreshPromise = doFetch('/auth/refresh', { method: 'POST' })
    .then(...)
    .finally(() => { refreshPromise = null })
}
await refreshPromise  // 이미 진행 중이면 같은 Promise를 기다림
```

Java에서는 `synchronized` 블록이나 `CompletableFuture`로 비슷한 문제를 풀지만, JavaScript는 싱글 스레드이므로 모듈 변수 하나로 자연스럽게 구현됩니다. 이미 완료된 Promise도 `await`할 수 있는 특성 덕분에 race condition 없이 동작합니다.

→ `api/client.ts`

---

## 인증 상태도 TanStack Query 캐시로 관리

로그인 성공 시 `setQueryData`로 즉시 캐시 저장, 로그아웃 시 `queryClient.clear()`로 전체 캐시 제거합니다.

→ `hooks/useAuth.ts`

---

# Part 4: 브라우저 플랫폼

## `IntersectionObserver` — 무한 스크롤

`useRef`로 DOM 노드를 잡고, `useEffect`에서 observer를 등록하고, cleanup 함수에서 반드시 해제합니다. **브라우저 API를 React 식으로 안전하게 묶는 패턴**입니다.

→ `pages/HomePage.tsx`

---

## `Date`와 `toISOString()` — 시간대 함정

`toISOString()`은 항상 UTC 문자열을 반환합니다. 사용자가 로컬 시간으로 입력한 값이 다른 시간대로 변환되어 전송될 수 있습니다.

코드를 읽을 때: 입력 값이 로컬 시간인지, 서버에 보내는 값은 UTC instant인지, 화면에 보여줄 때는 어떤 시간대로 해석하는지를 함께 봐야 합니다.

---

## 브라우저 이벤트 순서: `blur`가 `click`보다 먼저

드롭다운 항목을 클릭할 때 순서: `mousedown` → input `blur` → `mouseup` → `click`.

`onBlur`에서 드롭다운을 닫으면 `click` 시점에 항목이 이미 사라집니다. 그래서 `onMouseDown`에서 먼저 잡고 `preventDefault()`로 blur를 막습니다.

→ `components/TagInput.tsx`

---

## E2E와 접근성

`getByRole`, `getByLabel` 기반 셀렉터를 사용합니다. 접근성 라벨이 좋아질수록 테스트도 안정적이 됩니다. E2E는 **컴포넌트 접근성과 테스트 가능성을 함께 점검하는 도구**이기도 합니다.

→ `e2e/log-happy-path.spec.ts`

---

# 학습 순서 가이드

## TypeScript + React 문법 우선순위

1. `interface`, `type`, union type
2. discriminated union
3. generic, `async/await`, `Promise<T>`
4. `?.`, `??`, `as const`
5. 불변 상태 업데이트 (`...spread`)
6. React hook (`useState`, `useEffect`, `useMemo`, `useRef`)
7. `useSearchParams`, `URLSearchParams`
8. `useQuery`, `useMutation`, `queryKey`, `enabled`
9. `useInfiniteQuery`
10. 브라우저 API (`IntersectionObserver`, `Date`)
11. controlled component + 상태 끌어올리기
12. 브라우저 이벤트 순서 (`mousedown`, `blur`, `click`)
13. `useId`와 ARIA 속성

## 코드 읽기 추천 순서

DTO → Client → Service-like hook → Controller-like page 순서가 가장 읽기 쉽습니다.

1. `types/log.ts` — 핵심 타입 구조 파악
2. `api/client.ts` — API 호출 기본 방식 이해
3. `api/logs.ts` — CRUD 함수 파악
4. `hooks/useLogs.ts` — 서버 상태 관리 방식 이해
5. `pages/logFormState.ts` — 폼 상태 변환 이해
6. `components/FilterBar.tsx` — 제어 컴포넌트와 필터 props 구조
7. `pages/HomePage.tsx` — URL 상태, 무한 스크롤, 상태 분기
8. `hooks/useSuggestions.ts` — 조건부 쿼리와 캐시 키 설계
9. `components/TagInput.tsx` — controlled component, 이벤트 처리, 접근성
10. `pages/LogDetailPage.tsx`, `pages/LogFormPage.tsx`
11. `e2e/log-happy-path.spec.ts` — 사용자 흐름 검증

---

*이 문서는 Codex가 작성하고, Claude가 보강했습니다.*
