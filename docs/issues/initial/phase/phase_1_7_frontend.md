# Phase 1-7 Frontend — 타입 및 API 클라이언트

> 대상 독자: Java/Spring 경험이 있는 개발자. TypeScript 타입 시스템과 TanStack Query를 Spring과 비교하여 설명합니다.

---

## 무엇을 만들었나

백엔드 API를 호출하기 위한 프론트엔드 계층을 구성했습니다.

| 파일 | 역할 |
|------|------|
| `src/types/schema.ts` | **자동 생성** — `openapi.yml`에서 `npm run generate`로 생성. 직접 편집 금지 |
| `src/types/log.ts` | schema.ts alias + TypeScript 전용 Discriminated Union |
| `src/types/common.ts` | `CursorPage<T>` 제네릭 타입 |
| `src/api/client.ts` | 오류 파싱 수정 (백엔드 에러 포맷 대응) |
| `src/api/logs.ts` | 5개 API 함수 (`getLogs`, `getLog`, `createLog`, `updateLog`, `deleteLog`) |
| `src/hooks/useLogs.ts` | TanStack Query 훅 5개 |
| `src/api/logs.test.ts` | API 함수 단위 테스트 |
| `src/hooks/useLogs.test.tsx` | 훅 단위 테스트 |

---

## openapi-typescript — 타입의 단일 진실 공급원

프론트엔드 타입을 백엔드 소스코드를 보고 직접 쓰면 두 가지 문제가 생깁니다:

1. **동기화 문제**: 백엔드 API가 바뀌어도 프론트엔드 타입은 자동으로 틀어짐을 알 수 없다.
2. **기준 불명확**: 백엔드 소스와 openapi.yml 중 어느 쪽을 봐야 하는지 모호해진다.

`openapi-typescript`는 이 문제를 해결합니다. `openapi.yml`을 파싱해 TypeScript 타입을 자동으로 생성하므로, **openapi.yml이 유일한 기준**이 됩니다.

```
openapi.yml (백엔드 구현 후 작성)
    ↓ npm run generate
src/types/schema.ts  ← 자동 생성, 절대 직접 편집 금지
    ↓ import + alias
src/types/log.ts     ← 프로젝트 전용 타입 (alias + Discriminated Union)
    ↓ import
src/api/logs.ts, src/hooks/useLogs.ts
```

생성된 `schema.ts`는 `components['schemas']['CoffeeLogResponse']`처럼 OpenAPI 스키마 경로를 그대로 TypeScript 인터페이스로 표현합니다. `log.ts`에서 짧은 이름으로 alias해 사용합니다:

```typescript
// src/types/log.ts
import type { components } from './schema'

export type LogType = components['schemas']['LogType']           // 'cafe' | 'brew'
export type CoffeeLogResponse = components['schemas']['CoffeeLogResponse']
export type CreateLogInput = components['schemas']['CreateLogRequest']
```

### openapi.yml 정밀도가 중요한 이유

`required` 배열을 빠뜨리면 모든 필드가 optional(`?`)로 생성됩니다. Phase 1-7 작업 중 다음을 수정했습니다:

| 문제 | 수정 |
|------|------|
| `CoffeeLogResponse` 전 필드 optional | `required: [id, user_id, ...]` 추가 |
| `$ref` + `nullable: true` 조합이 `null` 미포함 | `roast_level`을 인라인 enum + `nullable: true`로 교체 |
| `ListLogsResponse.items` optional | `required: [items, has_next]` 추가 |

이 수정 후 `roast_level`의 생성 타입이 `RoastLevel`(null 불가)에서 `"light" \| "medium" \| "dark" \| null`로 정확해졌습니다.

---

## TypeScript Discriminated Union — Spring과 비교

Go 백엔드에서 `CoffeeLogFull`은 다음 구조를 가집니다:
```go
type CoffeeLogFull struct {
    CoffeeLog
    Cafe *CafeDetail
    Brew *BrewDetail
}
```

TypeScript에서 이를 그대로 `cafe?: CafeDetail | null, brew?: BrewDetail | null`로 표현하면 컴파일러는 `log.cafe`에 접근할 때 항상 null 체크를 요구합니다. **Discriminated Union**을 쓰면 더 안전합니다:

```typescript
// log_type 값으로 cafe/brew 필드의 존재를 컴파일러가 보장한다
export type CoffeeLogFull = CafeLogFull | BrewLogFull

if (log.log_type === 'cafe') {
  log.cafe.cafe_name  // ✅ 컴파일러가 log.cafe를 non-null로 앎
  log.brew            // ✅ 컴파일러가 brew 필드가 없음을 앎
}
```

Java에서 `instanceof` 검사 후 캐스팅하는 것과 유사하지만, TypeScript는 `if` 블록 안에서 자동으로 타입을 좁혀줍니다. Spring의 `@JsonSubTypes` + `@JsonTypeInfo` 어노테이션이 런타임 역직렬화를 처리하는 것처럼, TypeScript는 이를 컴파일 타임에 처리합니다.

OpenAPI 3.0은 이 패턴을 스키마로 표현하기 어렵습니다(`discriminator` 지원이 제한적). 따라서 `CoffeeLogFull` discriminated union은 생성된 `CoffeeLogResponse`를 기반으로 TypeScript 레벨에서만 강화합니다:

```typescript
// schema.ts (자동 생성) — cafe/brew 모두 optional
CoffeeLogResponse: { log_type?: 'cafe' | 'brew', cafe?: CafeDetail, brew?: BrewDetail, ... }

// log.ts (수동) — TypeScript로 의미를 강화
export type CoffeeLogFull =
  | (Omit<CoffeeLogResponse, 'log_type' | 'cafe' | 'brew'> & { log_type: 'cafe'; cafe: CafeDetail })
  | (Omit<CoffeeLogResponse, 'log_type' | 'cafe' | 'brew'> & { log_type: 'brew'; brew: BrewDetail })
```

---

## 계층 구조 — Spring MVC 비교

```
Spring:                       React (이 프로젝트):
────────────────────          ────────────────────────────
@Controller                   pages/ (컴포넌트)
    ↓ 호출                        ↓ 훅 호출
@Service                      hooks/useLogs.ts (TanStack Query)
    ↓ 호출                        ↓ API 함수 호출
Repository/DAO                api/logs.ts (fetch 함수)
    ↓ 호출                        ↓ 공통 클라이언트
DataSource                    api/client.ts (fetch + 헤더)
```

`hooks/useLogs.ts`는 Spring의 `@Service`와 비슷한 역할을 합니다. 캐싱, 로딩 상태, 오류 처리를 추상화합니다.

---

## TanStack Query — 서버 상태 관리

Spring에서는 서비스를 호출하면 즉시 결과를 받습니다. React에서는 비동기 데이터를 UI에 연결하는 방식이 다릅니다. TanStack Query가 이를 담당합니다:

```typescript
// 컴포넌트 없이 훅으로 데이터 조회
const { data, isLoading, error } = useLog('log-1')
```

내부적으로 TanStack Query는:
1. 같은 쿼리 키면 캐시를 반환합니다 (Spring의 `@Cacheable`과 유사)
2. 컴포넌트 마운트 시 자동으로 최신 데이터를 재조회합니다
3. 네트워크 재연결 시 자동 갱신합니다

### 쿼리 키

```typescript
const LOG_KEYS = {
  all: ['logs'],
  list: (params) => ['logs', 'list', params],
  detail: (id) => ['logs', 'detail', id],
}
```

쿼리 키는 캐시의 식별자입니다. `invalidateQueries({ queryKey: LOG_KEYS.all })`을 호출하면 `['logs']`로 시작하는 모든 캐시를 무효화합니다. Spring의 `@CacheEvict(value="logs", allEntries=true)`와 같습니다.

---

## useInfiniteQuery — 무한 스크롤 페이지네이션

`useQuery`는 단일 페이지를, `useInfiniteQuery`는 여러 페이지를 누적 관리합니다:

```typescript
export function useLogList(params = {}) {
  return useInfiniteQuery({
    queryKey: LOG_KEYS.list(params),
    queryFn: ({ pageParam }) => getLogs({ ...params, cursor: pageParam }),
    getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
    initialPageParam: undefined as string | undefined,  // TanStack Query v5 필수
  })
}
```

- `initialPageParam`: 첫 페이지 요청 시 `cursor`에 들어갈 값 (`undefined` = 커서 없음)
- `getNextPageParam`: 마지막 페이지의 응답에서 다음 페이지 커서를 추출. `null`이면 `undefined`를 반환해야 `hasNextPage`가 `false`가 됨
- `data.pages`: 각 페이지 응답이 배열로 쌓임. 렌더링 시 `pages.flatMap(p => p.items)`로 펼쳐서 사용

### TanStack Query v4 → v5 변경 사항

v5에서 `initialPageParam`이 **필수**가 됐습니다. v4에서는 암묵적으로 `undefined`였지만, v5는 명시를 요구합니다. 이 프로젝트는 v5를 사용하므로 반드시 선언해야 합니다.

---

## 캐시 무효화 전략

CRUD 작업 후 캐시를 언제 무효화할지가 중요합니다:

| 훅 | 무효화 범위 | 이유 |
|-----|------------|------|
| `useCreateLog` | `LOG_KEYS.all` | 새 기록이 생겼으니 목록 전체를 새로 조회 |
| `useUpdateLog` | detail + all | 상세와 목록 모두 변경됨 |
| `useDeleteLog` | `LOG_KEYS.all` | 삭제된 항목이 목록에서 사라져야 함 |

`LOG_KEYS.all = ['logs']`를 최상위 키로 두면 `invalidateQueries({ queryKey: ['logs'] })`로 모든 로그 관련 캐시를 한 번에 무효화할 수 있습니다. 이 패턴은 계층적 쿼리 키의 핵심입니다.

---

## 테스트 전략

### API 함수 테스트 (`logs.test.ts`)
`request` 함수를 `vi.mock`으로 교체하고, API 함수가 올바른 경로/메서드/본문으로 호출하는지 검증합니다:

```typescript
vi.mock('./client', () => ({ request: vi.fn() }))

it('log_type 필터를 쿼리 파라미터로 전달한다', async () => {
  await getLogs({ log_type: 'cafe' })
  expect(mockRequest).toHaveBeenCalledWith('/logs?log_type=cafe')
})
```

### 훅 테스트 (`useLogs.test.tsx`)
`renderHook` + `QueryClientProvider` 래퍼로 훅을 독립적으로 테스트합니다:

```typescript
const { result } = renderHook(() => useLog('log-1'), { wrapper: makeWrapper() })
await waitFor(() => expect(result.current.isSuccess).toBe(true))
```

**주의**: 각 테스트마다 새 `QueryClient`를 생성해야 합니다. 하나의 클라이언트를 공유하면 테스트 간 캐시가 오염됩니다.

---

## 다음 단계

Phase 1-8에서 이 훅들을 실제 컴포넌트(홈 페이지, 상세 페이지, 폼 페이지)에 연결하여 브라우저에서 커피 기록을 생성·조회·수정·삭제할 수 있는 UI를 완성합니다.
