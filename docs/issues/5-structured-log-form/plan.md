# Issue #5 — 단계적 입력: 로그 작성 폼 필드 구조화

## 목표

`LogFormPage.tsx`의 Cafe/Brew 필드 섹션을 "필수 영역"과 "선택 영역"으로 분리한다. 선택 영역은 "더 기록하기" 토글로 접히며, 수정 모드에서 기존 값이 있으면 자동으로 펼쳐진다.

백엔드 변경 없음. `docs/openapi.yml` 변경 없음. 프론트엔드 전용 작업.

---

## 영역 분류

필수/선택 분류 기준은 `docs/spec.md` Section 6.2에 정의되어 있다.

**공통 필수:** `recorded_at` (항상 노출)
**공통 선택:** `companions`, `memo` (토글 영역으로 이동)

**Cafe 필수:** `cafe_name`, `coffee_name`, `rating`
**Cafe 선택:** `location`, `bean_origin`, `bean_process`, `roast_level`, `tasting_tags`, `tasting_note`, `impressions`

**Brew 필수:** `bean_name`, `brew_method`, `rating`
**Brew 선택:** `bean_origin`, `bean_process`, `roast_level`, `roast_date`, `brew_device`, `coffee_amount_g`, `water_amount_ml`, `water_temp_c`, `brew_time_sec`, `grind_size`, `brew_steps`, `tasting_tags`, `tasting_note`, `impressions`

> `rating`은 API에서 nullable이지만, 이슈 요구사항에 따라 필수 영역에 배치한다 (항상 보이는 필드, 입력 필수 아님).

---

## 설계

### 접기/펼치기 상태 관리

`LogFormPage.tsx`에 `expanded` boolean state를 추가한다.

```
const [expanded, setExpanded] = useState(false)
```

- 생성 모드: `expanded = false` (기본 접힌 상태)
- 수정 모드: 선택 영역 필드에 값이 하나라도 있으면 `expanded = true`

수정 모드의 자동 펼침 판단은 기존 `useEffect` (hydrate 단계) 이후에 실행해야 한다. `log` 데이터가 hydrate된 시점에 선택 필드 값을 검사하여 `expanded`를 설정한다.

### 선택 영역 값 보존 규칙

접힌 상태에서도 선택 필드의 폼 상태는 유지된다. 토글은 순수하게 UI 가시성만 제어하며, `LogFormState`나 `buildLogPayload`는 수정하지 않는다. 접힌 상태의 필드는 DOM에서 숨기되 (`expanded && ...` 조건부 렌더링), 상태 객체의 값은 그대로 남아 있으므로 submit 시 함께 전송된다.

### 컴포넌트 구조 변경

현재 구조:
```
<form>
  <LogTypeSection />
  <CommonFieldsSection />        ← recorded_at, companions, memo
  <CafeFieldsSection /> 또는 <BrewFieldsSection />
</form>
```

변경 후 구조:
```
<form>
  <LogTypeSection />
  <필수 영역 Section>
    recorded_at                   ← CommonFieldsSection에서 이동
    cafe_name, coffee_name, rating  (또는 bean_name, brew_method, rating)
  </필수 영역 Section>
  <"더 기록하기" 토글 버튼>
  {expanded && (
    <선택 영역 Section>
      companions, memo            ← CommonFieldsSection에서 이동
      나머지 cafe/brew 필드
    </선택 영역 Section>
  )}
</form>
```

`CommonFieldsSection`은 해체한다. `recorded_at`은 필수 영역으로, `companions`와 `memo`는 선택 영역으로 각각 인라인 배치한다. 별도 컴포넌트로 분리하지 않고 `CafeFieldsSection`/`BrewFieldsSection` 내부에서 필수/선택을 구분하여 렌더링한다.

### CafeFieldsSection 변경

props에 `expanded`, `onToggle` 추가. 내부 구조:

1. 필수 영역 (Section 컴포넌트): `cafe_name`, `coffee_name`, `rating`
2. 토글 버튼: "더 기록하기" / "접기"
3. 선택 영역 (expanded일 때만 렌더링, Section 컴포넌트): `location`, `bean_origin`, `bean_process`, `roast_level`, `tasting_tags`, `tasting_note`, `impressions`, `companions`, `memo`

`companions`와 `memo`를 렌더링하려면 `form.companions`, `form.memo`에 대한 상태 업데이트가 필요하다. 이미 `form`과 `setForm`을 props로 받고 있으므로 추가 props 없이 접근 가능.

### BrewFieldsSection 변경

CafeFieldsSection과 동일한 패턴.

1. 필수 영역: `bean_name`, `brew_method`, `rating`
2. 토글 버튼
3. 선택 영역: `brew_device`, `grind_size`, `bean_origin`, `bean_process`, `roast_level`, `roast_date`, Recipe 블록 (coffee/water/temp/time), `tasting_tags`, `tasting_note`, `brew_steps`, `impressions`, `companions`, `memo`

### 토글 버튼 디자인

기존 코드베이스의 스타일 패턴을 따른다. `rounded-full border` 스타일의 버튼으로, 접힌 상태에서는 "더 기록하기", 펼친 상태에서는 "접기" 텍스트를 표시한다. 섹션 사이에 독립적으로 배치.

### 수정 모드 자동 펼침 판단

`logFormState.ts`에 선택 필드 값 존재 여부를 검사하는 헬퍼 함수를 추가한다.

```typescript
export function hasOptionalValues(state: LogFormState): boolean
```

Cafe: `location`, `beanOrigin`, `beanProcess`, `roastLevel`, `tastingTags.length`, `tastingNote`, `impressions` 중 하나라도 비어있지 않으면 `true`.
Brew: `beanOrigin`, `beanProcess`, `roastLevel`, `roastDate`, `brewDevice`, `coffeeAmountG`, `waterAmountMl`, `waterTempC`, `brewTimeSec`, `grindSize`, `tastingTags.length`, `tastingNote`, `brewSteps`(빈 문자열 제외), `impressions` 중 하나라도 비어있지 않으면 `true`.
공통: `companions.length > 0` 또는 `memo`가 비어있지 않으면 `true`.

이 함수는 hydrate 이후 한 번만 호출하여 `expanded` 초기값을 결정한다.

---

## 수정하지 않는 것

- `LogFormState` 인터페이스 — 필드 추가/삭제/이름 변경 없음
- `buildLogPayload`, `logToFormState`, `createEmptyFormState` — 로직 변경 없음
- `Section`, `Field`, `inputClassName`, `textareaClassName` — 공통 UI 유틸 그대로 사용
- `RatingInput`, `TagInput` — 그대로 사용
- 백엔드, OpenAPI 스키마

---

## 테스트 전략

- `logFormState.test.ts`: `hasOptionalValues` 헬퍼에 대한 단위 테스트 추가 (빈 상태 → false, 값 있는 상태 → true)
- 기존 테스트(`createEmptyFormState`, `buildLogPayload`, `logToFormState`): 변경 없이 통과해야 함
- `npm test`로 전체 프론트엔드 테스트 실행하여 기존 테스트 깨짐 없음을 확인
