# 코드 리뷰

## 리뷰 범위

- **브랜치**: `feat/9-draft-save`
- **비교 기준**: `main...HEAD` (8커밋)
- **변경 파일**:
  - `backend/db/migrations/008_add_status_to_coffee_logs.{up,down}.sql`
  - `backend/db/queries/coffee_logs.sql`, `suggestions.sql`
  - `backend/internal/db/coffee_logs.sql.go`, `models.go`
  - `backend/internal/domain/log.go`
  - `backend/internal/handler/log_handler.go`, `log_handler_test.go`
  - `backend/internal/repository/log_repository.go`, `log_repository_test.go`, `suggestion_repository.go`, `suggestion_repository_test.go`
  - `backend/internal/service/log_service.go`, `log_service_test.go`
  - `docs/backlog.md`, `docs/spec.md`, `docs/openapi.yml`
  - `docs/issues/9-draft-save/plan.md`, `tasks.md`
  - `frontend/src/api/logs.ts`
  - `frontend/src/components/LogCard.tsx`, `LogCard.test.tsx`
  - `frontend/src/pages/HomePage.tsx`, `LogDetailPage.tsx`, `LogFormPage.tsx`, `LogFormPage.test.tsx`
  - `frontend/src/pages/logFormState.ts`, `logFormState.test.ts`
  - `frontend/src/types/log.ts`, `schema.ts`

## 요약

`status` 컬럼 추가(draft/published)를 통한 드래프트 저장 기능 구현이다. 아키텍처 설계(NOT NULL + 빈 문자열 전략, service 레이어 status 전이 검증, suggestion 격리)는 plan.md와 충실히 일치하며, 테스트 커버리지도 service/handler/repository 전 레이어에 걸쳐 계획한 케이스 대부분을 달성했다. 다만 기존 `suggestion_repository_test.go`의 `status` 컬럼 누락으로 인한 테스트 격리 파괴 위험, `LogFormPage.test.tsx`의 임시 저장 버튼 관련 테스트 미작성, `LogDetailPage.tsx`의 로딩 중 UI 노출 문제, `LogCard.tsx`의 "View log" 문구 혼선 등 수정이 필요한 항목들이 존재한다.

---

## 발견 사항

### [High] 기존 suggestion 테스트에서 `status` 컬럼 누락으로 인한 DB 제약 위반 위험

- **파일**: `backend/internal/repository/suggestion_repository_test.go:37-55`, `57-80`, `82-105`, `107-135`, `137-161`, `163-187`, `193-217`, `219-240`, `242-263`
- **카테고리**: Quality
- **현재**: `008_add_status_to_coffee_logs.up.sql` 마이그레이션이 `status TEXT NOT NULL DEFAULT 'published' CHECK(status IN ('draft', 'published'))`를 추가했다. 그런데 기존 suggestion 테스트의 `INSERT INTO coffee_logs` 구문들은 `status` 컬럼을 명시하지 않는다. SQLite에서 `DEFAULT` 절이 있으므로 런타임에는 통과하지만, 마이그레이션 순서나 SQLite 버전에 따라 `DEFAULT` 처리 여부가 달라질 수 있다. 더 큰 문제는 새로 추가된 `TestGetTagSuggestions_DraftLog_Excluded`, `TestGetCompanionSuggestions_DraftLog_Excluded` 테스트는 `status`를 명시하는데, 바로 위 기존 테스트들은 명시하지 않아 코드 스타일이 일관되지 않으며 차후 테스트 픽스처가 잘못된 status로 삽입될 위험이 있다.
- **제안**: 기존 `INSERT INTO coffee_logs` 구문에 `status` 컬럼과 값을 명시적으로 추가한다.

  ```go
  // 기존
  `INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, created_at, updated_at)
   VALUES (?, ?, ?, ?, ?, ?, ?)`,
  "log-1", testUserID, now, "[]", "cafe", now, now,

  // 변경
  `INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, status, created_at, updated_at)
   VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
  "log-1", testUserID, now, "[]", "cafe", "published", now, now,
  ```
- **근거**: AGENTS.md "항상 테스트를 작성할 것" 정책 및 테스트 격리 원칙. 기존 suggestion 테스트가 `status` 필터 동작을 가정하고 있으므로 픽스처가 정확한 status 값을 갖도록 보장해야 한다.

---

### [High] `LogFormPage.test.tsx`에 임시 저장 관련 테스트 미작성

- **파일**: `frontend/src/pages/LogFormPage.test.tsx`
- **카테고리**: Quality
- **현재**: `LogFormPage.test.tsx`는 clone/recipe-picker 모드 테스트만 포함하고, plan.md와 tasks.md에서 명시한 임시 저장 버튼 관련 케이스(신규 cafe 폼에서 "임시 저장" 표시, 빈 폼에서 disabled, cafe_name 입력 후 enabled, 클릭 시 `createLog`에 `status: 'draft'` 전달 및 홈으로 navigate, published 수정 모드에서 버튼 미표시, draft 수정 모드에서 표시 및 클릭 시 `updateLog`에 `status: 'draft'` 전달)가 전혀 구현되어 있지 않다.
- **제안**: 아래 케이스를 `LogFormPage.test.tsx`에 추가한다.

  ```tsx
  describe('임시 저장 버튼', () => {
    it('신규 cafe 폼에서 "임시 저장" 버튼이 표시된다', () => {
      renderNewMode()
      expect(screen.getByRole('button', { name: '임시 저장' })).toBeInTheDocument()
    })

    it('cafe_name/coffee_name이 모두 비어있으면 "임시 저장" 버튼이 disabled다', () => {
      renderNewMode()
      expect(screen.getByRole('button', { name: '임시 저장' })).toBeDisabled()
    })

    it('cafe_name 입력 후 "임시 저장"이 활성화된다', async () => {
      renderNewMode()
      fireEvent.change(screen.getByLabelText(/Cafe name/), { target: { value: '블루보틀' } })
      expect(screen.getByRole('button', { name: '임시 저장' })).toBeEnabled()
    })

    it('published 수정 모드에서는 "임시 저장" 버튼이 없다', () => {
      // useLog mock을 published 로그로 오버라이드하는 테스트
      // ...
    })
  })
  ```
- **근거**: AGENTS.md "기능 추가/수정 시 항상 테스트 작성" 정책. plan.md 테스트 전략 섹션에 명시된 케이스들이 미구현 상태다.

---

### [Medium] `LogDetailPage.tsx`에서 드래프트 redirect 중 UI가 brief하게 노출됨

- **파일**: `frontend/src/pages/LogDetailPage.tsx:78-82`
- **카테고리**: Quality
- **현재**: `useEffect`로 `log?.status === 'draft'`를 감지해 redirect하지만, `log`가 로드되기 전까지 빈 상태가 렌더되고, 로드 완료 후 `useEffect`가 실행되므로 draft 로그 상세 내용이 한 프레임 이상 노출될 수 있다. 특히 `log`가 draft이면서 `isLoading`이 false인 시점에 `{log ? <div>...상세 내용...</div>}` 블록이 렌더된 뒤 redirect 처리가 일어난다.

  ```tsx
  // 현재: 렌더 이후 effect에서 redirect
  useEffect(() => {
    if (log?.status === 'draft' && id) {
      navigate(`/logs/${id}/edit`, { replace: true })
    }
  }, [log?.status, id, navigate])

  // ...

  {log ? (
    <div className="space-y-6">
      {/* draft 상세 내용이 한 프레임 노출될 수 있다 */}
    </div>
  ) : null}
  ```
- **제안**: `log`가 있고 draft인 경우 상세 컨텐츠 렌더를 건너뛴다.

  ```tsx
  // 로딩 중이거나 드래프트 redirect 대기 중이면 컨텐츠를 렌더하지 않는다
  {log && log.status !== 'draft' ? (
    <div className="space-y-6">
      {/* 상세 내용 */}
    </div>
  ) : null}
  ```
- **근거**: plan.md "드래프트는 상세 화면에서 보여줄 의미 있는 정보가 적으므로 곧바로 수정 폼으로 보낸다"는 설계 의도와 일치시키기 위함. draft 상세 내용이 순간적으로 노출되는 것은 의도하지 않은 UX다.

---

### [Medium] `normalizeListFilter`의 `limit` 중복 검사 (dead code)

- **파일**: `backend/internal/service/log_service.go:401-414`
- **카테고리**: Quality
- **현재**: `limit == 0` 조건이 두 번 검사된다. 첫 번째는 기본값 할당(`limit = defaultListLimit`)이고, 두 번째(`limit == 0`인 경우 ValidationError 반환)는 첫 번째 블록이 이미 0을 `defaultListLimit`으로 치환했으므로 도달 불가능한 코드다.

  ```go
  limit := filter.Limit
  if limit == 0 {
      limit = defaultListLimit  // 0을 defaultListLimit으로 치환
  }
  if limit < 0 {
      return ..., newValidationError("limit", "0 이상이어야 합니다")
  }
  if limit > maxListLimit {
      return ..., newValidationError("limit", ...)
  }
  if limit == 0 {  // 이 분기는 절대 실행되지 않는다
      return ..., newValidationError("limit", "0보다 커야 합니다")
  }
  ```
- **제안**: 도달 불가능한 마지막 `if limit == 0` 블록을 제거한다. 음수 처리는 `limit < 0`으로 충분하다.

  ```go
  limit := filter.Limit
  if limit == 0 {
      limit = defaultListLimit
  }
  if limit < 0 {
      return repository.ListFilter{}, 0, newValidationError("limit", "0 이상이어야 합니다")
  }
  if limit > maxListLimit {
      return repository.ListFilter{}, 0, newValidationError("limit", fmt.Sprintf("%d 이하여야 합니다", maxListLimit))
  }
  ```
- **근거**: 이 dead code는 기존 PR에서 이미 존재했을 수 있으나, 이번 PR의 `normalizeListFilter` 수정 범위에 포함된다. 불필요한 조건문은 코드 가독성을 해치고 잘못된 이해를 유발할 수 있다.

---

### [Medium] `LogCard.tsx`의 "View log" 문구가 draft 카드에서 혼선 유발

- **파일**: `frontend/src/components/LogCard.tsx:173`
- **카테고리**: Quality
- **현재**: 카드 하단 footer에 `<span>View log</span>`이 항상 렌더된다. draft 카드는 클릭 시 상세 페이지가 아닌 수정 폼(`/logs/:id/edit`)으로 이동하는데, "View log"라는 문구는 상세 보기 의도를 암시해 사용자에게 혼선을 준다.

  ```tsx
  <span className="transition group-hover:translate-x-1">View log</span>
  ```
- **제안**: draft 여부에 따라 문구를 분기한다.

  ```tsx
  <span className="transition group-hover:translate-x-1">
    {isDraft ? '이어서 작성' : 'View log'}
  </span>
  ```
- **근거**: plan.md "드래프트 카드의 to 분기" 설계에서 클릭 시 수정 폼으로 이동하는 것을 명시했다. 링크 목적지와 문구가 불일치하면 사용자가 드래프트 상태를 오해할 수 있다.

---

### [Low] `buildLogPayload`에서 draft 모드임에도 `normalizeText`로 빈 문자열을 `undefined`로 변환

- **파일**: `frontend/src/pages/logFormState.ts:329-330`
- **카테고리**: Quality
- **현재**: `buildLogPayload`에서 cafe draft 모드일 때 `cafe_name`과 `coffee_name`은 `.trim()`으로 빈 문자열 그대로 전달하지만, 이 함수 내 옵셔널 필드(location, beanOrigin 등)는 `normalizeText`를 통해 빈 문자열이 `undefined`로 변환된다. 이 자체는 의도된 동작이나(plan.md 6. 선택 필드 처리 정책과 일치), 함수 내 주석이나 옵션 분기가 없어 "draft이면 빈 문자열 그대로"라는 설명이 cafe_name/coffee_name에만 국한된 것임을 파악하기 어렵다.
- **제안**: `buildLogPayload` 함수 내 관련 블록에 주석을 추가한다.

  ```ts
  if (state.logType === 'cafe') {
    payload.cafe = {
      // draft/published 모두 trim 후 그대로 전달한다.
      // draft는 백엔드 draft 검증기에 필수 필드 검증을 위임한다(plan.md 6항 참조).
      cafe_name: state.cafe.cafeName.trim(),
      coffee_name: state.cafe.coffeeName.trim(),
      tasting_tags: state.cafe.tastingTags,
    }
    // 옵셔널 필드는 status와 무관하게 빈 값이면 undefined로 정규화한다.
    const location = normalizeText(state.cafe.location)
    // ...
  }
  ```
- **근거**: AGENTS.md "주석은 한국어로"(기술 용어는 원문 유지) 정책. 의도를 명시적으로 문서화하면 향후 기여자가 draft/published 분기 누락을 실수로 추가하는 것을 방지할 수 있다.

---

### [Low] `RecipePickerModal`이 published 로그만 표시하지 않음

- **파일**: `frontend/src/pages/LogFormPage.tsx:230`
- **카테고리**: Quality
- **현재**: `RecipePickerModal` 내부의 `useLogList({ log_type: 'brew' })` 호출에 `status` 파라미터가 없다. 기본값이 published이므로 현재는 안전하지만, 코드만 보면 draft brew 로그가 레시피 목록에 포함될 가능성을 배제하기 어렵다. plan.md "통계/자동완성 격리" 섹션에서 "기본 목록 호출만으로 published만 집계된다"라고 명시하고 있으나, 의도를 코드에서 명시적으로 표현하지 않는다.

  ```tsx
  // 현재
  const { data, ... } = useLogList({ log_type: 'brew' })

  // 제안: 의도를 명시
  const { data, ... } = useLogList({ log_type: 'brew', status: 'published' })
  ```
- **제안**: `status: 'published'`를 명시적으로 전달한다.
- **근거**: AGENTS.md 코드 컨벤션상 기본값 의존보다 명시적 표현이 선호된다. 레시피 목록에 draft brew 로그(미완성 bean_name)가 표시되면 혼란을 줄 수 있다.

---

## 액션 아이템

1. **[High]** `backend/internal/repository/suggestion_repository_test.go`의 기존 `INSERT INTO coffee_logs` 구문 9개 모두에 `status, 'published'`를 컬럼/값 목록에 추가한다. (37-55, 57-80, 82-105, 107-135, 137-161, 163-187, 193-217, 219-240, 242-263 라인 부근)

2. **[High]** `frontend/src/pages/LogFormPage.test.tsx`에 임시 저장 버튼 테스트를 추가한다: (a) 신규 cafe 폼에서 "임시 저장" 버튼 표시, (b) 빈 폼 → disabled, (c) cafe_name 입력 → enabled, (d) 클릭 → `createLog` mock에 `status: 'draft'` 전달 및 홈으로 navigate, (e) published 수정 모드 → 버튼 미표시, (f) draft 수정 모드 → 버튼 표시 + 클릭 → `updateLog`에 `status: 'draft'` 전달.

3. **[Medium]** `frontend/src/pages/LogDetailPage.tsx`의 상세 컨텐츠 렌더 조건을 `{log && log.status !== 'draft' ? ... : null}`로 변경하여 draft redirect 중 상세 내용이 순간 노출되지 않도록 한다. (159번 라인 부근)

4. **[Medium]** `backend/internal/service/log_service.go`의 `normalizeListFilter`에서 도달 불가능한 `if limit == 0` 블록(411-413 라인)을 제거한다.

5. **[Medium]** `frontend/src/components/LogCard.tsx`의 173번 라인 `<span>View log</span>`을 `<span>{isDraft ? '이어서 작성' : 'View log'}</span>`으로 변경한다.

6. **[Low]** `frontend/src/pages/LogFormPage.tsx`의 `RecipePickerModal` 내 `useLogList({ log_type: 'brew' })` 호출을 `useLogList({ log_type: 'brew', status: 'published' })`로 변경하여 의도를 명시한다. (230번 라인 부근)

7. **[Low]** `frontend/src/pages/logFormState.ts`의 `buildLogPayload` cafe 블록 시작부에 draft 위임 의도를 설명하는 한국어 주석을 추가한다. (328-330번 라인 부근)
