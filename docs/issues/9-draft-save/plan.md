# Issue #9 — 드래프트 저장

## 목표

로그를 완성하지 못한 상태에서도 카페 이름이나 원두 이름 같은 일부 필드만 채운 채로 임시 저장(`status = 'draft'`)할 수 있게 한다. 마시는 순간 빠르게 메모만 남기고 나중에 이어서 완성하는 워크플로우를 지원하며, 드래프트 로그가 통계나 자동완성 같은 파생 데이터에 영향을 주지 않도록 격리한다.

이 이슈는 Epic #10(기록하기 더 쉽게)의 마지막 서브 이슈로, 단계적 입력(#5), 복제(#6), 프리셋(#7), 레시피 템플릿(#8)에 이어 "필수 입력 자체를 미루는" 마지막 안전망을 제공한다.

---

## 데이터 모델 변경

### `coffee_logs` 테이블에 `status` 컬럼 추가

새 마이그레이션 `008_add_status_to_coffee_logs`을 추가한다.

```sql
ALTER TABLE coffee_logs
  ADD COLUMN status TEXT NOT NULL DEFAULT 'published'
  CHECK(status IN ('draft', 'published'));
```

설계 결정 — 필드 위치를 `coffee_logs`(공통 테이블)에 둔다:
- 드래프트는 cafe/brew 공통 개념이며, 어떤 서브 디테일도 미작성 가능하다.
- 목록 조회 시 status 기반 필터링/섹션 분리가 필요한데, 이는 `coffee_logs`만 스캔하면 된다.
- 기존 데이터는 모두 완성된 published 로그이므로 `DEFAULT 'published'`로 안전하게 backfill 된다.

`down.sql`은 단순 `ALTER TABLE coffee_logs DROP COLUMN status;`로 작성한다(SQLite 3.35+ 지원).

### 인덱스

P0 단계에서는 추가 인덱스 없이 기존 PK + `(user_id, recorded_at, id)` 정렬에 의존한다. 사용자당 드래프트가 수십 건 이하로 유지될 것으로 가정하며, 향후 필요 시 `(user_id, status, recorded_at)` 복합 인덱스 추가를 고려한다(backlog 후보).

### 서브 디테일 테이블의 NOT NULL 제약 검토

`cafe_logs.cafe_name`, `cafe_logs.coffee_name`, `brew_logs.bean_name`, `brew_logs.brew_method`는 현재 모두 `NOT NULL`이다. 드래프트는 이 중 일부만 채워져 있으므로 다음 두 가지 전략 중 하나가 필요하다.

**채택 전략 A — 빈 문자열 허용:**
- 도메인/서비스 레이어에서 드래프트 검증 시 "둘 중 하나 이상은 채워져야 한다"는 규칙만 적용한다.
- DB는 `''` 빈 문자열을 허용한다(NOT NULL 위반 아님).
- 채우지 않은 필드는 빈 문자열로 저장한다.
- 마이그레이션 변경 없이 기존 스키마 유지.

**전략 B(미채택) — NULL 허용:**
- 컬럼을 `NULL` 허용으로 변경.
- 도메인 타입(`CafeDetail.CafeName` 등)이 string에서 `*string`으로 바뀌어 영향 범위가 너무 크다.

전략 A를 채택한다. 도메인 코드 변경 최소화, 기존 published 로그의 의미 보존(빈 문자열은 published에서는 여전히 검증으로 차단), 응답 직렬화 시 추가 분기 불필요라는 이점이 있다.

### 도메인 모델 (Go)

`backend/internal/domain/log.go`의 `CoffeeLog`에 `Status` 필드 추가:

```go
type LogStatus string

const (
    LogStatusDraft     LogStatus = "draft"
    LogStatusPublished LogStatus = "published"
)

type CoffeeLog struct {
    // 기존 필드...
    Status LogStatus
    // 기존 필드...
}
```

domain은 단순 enum 정의만 추가하고, 검증/전이 규칙은 service 레이어가 책임진다.

---

## API 계약 변경 (OpenAPI)

`docs/openapi.yml` 변경:

1. **새 스키마 `LogStatus`:**
   ```yaml
   LogStatus:
     type: string
     enum: [draft, published]
   ```

2. **`CoffeeLogResponse`에 `status` 추가** — required로 포함. 기존 published 데이터는 백엔드가 `'published'`로 응답.

3. **`CreateLogRequest`에 `status` 추가** — optional, default `published`. 단, 클라이언트가 `draft`를 명시하면 드래프트 검증 규칙이 적용된다.

4. **`UpdateLogRequest`에 `status` 추가** — optional. 생략하면 기존 status 유지(현 동작과 호환).

5. **`GET /api/v1/logs`에 `status` 쿼리 파라미터 추가:**
   ```yaml
   - name: status
     in: query
     schema:
       type: string
       enum: [draft, published, all]
       default: published
     description: |
       기본값 published(완성된 로그만). draft는 작성 중 로그만,
       all은 둘 다 반환.
   ```

   기본값을 `published`로 두는 이유: 통계/인사이트 등 기존 호출 지점은 변경 없이 동작해야 한다. 드래프트 섹션이 새로 추가된 홈 화면만 명시적으로 `status=draft`를 전달한다.

6. **`CafeDetail`, `BrewDetail`의 required 필드 완화 — 해당 사항 없음.**
   드래프트 저장에서도 OpenAPI 상의 `cafe.cafe_name`, `cafe.coffee_name`, `brew.bean_name`, `brew.brew_method`는 여전히 required로 둔다. 드래프트는 빈 문자열 `""`을 허용하는 방식으로 처리하므로 OpenAPI required를 완화할 필요가 없다. 클라이언트는 미입력 필드를 `""`로 보낸다.

타입 재생성: `cd frontend && npm run generate`.

---

## 백엔드 설계

### Service 레이어 (검증 분기)

`backend/internal/service/log_service.go`의 `normalizeCreateRequest`, `normalizeUpdateRequest`를 status에 따라 검증 강도를 달리하도록 확장한다.

**핵심 분기:**

```go
switch normalizedReq.Status {
case domain.LogStatusDraft:
    return validateDraftDetail(logType, req.Cafe, req.Brew)
case domain.LogStatusPublished:
    return validatePublishedDetail(logType, req.Cafe, req.Brew) // 기존 로직
}
```

**드래프트 검증 규칙:**
- `log_type` 필수 (기존과 동일)
- `recorded_at` 필수 (기존과 동일, RFC3339)
- cafe 드래프트: `cafe.cafe_name`이나 `cafe.coffee_name` 중 하나 이상이 비어있지 않아야 함
- brew 드래프트: `brew.bean_name`이나 `brew.brew_method` 중 하나 이상이 비어있지 않아야 함
- 그 외 cafe/brew 필드는 모두 optional, 빈 문자열 허용
- rating, recorded_at 등 형식 검증(0.5 단위, RFC3339)은 값이 있을 때만 적용 — published와 동일

**published 검증 규칙:** 기존 그대로 유지(`validateRequiredString`로 cafe_name, coffee_name, bean_name 모두 필수).

위 분기를 위해 normalize 함수에 `status` 파라미터를 추가하거나, `normalizeCafeDetailDraft` / `normalizeBrewDetailDraft`를 신규 추가한다. 후자(별도 함수)가 기존 검증 함수를 변경하지 않으므로 회귀 위험이 낮다 — **별도 함수 방식을 선택한다.**

### Service 레이어 (status 전이 검증)

`UpdateLog`에서 status 전이를 검사한다.

```go
// existing.Status 와 normalizedReq.Status 비교
if existing.Status == LogStatusPublished && normalizedReq.Status == LogStatusDraft {
    return ValidationError{Field: "status", Message: "발행된 로그를 드래프트로 되돌릴 수 없습니다"}
}
```

`UpdateLogRequest`의 `Status`가 빈 값이면 `existing.Status`를 그대로 쓴다(기존 호환).

### Service 레이어 (목록 필터)

`ListLogsFilter`에 `Status` 필드 추가:

```go
type ListLogsFilter struct {
    // 기존 필드
    Status *string // "draft", "published", "all". nil이면 published만(기본값)
}
```

`normalizeListFilter`에서 status 파라미터를 검증하고 repository에 전달한다.

### Repository 레이어

1. **`InsertLog`, `GetLogByID`, `ListLogs`, `UpdateLog` 쿼리에 `status` 컬럼 추가.**
   - sqlc 쿼리 (`db/queries/coffee_logs.sql`)에서 SELECT/INSERT/UPDATE에 `status` 추가.
   - `sqlc generate` 재실행 필요.
   - `ListLogs`는 raw SQL(`log_repository.go::ListLogs`)이므로 `SELECT ... status ...`와 status 필터 분기를 직접 추가한다:
     ```go
     if filter.Status == nil || *filter.Status == "published" {
         query += ` AND status = 'published'`
     } else if *filter.Status == "draft" {
         query += ` AND status = 'draft'`
     }
     // "all" 인 경우 필터 없음
     ```
   - `coffeeLogToFull`에서 `Status: domain.LogStatus(row.Status)` 매핑.

2. **`ListFilter` 구조체에 `Status *string` 추가.**

3. **`batchLoadCafe`, `batchLoadBrew`는 변경 불필요** — status는 `coffee_logs`에만 있으므로.

4. **`Suggestion` 쿼리는 published만 대상으로 한다.** `db/queries/suggestions.sql`에서 `coffee_logs.status = 'published'` 조건을 추가. 현재 suggestion repository는 raw SQL(`json_each`)을 쓰므로 `internal/repository/suggestion_repository.go`에서도 동일하게 처리.

### Handler 레이어

1. **`coffeeLogResponse` 구조체에 `Status string` 필드 추가** (json tag: `"status"`, omitempty 없이 항상 포함).

2. **`createLogRequest`, `updateLogRequest`에 `Status *string` 추가.** 빈 값이거나 누락 시 service가 기본값 처리.

3. **`logToResponse`에서 `Status: string(log.Status)` 매핑.**

4. **`ListLogs` 핸들러: 쿼리스트링 `status` 파싱 후 service 필터로 전달.**
   - 빈 값이면 service 기본값(`published`).
   - 허용 값: `published`, `draft`, `all`.

5. **`writeServiceError`에서 status 전이 위반은 ValidationError로 자동 처리됨** (별도 매핑 불필요).

---

## 프론트엔드 설계

### 타입

`frontend/src/types/log.ts`에서 `LogStatus`를 schema에서 export. discriminated union(`CafeLogFull`, `BrewLogFull`)에는 `status: LogStatus`가 자동으로 포함된다(schema.ts에서 그대로 흘러옴).

### API 클라이언트

`frontend/src/api/logs.ts`:

1. **`ListLogsParams`에 `status?: 'draft' | 'published' | 'all'` 추가.**
2. **`getLogs`가 `status` 쿼리스트링을 전달하도록 확장.**
3. **`createLog`, `updateLog`는 input 타입 자동 갱신** — `CreateLogInput`, `UpdateLogInput`에 status가 포함되므로 별도 함수 시그니처 변경 없음.

### Hooks

`frontend/src/hooks/useLogs.ts`:

- `useLogList(params)`는 그대로 사용. 호출부에서 `status` 옵션을 넘겨 두 종류 목록(draft, published)을 별도 캐시로 운영한다.
- 쿼리 키에 자동으로 params가 포함되므로 캐시 분리는 자연스럽게 이루어진다.
- mutation(`useCreateLog`, `useUpdateLog`)은 변경 없음. 다만 invalidate 키 `LOG_KEYS.all`이 모든 목록(draft, published)을 무효화하므로 충분.

### `logFormState.ts`

1. **`LogFormState`에 `status: LogStatus` 필드 추가:**
   - `createEmptyFormState`에서 `status: 'published'` 기본값.
   - `logToFormState`에서 `status: log.status`로 채움.
   - `cloneToFormState`, `recipeToFormState`, `presetToFormState`는 모두 `status: 'published'`로 리셋(드래프트 복제 시에도 새 로그는 발행 의도로 시작).

2. **`buildLogPayload(state, options)` 시그니처 확장:**
   ```ts
   interface BuildOptions {
     status: LogStatus // 호출자가 명시 (저장 버튼별로 결정)
   }
   ```
   - draft 모드일 때는 trim 후에도 빈 문자열을 그대로 보낸다(필수 필드 검증을 백엔드 draft validator에 위임).
   - published 모드는 기존 로직 유지.

3. **`canSaveAsDraft(state): boolean` 헬퍼 추가:**
   - cafe: `state.cafe.cafeName.trim() || state.cafe.coffeeName.trim()`
   - brew: `state.brew.beanName.trim() || state.brew.brewMethod` (brewMethod는 `createEmptyFormState`에서 `pour_over` 기본값이므로 항상 truthy. 그러나 사용자가 명시적으로 선택하지 않은 상태를 구분할 수 없다는 한계가 있다. 본 이슈에서는 brewMethod가 기본값으로라도 존재하면 드래프트 저장 가능으로 간주한다.)

### `LogFormPage.tsx`

1. **상단 actions에 "임시 저장" 버튼 추가.**
   - 위치: 기존 "기록 추가" / "변경 저장" 버튼 왼쪽.
   - 비활성화 조건: `!canSaveAsDraft(form)` 또는 mutation pending 중.
   - 클릭 시 `buildLogPayload(form, { status: 'draft' })`로 페이로드를 만들고 createMutation/updateMutation 호출.

2. **이미 published 상태인 로그를 수정 중일 때는 "임시 저장" 버튼을 숨긴다.**
   - 백엔드 검증(`published → draft 금지`)과 일치.
   - 조건: `isEditMode && log?.status === 'published'`.
   - 즉, 임시 저장 버튼이 보이는 경우는: (a) 신규 작성, (b) 신규 작성에서 onSelect 등으로 채운 폼, (c) 기존 draft 수정 중.

3. **저장 후 라우팅:**
   - draft 저장 성공 → 홈(`/`)으로 이동(목록의 드래프트 섹션에서 확인 가능).
   - published 저장 성공 → 기존과 동일하게 `/logs/:id`로 이동.

4. **드래프트 → published 발행 흐름:**
   - 드래프트를 열어 "변경 저장" 버튼을 누르면 `buildLogPayload(form, { status: 'published' })`로 호출되고, 백엔드가 published 검증(필수 필드 모두)을 수행한다.
   - 검증 실패 시 기존 ValidationError 인라인 표시 메커니즘이 그대로 적용된다.

5. **수정 모드 진입 시 form.status를 백엔드 응답에서 가져온다** — `logToFormState(log)`에서 자동.

### `HomePage.tsx`

1. **두 개의 `useLogList` 호출:**
   ```ts
   const drafts = useLogList({ limit: 12, status: 'draft' })
   const published = useLogList({ status: 'published', /* 기존 필터 */ })
   ```

2. **렌더 순서:**
   - 드래프트가 1건 이상이면 상단에 "작성 중인 기록" 섹션 헤더와 함께 카드 리스트.
   - 그 아래 기존 FilterBar + published 목록.
   - 드래프트 섹션은 무한 스크롤 없이 초기 페이지(최대 12건)만 표시한다. 사용자당 드래프트가 많지 않다고 가정.

3. **드래프트 섹션 빈 상태:** 드래프트가 0건이면 섹션 자체를 렌더하지 않는다(시각적 노이즈 최소화).

### `LogCard.tsx`

1. **드래프트용 시각 구분:**
   - `log.status === 'draft'`일 때 카드 외곽 테두리를 점선(`border-dashed`)으로, 좌측 상단에 "작성 중" 뱃지 추가.
   - 클릭 시 상세가 아닌 수정 폼(`/logs/:id/edit`)으로 이동한다(드래프트는 상세 화면이 의미 없음).
     - 구현: `to={log.status === 'draft' ? \`/logs/${log.id}/edit\` : \`/logs/${log.id}\`}`.

2. **드래프트 카드의 결측 필드 표시:**
   - 카드 제목(`title`)이 빈 문자열일 수 있으므로 fallback: `title || '제목 미작성'`.
   - subtitle도 동일하게 fallback.
   - rating이 없으면 RatingDisplay가 빈 표시 처리(기존 동작 유지).

3. **드래프트 카드의 복제/삭제 버튼:**
   - 복제 버튼은 드래프트에도 노출하되, 복제 결과는 `status='published'`인 새 폼으로 시작한다.
   - 삭제는 일반 로그와 동일.

### `LogDetailPage.tsx`

드래프트 로그 상세 진입은 LogCard에서 막혀 있지만, 직접 URL을 입력하는 케이스가 있을 수 있다. 다음 둘 중 하나를 적용한다.

- **선택 1 (간단):** 상세 페이지에서 `log.status === 'draft'`이면 자동으로 `/logs/:id/edit`로 redirect.
- **선택 2:** 드래프트 상태 안내 + "이어서 작성" 버튼만 표시.

본 이슈에서는 **선택 1**(redirect)을 채택한다 — 드래프트는 "아직 보여줄 게 없는 상태"이므로 상세 화면을 별도 디자인할 가치가 낮다.

---

## 통계/자동완성 격리

- **Suggestion 쿼리**: `coffee_logs.status = 'published'` 조건을 모든 자동완성 쿼리에 추가한다.
- **프리셋 last_used_at 갱신은 영향 없음**: 사용자가 "프리셋으로 작성"을 누르면 `usePreset` 호출이 수반되는데, 그 결과로 만들어지는 로그가 draft든 published든 프리셋의 사용 기록은 동일하게 갱신된다. 본 이슈에서는 변경 없음.
- **목록 기본 status=published**: 향후 통계/인사이트 화면이 추가되더라도 기본 목록 호출만으로 published만 집계된다.

---

## UX 디테일

- **임시 저장 버튼 라벨:** "임시 저장" (한국어). pending 상태에서 "저장 중...".
- **드래프트 카드 뱃지:** 점선 테두리 + 좌측 상단의 amber 배경 작은 뱃지 ("작성 중").
- **임시 저장 disabled 안내:** 드래프트 최소 조건(cafe_name/coffee_name 또는 bean_name) 미충족 시 버튼이 disabled되며, hover 툴팁이나 헬퍼 텍스트로 "카페 이름이나 메뉴 이름을 한 가지 이상 입력해주세요"를 안내한다(본 이슈에서는 단순 disabled로 충분).
- **수정 모드의 published 로그**에서는 임시 저장 버튼을 숨겨 사용자가 published → draft 회귀를 시도하지 않도록 한다.

---

## 수정하지 않는 것

- **기존 cafe/brew 서브 테이블 스키마** — `cafe_logs.cafe_name` NOT NULL 등 그대로 유지.
- **프리셋 도메인** — preset 테이블에는 status 개념이 없다.
- **자동완성 응답 포맷** — published만 집계하도록 SQL만 변경하고 응답 형태는 동일.
- **인증/세션 흐름** — 드래프트도 일반 로그와 동일한 사용자 인증 사용.
- **기존 published 로그의 데이터** — 마이그레이션이 `DEFAULT 'published'`로 backfill만 수행.
- **삭제 API** — 드래프트 삭제는 기존 `DELETE /api/v1/logs/:id`를 그대로 사용한다.

---

## 위험 요소

1. **NOT NULL + 빈 문자열 전략의 모호성:**
   - 드래프트의 `cafe_name = ""` 와 published의 `cafe_name = ""` 를 DB만 보고 구분할 수 없다.
   - 완화책: 도메인/검증 레이어에서 `status`로 분기하므로 실제로 published가 빈 cafe_name을 가질 수 없다.
   - 회귀 테스트로 published 검증이 빈 문자열을 거부하는지 확인.

2. **brew의 `brew_method` 기본값 문제:**
   - `createEmptyFormState`에서 `brewMethod: 'pour_over'` 기본값이 들어 있어, 사용자가 brew_method를 의도적으로 선택했는지 구분 어려움.
   - 본 이슈에서는 무시하고 진행 — 드래프트 단계에서 brew_method가 임의 값이어도 사용자가 나중에 수정할 수 있다.
   - 개선이 필요하면 backlog 항목으로 추가.

3. **자동완성 쿼리에 `status` 조건 누락 위험:**
   - 향후 자동완성 쿼리를 추가할 때 `status='published'` 조건을 누락하면 드래프트 데이터가 새어나갈 수 있다.
   - 완화책: suggestion repository 테스트에 "draft 로그의 태그/companion이 응답에 포함되지 않는다" 케이스 추가.

4. **OpenAPI required 호환성:**
   - `cafe.cafe_name`이 OpenAPI에서 required인데, 드래프트 클라이언트가 빈 문자열로 보낸다.
   - OpenAPI required는 "필드 존재"를 의미하지 "비어 있지 않다"를 강제하지 않으므로 문제 없음.

---

## 테스트 전략

### 백엔드

**Unit (service):**
- `normalizeCreateRequest`/`normalizeUpdateRequest`의 status별 분기:
  - cafe draft: cafe_name만 채워짐 → 통과
  - cafe draft: coffee_name만 채워짐 → 통과
  - cafe draft: 둘 다 비어 있음 → ValidationError
  - brew draft: bean_name만 채워짐 → 통과
  - brew draft: 둘 다 비어 있음 → ValidationError(단, brew_method가 항상 enum에서 들어오므로 실질적으로는 brewMethod도 검사)
  - published 검증은 status='published'에서 기존 동작 유지 (회귀 테스트)
- status 전이:
  - published → draft 차단
  - draft → published 허용 (단, published 검증 규칙 재적용)
  - draft → draft 허용
- ListLogsFilter status 정규화: published(기본), draft, all 외의 값은 ValidationError

**Integration (repository):**
- `008_add_status_to_coffee_logs` 마이그레이션 적용 후 기존 log CRUD 정상 동작
- ListLogs status 필터:
  - draft 1건 + published 2건 시드
  - status 미지정 → 2건 (published만)
  - status='draft' → 1건
  - status='all' → 3건
- Suggestion 쿼리 회귀:
  - draft 로그에만 있는 tasting_tag가 자동완성 결과에 포함되지 않는지

**Handler:**
- POST /api/v1/logs status='draft' + 빈 cafe_name + coffee_name 채움 → 201
- POST /api/v1/logs status='draft' + 빈 cafe_name + 빈 coffee_name → 400
- PUT /api/v1/logs/{id} 기존 published를 draft로 변경 시도 → 400
- GET /api/v1/logs?status=draft → 드래프트만
- GET /api/v1/logs (status 미지정) → published만
- GET /api/v1/logs?status=invalid → 400

### 프론트엔드

**Unit (logFormState):**
- `canSaveAsDraft`:
  - 빈 폼 → cafe는 false, brew는 (brewMethod 기본값 때문에) true
  - cafe_name 입력 후 → true
  - coffee_name 입력 후 → true
- `buildLogPayload(state, { status: 'draft' })`:
  - 빈 필드를 빈 문자열로 그대로 보냄
  - status 필드가 페이로드에 포함됨
- `logToFormState`가 응답의 status를 폼 상태로 옮김

**Component (LogFormPage):**
- 신규 작성 폼에 "임시 저장" 버튼 표시
- cafe_name, coffee_name 모두 비어 있으면 임시 저장 disabled
- cafe_name 입력 시 enabled
- 신규 draft 저장 → 홈으로 이동
- 드래프트 수정 모드에서 "임시 저장" 표시, "변경 저장"도 표시
- 기존 published 수정 모드에서 "임시 저장" 미표시

**Component (HomePage):**
- 드래프트 0건: 드래프트 섹션 미렌더
- 드래프트 N건 + published M건: 두 섹션 모두 렌더, 드래프트가 위에

**Component (LogCard):**
- status='draft' 카드: 점선 테두리, "작성 중" 뱃지 표시
- 드래프트 카드 클릭 → `/logs/:id/edit`
- published 카드 클릭 → `/logs/:id` (회귀 테스트)

**E2E (PR 직전 수동):**
1. 새 cafe 폼 → cafe_name만 입력 → 임시 저장 → 홈에 드래프트 섹션 노출 확인
2. 드래프트 카드 탭 → 수정 폼 진입, 이전 입력값 채워짐 확인
3. 나머지 필드 채우고 변경 저장 → published로 전환, 홈의 드래프트 섹션에서 사라지고 published 목록에 등장
4. brew 드래프트도 동일 흐름 확인
5. 수정 폼에서 다시 임시 저장 시 status가 draft로 유지되는지
6. 드래프트 삭제 → 홈에서 사라지는지
