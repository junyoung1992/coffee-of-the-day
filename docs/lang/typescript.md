# TypeScript 학습 가이드 — 프로젝트 코드 따라가기

> 대상 독자: Java/Spring 개발자
> 목표: Coffee of the Day 프론트 코드를 읽을 때 필요한 TypeScript와 React 문법을 빠르게 익힌다.

---

## 1. 먼저 감 잡기: TypeScript를 Java와 어떻게 대응해서 보면 되나

TypeScript는 Java처럼 **정적 타입**을 가지지만, 실행 환경은 JVM이 아니라 브라우저/Node.js입니다.  
즉, 문법은 Java와 다르지만 "컴파일 전에 타입으로 실수를 줄인다"는 목적은 비슷합니다.

이 프로젝트 기준 대응은 대략 이렇습니다.

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

즉, Java처럼 모든 걸 class로 만들기보다:

- 데이터 모양은 `type`, `interface`
- 동작은 함수
- 상태 관리는 hook

방식으로 읽는 편이 자연스럽습니다.

---

## 2. `interface`와 `type` — 언제 무엇을 쓰나

### `interface`

객체 구조를 설명할 때 자주 씁니다.

```ts
export interface CursorPage<T> {
  items: T[]
  next_cursor: string | null
  has_next: boolean
}
```

Java로 보면:

```java
public class CursorPage<T> {
    List<T> items;
    String nextCursor;
    boolean hasNext;
}
```

과 비슷한 DTO 정의입니다.

### `type`

조합형 타입을 만들 때 많이 씁니다.

```ts
export type FormLogType = 'cafe' | 'brew'
```

Java enum과 비슷하지만, 더 가볍고 유연합니다.

또는 이런 식으로 별칭도 만듭니다.

```ts
export type CreateLogInput = components['schemas']['CreateLogRequest']
```

즉, generated type에 프로젝트 전용 이름을 붙이는 용도로도 씁니다.

---

## 3. union type — "둘 중 하나"를 타입으로 표현

TypeScript에서 가장 중요한 기능 중 하나입니다.

```ts
type FormLogType = 'cafe' | 'brew'
```

의미:
- 값은 `'cafe'` 또는 `'brew'` 둘 중 하나

Java에서는 enum으로 표현할 부분이지만, TypeScript에서는 문자열 literal union을 자주 씁니다.

### 왜 좋은가

이 값이 `string`이면 아무 문자열이나 들어갈 수 있습니다.  
하지만 `'cafe' | 'brew'`로 좁혀두면 오타를 컴파일 단계에서 잡아줍니다.

---

## 4. discriminated union — 이 프로젝트에서 가장 중요한 TypeScript 기능

이 프로젝트의 핵심 타입은 이것입니다.

```ts
export type CoffeeLogFull = CafeLogFull | BrewLogFull
```

그리고 각 타입은 `log_type`으로 구분됩니다.

```ts
if (log.log_type === 'cafe') {
  log.cafe.cafe_name
}
```

이걸 **discriminated union**이라고 합니다.

Java에서 비슷한 느낌은:

- `instanceof`
- sealed interface + record subtype

입니다.

예를 들어 Java라면:

```java
if (log instanceof CafeLogFull cafe) {
    cafe.getCafe().getCafeName();
}
```

처럼 분기할 수 있는데, TypeScript는 `log_type` 값 검사만으로 비슷한 narrowing이 가능합니다.

### 왜 중요한가

이 기능 덕분에:

- `cafe` 로그에서는 `log.cafe`가 있다고 컴파일러가 확신
- `brew` 로그에서는 `log.brew`가 있다고 컴파일러가 확신

즉, null 체크를 매번 강요하지 않습니다.

---

## 5. optional chaining (`?.`)와 nullish coalescing (`??`)

프론트 코드에서 매우 자주 등장합니다.

### optional chaining

```ts
data?.pages
log?.log_type
```

의미:
- 왼쪽 값이 `null` 또는 `undefined`면 에러를 내지 않고 `undefined` 반환

Java에서 `Optional.map(...)`을 가볍게 쓰는 느낌과 비슷합니다.

### nullish coalescing

```ts
const baseUrl = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'
```

의미:
- 왼쪽이 `null` 또는 `undefined`일 때만 오른쪽 기본값 사용

Java로 치면:

```java
String value = envValue != null ? envValue : defaultValue;
```

와 비슷합니다.

### `||`와 왜 다른가

`||`는 빈 문자열, 0, false도 fallback으로 취급합니다.  
`??`는 정말 `null`/`undefined`일 때만 fallback합니다.

이 차이는 optional field 처리에서 꽤 중요합니다.

---

## 6. generic (`<T>`) — Java generic과 거의 같습니다

```ts
export interface CursorPage<T> {
  items: T[]
  next_cursor: string | null
  has_next: boolean
}
```

Java의:

```java
class CursorPage<T> {
    List<T> items;
}
```

와 같은 개념입니다.

이 프로젝트에서 generic이 쓰이는 주요 위치:

- `CursorPage<T>`
- `request<T>()`
- TanStack Query 반환 타입

### `request<T>()`

```ts
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  ...
}
```

즉, 호출하는 쪽이 "이 API는 어떤 타입을 돌려줄지"를 타입으로 알려줍니다.

Java의 제네릭 메서드와 거의 같은 사고방식입니다.

---

## 7. `import type` — 타입만 가져올 때 쓰는 문법

```ts
import type { CoffeeLogFull, CreateLogInput } from '../types/log'
```

의미:
- 런타임 값은 가져오지 않고, 타입 정보만 import

Java는 타입 정보와 runtime symbol이 같은 세계에 있지만, TypeScript는 build 단계에서 타입이 사라집니다.  
그래서 타입 전용 import를 구분할 수 있습니다.

실무에서는:

- 번들 최적화
- 의도 표현

측면에서 유용합니다.

---

## 8. `as const` — 값을 더 좁은 타입으로 고정

```ts
export const brewMethodOptions = [
  { label: 'Pour Over', value: 'pour_over' },
  ...
] as const
```

`as const`가 없으면 `value` 타입은 그냥 `string`이 되기 쉽습니다.  
`as const`를 붙이면 `'pour_over'` 같은 literal 타입으로 유지됩니다.

즉, Java enum 상수처럼 더 좁고 안전한 값으로 다루고 싶을 때 유용합니다.

이 프로젝트에서는:

- 옵션 목록
- query key
- 타입 분기용 상수

에 많이 쓰입니다.

---

## 9. `async/await` — 비동기 코드를 순차 코드처럼 읽기

프론트 API와 이벤트 처리에서 많이 씁니다.

```ts
export async function request<T>(...) {
  const res = await fetch(...)
  ...
}
```

또는:

```ts
async function handleDelete() {
  await deleteMutation.mutateAsync(id)
  navigate('/')
}
```

Java에서 `CompletableFuture`보다 문법이 훨씬 직관적이라고 보면 됩니다.

중요한 점:
- 함수는 실제로 `Promise`를 반환합니다.
- `await`는 Promise가 끝날 때까지 기다린 뒤 결과를 꺼내는 문법입니다.

---

## 10. 객체/배열 불변 업데이트

React 상태는 직접 수정하지 않고 새 값을 만들어 교체합니다.

예:

```ts
setForm((prev) => ({
  ...prev,
  logType,
}))
```

또는 배열도:

```ts
const nextSteps = [...prev.brew.brewSteps]
```

Java에서 immutable DTO를 새로 만들어 교체하는 사고와 비슷합니다.

### 왜 이렇게 하나

React는 참조가 바뀌는지를 보고 다시 렌더링할지 판단하는 경우가 많습니다.  
기존 객체를 직접 수정하면 상태 변경을 놓치기 쉽습니다.

즉:

- `prev.logType = 'brew'` 같은 직접 변경은 피함
- spread(`...`)로 새 객체/배열 생성

이 패턴이 기본입니다.

---

## 11. React hook — Phase 1-2에서 꼭 알아야 할 4개

### `useState`

컴포넌트 내부 상태를 저장합니다.

```ts
const [form, setForm] = useState(() => createEmptyFormState())
```

Java/Spring에는 직접 대응되는 개념은 없지만, 서버 세션이 아니라 **브라우저 화면 내부 상태**라고 생각하면 됩니다.

### `useEffect`

외부 세계와 동기화할 때 사용합니다.

```ts
useEffect(() => {
  ...
}, [isEditMode, log])
```

예:
- 서버에서 받아온 데이터를 폼 상태에 주입
- DOM API 사용
- observer 등록/해제

Spring으로 비유하면 lifecycle callback 느낌이지만, 더 UI 중심입니다.

### `useMemo`

비싼 계산 결과를 필요할 때만 다시 계산합니다.

```ts
const logs = useMemo(
  () => data?.pages.flatMap((page) => page.items) ?? [],
  [data?.pages],
)
```

매 렌더마다 다시 펼치지 않도록 하는 최적화입니다.

### `useRef`

렌더 사이에 유지되는 mutable 값이나 DOM 참조를 저장합니다.

```ts
const sentinelRef = useRef<HTMLDivElement | null>(null)
```

이 프로젝트에서는:

- `IntersectionObserver` 대상 DOM 참조
- hydration 여부 추적

용도로 씁니다.

---

## 12. URL을 상태 저장소로 쓰기: `useSearchParams`

Phase 2부터는 필터 상태를 컴포넌트 `useState`가 아니라 URL 쿼리 파라미터에 저장합니다.

```ts
const [searchParams, setSearchParams] = useSearchParams()
const logType = parseLogType(searchParams.get('log_type'))
```

Java/Spring으로 비유하면, 서버가 `@RequestParam`으로 화면 상태를 복원하는 것과 비슷합니다.

이 방식의 장점:

- 새로고침해도 상태가 유지됨
- 뒤로가기/앞으로가기와 자연스럽게 연결됨
- URL 공유만으로 같은 필터 상태를 재현 가능

즉, **공유 가능하고 북마크 가능한 상태는 URL에 두는 편이 더 자연스럽다**고 이해하면 됩니다.

### `URLSearchParams`

```ts
setSearchParams((prev) => {
  const next = new URLSearchParams(prev)
  next.set('date_from', value)
  return next
})
```

`URLSearchParams`는 브라우저가 제공하는 쿼리 문자열 조작 객체입니다.  
기존 값을 복사한 뒤 일부 키만 바꾸는 방식은 React의 불변 업데이트와 같은 사고입니다.

---

## 13. `useInfiniteQuery` — 페이지를 배열로 쌓는 서버 상태 관리

Phase 2 목록은 `useQuery`가 아니라 `useInfiniteQuery`를 사용합니다.

```ts
return useInfiniteQuery({
  queryKey: LOG_KEYS.list(params),
  queryFn: ({ pageParam }) => getLogs({ ...params, cursor: pageParam }),
  getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
})
```

여기서 중요한 개념:

- `pageParam`: 다음 페이지 요청에 사용할 커서
- `data.pages`: 지금까지 받아온 페이지 배열
- `getNextPageParam`: 마지막 응답에서 다음 커서를 꺼내는 함수

Java/Spring으로 치면, cursor pagination 응답을 클라이언트 메모리 위에 누적하는 느낌입니다.

### 왜 `null`이 아니라 `undefined`를 반환하나

API 응답은 `next_cursor: string | null`입니다.  
하지만 TanStack Query는 "다음 페이지 없음"을 `undefined`로 해석합니다.

즉:

- API 계약: `null`
- 라이브러리 계약: `undefined`

처럼 **경계마다 값 없음 표현이 다를 수 있다**는 점을 이해해야 합니다.

---

## 14. 브라우저 API와 hook 결합: `IntersectionObserver`

Phase 2 무한 스크롤은 `IntersectionObserver`를 사용합니다.

```ts
const sentinelRef = useRef<HTMLDivElement | null>(null)

useEffect(() => {
  const node = sentinelRef.current
  if (!node || !hasNextPage) return

  const observer = new IntersectionObserver((entries) => {
    if (entries.some((entry) => entry.isIntersecting)) {
      void fetchNextPage()
    }
  })

  observer.observe(node)
  return () => observer.disconnect()
}, [fetchNextPage, hasNextPage])
```

읽는 포인트는 세 가지입니다.

1. `useRef`로 DOM 노드를 잡는다
2. `useEffect`에서 observer를 등록한다
3. cleanup 함수에서 반드시 해제한다

즉, 이 코드는 "React 문법"만이 아니라 **브라우저 API를 React 식으로 안전하게 묶는 패턴**입니다.

---

## 15. `Date`와 `toISOString()` — 로컬 시간대 함정

Phase 2에서는 날짜 필터와 `recorded_at` 입력 때문에 JavaScript `Date`의 함정도 알아야 합니다.

```ts
const parsed = new Date('2026-03-29T09:30')
parsed.toISOString()
```

이때 `toISOString()`은 항상 UTC 문자열을 반환합니다.  
즉, 사용자가 로컬 시간으로 입력한 값이 다른 시간대로 변환되어 전송될 수 있습니다.

Java로 치면 `LocalDateTime`을 다루다가 어느 순간 `Instant`로 변환되는 상황과 비슷합니다.

그래서 프론트 코드를 읽을 때는:

- 입력 값이 로컬 시간인지
- 서버에 보내는 값은 UTC instant인지
- 다시 화면에 보여줄 때는 어떤 시간대로 해석하는지

를 함께 봐야 합니다.

---

## 16. 접근성 속성과 E2E 안정성은 같이 간다

Phase 2에서는 Playwright E2E가 추가되면서 `getByRole`, `getByLabel` 기반 셀렉터를 적극적으로 사용합니다.

```ts
await page.getByRole('button', { name: '브루' }).click()
await page.getByLabel('Brew step 1').fill('30초 뜸들이기 후 천천히 프레스')
```

이 방식이 좋은 이유:

- 화면 텍스트 위치에 덜 의존함
- 중복 요소가 있어도 의미 기반으로 찾기 쉬움
- 접근성 라벨이 좋아질수록 테스트도 안정적이 됨

즉, E2E는 **컴포넌트 접근성과 테스트 가능성을 함께 점검하는 도구**이기도 합니다.

---

## 17. `Promise<T>`와 API 계층 읽는 법

프론트 API 함수는 보통 이렇게 생깁니다.

```ts
export function getLog(id: string): Promise<CoffeeLogFull> {
  return request(`/logs/${id}`)
}
```

즉:

- 지금 당장 `CoffeeLogFull`이 있는 게 아니라
- 나중에 resolve될 `Promise<CoffeeLogFull>`가 반환됨

Java에서는 `CompletableFuture<CoffeeLogFull>` 같은 느낌입니다.

---

## 18. 함수형 스타일이 많다

TypeScript/React 코드는 Java보다 함수형 스타일이 더 자주 나옵니다.

예:

```ts
pages.flatMap((page) => page.items)
array.map(...)
array.filter(Boolean)
```

Java Stream API에 익숙하면 이해가 빠릅니다.

대응 관계:

| TypeScript | Java Stream |
|------|------|
| `map` | `map` |
| `filter` | `filter` |
| `flatMap` | `flatMap` |

예를 들어:

```ts
state.brew.brewSteps.map((step) => step.trim()).filter(Boolean)
```

은 Java로 보면:

```java
steps.stream()
    .map(String::trim)
    .filter(s -> !s.isBlank())
```

와 비슷합니다.

---

## 19. 타입 단언(`as ...`)은 "컴파일러에게 알려주기"

이 프로젝트에서 종종 보이는 문법:

```ts
event.target.value as LogFormState['brew']['brewMethod']
```

의미:
- 개발자가 "이 값은 이 타입으로 봐도 된다"고 컴파일러에게 알려주는 것

Java의 강제 캐스팅과 비슷하지만, 런타임 변환이 아니라 **타입 검사 보조**에 가깝습니다.

주의:
- 남용하면 타입 안전성이 약해집니다.
- 가능하면 실제 타입 흐름으로 해결하고, 필요한 경우에만 쓰는 것이 좋습니다.

---

## 20. TypeScript에서 `null`과 `undefined`

Java 개발자가 자주 헷갈리는 부분입니다.

- `null`: 명시적으로 값이 없음
- `undefined`: 아직 할당되지 않았거나 존재하지 않음

이 프로젝트는 둘 다 다룹니다.

예:

```ts
next_cursor: string | null
```

또는:

```ts
getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined
```

즉:
- API 응답에서는 `null`
- TanStack Query에 넘길 때는 `undefined`

처럼 라이브러리 계약에 맞춰 구분합니다.

---

## 21. `enabled`로 "훅은 항상 호출하되, 요청만 조건부로" 만들기

Phase 3 자동완성 훅에서 가장 먼저 봐야 할 포인트는 이것입니다.

```ts
function useSuggestions(type: 'tags' | 'companions', q: string) {
  return useQuery({
    queryKey: ['suggestions', type, q],
    queryFn: () => getSuggestions(type, q),
    staleTime: 30_000,
    enabled: q.length > 0,
  })
}
```

React에서는 hook을 조건문 안에서 호출하면 안 됩니다.

```ts
// 잘못된 예시
if (q.length > 0) {
  useQuery(...)
}
```

대신 **hook 호출은 항상 유지하고, 내부 실행만 `enabled`로 제어**합니다.

이 사고방식은 Java/Spring에는 바로 대응되는 문법이 없어서 초기에 낯설 수 있습니다.
핵심은 "호출 위치의 안정성"이 React 규칙이고, "실행 여부"는 TanStack Query 옵션으로 푼다는 점입니다.

---

## 22. `queryKey`는 함수 인자가 아니라 캐시 식별자다

자동완성은 검색어가 한 글자만 달라져도 다른 결과를 돌려줘야 합니다.

```ts
queryKey: ['suggestions', type, q]
```

이 배열은 단순한 옵션이 아니라, **이 요청 결과를 어떤 이름으로 캐시에 저장할지 정하는 키**입니다.

예:

- `['suggestions', 'tags', '초']`
- `['suggestions', 'tags', '초콜']`
- `['suggestions', 'companions', '지']`

이 셋은 모두 다른 캐시 엔트리입니다.

Java/Spring으로 치면 메서드 인자 기반 캐시 키 생성과 비슷합니다.
다만 TanStack Query는 `@Cacheable` 애노테이션 대신, 프론트 코드에서 이 키를 직접 선언합니다.

Phase 3 코드를 읽을 때는 `queryFn`만 보지 말고:

- 어떤 값이 캐시 키에 들어가는가
- 어떤 값이 바뀌면 새 요청이 발생하는가
- 어떤 값은 같아서 캐시를 재사용하는가

를 함께 봐야 합니다.

---

## 23. controlled component와 state ownership

`TagInput`은 내부에서 태그 배열을 최종 소유하지 않습니다.

```ts
interface TagInputProps {
  value: string[]
  onChange: (tags: string[]) => void
  suggestions?: string[]
  onQueryChange?: (q: string) => void
}
```

즉:

- 실제 태그 목록은 부모(`LogFormPage`)가 가진다
- `TagInput`은 현재 입력 중인 문자열과 드롭다운 열림 상태만 가진다

이걸 controlled component라고 합니다.

Java/Spring MVC 식으로 비유하면:

- `TagInput`은 입력 UI 조각
- 부모 페이지는 form backing object를 들고 있는 controller 역할

과 비슷합니다.

이 방식의 장점:

- 폼 제출 시 부모 상태만 읽으면 된다
- 태그 목록과 API payload 구조를 맞추기 쉽다
- 재사용 컴포넌트가 특정 API 훅에 덜 묶인다

Phase 3에서 `companionsText: string`이 `companions: string[]`로 바뀐 것도 같은 방향입니다.
UI 내부 표현과 실제 데이터 표현을 가깝게 붙이면 변환 코드가 줄어듭니다.

---

## 24. 브라우저 이벤트 순서: `blur`가 `click`보다 먼저 올 수 있다

자동완성 드롭다운을 처음 만들 때 자주 겪는 문제입니다.

```ts
function handleOptionMouseDown(e: React.MouseEvent, tag: string) {
  e.preventDefault()
  addTag(tag)
  inputRef.current?.focus()
}
```

왜 `onClick`이 아니라 `onMouseDown`일까요?

드롭다운 항목을 클릭할 때는 보통 이런 순서가 됩니다.

1. `mousedown`
2. input의 `blur`
3. `mouseup`
4. `click`

만약 `onBlur`에서 드롭다운을 닫아버리면, `click` 시점에는 항목이 이미 사라졌을 수 있습니다.
그래서 Phase 3 구현은 `mousedown`에서 먼저 잡고 `preventDefault()`로 blur를 막습니다.

이건 TypeScript 문법이라기보다 브라우저 이벤트 모델을 React 코드 안에서 다루는 패턴입니다.
하지만 실제 프론트 구현에서는 이런 지점이 문법보다 더 중요할 때가 많습니다.

---

## 25. `useId`와 ARIA 연결

Phase 3 `TagInput`은 단순히 보이기만 하는 드롭다운이 아니라, input과 suggestion list를 접근성 속성으로 연결합니다.

```ts
const listboxId = useId()

<input
  aria-autocomplete="list"
  aria-controls={open ? listboxId : undefined}
/>

<ul id={listboxId} role="listbox">
  <li role="option" aria-selected={false}>...</li>
</ul>
```

`useId()`는 컴포넌트 인스턴스마다 안정적인 고유 ID를 만들어줍니다.

이 패턴의 의미:

- DOM이 가까이 있다고 해서 스크린 리더가 관계를 자동 이해하지는 않는다
- `aria-controls`, `role="listbox"`, `role="option"`으로 의미를 연결해줘야 한다
- 접근성 속성이 좋아질수록 테스트도 `getByRole` 기반으로 더 안정적이 된다

즉, 접근성은 "나중에 붙이는 부가 기능"이 아니라 컴포넌트 설계 일부입니다.

---

## 26. 이 프로젝트에서 꼭 익혀야 할 TypeScript 우선순위

Phase 1-3 프론트를 리뷰하려면 아래 순서가 가장 효율적입니다.

1. `interface`, `type`
2. union type
3. discriminated union
4. generic
5. `async/await`
6. `?.`, `??`
7. `as const`
8. 불변 상태 업데이트 (`...spread`)
9. React hook (`useState`, `useEffect`, `useMemo`, `useRef`)
10. `useSearchParams`, `URLSearchParams`
11. `useInfiniteQuery`
12. 브라우저 API (`IntersectionObserver`, `Date`)
13. `Promise<T>`
14. `useQuery`의 `enabled`, `queryKey`
15. controlled component + 상태 끌어올리기
16. 브라우저 이벤트 순서 (`mousedown`, `blur`, `click`)
17. `useId`와 ARIA 속성

이 정도면 Phase 3 자동완성 UI까지 대부분 읽을 수 있습니다.

---

## 27. 실제 코드 읽기 추천 순서

처음부터 큰 페이지 컴포넌트를 읽기보다 아래 순서가 좋습니다.

1. `frontend/src/types/log.ts`
   핵심 타입 구조 파악
2. `frontend/src/api/client.ts`
   API 호출 기본 방식 이해
3. `frontend/src/api/logs.ts`
   CRUD 함수 파악
4. `frontend/src/hooks/useLogs.ts`
   서버 상태 관리 방식 이해
5. `frontend/src/pages/logFormState.ts`
   폼 상태 변환 이해
6. `frontend/src/components/FilterBar.tsx`
   제어 컴포넌트와 필터 props 구조 이해
7. `frontend/src/pages/HomePage.tsx`
   URL 상태, 무한 스크롤, 상태 분기 이해
8. `frontend/src/api/suggestions.ts`
   query string 조립과 생성 타입 사용 방식 확인
9. `frontend/src/hooks/useSuggestions.ts`
   조건부 쿼리와 캐시 키 설계 확인
10. `frontend/src/components/TagInput.tsx`
   controlled component, 이벤트 처리, 접근성 확인
11. `frontend/src/pages/LogDetailPage.tsx`
12. `frontend/src/pages/LogFormPage.tsx`
13. `frontend/e2e/log-happy-path.spec.ts`
   실제 사용자 흐름 검증 방식 이해

이 순서가 Spring 개발자에게 가장 읽기 쉽습니다.  
DTO → Client → Service-like hook → Controller-like page 순서로 내려오기 때문입니다.

---

## 28. 마지막 정리

이 프로젝트의 TypeScript 코드는 "클래스 기반 프론트엔드"가 아니라 **함수 + 타입 중심 프론트엔드**입니다.  
처음에는 Java보다 덜 구조적처럼 보일 수 있지만, 실제로는 타입과 함수가 역할을 나눠 갖고 있습니다.

특히 꼭 이해해야 할 핵심은 세 가지입니다.

- union type
- discriminated union
- React hook 기반 상태 관리

Phase 2부터는 여기에 다음이 추가됩니다.

- URL을 상태 저장소로 쓰는 방식
- cursor 기반 무한 스크롤
- 브라우저 API와 React hook의 결합
- 날짜/시간대 처리의 함정

Phase 3부터는 여기에 다음이 더 붙습니다.

- hook 호출 규칙을 지키면서 요청만 조건부로 실행하는 방식
- `queryKey`로 캐시 경계를 직접 설계하는 방식
- 상태를 부모가 소유하는 controlled component 설계
- `blur`/`mousedown` 같은 브라우저 이벤트 순서를 고려한 UI 처리
- `useId`와 ARIA를 통한 접근성 연결

이 흐름까지 잡히면 Phase 1-3 프론트 코드는 훨씬 읽기 쉬워집니다.

---

이 문서는 Codex가 작성했습니다.
