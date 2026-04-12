# Refactoring — Issue #8 코드 리뷰 후속 작업

> ARIA/Escape 접근성(DEBT-7 범위)은 제외한다.

---

## 1. RecipePickerModal 에러 상태 처리 [High]

- Target: `frontend/src/pages/LogFormPage.tsx`
- `useLogList`에서 `isError`를 destructure하고, `isLoading` 분기 다음에 에러 UI를 추가한다.
- 현재: 네트워크 오류 시 "아직 brew 기록이 없습니다."로 잘못 표시됨.

---

## 2. RecipePickerModal 불필요한 API 호출 방지 [Medium]

- Target: `frontend/src/pages/LogFormPage.tsx`
- `RecipePickerSection`에서 `<RecipePickerModal>`을 항상 렌더링하는 대신 `{open && <RecipePickerModal ... />}` 패턴으로 변경한다.
- 현재: 모달이 닫혀있어도 `useLogList` 호출됨.

---

## 3. brewMethodLabelMap 중복 제거 [Medium]

- Target: `frontend/src/utils/brewMethod.ts` (신규)
- `BREW_METHOD_LABELS` 상수를 정의하고 아래 3곳의 중복 정의를 import로 교체한다.
  - `frontend/src/pages/LogFormPage.tsx` (인라인 `brewMethodLabelMap`)
  - `frontend/src/components/LogCard.tsx`
  - `frontend/src/pages/LogDetailPage.tsx`

---

## 4. `as BrewLogFull` 타입 단언 정리 [Low]

- Target: `frontend/src/pages/LogFormPage.tsx`
- `logs` 변수 선언 시 `as BrewLogFull[]`로 한 번만 단언하여 이후 3곳의 반복 단언을 제거한다.

---

## 5. 테스트 fixture 중복 정리 [Low]

- Target: `frontend/src/pages/logFormState.test.ts`
- `recipeToFormState` describe 내 `brewLog` fixture를 `cloneToFormState`의 것과 통합하여 파일 상단 공유 fixture로 추출한다.

---

## 6. RecipePicker 컴포넌트 테스트 추가 [High]

- Target: `frontend/src/pages/LogFormPage.test.tsx`
- `vi.mock('../hooks/useLogs', ...)`에 `useLogList` mock 추가
- 테스트 케이스:
  - brew 새 로그 모드에서 "이전 레시피 불러오기" 버튼 표시
  - cafe 모드에서 버튼 미표시
  - 수정 모드에서 버튼 미표시
  - clone 모드에서 버튼 미표시

태스크 1-5 완료 후 실행한다 (구현 변경이 확정된 뒤 테스트 작성).
