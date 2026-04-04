# 코드 리뷰: #5 단계적 입력 + #6 최근 기록 복제

## Issue #5: 단계적 입력 — 로그 작성 폼 필드 구조화

전반적으로 요구사항을 잘 충족하고 있다.

### 발견 사항

| 심각도 | 파일 | 내용 |
|--------|------|------|
| **Medium** | `LogFormPage.tsx` | `Recorded at` 필드가 `CafeFieldsSection`과 `BrewFieldsSection`에 복사-붙여넣기로 중복 존재 — 공통 컴포넌트 추출 권장 |
| **Low** | `LogFormPage.tsx` | `ToggleButton`에 `aria-expanded` 속성 누락 — 접근성 개선 필요 |
| **Low** | `LogFormPage.tsx` | 선택 영역을 접었다 펼치면 `companionQuery` 등 내부 state가 리셋됨 (unmount/remount) |
| **Low** | `logFormState.test.ts` | `memo`에 값을 넣었을 때 `hasOptionalValues`가 `true`를 반환하는 테스트 누락 |
| **Low** | `logFormState.test.ts` | cafe optional 필드 중 `tastingTags`만 테스트 — 다른 필드도 최소 1개 커버 권장 |
| **Info** | `LogFormPage.tsx` | 토글을 접어도 선택 영역 입력값이 유지되는 동작이 의도적인지 주석 명시 권장 |

### 상세

#### `LogFormPage.tsx` — `Recorded at` 중복 (Medium)

`CafeFieldsSection`과 `BrewFieldsSection`에 완전히 동일한 `Recorded at` 입력 필드가 복사-붙여넣기로 존재한다. `OptionalCommonFields`처럼 공통 컴포넌트로 추출하거나 별도 컴포넌트로 분리하는 것이 바람직하다. 한쪽만 수정하고 다른 쪽을 빠뜨리는 실수가 발생할 수 있다.

#### `LogFormPage.tsx` — `aria-expanded` 누락 (Low)

`ToggleButton`은 접기/펼치기를 제어하는 버튼이지만 `aria-expanded` 속성이 없다. Screen reader 사용자를 위해 추가 권장:

```tsx
<button
  type="button"
  onClick={onToggle}
  aria-expanded={expanded}
  ...
>
```

#### `LogFormPage.tsx` — 선택 영역 unmount 시 query state 리셋 (Low)

선택 영역이 닫혔다가 다시 열리면 `OptionalCommonFields`가 unmount/remount되면서 `companionQuery`, `tagsQuery` state가 리셋된다. 실질적 영향은 미미하지만(사용자가 입력 중간에 토글을 접을 가능성이 낮으므로) 인지해 둘 필요는 있다.

#### `logFormState.test.ts` — 테스트 커버리지 보강 (Low)

- `hasOptionalValues`에서 `memo`에 값을 넣었을 때 `true`를 반환하는 테스트가 없다. `companions`만 테스트되어 있다.
- cafe 쪽은 `tastingTags`만 테스트했다. brew 쪽은 `coffeeAmountG`와 `brewSteps` 두 가지를 테스트한 것과 대비된다.

---

## Issue #6: 최근 기록 복제

기능적 버그 없이 깔끔하게 구현되어 있다. 테스트도 unit/integration/E2E 세 레벨 모두 갖추었다.

### 발견 사항

| 심각도 | 파일 | 내용 |
|--------|------|------|
| **Medium** | `LogCard.tsx` | `<Link>` 내부에 `<button>` 중첩 — HTML 표준 위반으로 접근성 이슈 |
| **Medium** | `logFormState.ts` | `cloneToFormState()`에서 `logToFormState()` 반환 중첩 객체를 직접 mutation — spread로 얕은 복사 권장 |
| **Low** | `Layout.tsx` | 헤더에서 글로벌 "New Log" 링크 제거로 홈 외 페이지에서 신규 기록 진입점이 사라짐 — 의도된 결정인지 확인 필요 |
| **Low** | `logFormState.test.ts` | `tasting_note` 복제 여부 assertion 누락 |
| **Low** | `LogFormPage.test.tsx` | rating 리셋 여부 검증 테스트 없음 |
| **Low** | 테스트 공통 | `cafeLog`, `brewLog` fixture가 두 테스트 파일에 중복 — 공유 fixture 추출 권장 |
| **Info** | E2E 테스트 | 주석에 "다시 쓰기"라 되어 있지만 실제 버튼 텍스트는 "복제" — 주석 수정 필요 |
| **Info** | `LogFormPage.tsx` | `LogTypeSection`의 `isEditMode` prop이 실제로는 "변경 불가" 의미 — `disabled` 또는 `locked`로 rename 고려 |

### 상세

#### `LogCard.tsx` — `<Link>` 내부에 `<button>` 중첩 (Medium)

```tsx
<Link to={`/logs/${log.id}`} ...>
  ...
  <button type="button" onClick={(e) => { e.preventDefault(); e.stopPropagation(); ... }}>
    복제
  </button>
  ...
</Link>
```

HTML 표준상 `<a>` 안에 `<button>`을 넣는 것은 유효하지 않다. `e.preventDefault()`와 `e.stopPropagation()`으로 동작은 막고 있지만:
- 스크린 리더에서 혼란을 줄 수 있다
- 일부 브라우저에서 키보드 탐색 시 예상치 못한 동작이 발생할 수 있다

카드 전체를 `<div>`로 만들고 제목 부분만 별도 `<Link>`로 처리하거나, 복제 버튼을 카드 밖으로 빼는 구조 변경을 권장한다. 기존 카드 구조에 대한 리팩터링이므로 별도 이슈로 관리하는 것이 적절하다.

#### `logFormState.ts` — 중첩 객체 직접 mutation (Medium)

```typescript
export function cloneToFormState(log: CoffeeLogFull, now = new Date()): LogFormState {
  const state = logToFormState(log)
  state.recordedAt = toDateTimeLocal(now)  // 직접 mutation
  state.companions = []
  state.memo = ''
  ...
}
```

`logToFormState()`가 현재는 새 객체를 만들어 반환하므로 실질적 버그는 아니지만, `state.cafe.rating = ''` 같은 코드는 중첩 객체를 직접 변경하고 있어 향후 `logToFormState()` 내부에서 객체 재사용이 발생하면 의도치 않은 부작용이 생길 수 있다. 방어적으로 spread를 사용하는 것을 권장:

```typescript
// 개선 예시
if (state.logType === 'cafe') {
  state.cafe = { ...state.cafe, rating: '', impressions: '' }
}
```

#### `Layout.tsx` — 글로벌 "New Log" 제거 (Low)

헤더에서 "New Log" 링크를 제거하고 HomePage의 action 영역으로 "기록 추가" 버튼을 이동했다. 로그 상세 페이지나 폼 페이지 등 다른 곳에서는 "기록 추가" 버튼에 접근할 수 없게 된다. 의도된 결정인지 확인 필요.

#### 테스트 커버리지 보강 (Low)

- `tasting_note`가 복제되는지 명시적으로 assert하는 테스트 케이스가 없다. `cafe.tastingNote`와 `brew.tastingNote` 값이 유지되는지 확인하는 assertion 추가 권장.
- rating이 리셋되었는지 확인하는 integration test가 없다. RatingInput이 커스텀 컴포넌트라 직접 검증이 어려울 수 있지만, 가능하다면 추가 권장.
- `cafeLog`, `brewLog` fixture가 `logFormState.test.ts`와 `LogFormPage.test.tsx`에 완전히 중복된다. 공유 fixture 파일로 추출하면 유지보수가 편해진다.

#### E2E 주석 불일치 (Info)

주석에 "상세 화면에서 '다시 쓰기' 클릭"이라고 되어 있지만 실제 버튼 텍스트는 `'복제'`이다. 주석 수정 필요.

#### `LogTypeSection` prop 이름 (Info)

Clone 모드에서 log type 변경을 막기 위해 `isEditMode`를 전달하지만, 이 prop 이름이 실제로는 "수정 불가" 여부를 의미한다. `disabled` 또는 `locked`로 변경하면 의미가 더 명확해진다.

---

## 두 이슈 공통 — 개선 권장 사항

1. **`<Link>` 안에 `<button>` 중첩 문제**: 카드 구조 리팩터링이 필요하므로 별도 이슈로 관리 권장.
2. **`cloneToFormState()` 직접 mutation**: 현재 동작에는 문제없지만, `logToFormState()` 변경 시 side effect가 생길 수 있어 방어적 복사 추천.
3. **테스트 fixture 중복**: 현재 규모에서는 관리 가능하지만, fixture 변경 시 양쪽 동기화 비용이 있다.
