# Tasks — Issue #3 모바일 UX 개선

> Phase 간 의존 관계 없음. 독립적으로 구현 가능하다.
> 상세 설계 배경은 `plan.md`를 참조한다.

---

## Phase 1 — HTML 메타데이터 및 아이콘

- [x] **타이틀 변경**
  - `frontend/index.html`: `<title>frontend</title>` → `<title>Coffee of the Day</title>`

- [x] **Favicon 교체**
  - `frontend/public/favicon.svg`: Vite 기본 번개 아이콘 → 커피잔 SVG (stone-950/amber-900 팔레트)
  - Vite 보일러플레이트 파일 삭제: `icons.svg`, `App.tsx`, `App.css`, `assets/`

> Apple Touch Icon + Web App Manifest는 PNG 이미지 준비 후 별도 진행 (이슈 #3 댓글 참조)

---

## Phase 2 — 모바일 레이아웃 수정

- [x] **헤더 버튼 개행 방지**
  - `frontend/src/components/Layout.tsx`: 헤더 영역(line 33-45)의 `New Log` 링크와 `로그아웃` 버튼 className에 `whitespace-nowrap` 추가

- [x] **섹션 헤더 레이아웃 개선**
  - `frontend/src/components/Layout.tsx`: 섹션 헤더 영역(line 52-64)
  - `h1` 태그: `text-3xl` → `text-2xl sm:text-3xl` (모바일 타이틀 크기 축소, 기존 `sm:text-4xl`은 제거)
  - 액션 버튼 래퍼(`div.flex.shrink-0`): 버튼 요소에 `whitespace-nowrap` 추가, 모바일에서 버튼 전체 너비 적용(`w-full sm:w-auto`)

- [x] **페이지별 액션 버튼 점검**
  - 아래 파일에서 `Layout`의 `actions` prop에 전달하는 모든 `<Link>`, `<button>`에 `whitespace-nowrap` 추가
  - `frontend/src/pages/HomePage.tsx` (line 132-143): `오늘의 기록 추가`, `빠른 추가`
  - `frontend/src/pages/LogDetailPage.tsx` (line 77-91): `목록으로`, `수정`
  - `frontend/src/pages/LogFormPage.tsx` (line 752-765): `상세로`/`목록으로`, `변경 저장`/`기록 추가`
  - 주의: Layout 헤더의 `New Log`, `로그아웃`은 이전 태스크에서 처리 완료

---

## Phase 3 — 스크롤 동작 수정

- [x] **ScrollRestoration 적용**
  - `frontend/src/router.tsx`: `RootLayout` 컴포넌트 추가, 기존 라우트를 `children`으로 감쌈
  - `ScrollRestoration`으로 페이지 전환 시 스크롤 초기화 + 뒤로가기 시 복원

- [x] **Scroll to Top 버튼 추가**
  - `frontend/src/components/ScrollToTop.tsx`: 스크롤 300px 이상 시 우측 하단에 반투명 화살표 버튼 표시
  - `requestAnimationFrame` + `easeInOutCubic`으로 고정 500ms 내 스크롤 이동 (거리 무관)
  - `RootLayout`에서 렌더링하여 모든 페이지에 적용

---

## Phase 4 — 조회일자 기본값

- [x] **날짜 기본값 설정**
  - `frontend/src/pages/HomePage.tsx` (line 28-29)
  - `searchParams.get('date_from') ?? ''` → `searchParams.get('date_from') ?? getDefaultDateFrom()`
  - `searchParams.get('date_to') ?? ''` → `searchParams.get('date_to') ?? getDefaultDateTo()`
  - `getDefaultDateFrom()`: 당월 1일 (`YYYY-MM-01` 형식)
  - `getDefaultDateTo()`: 오늘 (`YYYY-MM-DD` 형식)
  - 헬퍼 함수는 컴포넌트 밖(모듈 스코프)에 선언
  - 기본값을 URL 파라미터에 쓰지 않는다 — URL이 깨끗하게 유지되어야 함
  - FilterBar는 props로 계산된 값을 받으므로 추가 수정 불필요

---

## 리팩토링 (코드 리뷰 반영)

- [x] **`getDefaultDateTo()` 타임존 버그 수정**: UTC `toISOString()` → 로컬 시간 기반으로 변경
- [x] **날짜 헬퍼 모듈 분리**: `src/utils/date.ts`로 추출, `formatLocalDate` 공통 헬퍼 포함
- [x] **날짜 헬퍼 단위 테스트**: `src/utils/date.test.ts` (KST 자정 근방 케이스 포함)
- [x] **ScrollToTop throttling**: `requestAnimationFrame` 기반 scroll 이벤트 throttle 적용
- [x] **ScrollToTop transition**: `opacity` + `pointer-events`로 부드러운 나타남/사라짐 전환
- [x] **ScrollToTop 단위 테스트**: `src/components/ScrollToTop.test.tsx` (visible 토글, 버튼 클릭)
