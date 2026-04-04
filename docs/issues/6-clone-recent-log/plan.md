# #6 최근 기록 복제 — 구현 계획

## 개요

기존 로그를 복제하여 새 로그 작성 폼의 초기값으로 채워주는 프론트엔드 전용 기능.
백엔드 변경 없이 기존 create API(`POST /api/v1/logs`)를 그대로 사용한다.

## 핵심 설계 결정

### 데이터 전달 방식: React Router state

`navigate('/logs/new', { state: { cloneFrom: log } })` 패턴을 사용한다.

- URL에 query param을 노출하지 않아 깔끔하다.
- LogCard에서는 이미 `log` 객체를 props로 갖고 있어 별도 API 호출 불필요.
- LogDetailPage에서도 `useLog(id)`로 이미 조회한 데이터를 그대로 넘긴다.
- 새 라우트 추가 불필요 — 기존 `/logs/new` 라우트를 재활용.

### 폼 초기화: `cloneToFormState()` 함수 신설

기존 `logToFormState()`는 원본 데이터를 있는 그대로 변환한다.
복제 시에는 특정 필드를 리셋해야 하므로, `logToFormState()`를 호출한 뒤 리셋 로직을 적용하는 `cloneToFormState()` 래퍼를 `logFormState.ts`에 추가한다.

리셋 대상: `recordedAt`(오늘 날짜), `rating`(빈 값), `memo`(빈 값), `companions`(빈 배열).
나머지 필드(log_type, cafe/brew 전용 필드, tasting_tags, brew_steps 등)는 원본 값 유지.

### LogCard 액션 메뉴

현재 LogCard는 전체가 `<Link>` 래퍼로 감싸져 있다.
복제 버튼 추가를 위해 카드 구조를 변경한다:
- 카드 하단 footer 영역에 "복제" 버튼 추가.
- 복제 버튼은 `e.preventDefault()`로 Link 네비게이션을 막고, clone 네비게이션을 수행한다.

## 영향 범위

| 파일 | 변경 내용 |
|------|----------|
| `frontend/src/pages/logFormState.ts` | `cloneToFormState()` 함수 추가 |
| `frontend/src/pages/LogFormPage.tsx` | `useLocation()` state에서 clone 데이터 수신, 폼 초기화 로직 추가 |
| `frontend/src/pages/LogDetailPage.tsx` | 헤더 액션에 "이 기록으로 다시 쓰기" 버튼 추가 |
| `frontend/src/components/LogCard.tsx` | 카드 footer에 "복제" 버튼 추가 |

## 테스트 전략

- `logFormState.ts`: `cloneToFormState()` 단위 테스트 — 복제/리셋 필드 규칙 검증
- `LogFormPage.tsx`: clone state가 전달되었을 때 폼이 올바르게 초기화되는지 통합 테스트
- E2E: 상세 화면 → 복제 → 저장 → 새 로그 생성 확인
