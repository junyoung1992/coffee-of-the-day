# Issue #8 — 브루 레시피 템플릿

## 목표

브루 로그 작성 시 이전 브루 로그의 레시피 필드만 선택적으로 불러와 폼을 초기화할 수 있게 한다. 홈브루잉에서 같은 원두로 파라미터를 미세 조정하는 워크플로우를 지원한다.

---

## 핵심 설계: 프론트엔드 전용 기능

이 기능은 **백엔드 변경 없이** 구현한다. 기존 `GET /api/v1/logs?log_type=brew` API가 brew 로그 목록을 이미 반환하며, 각 로그에 레시피 필드가 모두 포함되어 있다. 프론트엔드에서 목록을 조회하고 사용자가 선택한 로그에서 레시피 필드만 추출하여 폼에 채우면 된다.

**기존 패턴 활용:** `logFormState.ts`에 `cloneToFormState()`와 `presetToFormState()`가 이미 존재한다. 레시피 불러오기는 이 둘의 중간 형태로, `recipeToFormState()` 함수를 새로 추가한다.

---

## 레시피 필드 매핑

기존 `BrewLogFull`에서 불러올 필드와 리셋할 필드를 명확히 구분한다.

**불러오는 필드 (brew 객체에서):**
- `bean_name`, `brew_method`, `brew_device`, `coffee_amount_g`, `water_amount_ml`, `water_temp_c`, `brew_time_sec`, `grind_size`, `brew_steps`
- `bean_origin`, `bean_process`, `roast_level`, `roast_date` (원두 정보도 레시피의 일부로 포함)

**리셋하는 필드:**
- `rating`, `memo`, `companions`, `tasting_tags`, `tasting_note`, `impressions`
- `recorded_at` -> 현재 시각으로 초기화

이 매핑은 issue 요구사항의 테이블을 따른다. `cloneToFormState()`가 거의 모든 필드를 복제한 뒤 일부만 리셋하는 것과 반대로, `recipeToFormState()`는 빈 폼에서 시작하여 레시피 필드만 채운다.

---

## UI 설계

### 진입점

브루 로그 작성 폼(`LogFormPage.tsx`)에서 `logType === 'brew'`일 때, `BrewFieldsSection` 필수 영역 위 또는 `PresetSection`과 같은 위치에 "이전 레시피 불러오기" 버튼을 배치한다.

**배치 위치:** `PresetSection` 아래, `BrewFieldsSection` 위. `PresetSection`과 동일한 레벨의 독립 섹션으로 렌더링한다. 새 로그 작성 모드에서만 표시하고, 수정 모드와 clone 모드에서는 숨긴다.

### 레시피 선택 모달

버튼 클릭 시 모달을 열어 최근 brew 로그 목록을 표시한다.

**모달 내용:**
- 제목: "이전 레시피 불러오기"
- 목록: 최근 brew 로그를 최신순으로 표시. 각 항목에 `bean_name + brew_method + recorded_at` 표시
- 페이지네이션: 기존 `useLogList({ log_type: 'brew' })` 무한 스크롤 훅을 활용
- 선택 시: 모달 닫힘 + 레시피 필드 채움 + 선택 영역 자동 펼침(레시피 값이 존재하므로 `hasOptionalValues()`가 true 반환)

**모달 구현 방식:** 현재 프로젝트에는 공통 Modal 컴포넌트가 없다(DEBT-7 참조). 이번에도 `PresetsPage.tsx`의 `EditModal`과 유사한 인라인 모달을 구현한다. 공통 Modal 컴포넌트 추출은 이 issue의 범위가 아니다.

### 덮어쓰기 경고

사용자가 이미 brew 필드를 입력한 상태에서 레시피를 불러오면 기존 값이 덮어쓰여진다. DEBT-6과 동일한 이슈이며, 이 issue에서는 프리셋과 동일한 방식(즉시 덮어쓰기)으로 동작한다. 향후 DEBT-6에서 일괄적으로 확인 UX를 추가할 때 함께 개선한다.

---

## 구현 위치 상세

### `logFormState.ts`

`recipeToFormState(log: BrewLogFull, now?: Date): LogFormState` 함수 추가. 위치는 `cloneToFormState()` 바로 아래.

구현 전략: `createEmptyFormState(now)`로 빈 폼을 만들고, `logType`을 `'brew'`로 설정한 뒤, brew 객체의 레시피 필드만 복사한다. `logToFormState()`의 brew 분기에서 레시피 관련 필드만 가져오는 서브셋이다.

### `LogFormPage.tsx`

1. `RecipePickerSection` 컴포넌트 추가: `PresetSection`과 유사한 구조의 섹션. "이전 레시피 불러오기" 버튼 하나를 렌더링한다.
2. `RecipePickerModal` 컴포넌트 추가: brew 로그 목록을 표시하는 모달. `useLogList({ log_type: 'brew' })`를 호출하여 데이터를 가져온다.
3. `LogFormPage` 메인 컴포넌트에서 `recipeToFormState` import 및 사용. clone/edit 모드가 아니고 `logType === 'brew'`일 때 `RecipePickerSection`을 렌더링한다.

### `useLogs.ts`

변경 없음. 기존 `useLogList`가 `log_type` 파라미터를 이미 지원한다.

---

## 수정하지 않는 것

- **백엔드**: API, DB, 서비스 레이어 일체 변경 없음
- **`docs/openapi.yml`**: API 스키마 변경 없음
- **`docs/spec.md`**: 레시피 불러오기 기능은 spec에 새 섹션으로 추가해야 하지만, UI 동작 규칙의 확장이므로 구현 후 반영
- **기존 컴포넌트**: `LogCard`, `LogDetailPage`, `BrewFieldsSection`, `CafeFieldsSection` 등 기존 컴포넌트는 변경하지 않음
- **타입 파일**: `types/log.ts`, `types/schema.ts` 변경 없음

---

## 테스트 전략

### Unit 테스트

1. **`logFormState.test.ts`**: `recipeToFormState()` 함수 테스트
   - brew 로그에서 레시피 필드만 정확히 채워지는지 검증
   - `rating`, `tasting_tags`, `tasting_note`, `impressions`, `memo`, `companions`가 빈 값인지 검증
   - `recorded_at`이 현재 시각으로 초기화되는지 검증
   - `logType`이 `'brew'`로 설정되는지 검증

2. **`LogFormPage.test.tsx`**: RecipePickerSection/Modal 통합 테스트
   - brew 모드에서 "이전 레시피 불러오기" 버튼이 표시되는지
   - cafe 모드에서는 버튼이 표시되지 않는지
   - 수정 모드/clone 모드에서는 버튼이 표시되지 않는지
   - 모달에서 로그 선택 시 레시피 필드가 채워지는지

### E2E 테스트

PR 전에 수동으로 확인:
- brew 로그 작성 화면에서 레시피 불러오기 -> 선택 -> 필드 채움 -> 수정 -> 저장 흐름
