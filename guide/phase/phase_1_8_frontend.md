# Phase 1-8 Frontend — 페이지 및 컴포넌트 연결

> 대상 독자: Java/Spring 경험은 있지만 React UI 조합 경험은 적은 개발자.
> 이번 문서는 "데이터 계층이 준비된 뒤 실제 화면을 어떻게 조립했는가"에 집중합니다.

---

## 무엇을 만들었나

Phase 1-7에서 준비한 타입, API 함수, TanStack Query 훅을 실제 화면에 연결했습니다.

| 파일 | 역할 |
|------|------|
| `src/components/Layout.tsx` | 공통 헤더와 페이지 프레임 |
| `src/components/LogCard.tsx` | 홈 목록 카드. `cafe` / `brew` 분기 렌더링 |
| `src/components/RatingDisplay.tsx` | 0.5 단위 별점 표시 |
| `src/components/RatingInput.tsx` | 0.5 단위 별점 입력 |
| `src/pages/HomePage.tsx` | 무한 스크롤 목록 화면 |
| `src/pages/LogDetailPage.tsx` | 상세 조회 + 삭제 |
| `src/pages/LogFormPage.tsx` | 생성/수정 통합 폼 |
| `src/pages/logFormState.ts` | 응답 ↔ 폼 상태 ↔ API 요청 본문 변환 |
| `src/pages/logFormState.test.ts` | 폼 상태 변환 테스트 |
| `src/components/*.test.tsx` | 카드/별점 입력 동작 테스트 |

---

## 왜 컴포넌트를 먼저 쪼갰나

Spring MVC에서 `Controller` 안에 HTML 조립, DTO 변환, 서비스 호출을 한 번에 몰아넣으면 유지보수가 빠르게 어려워집니다. React도 같습니다.

이번 단계는 화면을 세 층으로 나눴습니다.

1. `pages/`
   화면 단위 흐름을 담당합니다. 라우터 파라미터 확인, 어떤 훅을 호출할지 결정, 성공 후 어디로 이동할지 같은 "페이지 오케스트레이션"을 맡습니다.
2. `components/`
   재사용 가능한 화면 조각입니다. 예를 들어 `LogCard`, `RatingDisplay`, `RatingInput`은 어느 페이지에서도 다시 쓸 수 있습니다.
3. `logFormState.ts`
   폼 전용 변환 계층입니다. 이 파일이 중요한 이유는, 브라우저 입력값은 문자열이지만 백엔드 요청 본문은 숫자/배열/optional 필드 조합이기 때문입니다.

즉, Spring 기준으로 보면:

```text
@Controller          -> pages/
DTO Mapper           -> logFormState.ts
View Fragment        -> components/
@Service 호출        -> hooks/useLogs.ts
```

---

## HomePage — 서버 상태를 화면 카드로 펼치기

홈 화면은 `useLogList()`의 `data.pages`를 펼쳐 카드 목록으로 렌더링합니다.

```tsx
const logs = data?.pages.flatMap((page) => page.items) ?? []
```

이 패턴은 Spring에서 `Slice<Page<T>>`를 받아 최종 ViewModel 리스트로 평탄화하는 것과 비슷합니다.

여기서 중요한 점은:

- 목록 UI는 `CursorPage<T>` 내부 구조를 알 필요가 없습니다.
- 다음 페이지 로딩 여부는 `hasNextPage`만 보면 됩니다.
- 스크롤 하단 감지는 `IntersectionObserver`가 담당하고, 실제 요청은 `fetchNextPage()`가 담당합니다.

즉, 스크롤 감지와 서버 페이지네이션을 분리했습니다. 이 분리를 해두면 나중에 "Load more 버튼만 사용" 또는 "자동 감지 + 버튼 fallback" 같은 UX 변경이 쉬워집니다.

---

## LogDetailPage — 타입 분기 렌더링

상세 페이지는 `CoffeeLogFull` discriminated union의 장점을 가장 직접적으로 보여주는 화면입니다.

```tsx
if (log.log_type === 'cafe') {
  log.cafe.cafe_name
}

if (log.log_type === 'brew') {
  log.brew.brew_method
}
```

Java라면 `instanceof`나 별도 DTO 계층이 필요할 수 있는 부분을, TypeScript는 `log_type` 검사만으로 좁혀줍니다.

이 페이지는 다음 책임만 가집니다.

- 단건 조회
- `cafe` / `brew` 분기 출력
- 삭제 액션
- 삭제 성공 후 목록으로 이동

삭제는 서버 상태 변경이므로 `useDeleteLog()` mutation을 사용합니다. 성공 후 목록으로 돌아가면, 이전 단계에서 정의한 query invalidation 덕분에 목록이 최신 상태로 다시 맞춰집니다.

---

## LogFormPage — 생성/수정 통합 폼의 핵심

이번 단계에서 가장 중요한 설계는 "화면 입력 상태"와 "API 요청 본문"을 분리한 점입니다.

브라우저 폼은 본질적으로 문자열 중심입니다.

- 숫자 입력도 `event.target.value`는 문자열
- 태그 입력도 결국 쉼표로 구분한 문자열
- `datetime-local`도 백엔드가 원하는 RFC3339와는 형식이 다름

이 상태로 컴포넌트 안에서 바로 API 요청을 만들면 조건문과 형 변환이 JSX 여기저기에 퍼집니다. 그래서 `logFormState.ts`를 따로 두고 세 가지 함수를 만들었습니다.

### 1. `createEmptyFormState()`

신규 작성 기본값을 만듭니다. 브루 단계는 최소 1개의 빈 step을 넣어, 사용자가 "추가 버튼을 먼저 눌러야만 입력할 수 있는" 불필요한 마찰을 없앴습니다.

### 2. `logToFormState()`

조회한 응답을 수정 폼에 주입합니다.

- 배열 → 쉼표 문자열
- 숫자 → 문자열
- `null` → 빈 문자열

이것은 Spring에서 Entity/Response DTO를 HTML Form backing object로 다시 매핑하는 과정과 비슷합니다.

### 3. `buildLogPayload()`

폼 draft를 서버 요청 본문으로 변환합니다.

- 쉼표 문자열 → `string[]`
- 빈 문자열 → optional 필드 제거
- 숫자 문자열 → `number`
- `datetime-local` → ISO 문자열
- `brew_steps`의 빈 단계 제거

이 함수를 분리해둔 덕분에 페이지 컴포넌트는 "입력 수집"에 집중하고, 변환 규칙은 테스트 가능한 순수 함수로 남았습니다.

---

## 왜 `RatingDisplay`와 `RatingInput`을 분리했나

별점은 보기와 입력의 요구사항이 다릅니다.

- 보기: 현재 점수를 읽기 쉽고 compact해야 함
- 입력: 0.5 단위 선택과 초기화가 쉬워야 함

하나의 컴포넌트로 합치면 prop이 불필요하게 많아지고 조건 분기가 복잡해집니다. 이번 구조는 Spring의 읽기용 DTO / 쓰기용 Form Object를 분리하는 사고와 비슷합니다.

---

## 테스트 전략

이번 프론트 작업은 "페이지가 예쁘게 보이는가"보다 "상태 변환과 사용자 입력이 안전한가"를 먼저 검증했습니다.

### `logFormState.test.ts`

가장 중요한 테스트입니다.

- 신규 폼 기본값 생성
- cafe 입력이 올바른 요청 본문으로 변환되는지
- brew 입력에서 빈 step 제거, 숫자 변환이 되는지
- 상세 응답을 수정용 폼 상태로 되돌릴 수 있는지

이 테스트는 Java에서 DTO mapper 단위 테스트를 두는 것과 같은 의미입니다. 실제로 폼 버그의 상당수는 렌더링보다 변환 단계에서 발생합니다.

### `LogCard.test.tsx`, `RatingInput.test.tsx`

재사용 컴포넌트의 핵심 상호작용만 검증했습니다.

- 카드가 타입별 핵심 정보를 제대로 노출하는지
- 별점 입력이 클릭 값을 부모로 올바르게 전달하는지

---

## 구현 중 중요했던 제약

### 수정 시 `log_type` 변경 금지

백엔드 서비스가 기존 로그 타입 변경을 금지하므로, 프론트도 수정 모드에서는 탭 전환을 막았습니다. 백엔드 규칙을 UI가 반영해야 사용자가 "저장 직전에 거절당하는" 경험을 줄일 수 있습니다.

### 브루 step은 순서가 곧 의미

브루 단계는 단순 문자열 배열이 아니라 순서가 중요한 데이터입니다. 그래서 추가/삭제뿐 아니라 위/아래 이동까지 초기 단계부터 넣었습니다. 이 구조가 있어야 이후 레시피 UX 고도화 때도 기반을 다시 바꿀 필요가 없습니다.

---

## 다음 단계

Phase 2에서는 이번에 만든 기본 CRUD 화면 위에서 다음을 고도화하면 됩니다.

- 홈 화면 필터 (`log_type`, 날짜 범위)
- 브루 폼 레시피 UX 강화
- 무한 스크롤 로딩 상태/스켈레톤 개선
- URL 쿼리 파라미터와 필터 상태 연결
