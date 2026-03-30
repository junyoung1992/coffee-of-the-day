# Refactoring — Issue #3 모바일 UX 개선

> `code_review.md`의 지적 사항을 기반으로 한 리팩토링 계획.

---

## 필수 수정

### 1. `getDefaultDateTo()` 타임존 버그 수정

**파일:** `frontend/src/pages/HomePage.tsx`

`toISOString()`은 UTC 기준이므로 KST 00:00~08:59 사이에 어제 날짜를 반환한다.
`getDefaultDateFrom()`은 로컬 시간(`getFullYear()`, `getMonth()`)을 사용하므로 두 함수 간 시간대가 불일치한다.

**수정:** `getDefaultDateTo()`도 로컬 시간 기반으로 변경한다.

```tsx
// 변경 전
function getDefaultDateTo(): string {
  return new Date().toISOString().slice(0, 10)
}

// 변경 후
function getDefaultDateTo(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}
```

두 함수의 날짜 포맷팅 로직이 중복되므로, 공통 헬퍼(`formatLocalDate`)를 추출하여 사용한다.

### 2. 테스트 추가

AGENTS.md 규칙에 따라 기능 변경 시 테스트를 작성해야 한다.

**대상:**
- `getDefaultDateFrom` / `getDefaultDateTo`: 순수 함수. 로컬 날짜 기반 포맷이 올바른지 검증.
  - 시스템 시간을 모킹(`vi.useFakeTimers`)하여 KST 자정 근방 케이스 포함.
- `ScrollToTop` 컴포넌트: `visible` 상태 토글(스크롤 300px 기준)과 버튼 클릭 시 `animateScrollToTop` 호출 검증.

**파일 구조:**
- `frontend/src/pages/__tests__/HomePage.test.ts` — 날짜 헬퍼 테스트 (기존 파일이 있으면 추가, 없으면 생성)
- `frontend/src/components/__tests__/ScrollToTop.test.tsx` — 컴포넌트 테스트

---

## 선택 개선

### 3. ScrollToTop scroll 이벤트 throttling

**파일:** `frontend/src/components/ScrollToTop.tsx`

`scroll` 이벤트 핸들러에 `requestAnimationFrame` 기반 throttle을 적용한다.
React가 동일 값 `setState`를 건너뛰므로 현재 문제는 없지만, 긴 목록 대비 방어 처리.

### 4. ScrollToTop 나타남/사라짐 transition

**파일:** `frontend/src/components/ScrollToTop.tsx`

현재 `if (!visible) return null`로 DOM에서 제거하므로 `transition` 클래스가 동작하지 않는다.
항상 렌더링하되 `opacity` + `pointer-events`로 전환하여 부드러운 나타남/사라짐을 적용한다.

---

## 체크리스트

### 필수
- [x] `getDefaultDateTo()` 로컬 시간 기반으로 수정
- [x] 날짜 포맷팅 공통 헬퍼(`formatLocalDate`) 추출 → `src/utils/date.ts`
- [x] 날짜 헬퍼 단위 테스트 작성 (KST 자정 근방 케이스 포함) → `src/utils/date.test.ts`
- [x] ScrollToTop 단위 테스트 작성 (visible 토글, 버튼 클릭) → `src/components/ScrollToTop.test.tsx`

### 선택
- [x] ScrollToTop scroll 이벤트 `requestAnimationFrame` throttling
- [x] ScrollToTop 나타남/사라짐 opacity transition 적용

### 마무리
- [x] 전체 테스트 실행 확인 (`npm test`) — 13파일 68테스트 통과
- [x] tasks.md 업데이트
