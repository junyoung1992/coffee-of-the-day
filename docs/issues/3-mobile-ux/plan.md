# Issue #3 — 모바일 UX 개선 및 기본 메타데이터 수정

## 목표

iPhone 16(393px) Safari에서 모든 화면이 자연스럽게 동작하고, 브라우저 탭·파비콘·홈 화면 추가 시 앱 정체성이 드러나도록 프론트엔드 프레젠테이션 레이어만 수정한다.

백엔드, `docs/openapi.yml`, `docs/spec.md` 변경 없음.

---

## Phase 1 — HTML 메타데이터 및 아이콘

### 1-1. 타이틀 변경

**파일:** `frontend/index.html`

`<title>frontend</title>` → `<title>Coffee of the Day</title>`

### 1-2. Favicon 교체

**파일:** `frontend/public/favicon.svg`

현재 파일은 Vite 기본 번개 아이콘(보라색)이다. 커피 테마 SVG로 교체한다.

### 1-3. Apple Touch Icon + Web App Manifest — 후속 작업

> **이 Phase의 apple-touch-icon(180x180 PNG)과 manifest는 구현 범위에서 제외한다.**
> PNG 이미지 생성이 필요하므로, 나머지 작업 완료 후 별도로 아이콘을 준비하여 추가한다.
> 상세 스펙은 이슈 #3 댓글에 기록한다.

구현 시 필요한 사항:
- `frontend/public/apple-touch-icon.png` — 180x180 PNG (iOS 필수 형식)
- `frontend/public/manifest.json` — `background_color`/`theme_color`는 `#faf4eb` (`index.css` 배경색)
- `frontend/index.html`에 `<link rel="apple-touch-icon">`, `<link rel="manifest">`, `<meta name="apple-mobile-web-app-title">` 추가

---

## Phase 2 — 모바일 레이아웃 수정

### 2-1. 헤더 버튼 텍스트 개행 방지

**파일:** `frontend/src/components/Layout.tsx`

헤더의 `New Log`, `로그아웃` 버튼에 `whitespace-nowrap`을 추가한다.

### 2-2. 섹션 헤더 타이틀 + 액션 레이아웃

**파일:** `frontend/src/components/Layout.tsx`

현재 구조:
```
flex-col → sm:flex-row sm:items-end sm:justify-between
  ├─ 타이틀 영역 (max-w-2xl, text-3xl sm:text-4xl)
  └─ 액션 버튼 영역 (flex shrink-0 flex-wrap gap-3)
```

수정 사항:
- 타이틀 크기: `text-3xl` → `text-2xl sm:text-3xl` (모바일에서 축소)
- 액션 버튼에 `whitespace-nowrap` 추가
- 모바일에서 버튼을 가로 전체 너비로 배치: `w-full sm:w-auto`

### 2-3. 각 페이지 액션 버튼 점검

Layout에 `actions`를 전달하는 모든 페이지의 버튼에 `whitespace-nowrap`을 추가한다.

| 파일 | 버튼 텍스트 |
|------|------------|
| `frontend/src/pages/HomePage.tsx` | `오늘의 기록 추가`, `빠른 추가` |
| `frontend/src/pages/LogDetailPage.tsx` | `목록으로`, `수정` |
| `frontend/src/pages/LogFormPage.tsx` | `상세로`/`목록으로`, `변경 저장`/`기록 추가` |

---

## Phase 3 — 스크롤 동작 수정

**파일:** `frontend/src/router.tsx`

React Router v6.4+의 `ScrollRestoration` 컴포넌트를 사용한다. `createBrowserRouter`에서 이를 사용하려면 루트 레이아웃 컴포넌트가 필요하다.

기존 라우트를 루트 레이아웃의 `children`으로 감싼다:

```tsx
import { Outlet, ScrollRestoration } from 'react-router-dom'

function RootLayout() {
  return (
    <>
      <ScrollRestoration />
      <Outlet />
    </>
  )
}

export const router = createBrowserRouter([
  {
    element: <RootLayout />,
    children: [
      { path: '/login', element: <LoginPage /> },
      { path: '/register', element: <RegisterPage /> },
      {
        element: <ProtectedRoute />,
        children: [
          { path: '/', element: <HomePage /> },
          { path: '/logs/new', element: <LogFormPage /> },
          { path: '/logs/:id', element: <LogDetailPage /> },
          { path: '/logs/:id/edit', element: <LogFormPage /> },
        ],
      },
    ],
  },
])
```

`ScrollRestoration`은 `useEffect` + `scrollTo(0,0)` 대비 뒤로가기 시 스크롤 복원도 처리한다.

---

## Phase 4 — 조회일자 기본값

**파일:** `frontend/src/pages/HomePage.tsx`

URL에 `date_from`/`date_to` 파라미터가 없을 때 당월 1일 ~ 오늘을 기본값으로 사용한다.

```tsx
// 변경 전
const dateFrom = searchParams.get('date_from') ?? ''
const dateTo = searchParams.get('date_to') ?? ''

// 변경 후
function getDefaultDateFrom(): string {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-01`
}

function getDefaultDateTo(): string {
  return new Date().toISOString().slice(0, 10)
}

const dateFrom = searchParams.get('date_from') ?? getDefaultDateFrom()
const dateTo = searchParams.get('date_to') ?? getDefaultDateTo()
```

기본값은 URL 파라미터에 쓰지 않는다. URL이 깨끗하게 유지되어 공유 링크와 뒤로가기에 영향이 없다.

FilterBar는 이미 props로 `dateFrom`/`dateTo`를 받으므로 추가 수정이 불필요하다.

---

## 제약 조건

- `apple-touch-icon.png`는 반드시 PNG 180x180이어야 한다. iOS는 SVG를 지원하지 않는다.
- `manifest.json`의 `background_color`/`theme_color`는 `index.css`의 배경색과 일치시킨다.
- 날짜 기본값은 URL에 반영하지 않는다. URL 파라미터가 명시적으로 있을 때만 해당 값을 사용한다.
- Phase 간 의존 관계 없음. 독립적으로 구현 가능하다.

## 결정 사항

| 항목 | 결정 | 이유 |
|------|------|------|
| Favicon | SVG 직접 작성 | 외부 의존성 없이 커피 테마에 맞는 간결한 아이콘 |
| Apple Touch Icon + Manifest | 후속 작업으로 분리 | PNG 이미지 생성이 필요하므로 별도 진행 |
| 스크롤 초기화 | `ScrollRestoration` | React Router 공식 솔루션. 뒤로가기 시 복원도 처리 |
| 날짜 기본값 | 당월 1일 ~ 오늘 | 최근 기록을 바로 볼 수 있는 적절한 범위 |
| 기본값 URL 반영 | URL에 쓰지 않음 | 깨끗한 URL 유지, 공유 링크 안정성 |
