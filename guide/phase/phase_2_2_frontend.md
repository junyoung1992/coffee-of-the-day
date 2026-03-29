# Phase 2-2 Frontend — 목록 필터 (URL 쿼리 파라미터 기반)

## 무엇을 만들었나

홈 화면에 두 가지 필터를 추가했습니다.

1. **로그 타입 탭** — 전체 / 카페 / 브루 버튼 그룹
2. **날짜 범위** — `date_from`, `date_to` 날짜 입력 필드

핵심 설계 원칙: **필터 상태를 URL 쿼리 파라미터에 저장한다**.

결과적으로 `?log_type=cafe&date_from=2026-03-01` 같은 URL을 북마크하거나 공유하면 동일한 필터 상태가 복원됩니다.

---

## 왜 URL을 상태 저장소로 사용하는가

### Java/Spring 비유

Spring MVC에서 `@RequestParam`으로 쿼리 파라미터를 받아 뷰를 렌더링하는 것과 같습니다. 서버가 상태를 URL에서 읽듯, 클라이언트도 URL에서 필터 상태를 읽습니다.

### React에서 `useState` vs URL 파라미터

```tsx
// ❌ useState 방식 — 새로고침/공유 시 상태 사라짐
const [logType, setLogType] = useState<LogType | undefined>()

// ✅ URL 방식 — 새로고침/뒤로가기/공유 모두 동작
const [searchParams, setSearchParams] = useSearchParams()
const logType = parseLogType(searchParams.get('log_type'))
```

`useSearchParams`는 `useState`와 완전히 같은 인터페이스(`[값, setter]`)를 가지지만, 내부적으로 브라우저 URL을 읽고 씁니다.

---

## 파일 구조

```
frontend/src/
├── components/
│   └── FilterBar.tsx          ← 새로 추가: 필터 UI 컴포넌트
├── pages/
│   └── HomePage.tsx           ← 수정: URL 파라미터 연동
```

---

## FilterBar 컴포넌트 설계

### 비제어(Uncontrolled) vs 제어(Controlled) 선택

`FilterBar`는 **제어 컴포넌트(Controlled Component)**로 설계했습니다. 상태를 직접 보유하지 않고, 부모(`HomePage`)로부터 현재 값과 변경 핸들러를 props로 받습니다.

```tsx
interface FilterBarProps {
  logType: LogType | undefined
  dateFrom: string
  dateTo: string
  onLogTypeChange: (value: LogType | undefined) => void
  onDateFromChange: (value: string) => void
  onDateToChange: (value: string) => void
}
```

이 패턴은 Spring의 바인딩 폼과 유사합니다. 뷰는 현재 값을 표시하고, 이벤트를 상위로 올려 상태 변경을 위임합니다.

---

## URL 파라미터 업데이트 방법

### 함수형 업데이트로 기존 파라미터 보존

```tsx
const handleLogTypeChange = useCallback(
  (value: LogType | undefined) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)  // 기존 파라미터 복사
      if (value) {
        next.set('log_type', value)
      } else {
        next.delete('log_type')  // 전체 선택 시 파라미터 제거 → 깔끔한 URL
      }
      return next
    })
  },
  [setSearchParams],
)
```

`prev`를 복사한 뒤 특정 키만 변경하면, 나머지 파라미터(예: `date_from`)는 유지됩니다. 이 패턴은 React의 `setState(prev => ({ ...prev, key: value }))` 와 동일한 발상입니다.

---

## TanStack Query 캐시 키 자동 갱신

기존 `useLogList` 훅은 파라미터 전체를 쿼리 키에 포함합니다.

```tsx
// useLogs.ts
const LOG_KEYS = {
  list: (params: ListLogsParams) => ['logs', 'list', params] as const,
}

export function useLogList(params: Omit<ListLogsParams, 'cursor'> = {}) {
  return useInfiniteQuery({
    queryKey: LOG_KEYS.list(params),  // params가 바뀌면 키가 바뀐다
    ...
  })
}
```

`HomePage`에서 URL 파라미터를 읽어 `useLogList`에 전달하면:

```tsx
useLogList({
  limit: 12,
  log_type: logType,          // URL에서 읽은 값
  date_from: dateFrom || undefined,
  date_to: dateTo || undefined,
})
```

필터가 변경되면 → URL 업데이트 → 컴포넌트 리렌더 → `params` 객체 변경 → 쿼리 키 변경 → TanStack Query가 자동으로 새 API 요청 실행.

**별도 `invalidateQueries` 없이 자동으로 동작합니다.**

---

## URL 파라미터 파싱 시 타입 안전성

백엔드 API가 `log_type`으로 `'cafe' | 'brew'`만 허용하므로, URL에서 읽은 문자열을 검증합니다.

```tsx
function parseLogType(value: string | null): LogType | undefined {
  if (value === 'cafe' || value === 'brew') return value
  return undefined  // 잘못된 값은 '전체'로 fallback
}
```

URL은 사용자가 직접 수정할 수 있으므로, URL에서 오는 값은 항상 검증이 필요합니다. Spring의 `@RequestParam`에 `Enum` 타입을 지정하고 `ConversionService`가 변환하는 것과 같은 역할입니다.

---

## 테스트 전략

`FilterBar`는 순수 UI 컴포넌트이므로 단위 테스트로 충분합니다.

- 탭 렌더링 확인
- 현재 활성 탭 스타일 확인 (`bg-white` 클래스 존재 여부)
- 클릭 이벤트 시 핸들러 호출 확인 (`vi.fn()` + `fireEvent.click`)
- 날짜 입력 변경 시 핸들러 호출 확인 (`fireEvent.change`)

`HomePage` 자체의 통합 테스트는 `useSearchParams`를 포함한 Router 컨텍스트가 필요하고, TanStack Query까지 모킹해야 하므로 범위 밖으로 두었습니다.
