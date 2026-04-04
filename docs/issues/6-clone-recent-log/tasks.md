# #6 최근 기록 복제 — 태스크

## 1. `cloneToFormState()` 함수 추가

파일: `frontend/src/pages/logFormState.ts`

`logToFormState(log)`를 호출한 뒤, 이슈 명세에 따라 필드를 리셋하는 래퍼 함수를 export한다.

```ts
export function cloneToFormState(log: CoffeeLogFull, now = new Date()): LogFormState
```

동작:
- `logToFormState(log)` 호출로 원본 폼 상태 생성
- `recordedAt`을 `now` 기준 datetime-local 포맷으로 교체
- `companions`를 빈 배열로 리셋
- `memo`를 빈 문자열로 리셋
- `cafe.rating` 또는 `brew.rating`을 빈 문자열로 리셋 (logType에 따라)
- `cafe.impressions` 또는 `brew.impressions`를 빈 문자열로 리셋 (logType에 따라)

나머지(log_type, cafe/brew 전용 필드, tasting_tags, tasting_note, brew_steps, 레시피 수치)는 원본 유지.

## 2. `cloneToFormState()` 단위 테스트

파일: `frontend/src/pages/logFormState.test.ts` (기존 파일이 있으면 추가, 없으면 생성)

cafe 타입과 brew 타입 각각에 대해:
- 복제 대상 필드가 원본 값을 유지하는지 검증
- 리셋 대상 필드(recorded_at, rating, memo, companions, impressions)가 초기화되는지 검증

## 3. LogFormPage에 clone 모드 추가

파일: `frontend/src/pages/LogFormPage.tsx`

변경 사항:
- `useLocation()`을 import하고, `location.state?.cloneFrom`에서 `CoffeeLogFull` 타입 데이터를 읽는다.
- `isCloneMode` 변수 도출: `!isEditMode && location.state?.cloneFrom != null`
- 기존 `useEffect`(edit 모드 hydrate) 아래에 clone 모드용 초기화 로직 추가:
  - `isCloneMode`이고 아직 hydrate되지 않았으면 `cloneToFormState(cloneFrom)` 호출
  - `setForm(formState)`, `setExpanded(hasOptionalValues(formState))` 적용
  - `hydratedLogIDRef`를 활용하여 중복 hydrate 방지 (clone 전용 sentinel 값 사용, 예: `'clone'`)
- Layout title: clone 모드일 때 '기록 복제'로 표시
- Layout description: clone 모드일 때 적절한 안내 문구

## 4. LogDetailPage에 복제 버튼 추가

파일: `frontend/src/pages/LogDetailPage.tsx`

변경 사항:
- `useNavigate()`는 이미 import되어 있으므로 추가 import 불필요
- Layout `actions` 영역에 "이 기록으로 다시 쓰기" 버튼 추가 (수정 버튼 옆)
- 클릭 시 `navigate('/logs/new', { state: { cloneFrom: log } })` 호출
- `log`가 로드되었을 때만 버튼 표시 (`id && log` 조건)

## 5. LogCard에 복제 버튼 추가

파일: `frontend/src/components/LogCard.tsx`

변경 사항:
- `useNavigate`를 import
- 카드 하단 footer 영역(현재 "Detail" / "View log" 텍스트가 있는 곳)에 "복제" 버튼 추가
- 복제 버튼 클릭 시 `e.preventDefault()`로 Link 이벤트 전파 중단, `navigate('/logs/new', { state: { cloneFrom: log } })` 호출
- 버튼 스타일: 기존 디자인 시스템과 일관된 작은 텍스트 버튼

## 6. 통합 테스트

LogFormPage clone 모드 통합 테스트:
- clone state로 진입 시 폼 필드가 올바르게 채워지는지 검증
- recordedAt이 오늘 날짜인지 검증
- rating, memo, companions가 비어있는지 검증
- log_type이 올바르게 설정되는지 검증

## 7. E2E 테스트

상세 화면에서 복제 → 저장까지의 전체 플로우:
- 상세 화면에서 "이 기록으로 다시 쓰기" 클릭
- 폼이 복제된 데이터로 채워져 열리는지 확인
- 저장 후 새 로그가 생성되고 원본은 변경되지 않는지 확인
