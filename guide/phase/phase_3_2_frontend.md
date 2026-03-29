# Phase 3-2 Frontend 학습 문서

> 자동완성 컴포넌트 구현을 통해 배운 내용을 정리합니다.
> 주요 주제: TagInput 컴포넌트 설계, blur/mousedown 타이밍 문제, TanStack Query의 조건부 실행, 폼 상태 타입 리팩토링.

---

## 1. 기능 개요

유저가 `tasting_tags`나 `companions` 필드에 입력하면, 과거에 사용한 값 중 일치하는 것을 드롭다운으로 제안한다.

```
입력: "초" 타이핑
  → GET /api/v1/suggestions/tags?q=초
  ← { "suggestions": ["초콜릿", "초록사과"] }
  → 드롭다운에 뱃지 형태로 표시
  → 선택하면 태그 뱃지로 추가
```

구현 목록:
- `src/api/suggestions.ts` — 자동완성 API 함수
- `src/hooks/useSuggestions.ts` — TanStack Query 훅
- `src/components/TagInput.tsx` — 태그 입력 + 드롭다운 + 뱃지 컴포넌트
- `LogFormPage` — `tasting_tags`, `companions` 필드에 `TagInput` 적용
- `logFormState.ts` — 폼 상태 타입을 `string`에서 `string[]`로 변경

---

## 2. 폼 상태 타입 리팩토링 — `string`에서 `string[]`로

### 이전 구조

기존 `LogFormState`는 태그와 동반자를 쉼표 구분 텍스트로 관리했다.

```typescript
interface LogFormState {
  companionsText: string    // "민수, 지연"
  cafe: {
    tastingTagsText: string // "초콜릿, 체리"
  }
}
```

제출 시 `buildLogPayload`에서 이를 배열로 변환했다.

```typescript
companions: splitCommaSeparated(state.companionsText),
// "민수,  지연 " → ["민수", "지연"]
```

### 변경 후 구조

`TagInput` 컴포넌트는 `string[]`를 직접 받고 돌려준다. 이에 맞춰 폼 상태 타입도 `string[]`로 변경했다.

```typescript
interface LogFormState {
  companions: string[]   // ["민수", "지연"]
  cafe: {
    tastingTags: string[] // ["초콜릿", "체리"]
  }
}
```

`buildLogPayload`의 변환 단계가 제거된다.

```typescript
// 변경 전
companions: splitCommaSeparated(state.companionsText),

// 변경 후
companions: state.companions,  // 이미 배열이므로 변환 불필요
```

**왜 이 방향이 더 나은가**: 내부 표현과 API 요청 형식이 같아질수록 변환 코드가 줄어든다. 변환이 필요한 지점이 줄수록 버그가 생길 여지도 줄어든다. 사용자가 직접 타이핑한 텍스트를 분리하는 게 아니라 UI에서 이미 확정된 태그 목록을 관리하는 것이므로, `string[]`이 의미론적으로도 더 정확하다.

**Spring에서의 대응**: DTO를 설계할 때 `String tags = "a,b,c"`로 받아서 서비스 레이어에서 분리하는 패턴은 흔하다. 하지만 `List<String> tags`로 직접 받으면 서비스가 더 단순해지는 것과 같은 원리다.

---

## 3. TanStack Query — 조건부 쿼리 실행

### enabled 옵션

자동완성 API는 유저가 무언가를 입력했을 때만 호출해야 한다. TanStack Query는 `enabled` 옵션으로 쿼리 실행 조건을 선언할 수 있다.

```typescript
function useSuggestions(type: 'tags' | 'companions', q: string) {
  return useQuery({
    queryKey: ['suggestions', type, q],
    queryFn: () => getSuggestions(type, q),
    staleTime: 30_000,
    enabled: q.length > 0,  // 입력이 비어있으면 실행하지 않는다
  })
}
```

`enabled: false`이면 컴포넌트가 마운트되어도 `queryFn`을 호출하지 않는다. `useLog(id)`에서 `id`가 없을 때 쿼리를 건너뛰는 것과 같은 패턴이다.

**Spring에서의 대응**: 조건부 캐시 조회 같은 개념이다. Spring Cache에서 `@Cacheable(condition = "#q != null && !#q.isEmpty()")`와 유사하다. 다만 TanStack Query는 클라이언트 상태 관리이므로, 조건이 false일 때는 캐시 접근 자체를 생략한다.

### queryKey 설계

`queryKey: ['suggestions', type, q]`는 `type`과 `q`가 다르면 다른 캐시 엔트리로 취급된다. "초"를 입력했다가 "초콜"로 확장하면 두 쿼리 결과가 각각 캐시된다. 뒤로 지우면 이전 결과를 즉시 보여준다.

```
q="초"   → queryKey: ['suggestions', 'tags', '초']   → 캐시에 저장
q="초콜" → queryKey: ['suggestions', 'tags', '초콜'] → 새 요청
q="초"   → queryKey: ['suggestions', 'tags', '초']   → 캐시 히트 (재요청 없음)
```

### staleTime

`staleTime: 30_000`(30초)으로 설정했다. 자동완성 데이터는 유저의 새 기록이 저장되기 전까지 변하지 않으므로, 짧은 시간 동안 캐시를 재사용해도 문제없다. 0으로 설정하면 동일한 `q`를 입력할 때마다 API를 재호출한다.

---

## 4. TagInput 컴포넌트 설계

### 인터페이스

```typescript
interface TagInputProps {
  value: string[]
  onChange: (tags: string[]) => void
  suggestions?: string[]
  placeholder?: string
  onQueryChange?: (q: string) => void
}
```

`value`와 `onChange`로 제어되는 controlled component다. 내부 상태는 입력 중인 텍스트(`inputValue`)와 드롭다운 열림 여부(`open`)뿐이다. 태그 목록은 부모(`LogFormState`)가 소유한다.

**왜 controlled인가**: 폼 제출 시 `LogFormState`에서 직접 태그 배열을 읽어야 하기 때문이다. 태그 목록을 `TagInput` 내부 상태로 두면 부모가 이를 읽기 위해 `ref`나 콜백이 필요해진다. React에서 폼 데이터는 상위 컴포넌트나 상태 관리 계층이 소유하는 것이 기본 패턴이다.

### 키보드 UX

```typescript
function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
  if (e.key === 'Enter' || e.key === ',') {
    e.preventDefault()
    addTag(inputValue)         // 엔터 또는 쉼표로 태그 확정
  } else if (e.key === 'Backspace' && inputValue === '') {
    removeTag(value.length - 1) // 입력 비었을 때 Backspace로 마지막 태그 삭제
  } else if (e.key === 'Escape') {
    setOpen(false)              // 드롭다운 닫기
  }
}
```

`Enter` 키에 `e.preventDefault()`를 호출하는 이유: `TagInput`이 `<form>` 안에 있으므로 `Enter` 기본 동작은 폼 제출이다. 태그 추가만 처리하고 폼 제출은 막아야 한다.

---

## 5. blur/mousedown 타이밍 문제와 해결책

자동완성 컴포넌트를 만들 때 가장 흔히 마주치는 문제다.

### 문제 상황

드롭다운 항목을 클릭하면 다음 순서로 이벤트가 발생한다.

```
1. mousedown (항목 위에서 마우스 버튼 누름)
2. blur      (input이 포커스를 잃음)
3. mouseup
4. click
```

`onBlur`에서 드롭다운을 닫으면 `blur`가 `click`보다 먼저 발생하므로, 드롭다운이 닫힌 후 `click`이 발생한다. 이미 사라진 요소의 `click`은 무시된다. 결과적으로 항목을 클릭해도 아무 일도 일어나지 않는다.

```
blur → 드롭다운 닫힘 → click 발생 → 이미 없는 요소이므로 무시
```

### 해결: mousedown에서 preventDefault

```typescript
function handleOptionMouseDown(e: React.MouseEvent, tag: string) {
  e.preventDefault()  // blur 발생을 막는다
  addTag(tag)
  inputRef.current?.focus()
}
```

`mousedown`에서 `preventDefault()`를 호출하면 브라우저가 포커스 이동을 처리하지 않는다. input은 blur되지 않고, 드롭다운도 닫히지 않은 채 `addTag`가 호출된다.

```
mousedown + preventDefault → blur 발생 안 함 → addTag 호출 → 드롭다운 닫힘
```

이 패턴은 `<select>` 대체 컴포넌트, 날짜 선택기, 색상 선택기 등 "입력 외부 영역 클릭 시 닫힘" UX가 필요한 모든 컴포넌트에서 공통으로 사용된다.

---

## 6. 접근성 — useId와 aria 속성

스크린 리더를 위해 input과 드롭다운 리스트를 연결해야 한다. ARIA의 combobox 패턴에서는 `aria-controls`로 연결한다.

```typescript
const listboxId = useId()  // React가 고유 ID를 생성한다

<input
  aria-autocomplete="list"
  aria-controls={open ? listboxId : undefined}
/>

<ul id={listboxId} role="listbox">
  <li role="option" aria-selected={false}>...</li>
</ul>
```

`useId()`는 React 18에서 추가된 훅이다. 같은 컴포넌트가 여러 번 렌더링될 때 각 인스턴스마다 고유한 ID를 생성한다. `Math.random()`이나 전역 카운터로 ID를 만들면 서버 사이드 렌더링에서 hydration 불일치가 발생할 수 있는데, `useId()`는 이를 방지한다.

**왜 aria-controls가 필요한가**: `input`과 `ul`은 DOM상 인접해 있지만 스크린 리더는 이 관계를 알지 못한다. `aria-controls`가 있어야 스크린 리더가 "이 input의 제안 목록은 저 리스트다"라는 것을 이해하고 적절히 안내할 수 있다.

---

## 7. onQueryChange 패턴 — 쿼리 상태를 부모로 올리기

`TagInput`은 API를 직접 호출하지 않는다. 대신 `onQueryChange` 콜백으로 현재 입력값을 부모에게 알리고, 부모가 훅을 통해 제안 목록을 가져온다.

```typescript
// LogFormPage (부모)
const [tagsQuery, setTagsQuery] = useState('')
const { data: tagSuggestions = [] } = useTagSuggestions(tagsQuery)

<TagInput
  value={form.cafe.tastingTags}
  onChange={(tags) => setForm(...)}
  suggestions={tagSuggestions}         // 부모가 제안 목록을 주입
  onQueryChange={setTagsQuery}         // 입력값 변경을 부모에게 알림
/>
```

**왜 TagInput이 직접 훅을 호출하지 않는가**:

만약 `TagInput`이 `useTagSuggestions`를 직접 호출한다면, companions 필드에 쓸 때는 `useCompanionSuggestions`로 교체해야 한다. 즉, 용도에 따라 내부 로직이 달라진다.

`onQueryChange` 패턴을 쓰면 `TagInput`은 어떤 종류의 제안인지 몰라도 된다. 부모가 훅 종류를 결정하고, `TagInput`은 순수하게 UI만 담당한다. 이를 **관심사 분리(separation of concerns)** 라고 한다.

**Spring에서의 대응**: `TagInput`은 View에 해당한다. 어떤 서비스를 호출할지는 Controller(부모 컴포넌트)가 결정한다. View가 직접 Service를 호출하지 않는 것과 같은 원리다.

---

## 8. YAML 파싱 오류 — description의 콜론

`openapi.yml`의 `recorded_at` 설명에 한국어 `예:` 텍스트가 있었다.

```yaml
# 오류 발생
description: RFC3339 datetime (예: 2026-03-29T10:00:00Z 또는 ...)
```

YAML 파서는 `: ` (콜론 + 공백) 패턴을 key-value 구분자로 인식한다. `예:` 뒤에 공백이 없더라도 파서가 혼란을 일으켰다.

```yaml
# 수정: 따옴표로 감싸기
description: "RFC3339 datetime (예: 2026-03-29T10:00:00Z 또는 ...)"
```

스칼라 문자열 값에 `:`, `#`, `{`, `}`, `[`, `]` 등 YAML 메타 문자가 포함될 경우 따옴표로 감싸야 한다. 이 오류로 `npm run generate`가 실패하고 `SuggestionsResponse` 타입이 `schema.ts`에 추가되지 않았다.

**수정 흐름**: `openapi.yml` 수정 → `npm run generate` 재실행 → `schema.ts`에 `SuggestionsResponse` 타입 생성 확인 → 프론트 구현 시작.

이 프로젝트의 워크플로우 원칙대로, 타입을 손으로 쓰지 않고 openapi.yml → 생성 순서를 지켰다.

---

## 9. 이미 추가된 태그 필터링

드롭다운에서 이미 추가된 태그는 제외해야 한다. 같은 태그를 두 번 추가하는 것은 의미가 없기 때문이다.

```typescript
// TagInput 내부
const filteredSuggestions = suggestions.filter((s) => !value.includes(s))
```

`addTag` 함수도 중복 추가를 막는다.

```typescript
function addTag(tag: string) {
  const trimmed = tag.trim()
  if (!trimmed || value.includes(trimmed)) return  // 이미 있으면 무시
  onChange([...value, trimmed])
  ...
}
```

두 곳에서 중복을 막는 이유: `filteredSuggestions`는 드롭다운 UI에서만 필터링한다. 유저가 드롭다운을 쓰지 않고 직접 타이핑하는 경우도 있으므로, `addTag` 내부에서도 중복 체크를 한다.

---

## 10. Phase 3 완료 기준 정리

| 항목 | 구현 방식 |
|------|-----------|
| 태그 입력 시 이전 태그 자동완성 제안 | `useTagSuggestions` + `TagInput` |
| 동반자 입력 시 이전 이름 자동완성 제안 | `useCompanionSuggestions` + `TagInput` |
| 중복 태그 방지 | `filteredSuggestions` + `addTag` 내부 검사 |
| 키보드 UX (Enter/,/Backspace/Escape) | `handleKeyDown` |
| 빈 입력 시 API 호출 안 함 | `enabled: q.length > 0` |
| 태그 삭제 | 뱃지의 × 버튼 → `removeTag` |
