# Tasks — Issue #8 브루 레시피 템플릿

> 이 기능은 프론트엔드 전용이다. 백엔드/API 변경 없음.
> 기존 `cloneToFormState()`, `presetToFormState()` 패턴을 참고한다.
> Refer to `plan.md` for detailed design context.

---

## 1. recipeToFormState 함수 추가

- [ ] **`recipeToFormState()` 함수 구현**
  - Target: `frontend/src/pages/logFormState.ts`
  - `cloneToFormState()` 아래(line 229 부근)에 `recipeToFormState(log: BrewLogFull, now?: Date): LogFormState` 함수를 추가한다.
  - import에 `BrewLogFull`을 추가한다 (현재 `CoffeeLogFull`만 import됨, `types/log.ts`에서 가져온다).
  - 구현 로직:
    1. `createEmptyFormState(now)`로 빈 폼 생성
    2. `state.logType = 'brew'` 설정
    3. brew 객체에서 레시피 필드만 복사:
       - `beanName` <- `log.brew.bean_name`
       - `brewMethod` <- `log.brew.brew_method` (as BrewMethodValue)
       - `brewDevice` <- `log.brew.brew_device ?? ''`
       - `coffeeAmountG` <- `log.brew.coffee_amount_g ? String(log.brew.coffee_amount_g) : ''`
       - `waterAmountMl` <- `log.brew.water_amount_ml ? String(log.brew.water_amount_ml) : ''`
       - `waterTempC` <- `log.brew.water_temp_c ? String(log.brew.water_temp_c) : ''`
       - `brewTimeSec` <- `log.brew.brew_time_sec ? String(log.brew.brew_time_sec) : ''`
       - `grindSize` <- `log.brew.grind_size ?? ''`
       - `brewSteps` <- `log.brew.brew_steps && log.brew.brew_steps.length > 0 ? log.brew.brew_steps : ['']`
       - `beanOrigin` <- `log.brew.bean_origin ?? ''`
       - `beanProcess` <- `log.brew.bean_process ?? ''`
       - `roastLevel` <- `(log.brew.roast_level ?? '') as RoastLevelValue`
       - `roastDate` <- `log.brew.roast_date ?? ''`
    4. 채우지 않는 필드 (빈 값 유지): `rating`, `tastingTags`, `tastingNote`, `impressions`, `memo`, `companions`
  - 패턴 참조: `logToFormState()`의 brew 분기(line 184-207)에서 필드 매핑 방식을 동일하게 사용한다.

이 태스크는 독립적으로 실행 가능하다.

---

## 2. recipeToFormState 단위 테스트

- [ ] **테스트 케이스 추가**
  - Target: `frontend/src/pages/logFormState.test.ts`
  - 기존 `cloneToFormState` describe 블록 패턴을 참고하여 `recipeToFormState` describe 블록을 추가한다.
  - import에 `recipeToFormState`를 추가한다.
  - 테스트 케이스:
    1. "brew 로그에서 레시피 필드만 채운다" — 모든 레시피 필드가 원본 값으로 채워졌는지 검증
    2. "평가/메모/태그 필드는 비어 있다" — `rating`, `tastingTags`, `tastingNote`, `impressions`가 빈 값, `memo`와 `companions`가 빈 값인지 검증
    3. "recorded_at이 현재 시각으로 초기화된다" — `now` 파라미터와 일치하는지 검증
    4. "logType이 brew로 설정된다" — `state.logType === 'brew'` 검증
  - 테스트 데이터: 기존 테스트 파일의 `brewLog` fixture를 참고하거나, brew 타입의 `BrewLogFull` mock 데이터를 작성한다.

태스크 1에 의존한다.

---

## 3. RecipePickerModal 컴포넌트 구현

- [ ] **RecipePickerModal 컴포넌트 추가**
  - Target: `frontend/src/pages/LogFormPage.tsx`
  - `PresetSection` 컴포넌트 아래(line 212 부근)에 추가한다.
  - Props: `{ open: boolean; onClose: () => void; onSelect: (log: BrewLogFull) => void }`
  - 구현:
    1. `open`이 false면 `null` 반환
    2. `useLogList({ log_type: 'brew' })`를 호출하여 brew 로그 목록을 가져온다.
       - import 추가: `useLogList`를 `'../hooks/useLogs'`에서 import
    3. backdrop(반투명 오버레이) + 중앙 모달 패널 레이아웃
    4. 모달 내 목록: `data.pages.flatMap(p => p.items)`로 전체 로그를 펼쳐서 표시
    5. 각 항목: `bean_name`, `brew_method`(label 변환), `recorded_at`(포맷) 표시
       - `brew_method` label 변환: `LogCard.tsx`의 `brewMethodLabelMap`과 동일한 매핑을 인라인으로 정의하거나, 공유가 필요하면 별도 추출 (이 issue에서는 인라인으로 충분)
       - `recorded_at` 포맷: `Intl.DateTimeFormat('ko-KR', { dateStyle: 'medium' }).format(new Date(log.recorded_at))` 사용
    6. 항목 클릭 시: `onSelect(log as BrewLogFull)` 호출 (brew 필터된 결과이므로 타입 안전)
    7. "더 보기" 버튼: `hasNextPage`가 true이면 표시, 클릭 시 `fetchNextPage()` 호출
    8. 빈 상태: brew 로그가 없으면 "아직 brew 기록이 없습니다." 메시지 표시
    9. backdrop 클릭 또는 X 버튼으로 모달 닫기
  - 스타일: `PresetsPage.tsx`의 `EditModal` 패턴을 참고하되, 목록 형태에 맞게 조정한다.
    - 모달 최대 높이: `max-h-[70vh]`, 목록 영역 스크롤: `overflow-y-auto`

이 태스크는 독립적으로 실행 가능하다 (태스크 1과 병렬 가능).

---

## 4. RecipePickerSection 및 LogFormPage 통합

- [ ] **RecipePickerSection 컴포넌트 추가**
  - Target: `frontend/src/pages/LogFormPage.tsx`
  - `RecipePickerModal` 아래에 추가한다.
  - Props: `{ onSelect: (log: BrewLogFull) => void }`
  - 구현:
    1. `useState`로 모달 open 상태 관리
    2. "이전 레시피 불러오기" 버튼을 가진 섹션 렌더링 (스타일: `PresetSection`과 유사한 섹션 레이아웃)
    3. 버튼 클릭 시 모달 open
    4. 모달에서 로그 선택 시 `onSelect` 호출 + 모달 닫기

- [ ] **LogFormPage에 RecipePickerSection 배치**
  - Target: `frontend/src/pages/LogFormPage.tsx`
  - `LogFormPage` 메인 컴포넌트의 JSX에서 `PresetSection` 아래에 조건부 렌더링 추가
  - 조건: `!isEditMode && !isCloneMode && form.logType === 'brew'`
  - import 추가: `recipeToFormState`를 `'./logFormState'`에서 import, `BrewLogFull`을 `'../types/log'`에서 import
  - onSelect 핸들러:
    ```
    (log: BrewLogFull) => {
      const formState = recipeToFormState(log)
      setForm(formState)
      setExpanded(hasOptionalValues(formState))
    }
    ```
  - 주의: `PresetSection`은 `!isEditMode && !isCloneMode`일 때 표시되는데, `RecipePickerSection`은 추가로 `form.logType === 'brew'` 조건이 필요하다.

태스크 1, 3에 의존한다.

---

## 5. spec.md 업데이트

- [ ] **spec.md에 레시피 불러오기 섹션 추가**
  - Target: `docs/spec.md`
  - 6.3 로그 복제 아래에 "6.4 브루 레시피 불러오기" 섹션을 추가한다.
  - 내용:
    - 기능 설명: 브루 로그 작성 시 이전 브루 로그의 레시피 필드만 불러오는 기능
    - 진입점: 브루 로그 작성 폼에서 "이전 레시피 불러오기" 버튼
    - 필드 채움 규칙 테이블 (불러오는 필드 / 리셋하는 필드)
    - 동작 규칙: 새 로그 작성 모드에서만 표시, 수정/clone 모드에서는 숨김
  - `*Last updated:*` 행을 업데이트한다.

태스크 4 완료 후 실행한다 (구현이 확정된 뒤 spec에 반영).

---

## 6. 검증

- [ ] **단위 테스트 실행**
  - Command: `cd frontend && npm test`
  - `logFormState.test.ts`의 `recipeToFormState` 테스트가 모두 통과하는지 확인

- [ ] **기존 테스트 회귀 확인**
  - Command: `cd frontend && npm test`
  - `LogFormPage.test.tsx`의 기존 테스트가 깨지지 않는지 확인

- [ ] **수동 검증 항목**
  - brew 로그가 1개 이상 있는 상태에서:
    1. `/logs/new` 진입 -> brew 타입 선택 -> "이전 레시피 불러오기" 버튼 표시 확인
    2. 버튼 클릭 -> 모달에 brew 로그 목록이 최신순으로 표시되는지 확인
    3. 로그 선택 -> 레시피 필드(bean_name, brew_method, coffeeAmountG 등)가 채워지는지 확인
    4. rating, tasting_tags, memo, companions가 비어 있는지 확인
    5. 불러온 레시피를 수정할 수 있는지 확인 (예: grindSize 변경)
    6. 저장 -> 새 로그가 정상 생성되는지 확인
  - cafe 타입에서는 레시피 불러오기 버튼이 표시되지 않는지 확인
  - 수정 모드(`/logs/:id/edit`)에서는 버튼이 표시되지 않는지 확인
  - clone 모드에서는 버튼이 표시되지 않는지 확인

태스크 1-5 모두 완료 후 실행한다.
