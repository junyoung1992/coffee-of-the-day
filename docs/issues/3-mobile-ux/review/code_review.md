# Code Review — Issue #3 모바일 UX 개선

## 리뷰 범위

`feat/3-mobile-ux` 브랜치의 `main` 대비 전체 변경사항.
커밋: `483dbdf`(plan/tasks) ~ `3ab2d3a`(backlog 업데이트) 총 6커밋.

---

## 필수 수정 (머지 차단)

### 1. `getDefaultDateTo()` 타임존 버그

**파일:** `frontend/src/pages/HomePage.tsx:30-31`

```tsx
// 현재 코드 — UTC 기준
function getDefaultDateTo(): string {
  return new Date().toISOString().slice(0, 10)
}
```

`toISOString()`은 UTC 기준이다. KST(UTC+9)에서 00:00~08:59 사이에 호출하면 **어제 날짜**를 반환한다.
`getDefaultDateFrom()`은 로컬 시간(`getFullYear()`, `getMonth()`)을 사용하므로 두 함수 간 시간대 기준이 불일치한다.

**수정 방법:** `getDefaultDateFrom()`과 동일하게 로컬 시간 기반으로 변경한다.

```tsx
function getDefaultDateTo(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}
```

### 2. 테스트 누락

AGENTS.md 규칙: "Always write tests when adding or modifying functionality."

이번 변경에 테스트가 없다. 최소한 아래 항목에 대한 단위 테스트를 추가해야 한다.

- **`getDefaultDateFrom` / `getDefaultDateTo`**: 순수 함수이므로 단위 테스트가 쉽다. 위 타임존 버그 수정이 올바른지 검증하는 데도 필요하다.
- **`ScrollToTop` 컴포넌트**: `visible` 상태 토글 로직(스크롤 300px 기준) 검증.

`whitespace-nowrap` 추가 등 Tailwind 클래스 변경은 시각적 영역이므로 단위 테스트 대상이 아니다.

---

## 권장 개선 (선택)

### 3. ScrollToTop scroll 이벤트 throttling

**파일:** `frontend/src/components/ScrollToTop.tsx:34-41`

`scroll` 이벤트는 초당 수십~수백 회 발생한다. `{ passive: true }`는 브라우저 스크롤 성능을 보호하지만 `setVisible()` 호출 빈도는 줄이지 않는다. React가 동일 값의 `setState`는 리렌더를 건너뛰므로 현재 실사용에 문제는 없다.

목록이 길어질 경우를 대비하려면 `requestAnimationFrame` 기반 throttle을 적용할 수 있다:

```tsx
useEffect(() => {
  let ticking = false
  function handleScroll() {
    if (!ticking) {
      requestAnimationFrame(() => {
        setVisible(window.scrollY > 300)
        ticking = false
      })
      ticking = true
    }
  }
  window.addEventListener('scroll', handleScroll, { passive: true })
  return () => window.removeEventListener('scroll', handleScroll)
}, [])
```

### 4. ScrollToTop 나타남/사라짐 transition 미동작

**파일:** `frontend/src/components/ScrollToTop.tsx:47`

`if (!visible) return null`로 DOM에서 완전 제거하므로, className의 `transition`이 실제로 동작하지 않는다. 부드러운 나타남/사라짐을 원한다면 항상 렌더링하되 `opacity`와 `pointer-events`로 전환한다:

```tsx
return (
  <button
    type="button"
    onClick={scrollToTop}
    aria-label="맨 위로 스크롤"
    className={`fixed bottom-6 right-6 z-50 flex h-11 w-11 items-center justify-center rounded-full border border-amber-950/10 bg-white/70 text-stone-600 shadow-lg backdrop-blur-sm transition-opacity duration-300 hover:bg-white hover:text-stone-900 ${
      visible ? 'opacity-100' : 'pointer-events-none opacity-0'
    }`}
  >
    {/* svg */}
  </button>
)
```

의도적으로 transition 없이 즉시 토글하는 것이라면 `transition` 클래스를 제거하여 의도를 명확히 한다.

---

## 문제 없는 부분

- **`ScrollRestoration` + `RootLayout` 패턴** (`router.tsx`): React Router 공식 권장 방식을 정확히 따랐다. 기존 라우트를 `children`으로 감싸는 구조가 깔끔하다.
- **날짜 기본값을 URL에 반영하지 않는 설계** (`HomePage.tsx`): 공유 링크 안정성과 뒤로가기 동작을 보존한다.
- **`animateScrollToTop` 고정 duration** (`ScrollToTop.tsx`): 스크롤 거리 무관 500ms 고정으로 일관된 UX를 제공한다. `easeInOutCubic` 선택도 적절하다.
- **Favicon SVG** (`favicon.svg`): stone-950/amber-900 앱 팔레트와 색상이 일치한다.
- **Vite 보일러플레이트 삭제**: `App.tsx`, `App.css`, `assets/`, `icons.svg` 삭제 후 남은 참조 없음을 확인했다.
- **`whitespace-nowrap` 적용 범위**: Layout 헤더 버튼 + 각 페이지 actions 버튼 모두 빠짐없이 적용되었다.
- **섹션 헤더 반응형 레이아웃** (`Layout.tsx:64`): `w-full sm:w-auto`로 모바일에서 버튼이 전체 너비를 차지하도록 처리했다.

---

## 수정 작업 요약

| # | 항목 | 파일 | 구분 |
|---|------|------|------|
| 1 | `getDefaultDateTo()` 로컬 시간 기반으로 수정 | `HomePage.tsx` | 필수 |
| 2 | 날짜 기본값 함수 + ScrollToTop 단위 테스트 추가 | 신규 테스트 파일 | 필수 |
| 3 | scroll 이벤트 throttling | `ScrollToTop.tsx` | 선택 |
| 4 | ScrollToTop transition 동작 또는 transition 클래스 제거 | `ScrollToTop.tsx` | 선택 |
