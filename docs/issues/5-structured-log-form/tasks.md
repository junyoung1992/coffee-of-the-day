# Tasks — Issue #5 단계적 입력: 로그 작성 폼 필드 구조화

> 모든 태스크는 프론트엔드 전용. 상세 설계 배경은 `plan.md`를 참조한다.
> 태스크 1~2는 독립적으로 구현 가능. 태스크 3은 1, 2에 의존한다.

---

## 1. hasOptionalValues 헬퍼 추가

- [x] **헬퍼 함수 구현**
  - `frontend/src/pages/logFormState.ts`에 `hasOptionalValues(state: LogFormState): boolean` export 함수 추가
  - Cafe 선택 필드 검사: `location`, `beanOrigin`, `beanProcess`, `roastLevel`, `tastingTags`(length > 0), `tastingNote`, `impressions` 중 하나라도 값이 있으면 `true`
  - Brew 선택 필드 검사: `beanOrigin`, `beanProcess`, `roastLevel`, `roastDate`, `brewDevice`, `coffeeAmountG`, `waterAmountMl`, `waterTempC`, `brewTimeSec`, `grindSize`, `tastingTags`(length > 0), `tastingNote`, `brewSteps`(빈 문자열 제외 후 length > 0), `impressions` 중 하나라도 값이 있으면 `true`
  - 공통 선택 필드 검사: `companions`(length > 0) 또는 `memo`가 비어있지 않으면 `true`
  - `state.logType`에 따라 cafe 또는 brew 필드를 검사

- [x] **단위 테스트 추가**
  - `frontend/src/pages/logFormState.test.ts`에 `hasOptionalValues` describe 블록 추가
  - 케이스 1: `createEmptyFormState()` 결과 → `false`
  - 케이스 2: cafe 선택 필드에 값이 있는 경우 → `true` (예: `tastingTags: ['초콜릿']`)
  - 케이스 3: brew 선택 필드에 값이 있는 경우 → `true` (예: `coffeeAmountG: '18'`)
  - 케이스 4: 공통 선택 필드에 값이 있는 경우 → `true` (예: `companions: ['민수']`)
  - 케이스 5: brew `brewSteps`가 `['']`(빈 스텝만)인 경우 → `false` (기본값)

---

## 2. CafeFieldsSection / BrewFieldsSection 필수·선택 영역 분리

- [x] **CommonFieldsSection 해체**
  - `frontend/src/pages/LogFormPage.tsx`에서 `CommonFieldsSection` 컴포넌트 제거
  - `LogFormPage` render에서 `<CommonFieldsSection>` 호출부 제거
  - `recorded_at` 필드는 `LogFormPage`의 `<LogTypeSection>` 아래에 직접 렌더링 (필수 영역 시작 전, 또는 각 타입별 필수 영역 내부에 배치)

- [x] **CafeFieldsSection 구조 변경**
  - props 타입에 `expanded: boolean`, `onToggle: () => void` 추가
  - 필수 영역 (기존 `Section` 컴포넌트 사용):
    - `recorded_at` (datetime-local, required) — `form.recordedAt`
    - `cafe_name` (text, required)
    - `coffee_name` (text, required)
    - `rating` (RatingInput)
  - 토글 버튼: 필수 영역 Section과 선택 영역 Section 사이에 배치
    - 텍스트: `expanded ? '접기' : '더 기록하기'`
    - 스타일: `rounded-full border border-amber-950/10 bg-white px-4 py-2.5 text-sm font-semibold text-stone-700 transition hover:border-amber-900/25 hover:bg-amber-50/60 w-full`
  - 선택 영역 (`expanded`일 때만 렌더링, `Section` 컴포넌트 사용):
    - `location`, `roast_level`, `bean_origin`, `bean_process`
    - `tasting_tags` (TagInput, col-span-2)
    - `tasting_note` (textarea, col-span-2)
    - `impressions` (textarea, col-span-2)
    - `companions` (TagInput) — `form.companions`와 `useCompanionSuggestions` 사용
    - `memo` (textarea, col-span-2) — `form.memo`

- [x] **BrewFieldsSection 구조 변경**
  - props 타입에 `expanded: boolean`, `onToggle: () => void` 추가
  - 필수 영역 (Section):
    - `recorded_at` (datetime-local, required) — `form.recordedAt`
    - `bean_name` (text, required, col-span-2)
    - `brew_method` (button group, col-span-2)
    - `rating` (RatingInput, col-span-2)
  - 토글 버튼: CafeFieldsSection과 동일한 패턴
  - 선택 영역 (expanded일 때만 렌더링, Section):
    - `brew_device`, `grind_size`
    - `bean_origin`, `bean_process`
    - `roast_level`, `roast_date`
    - Recipe 블록 (기존 레이아웃 유지: coffee/ratio/water 인라인 + temp/time)
    - `tasting_tags` (TagInput, col-span-2)
    - `tasting_note` (textarea, col-span-2)
    - `brew_steps` (동적 배열, col-span-2) — 기존 step CRUD 로직 그대로 유지
    - `impressions` (textarea, col-span-2)
    - `companions` (TagInput) — `form.companions`와 `useCompanionSuggestions` 사용
    - `memo` (textarea, col-span-2) — `form.memo`
  - 주의: `useCompanionSuggestions` 훅은 기존 `CommonFieldsSection`에서 사용하던 것. `CafeFieldsSection`과 `BrewFieldsSection` 각각에서 import하여 사용

---

## 3. expanded 상태 관리 및 수정 모드 자동 펼침

- [x] **expanded state 추가**
  - `frontend/src/pages/LogFormPage.tsx`의 `LogFormPage` 컴포넌트에 `const [expanded, setExpanded] = useState(false)` 추가
  - `CafeFieldsSection`/`BrewFieldsSection`에 `expanded`와 `onToggle={() => setExpanded(prev => !prev)}` 전달

- [x] **수정 모드 자동 펼침**
  - 기존 `useEffect` (line 710-724, hydrate 로직) 내부에서 hydrate 완료 후 `hasOptionalValues` 호출
  - `logToFormState(log)` 결과를 변수에 저장 → `setForm(formState)` → `hasOptionalValues(formState)`가 `true`이면 `setExpanded(true)`
  - 생성 모드에서는 `expanded`가 `false`인 기본값 유지

---

## 4. 검증

- [x] **기존 테스트 통과 확인**
  - `cd frontend && npm test` 실행
  - `logFormState.test.ts`의 기존 테스트(`createEmptyFormState`, `buildLogPayload`, `logToFormState`) 통과
  - `hasOptionalValues` 신규 테스트 통과
  - `RatingInput.test.tsx`, `TagInput.test.tsx` 등 기존 컴포넌트 테스트 통과

- [x] **수동 검증 항목** (E2E 테스트 전 로컬 확인)
  - 생성 모드: 필수 필드만 보이는지, "더 기록하기" 클릭 시 선택 필드 펼쳐지는지
  - 생성 모드: 선택 영역에 값 입력 → 접기 → 다시 펼치기 → 값이 유지되는지
  - 생성 모드: 필수 필드만 입력 후 저장 가능한지
  - 수정 모드: 선택 필드에 값이 있는 기록 → 자동으로 펼쳐진 상태인지
  - 수정 모드: 선택 필드에 값이 없는 기록 → 접힌 상태인지
