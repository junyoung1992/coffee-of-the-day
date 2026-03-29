# Phase 3 리팩토링 학습 문서

> Phase 3 자동완성 구현에 대한 코드 리뷰 결과를 반영한 리팩토링 내용을 정리합니다.
> 주요 주제: prefix 검색 전환, IME 조합 입력 처리, debounce 적용, 테스트 공백 보완.

---

## 1. 리팩토링 배경

Phase 3 구현 완료 후 코드 리뷰에서 세 가지 핵심 문제가 발견됐다.

| 문제 | 증상 |
|---|---|
| contains 검색 (`LIKE '%q%'`) | 계획과 달리 prefix가 아닌 부분 일치 구현됨 |
| IME 조합 중 `Enter` 오동작 | 한국어 입력 중 미완성 글자가 태그로 추가됨 |
| debounce 없음 | 키 입력마다 API 요청 발생 |

추가로 `SuggestionHandler`와 `SQLiteSuggestionRepository`에 대한 테스트가 없었다.

---

## 2. prefix 검색으로 전환

### 변경 내용

```sql
-- 변경 전: 부분 일치 (contains)
WHERE ? = '' OR LOWER(tag) LIKE '%' || LOWER(?) || '%'

-- 변경 후: 전방 일치 (prefix)
WHERE ? = '' OR LOWER(tag) LIKE LOWER(?) || '%'
```

### 왜 prefix가 더 적합한가

커피 태그("초콜릿", "체리", "플로럴")나 이름("지수", "민준") 같은 자동완성 데이터는 대부분 앞 글자부터 입력하는 패턴이다. "초"를 입력했을 때 "다크초콜릿"이 나오는 것보다 "초콜릿"이 나오는 게 자연스럽다.

성능 측면에서도 prefix는 인덱스 활용이 가능하다.

```
LIKE 'q%'   → B-tree 인덱스를 탈 수 있다 (범위 스캔)
LIKE '%q%'  → 인덱스를 무시하고 전체 행을 스캔한다
```

현재는 `json_each()`로 JSON 배열을 실시간 펼치는 구조라 인덱스 효과가 제한적이지만, 나중에 정규화 테이블로 마이그레이션할 때 prefix 방식이 훨씬 최적화하기 쉽다.

**Spring에서의 대응**: JPA에서 `LIKE :q%`와 `LIKE %:q%`의 차이와 같다. `LIKE :q%`는 인덱스를 타지만 `LIKE %:q%`는 full table scan이다.

---

## 3. IME 조합 중 Enter 오동작 방지

### 문제

한국어, 중국어, 일본어 등의 입력은 IME(Input Method Editor)를 거친다. IME는 자음/모음을 조합해 완성된 글자를 만드는 과정을 거친다.

```
"ㅊ" 입력 → "초" 입력 → "촉" 입력 → "초코" 입력 → Enter (조합 확정)
```

이 조합 과정에서도 `keydown` 이벤트가 발생한다. 기존 코드는 조합 중인지 여부를 확인하지 않았기 때문에, 조합을 확정하려고 누른 `Enter`가 태그 추가 동작을 함께 실행했다.

```
유저 의도: "초콜릿" 완성 → Enter로 조합 확정
실제 동작: 조합 확정 + 미완성 "초콜릿" 태그 추가
```

### 해결

브라우저는 IME 조합 중임을 `KeyboardEvent.isComposing` 플래그로 알려준다.

```typescript
// 변경 전
function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
  if (e.key === 'Enter' || e.key === ',') {
    e.preventDefault()
    addTag(inputValue)
  }
}

// 변경 후
function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
  if (e.key === 'Enter' || e.key === ',') {
    // IME 조합 중(한국어, 중국어 등)에는 Enter/쉼표를 무시한다.
    // isComposing이 true인 동안은 문자가 아직 확정되지 않은 상태다.
    if (e.nativeEvent.isComposing) return
    e.preventDefault()
    addTag(inputValue)
  }
}
```

React의 합성 이벤트(`SyntheticEvent`)에서는 `e.nativeEvent`를 통해 브라우저 네이티브 이벤트에 접근한다. `isComposing`은 네이티브 `KeyboardEvent`의 속성이다.

**Spring에서의 대응**: 직접적인 대응 개념은 없다. 이는 브라우저 입력 처리에 특화된 문제다. 다만 "같은 이벤트인데 컨텍스트에 따라 다르게 처리한다"는 점에서, AOP의 `@Around`로 특정 조건일 때만 실제 로직을 실행하도록 제어하는 것과 개념이 유사하다.

---

## 4. debounce 적용

### 문제

기존 구현에서는 키 입력마다 즉시 API를 요청했다. "초콜릿"을 입력하면 "초", "초콜", "초콜릿" 각각에 대해 요청이 발생한다.

```
"초"   입력 → GET /suggestions/tags?q=초   (요청 1)
"초콜" 입력 → GET /suggestions/tags?q=초콜 (요청 2)
"초콜릿" 입력 → GET /suggestions/tags?q=초콜릿 (요청 3)
```

요청 1, 2의 응답은 대부분 무의미하다. 유저가 계속 타이핑 중이기 때문이다.

### debounce란

입력이 멈춘 뒤 일정 시간(여기서는 200ms)이 지나야 값을 반영하는 기법이다.

```
"초"   입력 (0ms)   → 타이머 시작
"초콜" 입력 (50ms)  → 타이머 리셋
"초콜릿" 입력 (100ms) → 타이머 리셋
200ms 경과           → 최종값 "초콜릿"으로 API 요청 1건
```

### useDebounce 훅 구현

```typescript
// src/hooks/useDebounce.ts
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedValue(value), delay)
    return () => clearTimeout(timer)  // 값이 바뀌면 이전 타이머를 취소한다
  }, [value, delay])

  return debouncedValue
}
```

`useEffect`의 cleanup 함수(`return () => clearTimeout(timer)`)가 핵심이다. `value`가 바뀔 때마다 이전 타이머를 취소하고 새 타이머를 시작한다. 결과적으로 마지막 변경 후 `delay`ms가 지나야 `debouncedValue`가 업데이트된다.

**Spring에서의 대응**: Spring에서는 `@Scheduled`나 메시지 큐의 배치 처리와 개념이 비슷하다. 들어오는 이벤트를 즉시 처리하지 않고, 일정 시간 모은 뒤 최신 값만 처리하는 방식이다.

### useSuggestions에 적용

```typescript
// 변경 전
function useSuggestions(type: 'tags' | 'companions', q: string) {
  return useQuery({
    queryKey: ['suggestions', type, q],
    queryFn: () => getSuggestions(type, q),
    enabled: q.length > 0,
  })
}

// 변경 후
function useSuggestions(type: 'tags' | 'companions', q: string) {
  const debouncedQ = useDebounce(q, 200)  // 200ms debounce 적용

  return useQuery({
    queryKey: ['suggestions', type, debouncedQ],  // debounce된 값으로 캐시 키 구성
    queryFn: () => getSuggestions(type, debouncedQ),
    enabled: debouncedQ.length > 0,  // debounce된 값 기준으로 enabled 판단
  })
}
```

`queryKey`와 `enabled` 모두 `debouncedQ`를 기준으로 사용해야 한다. `q`를 쓰면 타이핑할 때마다 캐시 키가 바뀌어 debounce 효과가 사라진다.

---

## 5. 테스트 보완

### 추가한 테스트 파일

| 파일 | 레이어 | 주요 검증 |
|---|---|---|
| `suggestion_handler_test.go` | HTTP 핸들러 | 응답 코드, JSON 직렬화, 오류 매핑 |
| `suggestion_repository_test.go` | DB 레이어 | prefix 매칭, 대소문자 무시, 빈도순, 유저 격리 |
| `TagInput.test.tsx` | UI 컴포넌트 | Enter/쉼표/IME/Backspace/드롭다운 선택 |
| `useSuggestions.test.tsx` | React 훅 | enabled 조건, API 호출 인수 |
| `useDebounce.test.ts` | 유틸 훅 | 타이밍, 연속 입력 시 이전 타이머 취소 |

### 핸들러 테스트 패턴

핸들러 테스트는 실제 서비스 구현 대신 스텁을 주입한다. HTTP 레이어 자체의 동작(파라미터 파싱, JSON 직렬화, 상태 코드 매핑)만 검증한다.

```go
type stubSuggestionService struct {
    tagsFunc       func(ctx context.Context, userID, q string) ([]string, error)
    companionsFunc func(ctx context.Context, userID, q string) ([]string, error)
}
```

스텁의 함수 필드를 테스트마다 다르게 설정해 필요한 시나리오를 만든다. 정상 응답, 빈 결과, 유효성 검사 오류, 서버 오류 등을 각각 독립적으로 테스트할 수 있다.

**Spring에서의 대응**: `@WebMvcTest`와 `@MockBean`으로 서비스를 모킹하고 컨트롤러만 테스트하는 패턴과 같다. 실제 서비스 로직은 서비스 레이어 테스트에서 따로 검증한다.

### 리포지토리 통합 테스트

리포지토리 테스트는 인메모리 SQLite를 사용해 실제 쿼리를 실행한다.

```go
func setupTestDB(t *testing.T) *sql.DB {
    db, _ := sql.Open("sqlite3", "file::memory:?cache=shared")
    // 마이그레이션 파일을 순서대로 실행해 스키마를 구성한다
    ...
}
```

prefix 검색의 핵심 검증:

```go
func TestGetTagSuggestions_PrefixMatch(t *testing.T) {
    // ["초콜릿", "체리", "다크초콜릿"] 삽입
    ...
    got, _ := repo.GetTagSuggestions(ctx, testUserID, "초")

    // prefix "초"는 "초콜릿"만 매칭해야 한다. "다크초콜릿"은 포함되면 안 된다.
    assert.Equal(t, []string{"초콜릿"}, got)
}
```

이 테스트가 없었다면 contains에서 prefix로 SQL을 바꾼 뒤 실수로 되돌려도 감지하지 못한다.

**Spring에서의 대응**: `@DataJpaTest`나 H2 인메모리 DB를 사용한 리포지토리 테스트와 같다.

### 프론트 IME 테스트

`fireEvent.keyDown`에 `isComposing: true`를 전달하면 브라우저 `KeyboardEventInit`에 해당 속성이 설정된다.

```typescript
// isComposing을 KeyboardEventInit에 직접 전달해야 nativeEvent.isComposing이 true로 설정된다.
fireEvent.keyDown(input, { key: 'Enter', isComposing: true })
expect(onChange).not.toHaveBeenCalled()
```

`{ nativeEvent: { isComposing: true } }`처럼 중첩해서 전달하면 작동하지 않는다. `fireEvent`는 `KeyboardEvent` 생성자에 직접 전달할 수 있는 `KeyboardEventInit` 속성을 기대한다.

### debounce 테스트에서 fake timer 사용

`useDebounce`는 `setTimeout`에 의존하므로, 실제 시간을 기다리지 않고 테스트하려면 fake timer를 사용한다.

```typescript
beforeEach(() => { vi.useFakeTimers() })
afterEach(() => { vi.useRealTimers() })

it('delay가 지난 후 새 값으로 업데이트된다', () => {
  const { result, rerender } = renderHook(({ value }) => useDebounce(value, 200), {
    initialProps: { value: '초' },
  })
  rerender({ value: '초콜릿' })

  act(() => { vi.advanceTimersByTime(200) })  // 200ms를 즉시 건너뜀

  expect(result.current).toBe('초콜릿')
})
```

`vi.advanceTimersByTime(200)`은 실제 200ms를 기다리는 게 아니라 타이머를 200ms만큼 앞당긴다. 테스트가 느려지지 않는다.

### fake timer + waitFor 데드락 문제

`useSuggestions` 테스트에서 fake timer와 TanStack Query의 `waitFor`를 함께 쓰면 데드락이 발생한다.

```
waitFor → 내부에서 setInterval로 폴링 시작
fake timer → setInterval 정지
waitFor → 영원히 대기
```

해결 방법은 관심사를 나눠 테스트하는 것이다.

- `useDebounce.test.ts` — debounce 타이밍 검증 (fake timer 사용)
- `useSuggestions.test.tsx` — `enabled` 조건과 API 호출 인수 검증 (fake timer 미사용)

`useDebounce`는 초기값을 `useState(value)`로 즉시 설정하므로, `useSuggestions`를 처음 렌더링할 때는 debounce 없이 초기 `q`가 바로 사용된다. 이 특성을 활용해 `useSuggestions` 테스트에서는 fake timer 없이 비동기 동작을 검증할 수 있다.

---

## 6. 리팩토링 요약

| 변경 | 파일 | 효과 |
|---|---|---|
| `LIKE '%q%'` → `LIKE 'q%'` | `suggestion_repository.go` | 계획 일치, 인덱스 활용 가능 구조 |
| `isComposing` 체크 추가 | `TagInput.tsx` | 한국어 입력 중 오동작 방지 |
| `useDebounce` 적용 | `useSuggestions.ts` | 키 입력마다 API 요청 방지 |
| `useDebounce` 훅 신설 | `useDebounce.ts` | 재사용 가능한 debounce 추상화 |
| 핸들러/리포지토리/컴포넌트/훅 테스트 추가 | 각 레이어 테스트 파일 | 회귀 방지, 경계 조건 문서화 |
