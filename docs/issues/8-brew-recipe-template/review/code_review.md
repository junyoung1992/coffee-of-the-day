# 코드 리뷰

## 리뷰 범위

- **브랜치**: `feat/8-brew-recipe-template`
- **비교 기준**: `main...feat/8-brew-recipe-template`
- **변경 파일**:
  - `frontend/src/pages/LogFormPage.tsx`
  - `frontend/src/pages/logFormState.ts`
  - `frontend/src/pages/logFormState.test.ts`
  - `docs/spec.md`
  - `docs/backlog.md`
  - `docs/issues/8-brew-recipe-template/plan.md` (신규)
  - `docs/issues/8-brew-recipe-template/tasks.md` (신규)

## 요약

이전 brew 로그의 레시피 필드만 선택적으로 불러오는 프론트엔드 전용 기능을 구현한 PR이다. `recipeToFormState()` 함수 설계와 단위 테스트 커버리지는 기존 패턴(`cloneToFormState`, `presetToFormState`)을 충실히 따르고 있다. 단, `RecipePickerModal` 내부에서 `isError` 상태를 처리하지 않고, `LogFormPage.test.tsx`에서 새 기능(`RecipePickerSection` 표시 조건, 모달 선택 후 폼 채움)에 대한 컴포넌트 테스트가 추가되지 않은 점이 주요 결함이다.

## 발견 사항

### [High] RecipePickerModal이 API 오류 상태를 처리하지 않는다

- **파일**: `frontend/src/pages/LogFormPage.tsx:240`
- **카테고리**: Quality
- **현재**: `useLogList`에서 `isError`를 destructure하지 않으며, 네트워크 오류나 API 실패 시 모달이 빈 로딩 상태로 멈추거나 빈 목록("아직 brew 기록이 없습니다.")으로 잘못 표시된다.
  ```tsx
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    useLogList({ log_type: 'brew' })
  ```
- **제안**: `isError`와 `error`를 함께 destructure하고, `isLoading` 분기 다음에 오류 상태를 처리한다.
  ```tsx
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading, isError } =
    useLogList({ log_type: 'brew' })

  // ...
  {isLoading ? (
    <p ...>불러오는 중...</p>
  ) : isError ? (
    <p className="py-8 text-center text-sm text-rose-500">목록을 불러오지 못했습니다.</p>
  ) : logs.length === 0 ? (
    <p ...>아직 brew 기록이 없습니다.</p>
  ) : ( ... )}
  ```
- **근거**: `isLoading`이 `false`이고 `data`가 없는 경우(오류 발생 시)에 `logs`가 `[]`가 되어 "아직 brew 기록이 없습니다." 메시지가 노출된다. 실제 데이터가 없는 경우와 오류 상황을 구분할 수 없어 사용자에게 잘못된 정보를 제공한다.

---

### [High] RecipePickerSection/Modal에 대한 컴포넌트 테스트가 없다

- **파일**: `frontend/src/pages/LogFormPage.test.tsx`
- **카테고리**: Quality
- **현재**: `LogFormPage.test.tsx`의 `vi.mock('../hooks/useLogs', ...)` 블록에 `useLogList`가 포함되어 있지 않아 기존 mock이 새 hook 호출을 커버하지 못한다. 또한 plan.md와 tasks.md에 명시된 아래 테스트 케이스가 모두 누락되어 있다:
  - brew 모드에서 "이전 레시피 불러오기" 버튼이 표시되는지
  - cafe 모드에서 버튼이 표시되지 않는지
  - 수정 모드/clone 모드에서 버튼이 표시되지 않는지
  - 모달에서 로그 선택 시 레시피 필드가 채워지는지
- **제안**:
  1. `vi.mock('../hooks/useLogs', ...)` 블록에 `useLogList` mock을 추가한다.
     ```tsx
     vi.mock('../hooks/useLogs', () => ({
       useLog: () => ({ data: undefined, error: null, isError: false, isLoading: false }),
       useCreateLog: () => ({ mutateAsync: vi.fn(), isPending: false, isError: false, error: null }),
       useUpdateLog: () => ({ mutateAsync: vi.fn(), isPending: false, isError: false, error: null }),
       useLogList: () => ({ data: undefined, fetchNextPage: vi.fn(), hasNextPage: false, isFetchingNextPage: false, isLoading: false, isError: false }),
     }))
     ```
  2. brew 타입 선택 후 "이전 레시피 불러오기" 버튼이 표시되는지 검증하는 테스트를 추가한다.
  3. cafe 타입 / 수정 모드 / clone 모드에서 버튼이 표시되지 않는지 검증하는 테스트를 추가한다.
- **근거**: AGENTS.md 규칙("Always write tests when adding or modifying functionality")과 plan.md에 명시된 테스트 전략을 충족하지 못한다. 현재 `useLogList` mock 누락으로 인해 기존 테스트가 우연히 통과하는 상황이다(기존 mock이 `useLogList`를 덮어쓰지 않아 실제 hook이 호출되지만 `QueryClient`가 초기화되어 있어 오류 없이 undefined를 반환).

---

### [Medium] brewMethodLabelMap이 세 곳에서 중복 정의되어 있다

- **파일**: `frontend/src/pages/LogFormPage.tsx:218`, `frontend/src/components/LogCard.tsx:5`, `frontend/src/pages/LogDetailPage.tsx:17`
- **카테고리**: Architecture
- **현재**: 동일한 brew method → 한국어/영어 레이블 매핑 객체가 세 파일에 독립적으로 정의되어 있다. 이번 PR에서 네 번째 인스턴스가 `LogFormPage.tsx`에 추가되었다.
- **제안**: `frontend/src/utils/brewMethod.ts` 파일을 생성하여 단일 정의로 통합한다.
  ```ts
  // frontend/src/utils/brewMethod.ts
  export const BREW_METHOD_LABELS: Record<string, string> = {
    pour_over: 'Pour Over',
    immersion: 'Immersion',
    aeropress: 'AeroPress',
    espresso: 'Espresso',
    moka_pot: 'Moka Pot',
    siphon: 'Siphon',
    cold_brew: 'Cold Brew',
    other: 'Other',
  }
  ```
  이후 `LogCard.tsx`, `LogDetailPage.tsx`, `LogFormPage.tsx`에서 import하여 사용한다. `logFormState.ts`의 `brewMethodOptions`도 이 파일에 함께 옮기는 것이 자연스럽다.
- **근거**: 이미 `LogCard.tsx`와 `LogDetailPage.tsx`에 중복이 존재했는데, 이번 PR에서 세 번째 중복이 추가되었다. brew method가 추가되거나 레이블이 변경될 때 모든 파일을 함께 수정해야 하므로 누락이 발생하기 쉽다. `docs/arch/frontend.md`의 디렉토리 구조에서 `utils/`는 "도메인 로직 유틸리티"를 담는 위치로 명시되어 있다.

---

### [Medium] RecipePickerModal이 항상 마운트되어 `open=false`일 때도 API를 호출한다

- **파일**: `frontend/src/pages/LogFormPage.tsx:240-243`
- **카테고리**: Performance
- **현재**: `RecipePickerSection`이 `RecipePickerModal`을 항상 렌더링하고, `RecipePickerModal` 내부에서 `useLogList`를 조건 없이 호출한 뒤 `if (!open) return null`로 UI만 숨긴다. 따라서 brew 타입 폼을 열면 사용자가 "이전 레시피 불러오기" 버튼을 누르기 전에 API 요청이 시작된다.
- **제안**: `RecipePickerSection`에서 `open` 상태가 `true`일 때만 `RecipePickerModal`을 렌더링하거나, `useLogList`에 `enabled: open` 옵션을 추가한다.

  방법 1 — 조건부 마운트:
  ```tsx
  // RecipePickerSection 내부
  {open && (
    <RecipePickerModal
      open={open}
      onClose={() => setOpen(false)}
      onSelect={(log) => { onSelect(log); setOpen(false) }}
    />
  )}
  ```

  방법 2 — `enabled` 옵션:
  ```tsx
  const { data, ... } = useLogList({ log_type: 'brew' }, { enabled: open })
  ```
  단, `useLogList`가 현재 `enabled` 옵션을 지원하지 않으므로 hook 시그니처 변경이 필요하다.

  방법 1이 더 간단하고 기존 코드를 덜 수정하므로 권장한다.
- **근거**: 이 앱은 single-user POC이므로 즉각적인 성능 문제는 없으나, 불필요한 네트워크 요청과 캐시 오염을 줄이는 것이 올바른 설계다. 모달 open 전 prefetch가 의도된 동작이라면 코드 주석으로 명시해야 한다.

---

### [Medium] RecipePickerModal에 ARIA 속성과 Escape 키 닫기가 없다

- **파일**: `frontend/src/pages/LogFormPage.tsx:247`
- **카테고리**: Quality
- **현재**: 모달 컨테이너 `<div>`에 `role="dialog"`, `aria-modal="true"`, `aria-labelledby`가 없으며, Escape 키로 모달을 닫는 동작이 구현되어 있지 않다.
- **제안**: backlog의 DEBT-7 해결 전이라도 최소한의 접근성 속성과 Escape 키 처리를 추가한다.
  ```tsx
  // 모달 패널 div에 추가
  <div
    role="dialog"
    aria-modal="true"
    aria-labelledby="recipe-picker-title"
    ...
  >
    <h3 id="recipe-picker-title" ...>이전 레시피 불러오기</h3>
  ```
  ```tsx
  // backdrop div에 Escape 키 처리 추가
  <div
    className="fixed inset-0 ..."
    onClick={onClose}
    onKeyDown={(e) => { if (e.key === 'Escape') onClose() }}
  >
  ```
- **근거**: `PresetsPage.tsx`의 `EditModal`에도 이 속성들이 없으며 DEBT-7로 등록되어 있다. 이번 PR에서 새 모달을 추가하면서 동일한 접근성 미비 사항을 그대로 이어받았다. 최소한 `role="dialog"`와 Escape 닫기는 즉시 적용 가능하다.

---

### [Low] `log as BrewLogFull` 타입 단언이 반복 사용된다

- **파일**: `frontend/src/pages/LogFormPage.tsx:280, 284, 287`
- **카테고리**: Quality
- **현재**: `useLogList({ log_type: 'brew' })`로 조회한 결과임에도 불구하고 `log as BrewLogFull` 단언이 세 곳에서 반복된다.
  ```tsx
  onClick={() => onSelect(log as BrewLogFull)}
  // ...
  {(log as BrewLogFull).brew.bean_name}
  // ...
  {brewMethodLabelMap[(log as BrewLogFull).brew.brew_method] ?? ...}
  ```
- **제안**: `flatMap` 결과를 명시적으로 타입 단언하여 반복을 제거한다.
  ```tsx
  const logs = (data?.pages.flatMap((p) => p.items) ?? []) as BrewLogFull[]
  ```
  이후 `log.brew.bean_name`으로 직접 접근할 수 있다.
- **근거**: 동일한 단언을 여러 번 반복하는 것은 불필요한 노이즈이며, 향후 타입이 변경될 때 수정 누락 위험이 있다.

---

### [Low] `recipeToFormState` 테스트에서 fixture가 `cloneToFormState` 테스트와 중복된다

- **파일**: `frontend/src/pages/logFormState.test.ts:240-268`
- **카테고리**: Quality
- **현재**: `recipeToFormState` describe 블록 안에 `brewLog` fixture가 새로 정의되어 있다. `cloneToFormState` 블록(line 140)에 거의 동일한 fixture가 이미 존재하며, 유일한 차이는 `bean_process: null` vs `bean_process: 'washed'`다.
- **제안**: 두 테스트에서 공유할 수 있도록 `brewLog` fixture를 describe 블록 밖 상단으로 추출하거나, 기존 `cloneToFormState`의 `brewLog`를 `recipeToFormState` 테스트에서도 재사용한다. `bean_process` 값의 차이가 의도적이라면 주석으로 이유를 명시한다.
- **근거**: fixture 중복은 유지보수 부담을 높인다. 타입이나 필드가 추가될 때 두 곳을 모두 수정해야 한다.

## 액션 아이템

1. [High] `frontend/src/pages/LogFormPage.tsx`의 `RecipePickerModal`에서 `useLogList` destructure에 `isError`를 추가하고, `isLoading` 분기 다음에 오류 상태 UI(`<p className="py-8 text-center text-sm text-rose-500">목록을 불러오지 못했습니다.</p>`)를 추가한다.

2. [High] `frontend/src/pages/LogFormPage.test.tsx`의 `vi.mock('../hooks/useLogs', ...)` 블록에 `useLogList` mock을 추가하고, brew 모드에서 "이전 레시피 불러오기" 버튼 표시 여부 및 cafe/수정/clone 모드에서 버튼 미표시를 검증하는 테스트 케이스를 추가한다.

3. [Medium] `frontend/src/utils/brewMethod.ts` 파일을 생성하여 `BREW_METHOD_LABELS` 상수를 정의하고, `LogFormPage.tsx`(line 218), `LogCard.tsx`(line 5), `LogDetailPage.tsx`(line 17)의 중복 정의를 제거한 뒤 import로 교체한다.

4. [Medium] `frontend/src/pages/LogFormPage.tsx`의 `RecipePickerSection`에서 `RecipePickerModal`을 `open` 상태가 `true`일 때만 렌더링하도록 변경한다(`{open && <RecipePickerModal ... />}`).

5. [Medium] `frontend/src/pages/LogFormPage.tsx`의 `RecipePickerModal` 모달 패널 `<div>`에 `role="dialog"`, `aria-modal="true"`, `aria-labelledby="recipe-picker-title"` 속성을 추가하고, backdrop `<div>`에 `onKeyDown={(e) => { if (e.key === 'Escape') onClose() }}` 핸들러를 추가한다.

6. [Low] `frontend/src/pages/LogFormPage.tsx` line 245에서 `const logs = (data?.pages.flatMap((p) => p.items) ?? []) as BrewLogFull[]`로 변경하여 이후 `log as BrewLogFull` 단언 세 곳을 제거한다.

7. [Low] `frontend/src/pages/logFormState.test.ts`의 `recipeToFormState` describe 블록 내 `brewLog` fixture를 `cloneToFormState` describe 블록의 `brewLog`와 통합하거나 상단으로 추출한다.
