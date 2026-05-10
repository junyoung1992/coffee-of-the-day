# Tasks — Issue #9 드래프트 저장

> 백엔드(마이그레이션 → 도메인 → 쿼리 → repository → service → handler) → OpenAPI → 프론트엔드(타입 재생성 → API → 폼 상태 → UI) 순서로 진행한다.
> 각 레이어의 테스트는 같은 단계에서 같이 작성한다.
> 자세한 설계 맥락은 `plan.md` 참조.

---

## 1. DB 마이그레이션 추가

- [ ] **`008_add_status_to_coffee_logs.up.sql` 생성**
  - Target: `backend/db/migrations/008_add_status_to_coffee_logs.up.sql`
  - 내용:
    ```sql
    ALTER TABLE coffee_logs
      ADD COLUMN status TEXT NOT NULL DEFAULT 'published'
      CHECK(status IN ('draft', 'published'));
    ```
  - `DEFAULT 'published'`로 기존 row 자동 backfill.

- [ ] **`008_add_status_to_coffee_logs.down.sql` 생성**
  - Target: `backend/db/migrations/008_add_status_to_coffee_logs.down.sql`
  - 내용:
    ```sql
    ALTER TABLE coffee_logs DROP COLUMN status;
    ```
  - SQLite 3.35+ 에서 DROP COLUMN을 지원한다. 프로젝트 README/배포 환경의 SQLite 버전을 신뢰한다.

이 태스크는 독립적으로 실행 가능하다.

---

## 2. 도메인 모델 확장

- [ ] **`LogStatus` 타입과 상수 추가**
  - Target: `backend/internal/domain/log.go`
  - `LogType` 정의 아래에 추가:
    ```go
    type LogStatus string

    const (
        LogStatusDraft     LogStatus = "draft"
        LogStatusPublished LogStatus = "published"
    )
    ```

- [ ] **`CoffeeLog` 구조체에 `Status` 필드 추가**
  - Target: `backend/internal/domain/log.go`
  - `CoffeeLog` 내부에 `Status LogStatus` 추가. 위치는 `LogType` 아래.

태스크 1과 병렬 가능.

---

## 3. sqlc 쿼리 갱신 및 생성

- [ ] **`coffee_logs.sql`의 InsertLog/GetLogByID/UpdateLog에 status 추가**
  - Target: `backend/db/queries/coffee_logs.sql`
  - `InsertLog`: 컬럼 목록과 VALUES에 `status` 추가
    ```sql
    -- name: InsertLog :exec
    INSERT INTO coffee_logs (id, user_id, recorded_at, companions, log_type, status, memo, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
    ```
  - `GetLogByID`: SELECT 컬럼 목록에 `status` 추가
  - `ListLogs`: 본 쿼리는 raw SQL로 대체될 예정이지만 sqlc generate 호환을 위해 SELECT에 status 추가하고 status 필터는 raw SQL로 처리.
  - `UpdateLog`: SET 절에 `status = ?` 추가
    ```sql
    -- name: UpdateLog :exec
    UPDATE coffee_logs
    SET recorded_at = ?, companions = ?, status = ?, memo = ?, updated_at = ?
    WHERE id = ? AND user_id = ?;
    ```

- [ ] **sqlc 코드 재생성**
  - Command: `cd backend && sqlc generate`
  - 생성된 `internal/db/*.sql.go`에 `Status` 필드와 파라미터가 반영되었는지 확인.

태스크 1, 2 완료 후 실행.

---

## 4. Suggestion 쿼리에 status 조건 추가

- [ ] **자동완성 쿼리에서 published만 집계하도록 변경**
  - Target: `backend/db/queries/suggestions.sql` 및/또는 `backend/internal/repository/suggestion_repository.go`
  - 모든 자동완성 SQL의 `coffee_logs` JOIN/WHERE 절에 `coffee_logs.status = 'published'` 추가.
  - sqlc 쿼리는 sqlc generate 재실행, raw SQL은 직접 수정.
  - 변경 후 기존 suggestion 테스트가 깨지지 않는지 확인 (테스트 fixture는 status='published' 기본값으로 생성되므로 회귀 없을 것으로 예상).

- [ ] **Suggestion 회귀 테스트 추가**
  - Target: `backend/internal/repository/suggestion_repository_test.go`
  - 테스트 케이스: 동일 사용자 소유의 draft 로그에 들어간 tasting_tag가 자동완성 응답에 포함되지 않는다.
  - 동일 사용자의 published 로그 태그와 draft 로그 태그를 모두 시드하고, 응답이 published 태그만 포함하는지 검증.

태스크 1, 3 완료 후 실행.

---

## 5. Repository 레이어 status 지원

- [ ] **`ListFilter`에 `Status *string` 추가**
  - Target: `backend/internal/repository/log_repository.go`
  - 기존 `ListFilter` 구조체에 `Status *string` 필드 추가.

- [ ] **`SQLiteLogRepository.CreateLog`에 status 전달**
  - Target: `backend/internal/repository/log_repository.go`
  - `qtx.InsertLog(...)` 호출 시 `Status: string(log.Status)` 추가.

- [ ] **`SQLiteLogRepository.UpdateLog`에 status 전달**
  - Target: `backend/internal/repository/log_repository.go`
  - `qtx.UpdateLog(...)` 호출 시 `Status: string(log.Status)` 추가.

- [ ] **`SQLiteLogRepository.ListLogs` raw SQL 수정**
  - Target: `backend/internal/repository/log_repository.go`
  - SELECT 컬럼 목록에 `status` 추가:
    ```go
    query := `SELECT id, user_id, recorded_at, companions, log_type, status, memo, created_at, updated_at
        FROM coffee_logs WHERE user_id = ?`
    ```
  - status 필터 분기 추가(LogType 필터 분기 위에 둔다):
    ```go
    if filter.Status == nil || *filter.Status == "published" {
        query += ` AND status = 'published'`
    } else if *filter.Status == "draft" {
        query += ` AND status = 'draft'`
    }
    // "all" 인 경우 조건 추가하지 않음
    ```
  - `rows.Scan(...)`에 `&statusStr` 추가하고 `f.Status = domain.LogStatus(statusStr)`로 매핑.

- [ ] **`coffeeLogToFull` 매핑 함수에 status 추가**
  - Target: `backend/internal/repository/log_repository.go`
  - `coffeeLogToFull(row db.CoffeeLog)`에서 `Status: domain.LogStatus(row.Status)` 추가.

- [ ] **`GetLogByID`도 status 반영**
  - Target: `backend/internal/repository/log_repository.go`
  - sqlc 자동 생성 쿼리가 status를 반환하므로 `coffeeLogToFull`만 갱신되면 자동 처리.

- [ ] **Repository 통합 테스트 추가**
  - Target: `backend/internal/repository/log_repository_test.go`
  - 기존 `setupTestDB`에서 마이그레이션 목록에 `008_add_status_to_coffee_logs.up.sql` 추가.
  - 신규 케이스:
    1. `CreateLog`에서 `Status = 'draft'`인 로그를 만들고 `GetLogByID`로 조회 시 status가 draft로 반환되는지
    2. `ListLogs(status=nil)` → published만 반환
    3. `ListLogs(status="draft")` → draft만 반환
    4. `ListLogs(status="all")` → 모두 반환
    5. `UpdateLog`로 status를 draft → published로 변경 시 정상 반영

태스크 3 완료 후 실행. 태스크 6과 병렬 가능(서비스 레이어 변경 전이라도 repository는 status를 그대로 통과시키도록).

---

## 6. Service 레이어 검증/전이/필터

- [ ] **`CreateLogRequest`, `UpdateLogRequest`에 `Status` 필드 추가**
  - Target: `backend/internal/service/log_service.go`
  - 두 구조체 모두 `Status domain.LogStatus` 필드 추가. 빈 값(`""`)은 기본값(`published`) 의미.

- [ ] **`ListLogsFilter`에 `Status *string` 추가**
  - Target: `backend/internal/service/log_service.go`
  - 빈 값/nil이면 기본값 `published`로 정규화.

- [ ] **`normalizeListFilter`에서 status 정규화**
  - Target: `backend/internal/service/log_service.go`
  - 허용 값: `"published"`, `"draft"`, `"all"`.
  - 허용 외 값은 `ValidationError("status", "published, draft, all 중 하나여야 합니다")`.
  - nil/빈 값 → `"published"`로 기본값 설정 후 repository.ListFilter에 전달.

- [ ] **`normalizeCreateRequest`에 status 분기 추가**
  - Target: `backend/internal/service/log_service.go`
  - status 정규화 헬퍼:
    ```go
    func validateLogStatus(field string, status domain.LogStatus) (domain.LogStatus, error) {
        if status == "" {
            return domain.LogStatusPublished, nil
        }
        switch status {
        case domain.LogStatusDraft, domain.LogStatusPublished:
            return status, nil
        default:
            return "", newValidationError(field, "draft 또는 published만 허용됩니다")
        }
    }
    ```
  - normalize 후 status에 따라 cafe/brew detail 검증 함수를 분기.

- [ ] **draft 전용 detail 검증 함수 추가**
  - Target: `backend/internal/service/log_service.go`
  - `normalizeCafeDetailDraft(detail *domain.CafeDetail) (*domain.CafeDetail, error)`:
    - detail이 nil이면 빈 CafeDetail로 시작.
    - cafe_name과 coffee_name 둘 다 trim 후 빈 값이면 `ValidationError("cafe", "카페 이름이나 메뉴 이름 중 하나는 입력해야 합니다")`.
    - 형식 검증(rating range, roast_level enum)은 값이 있을 때만 적용.
    - 모든 빈 문자열 필드는 그대로 빈 문자열로 저장(기존 `validateRequiredString`을 호출하지 않음).
  - `normalizeBrewDetailDraft(detail *domain.BrewDetail) (*domain.BrewDetail, error)`:
    - bean_name과 brew_method 둘 다 비어 있으면 ValidationError.
    - brew_method가 비어 있지 않으면 enum 검증.
    - 그 외 형식 검증은 값이 있을 때만 적용.

- [ ] **`normalizeCreateRequest` 분기 적용**
  - Target: `backend/internal/service/log_service.go`
  - 의사 코드:
    ```go
    status, err := validateLogStatus("status", req.Status)
    if err != nil { ... }
    normalized.Status = status

    switch logType {
    case domain.LogTypeCafe:
        if status == domain.LogStatusDraft {
            normalized.Cafe, err = normalizeCafeDetailDraft(req.Cafe)
        } else {
            normalized.Cafe, err = normalizeCafeDetail(req.Cafe) // 기존
        }
    case domain.LogTypeBrew:
        ...
    }
    ```

- [ ] **`normalizeUpdateRequest` 분기 + 전이 검증**
  - Target: `backend/internal/service/log_service.go`
  - existing log status를 인자로 받아 전이 검증:
    ```go
    func normalizeUpdateRequest(req UpdateLogRequest, existing domain.CoffeeLogFull) (UpdateLogRequest, error)
    ```
  - 호출부(`UpdateLog`)에서 `existing` 전체를 넘기도록 수정.
  - status 미지정(빈 값) → `existing.Status` 사용.
  - status 지정 + existing이 published + req가 draft → `ValidationError("status", "발행된 로그를 드래프트로 되돌릴 수 없습니다")`.
  - draft → published 또는 draft → draft 또는 published → published는 허용.
  - 검증된 status로 cafe/brew detail 검증 함수 분기(create와 동일).

- [ ] **`DefaultLogService.CreateLog`에서 status를 도메인 객체에 저장**
  - Target: `backend/internal/service/log_service.go`
  - `domain.CoffeeLogFull{ CoffeeLog: domain.CoffeeLog{ ..., Status: normalizedReq.Status, ... } }` 매핑 추가.

- [ ] **`DefaultLogService.UpdateLog`에서 status 매핑**
  - Target: `backend/internal/service/log_service.go`
  - `updated.Status = normalizedReq.Status` (전이 검증 통과한 값 사용).

- [ ] **`DefaultLogService.ListLogs`에서 status 정규화 결과를 repository로 전달**
  - Target: `backend/internal/service/log_service.go`
  - `repoFilter.Status = ...` 매핑.

- [ ] **Service 단위 테스트 추가**
  - Target: `backend/internal/service/log_service_test.go`
  - 케이스:
    1. cafe draft 생성 — cafe_name만 채움 → 통과
    2. cafe draft 생성 — 둘 다 빈 값 → ValidationError(field=cafe)
    3. brew draft 생성 — bean_name만 채움 → 통과
    4. brew draft 생성 — bean_name 빈 값, brew_method 누락 → ValidationError
    5. published 회귀 — 기존 모든 검증 그대로 동작
    6. UpdateLog: published → draft 시도 → ValidationError(field=status)
    7. UpdateLog: draft → published 시 published 검증 적용 (필수 필드 누락 시 실패)
    8. UpdateLog: status 미지정 → existing status 유지
    9. ListLogs filter status="invalid" → ValidationError
    10. ListLogs filter status=nil → repository로 "published" 전달

태스크 5 완료 후 실행.

---

## 7. Handler 레이어 status 노출

- [ ] **`coffeeLogResponse`에 `Status string` 추가**
  - Target: `backend/internal/handler/log_handler.go`
  - `LogType` 다음에 `Status string \`json:"status"\`` 추가.

- [ ] **`createLogRequest`, `updateLogRequest`에 `Status *string` 추가**
  - Target: `backend/internal/handler/log_handler.go`
  - 두 구조체 모두 `Status *string \`json:"status"\`` 추가 (omitempty 없이, nil이면 빈 값으로 인식).

- [ ] **`logToResponse`에서 status 매핑**
  - Target: `backend/internal/handler/log_handler.go`
  - `Status: string(log.Status)` 추가.

- [ ] **`CreateLog` 핸들러에서 status 전달**
  - Target: `backend/internal/handler/log_handler.go`
  - `service.CreateLogRequest` 생성 시 status 매핑:
    ```go
    var status domain.LogStatus
    if req.Status != nil {
        status = domain.LogStatus(*req.Status)
    }
    ```
  - service에 status 전달.

- [ ] **`UpdateLog` 핸들러에서 status 전달**
  - 동일 패턴.

- [ ] **`ListLogs` 핸들러에서 status 쿼리스트링 파싱**
  - Target: `backend/internal/handler/log_handler.go`
  - `q.Get("status")`가 비어 있지 않으면 `filter.Status = &s`로 전달.
  - 검증은 service에 위임.

- [ ] **Handler 단위 테스트 추가**
  - Target: `backend/internal/handler/log_handler_test.go`
  - 케이스:
    1. POST /logs with status="draft" + cafe_name 채움 → 201, 응답에 status="draft"
    2. POST /logs with status="draft" + cafe_name 빈 값 + coffee_name 빈 값 → 400 with field=cafe
    3. PUT /logs/{id} 기존 published → status="draft" 시도 → 400 with field=status
    4. GET /logs?status=draft → 드래프트 로그만 반환
    5. GET /logs (status 미지정) → published만 반환
    6. GET /logs?status=invalid → 400

태스크 6 완료 후 실행.

---

## 8. OpenAPI 스키마 업데이트

- [ ] **`LogStatus` enum 추가**
  - Target: `docs/openapi.yml`
  - `LogType` 정의 아래에 추가:
    ```yaml
    LogStatus:
      type: string
      enum: [draft, published]
      description: |
        draft = 작성 중 임시 저장. published = 발행됨. 기본값은 published.
    ```

- [ ] **`CoffeeLogResponse`에 `status` 필드 추가**
  - Target: `docs/openapi.yml`
  - `required` 배열에 `status` 추가, properties에 `$ref: '#/components/schemas/LogStatus'` 정의.

- [ ] **`CreateLogRequest`, `UpdateLogRequest`에 `status` 필드 추가**
  - Target: `docs/openapi.yml`
  - 두 스키마 모두 properties에 `status: $ref: '#/components/schemas/LogStatus'` 추가 (required에는 포함하지 않음, 기본값 published).

- [ ] **`GET /api/v1/logs`에 `status` 쿼리 파라미터 추가**
  - Target: `docs/openapi.yml`
  - parameters 배열에 추가:
    ```yaml
    - name: status
      in: query
      schema:
        type: string
        enum: [draft, published, all]
        default: published
      description: |
        목록 필터. 기본값 published(완성된 로그만).
        draft는 작성 중인 로그, all은 전체.
    ```

- [ ] **버전 번호 갱신**
  - Target: `docs/openapi.yml`의 `info.version`을 한 단계 올림 (현재 0.2.0 → 0.3.0).

태스크 7 완료 후 실행.

---

## 9. 프론트엔드 타입 재생성

- [ ] **타입 자동 생성**
  - Command: `cd frontend && npm run generate`
  - `frontend/src/types/schema.ts`에 `LogStatus`, `CoffeeLogResponse.status`, `CreateLogRequest.status`, `UpdateLogRequest.status`가 반영되었는지 확인.

- [ ] **`types/log.ts`에서 `LogStatus` 재수출**
  - Target: `frontend/src/types/log.ts`
  - 다음 라인 추가:
    ```ts
    export type LogStatus = components['schemas']['LogStatus']
    ```

태스크 8 완료 후 실행.

---

## 10. API 클라이언트 status 파라미터 지원

- [ ] **`ListLogsParams`에 status 추가**
  - Target: `frontend/src/api/logs.ts`
  - `status?: 'draft' | 'published' | 'all'` 필드 추가.
  - `getLogs`에서 `if (params.status) q.set('status', params.status)` 추가.

태스크 9 완료 후 실행.

---

## 11. logFormState에 status 통합

- [ ] **`LogFormState`에 `status: LogStatus` 추가**
  - Target: `frontend/src/pages/logFormState.ts`
  - import: `import type { LogStatus, ... } from '../types/log'`
  - `LogFormState` 인터페이스 상단에 `status: LogStatus` 추가.

- [ ] **`createEmptyFormState`에서 기본값 설정**
  - Target: `frontend/src/pages/logFormState.ts`
  - `status: 'published'` 추가 (신규 폼은 발행 의도로 시작, 사용자가 "임시 저장" 클릭 시 buildLogPayload에서 override).

- [ ] **`logToFormState`에서 status 전이**
  - Target: `frontend/src/pages/logFormState.ts`
  - `status: log.status` 매핑 추가.

- [ ] **`cloneToFormState`, `recipeToFormState`, `presetToFormState`는 status='published' 유지**
  - Target: `frontend/src/pages/logFormState.ts`
  - `createEmptyFormState`로부터 시작하므로 자연스럽게 published. 명시적으로 `state.status = 'published'`를 추가해 가독성 보강.

- [ ] **`buildLogPayload`에 status 옵션 추가**
  - Target: `frontend/src/pages/logFormState.ts`
  - 시그니처 변경:
    ```ts
    interface BuildLogPayloadOptions {
      status: LogStatus
    }
    export function buildLogPayload(state: LogFormState, options: BuildLogPayloadOptions): CreateLogInput
    ```
  - `payload.status = options.status`를 결과에 포함.
  - draft 모드일 때:
    - cafe: cafe_name과 coffee_name이 빈 문자열이라도 그대로 전송 (trim은 그대로 적용).
    - brew: bean_name이 빈 문자열이어도 전송.
    - rating, recorded_at 등 형식 변환은 published와 동일하게 수행.
  - published 모드는 기존 로직 유지.
  - 호출부(LogFormPage)는 두 군데에서 옵션을 명시한다.

- [ ] **`canSaveAsDraft(state: LogFormState): boolean` 헬퍼 추가**
  - Target: `frontend/src/pages/logFormState.ts`
  - 구현:
    ```ts
    export function canSaveAsDraft(state: LogFormState): boolean {
      if (state.logType === 'cafe') {
        return state.cafe.cafeName.trim() !== '' || state.cafe.coffeeName.trim() !== ''
      }
      // brew는 bean_name이나 brew_method가 채워져 있어야 함
      // brewMethod는 createEmptyFormState에서 기본값 'pour_over'이지만, 사용자가 변경 없이 그대로 두는 경우도 의도적 선택으로 간주
      return state.brew.beanName.trim() !== '' || state.brew.brewMethod !== ''
    }
    ```

- [ ] **`logFormState.test.ts`에 단위 테스트 추가**
  - Target: `frontend/src/pages/logFormState.test.ts`
  - 케이스:
    1. `createEmptyFormState().status === 'published'`
    2. `logToFormState`가 응답의 status('draft')를 폼 상태로 전이
    3. `cloneToFormState`/`recipeToFormState`/`presetToFormState`가 항상 `status='published'`로 리셋
    4. `buildLogPayload(state, { status: 'draft' })` — payload.status === 'draft'
    5. `buildLogPayload(state, { status: 'published' })` — 기존 동작 회귀
    6. `canSaveAsDraft`:
       - 빈 cafe 폼 → false
       - cafe_name 입력 후 → true
       - coffee_name 입력 후 → true
       - 빈 brew 폼(brewMethod 기본값 있음) → true
       - bean_name 입력 후 → true

태스크 10 완료 후 실행.

---

## 12. LogFormPage 임시 저장 버튼 및 라우팅

- [ ] **"임시 저장" 버튼 추가**
  - Target: `frontend/src/pages/LogFormPage.tsx`
  - import에 `canSaveAsDraft`, `LogStatus` 추가.
  - `Layout` actions 영역에 새 버튼:
    - 라벨: "임시 저장" (pending 시 "저장 중...")
    - className: 기본 ghost 스타일(예: `border` 기반, `inline-flex items-center justify-center whitespace-nowrap rounded-full border border-amber-900/15 bg-amber-50 px-4 py-2 text-sm font-semibold text-amber-950 transition hover:bg-amber-100`)
    - 위치: "기록 추가/변경 저장" 버튼의 왼쪽
    - 표시 조건: 신규 작성(`!isEditMode`) 또는 (`isEditMode && log?.status === 'draft'`)
    - disabled: `!canSaveAsDraft(form) || activeMutation.isPending || (isEditMode && isLoading)`
    - onClick: `handleDraftSave()` 호출

- [ ] **`handleDraftSave` 핸들러 작성**
  - Target: `frontend/src/pages/LogFormPage.tsx`
  - 로직:
    ```ts
    async function handleDraftSave() {
      setFieldErrors({})
      const payload = buildLogPayload(form, { status: 'draft' })
      try {
        if (isEditMode && id) {
          await updateMutation.mutateAsync(payload)
        } else {
          await createMutation.mutateAsync(payload)
        }
        navigate('/')
      } catch (err) {
        if (err instanceof ApiError && err.field) {
          setFieldErrors({ [err.field]: err.message })
        }
      }
    }
    ```

- [ ] **기존 `handleSubmit`은 published로 명시**
  - Target: `frontend/src/pages/LogFormPage.tsx`
  - `buildLogPayload(form)` 호출을 `buildLogPayload(form, { status: 'published' })`로 변경. (또는 published 발행 의도일 때 form.status를 published로 강제)

- [ ] **수정 모드 published 로그에서 "임시 저장" 버튼 숨김**
  - Target: `frontend/src/pages/LogFormPage.tsx`
  - 위 표시 조건이 이를 처리한다 — `log?.status === 'published' && isEditMode`이면 버튼 미렌더.

- [ ] **드래프트 → published 전환 흐름 자동 동작 확인**
  - 별도 코드 변경 없음. "변경 저장" 버튼이 buildLogPayload(..., { status: 'published' })를 호출하므로 백엔드가 자동으로 전환.

- [ ] **LogFormPage 컴포넌트 테스트 추가**
  - Target: `frontend/src/pages/LogFormPage.test.tsx`
  - 케이스:
    1. 신규 cafe 폼 — "임시 저장" 버튼 표시
    2. 빈 폼 — "임시 저장" disabled
    3. cafe_name 입력 후 — "임시 저장" enabled
    4. "임시 저장" 클릭 → `createLog` mock에 `status: 'draft'` 전달, 성공 후 홈(`/`)으로 navigate
    5. 신규 brew 폼 — "임시 저장" 버튼 표시
    6. 수정 모드 + log.status='published' — "임시 저장" 버튼 미표시
    7. 수정 모드 + log.status='draft' — "임시 저장" 버튼 표시, 클릭 시 `updateLog` 호출 with status='draft'
    8. 수정 모드 + log.status='draft' + "변경 저장" 클릭 → `updateLog`에 `status: 'published'` 전달

태스크 11 완료 후 실행.

---

## 13. HomePage 드래프트 섹션 추가

- [ ] **드래프트 목록 별도 호출**
  - Target: `frontend/src/pages/HomePage.tsx`
  - 기존 `useLogList` 호출 위에 추가:
    ```ts
    const { data: draftData } = useLogList({ status: 'draft', limit: 12 })
    const drafts = useMemo(() => draftData?.pages.flatMap((p) => p.items) ?? [], [draftData?.pages])
    ```
  - 기존 published 호출에 `status: 'published'`를 명시(기본값이지만 가독성 향상).

- [ ] **드래프트 섹션 렌더**
  - Target: `frontend/src/pages/HomePage.tsx`
  - FilterBar 위 또는 카운트 박스 아래에 조건부 렌더:
    ```tsx
    {drafts.length > 0 ? (
      <section className="space-y-3">
        <header className="flex items-center justify-between">
          <h2 className="text-base font-semibold text-stone-900">작성 중인 기록</h2>
          <span className="text-xs text-stone-500">{drafts.length}건</span>
        </header>
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {drafts.map((log) => <LogCard key={log.id} log={log} />)}
        </div>
      </section>
    ) : null}
    ```
  - 무한 스크롤은 적용하지 않고 첫 페이지만 표시.

- [ ] **HomePage 테스트 보강(가능하면)**
  - Target: 신규 또는 기존 `frontend/src/pages/HomePage.test.tsx`(없으면 생성)
  - 케이스 우선순위:
    1. drafts 0건 + published 1건 → 드래프트 섹션 미렌더
    2. drafts 1건 + published 1건 → 두 섹션 모두 렌더, 드래프트 섹션이 위
  - HomePage 테스트가 미존재하면 시간 제약 시 E2E 수동 검증으로 대체 가능.

태스크 12 완료 후 실행. 태스크 14와 병렬 가능.

---

## 14. LogCard 드래프트 시각 처리

- [ ] **드래프트 뱃지 + 점선 테두리**
  - Target: `frontend/src/components/LogCard.tsx`
  - `log.status === 'draft'`일 때 카드 컨테이너의 className에 `border-dashed`를 추가하고, 카드 우측 상단(또는 RatingDisplay 위치)에 "작성 중" 뱃지를 표시.
  - 예시:
    ```tsx
    const isDraft = log.status === 'draft'
    const containerClass = `... ${isDraft ? 'border-dashed' : ''} ...`
    ```
  - 뱃지 디자인: amber 배경 + 작은 라운드 + "작성 중" 한글.

- [ ] **드래프트 카드의 to 분기**
  - Target: `frontend/src/components/LogCard.tsx`
  - `<Link to={...}>`의 to를 다음과 같이 분기:
    ```tsx
    to={isDraft ? `/logs/${log.id}/edit` : `/logs/${log.id}`}
    ```

- [ ] **결측 필드 fallback**
  - Target: `frontend/src/components/LogCard.tsx`
  - `title`, `subtitle` 계산 시 빈 문자열 fallback:
    ```tsx
    const title =
      log.log_type === 'cafe'
        ? (log.cafe.coffee_name || '메뉴 미작성')
        : (log.brew.bean_name || '원두 미작성')
    ```
  - subtitle도 cafe_name이 빈 문자열일 수 있으므로 동일 처리.

- [ ] **LogCard 테스트 보강**
  - Target: `frontend/src/components/LogCard.test.tsx`
  - 케이스:
    1. status='draft' + 빈 coffee_name → "메뉴 미작성" 표시 + "작성 중" 뱃지 + 링크 to=`/logs/{id}/edit`
    2. status='published' → 기존 동작 회귀 (링크 to=`/logs/{id}`, 뱃지 없음)

태스크 12 완료 후 실행.

---

## 15. LogDetailPage 드래프트 redirect

- [ ] **드래프트 진입 시 수정 폼으로 redirect**
  - Target: `frontend/src/pages/LogDetailPage.tsx`
  - 로그 로드 후:
    ```tsx
    useEffect(() => {
      if (log?.status === 'draft') {
        navigate(`/logs/${log.id}/edit`, { replace: true })
      }
    }, [log, navigate])
    ```
  - replace: true로 history에 남기지 않음.

태스크 12 완료 후 실행. 태스크 13, 14와 병렬 가능.

---

## 16. 검증

- [ ] **백엔드 테스트 실행**
  - Command: `cd backend && go test ./...`
  - 모든 패키지 통과 확인. 마이그레이션, repository, service, handler 테스트가 모두 그린이어야 한다.

- [ ] **프론트엔드 단위 테스트 실행**
  - Command: `cd frontend && npm test`
  - 신규 추가된 logFormState/LogFormPage/LogCard 테스트 모두 통과.

- [ ] **타입 체크**
  - Command: `cd frontend && npm run build` 또는 `npx tsc --noEmit -p tsconfig.app.json`
  - TypeScript 에러 없음 확인.

- [ ] **E2E 수동 검증** (PR 생성 직전)
  1. 새 cafe 폼 진입 → cafe_name만 입력 → "임시 저장" → 홈 화면에 "작성 중인 기록" 섹션이 노출되는지
  2. 드래프트 카드의 외형(점선 테두리, "작성 중" 뱃지) 확인
  3. 드래프트 카드 탭 → 수정 폼이 이전 입력값과 함께 열림 확인
  4. 나머지 필드 채우고 "변경 저장" → published로 전환되어 드래프트 섹션에서 사라지고 일반 목록에 등장
  5. brew 드래프트도 동일 흐름으로 작동 확인
  6. 드래프트 수정 화면에서 "임시 저장" 버튼 노출, 클릭 시 status가 draft로 유지됨
  7. published 로그 수정 화면에서는 "임시 저장" 버튼이 표시되지 않음
  8. 드래프트 카드의 복제 버튼 → 복제된 새 폼은 published 의도로 시작 (draft 아님)
  9. 드래프트 삭제 → 홈에서 사라지는지
  10. 자동완성: 드래프트에만 입력한 tasting_tag가 자동완성 응답에 노출되지 않는지(개발자 도구 네트워크에서 확인)

- [ ] **백엔드 마이그레이션 다운 검증**
  - 로컬에서 `migrate down 1` 후 `migrate up 1` 실행 → 데이터 손실 없이 복구되는지 (선택 사항, 실험 환경)

태스크 1-15 모두 완료 후 실행.

---

## 17. 문서 정리 (PR 직전)

- [ ] **`docs/arch/` 갱신 필요 여부 확인**
  - 본 이슈에서 추가한 status 필드는 도메인 모델 확장 수준이며, 아키텍처 결정 변경이 아니므로 `docs/arch/backend.md`, `docs/arch/frontend.md`는 변경 불필요. 그러나 검토 단계에서 새로 추가된 도메인 분리 설계(예: NOT NULL + 빈 문자열 전략)를 backend.md "데이터 모델" 절에 한 줄 추가할지 검토.

- [ ] **`docs/spec.md` 최종 검토**
  - 6.5 드래프트 저장 섹션이 실제 구현과 일치하는지 확인. 차이 발생 시 spec.md 갱신 + Last updated 라인 갱신.

- [ ] **`docs/backlog.md` 정리**
  - 본 이슈로 발생한 후속 backlog 후보(예: 드래프트 자동 만료 정책, brewMethod 기본값 모호성, draft 카드 키보드 접근성)를 backlog에 추가할지 검토 후 항목 추가.

태스크 16 완료 후 실행.
