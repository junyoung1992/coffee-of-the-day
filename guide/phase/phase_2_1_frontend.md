# Phase 2-1 Frontend — 브루 폼 UI 고도화

> 대상 독자: Java/Spring 경험은 있지만 React UI 설계 경험은 적은 개발자.
> 이번 문서는 "같은 데이터를 어떤 컴포넌트로 표현하느냐가 UX를 얼마나 바꾸는가"에 집중합니다.

---

## 무엇을 만들었나

브루 폼(`BrewFieldsSection`)을 세 방향으로 개선했습니다.

| 개선 항목 | 변경 전 | 변경 후 |
|-----------|---------|---------|
| `brew_method` 선택 | `<select>` 드롭다운 | 버튼 그룹 (2열/4열 그리드) |
| `brew_device` 위치 | `grind_size`와 같은 행 | `brew_method` 버튼 그룹 바로 아래 |
| 레시피 비율 표시 | 별도 `<div>` 박스 (입력 후 아래에 노출) | 원두량 ↔ 비율 ↔ 물량 인라인 3열 카드 |

변경된 파일은 두 곳입니다.

- `src/pages/logFormState.ts` — `brewMethodOptions`에 `description` 필드 추가
- `src/pages/LogFormPage.tsx` — `BrewFieldsSection` 개선

---

## `<select>`를 왜 버튼 그룹으로 바꿨나

### 클릭 수 차이

`<select>`는 "열기 → 선택" 두 단계가 필요합니다. 버튼 그룹은 한 번 클릭으로 선택이 끝납니다.
선택지가 8개 이하이고 동시에 모두 보여줄 수 있다면, 버튼 그룹이 정보 밀도와 조작성 모두에서 유리합니다.

### Spring 관점으로 보기

백엔드에서 `@RequestParam` 검증을 `@EnumValidator`로 거는 것과 달리, 프론트에서 버튼 그룹은 "잘못된 값을 입력할 가능성 자체를 없앱니다". 유효한 선택지만 버튼으로 존재하므로, 사용자가 `brew_method`에 오탈자를 넣는 시나리오가 구조적으로 불가능합니다.

### 구현

버튼 그룹은 `brewMethodOptions` 배열을 `map()`으로 렌더링합니다.

```tsx
// logFormState.ts — 선택지 정의 (label + description + value)
export const brewMethodOptions = [
  { label: 'Pour Over', description: '핸드드립 계열', value: 'pour_over' },
  { label: 'Immersion', description: '침지 계열',     value: 'immersion' },
  // ...
] as const
```

```tsx
// LogFormPage.tsx — 버튼 그룹 렌더링
<div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
  {brewMethodOptions.map((option) => {
    const selected = form.brew.brewMethod === option.value
    return (
      <button
        key={option.value}
        type="button"
        onClick={() => setForm((prev) => updateBrewField(prev, 'brewMethod', option.value))}
        className={selected ? '...amber selected style...' : '...default style...'}
      >
        <p>{option.label}</p>
        <p>{option.description}</p>
      </button>
    )
  })}
</div>
```

두 가지 주목할 점이 있습니다.

1. **`type="button"` 명시**: `<form>` 안의 `<button>` 기본값은 `type="submit"`입니다. 명시하지 않으면 방식 선택 버튼 클릭 시 폼 제출이 트리거됩니다.

2. **`as const` 타입**: 배열에 `as const`를 붙이면 TypeScript가 `value`를 `string`이 아닌 `'pour_over' | 'immersion' | ...`으로 좁혀 추론합니다. 이 덕분에 `brewMethod` 필드 타입(`BrewMethodValue`)과 자동으로 호환됩니다.

---

## brew_device를 brew_method 옆에 놓은 이유

`brew_device`는 "선택한 추출 방식을 구체화하는 도구명"입니다.
예를 들어 방식이 `pour_over`라면 도구는 `Origami`, `V60`, `Chemex` 중 하나일 것입니다.

이전 레이아웃은 `brew_device`를 `grind_size`와 같은 행에 배치했습니다. 두 필드가 시각적으로 동등해 보이지만, 의미 관계는 다릅니다.

```
이전: [brew_method select] [bean_origin]
      [brew_device        ] [grind_size ]

이후: [     brew_method 버튼 그룹 (전체 너비)    ]
      [brew_device        ] [grind_size ]
```

`brew_method`의 시각적 무게(8개 버튼, 전체 너비)가 커진 덕분에, 바로 아래에 오는 `brew_device`가 자연스럽게 "이 방식의 구체적 도구"로 읽힙니다. 코드가 아닌 배치로 관계를 설명하는 방식입니다.

---

## 레시피 비율 인라인 표시 — 파생 상태(derived state)

### 기존 방식의 문제

기존에는 원두량과 물량을 입력하면 비율이 아래에 별도 박스로 표시됐습니다. 두 입력 필드 사이의 시각적 거리가 있어서, "이 두 값이 연동된다"는 것이 직관적이지 않았습니다.

### 인라인 3열 레이아웃

`grid-cols-[1fr_auto_1fr]` 3열 그리드로 가운데에 비율을 끼웁니다.

```tsx
<div className="grid grid-cols-[1fr_auto_1fr] items-end gap-3">
  <Field label="Coffee (g)">
    <input type="number" ... />
  </Field>

  {/* 중앙 컬럼: 비율 실시간 표시 */}
  <div className="flex flex-col items-center gap-1 pb-1">
    <span className="text-xs text-stone-400">ratio</span>
    <span className={`text-base font-bold tabular-nums ${ratio ? 'text-amber-900' : 'text-stone-300'}`}>
      {ratio ? `1 : ${ratio}` : '1 : —'}
    </span>
  </div>

  <Field label="Water (ml)">
    <input type="number" ... />
  </Field>
</div>
```

`grid-cols-[1fr_auto_1fr]`은 Tailwind의 임의값(arbitrary value) 문법입니다. 중앙 컬럼은 콘텐츠 너비만큼, 양쪽 컬럼은 나머지를 균등하게 나눠 가집니다. "두 요소 사이에 고정 크기 요소"를 배치할 때 유용한 패턴입니다.

### 파생 상태는 `useState`에 저장하지 않는다

비율은 원두량과 물량으로부터 계산됩니다. 이 값을 별도 `useState`로 관리하면 세 개의 상태가 항상 동기화된 상태여야 한다는 책임이 생깁니다. 동기화 버그의 전형적인 원인입니다.

대신 `useMemo`로 파생합니다.

```tsx
const ratio = useMemo(() => {
  const coffee = Number(form.brew.coffeeAmountG)
  const water  = Number(form.brew.waterAmountMl)
  if (!Number.isFinite(coffee) || !Number.isFinite(water) || coffee <= 0 || water <= 0) {
    return null
  }
  return (water / coffee).toFixed(1)
}, [form.brew.coffeeAmountG, form.brew.waterAmountMl])
```

`useMemo`는 의존성 배열의 값이 바뀔 때만 재계산합니다. Spring 관점으로 보면 "캐싱된 getter"에 해당합니다. 원본 데이터가 변하기 전까지 이전 결과를 재활용합니다.

> 규칙: 다른 상태로부터 계산될 수 있는 값은 상태로 저장하지 않는다. 이 원칙을 지키면 상태 동기화 버그 전체를 구조적으로 예방할 수 있습니다.

### `tabular-nums` — 숫자 표시 안정성

```tsx
<span className="... tabular-nums">1 : 15.2</span>
```

`tabular-nums`(CSS `font-variant-numeric: tabular-nums`)는 숫자 글리프를 고정 너비로 렌더링합니다. `1:9.0`에서 `1:10.0`으로 바뀔 때 문자 너비가 달라지면 주변 레이아웃이 흔들릴 수 있습니다. `tabular-nums`는 이 미세한 레이아웃 이동(CLS, Cumulative Layout Shift)을 방지합니다.

---

## 레이아웃 계층 요약

개선 후 `BrewFieldsSection`의 정보 계층은 아래와 같습니다.

```
브루 정보 섹션
├── 원두 이름 (전체 너비, 필수)
├── 추출 방식 버튼 그룹 (전체 너비, 필수)
├── Brew device | Grind size
├── Bean origin | Bean process
├── Roast level | Roast date
├── Recipe 서브 카드
│   ├── [Coffee g] — [1 : ratio] — [Water ml]
│   └── Water temp | Brew time
├── Tasting tags (전체 너비)
├── Tasting note (전체 너비)
├── Brew steps (동적 입력)
├── Impressions (전체 너비)
└── Rating (전체 너비)
```

"필수 + 식별 정보"가 상단에, "레시피 수치"가 하나의 카드로 묶여 중간에, "테이스팅·인상"이 하단에 위치합니다. 사용자가 폼을 위에서 아래로 내려가면서 자연스러운 순서로 채울 수 있습니다.

---

## 다음 단계

Phase 2-2에서는 홈 화면 필터를 구현합니다.

- `log_type` 탭 필터 (전체 / 카페 / 브루)
- 날짜 범위 필터
- 필터 상태를 URL 쿼리 파라미터에 반영
- TanStack Query 캐시 키와 필터 상태 연결
