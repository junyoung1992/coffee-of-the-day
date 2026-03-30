# Phase 1 리팩터링 결과

## 개요

이 문서는 `review/code_review_phase_1.md`에서 도출된 개선 항목을 Phase 2 진입 전에 처리한 결과를 정리한 것입니다.

---

## 즉시 수정 완료 (1–3)

### 1. SQLite foreign key 활성화

**파일:** `backend/cmd/server/main.go`

**변경 내용:**

```go
// 이전
db, err := sql.Open("sqlite3", cfg.DBPath)

// 이후
db, err := sql.Open("sqlite3", cfg.DBPath+"?_foreign_keys=on")
```

DSN 수준에서 `_foreign_keys=on`을 지정하여 connection pool의 모든 연결에 외래키 강제가 일괄 적용되도록 했습니다.
코드로 `PRAGMA foreign_keys = ON`을 직접 실행하면 특정 연결에만 적용될 수 있어, DSN 방식이 더 안전합니다.

---

### 2. 요청 body 검증 강화

**파일:** `backend/internal/handler/log_handler.go`

**변경 내용:** `CreateLog`, `UpdateLog` 두 핸들러 모두 적용

```go
// 이전
if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }

// 이후
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB 제한
dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()
if err := dec.Decode(&req); err != nil { ... }
```

- `MaxBytesReader`: 1 MiB 초과 요청을 사전에 차단합니다.
- `DisallowUnknownFields`: `coffee_nmae`처럼 오타가 있는 필드를 조용히 무시하지 않고 에러로 반환합니다.

---

### 3. Repository domain invariant 정리

**파일:** `backend/internal/repository/log_repository.go`

**변경 내용:** `loadDetail` 함수에서 상세 레코드 누락을 정상 상태로 두던 로직 수정

```go
// 이전 — ErrNoRows를 정상으로 처리
if errors.Is(err, sql.ErrNoRows) {
    return nil
}

// 이후 — 데이터 손상으로 간주하여 에러 반환
if errors.Is(err, sql.ErrNoRows) {
    return fmt.Errorf("cafe detail missing for log %s: %w", f.ID, ErrNotFound)
}
```

`cafe` / `brew` 로그에서 상세 레코드가 없는 경우는 partial failure 또는 수동 DB 변경에 의한 데이터 손상입니다.
이 상태를 조용히 허용하면 프론트의 discriminated union 타입이 런타임에 깨질 수 있으므로, 명시적인 에러로 승격했습니다.

---

## Phase 2 시작 전 수정 완료 (4–8)

### 4. 목록 조회 N+1 개선

**파일:** `backend/internal/repository/log_repository.go`

**문제:** `ListLogs`가 목록 1회 조회 후, 각 항목마다 `loadDetail`을 호출하는 N+1 패턴

**변경 내용:**

- `loadDetail` 루프 → `batchLoadCafe` / `batchLoadBrew` 분리 메서드로 교체
- 로그 ID를 타입별로 분류 후, `IN (?, ?, ...)` 쿼리로 한 번에 조회
- 결과: 목록 조회가 **최대 3쿼리**로 고정 (coffee_logs 1회 + cafe_logs 1회 + brew_logs 1회)

```go
// batchLoadCafe — IN 절로 카페 상세 일괄 조회
func (r *SQLiteLogRepository) batchLoadCafe(ctx context.Context, ids []string) (map[string]*domain.CafeDetail, error)

// batchLoadBrew — IN 절로 브루 상세 일괄 조회
func (r *SQLiteLogRepository) batchLoadBrew(ctx context.Context, ids []string) (map[string]*domain.BrewDetail, error)
```

---

### 5. `LogFormPage` 컴포넌트 분리

**파일:** `frontend/src/pages/LogFormPage.tsx`

**문제:** 400줄 이상의 단일 컴포넌트로, 공통 필드·카페 섹션·브루 섹션·스텝 조작이 모두 혼재

**변경 내용:** 아래 4개 서브 컴포넌트로 분리 (같은 파일 내)

| 컴포넌트 | 역할 |
|---|---|
| `LogTypeSection` | 로그 유형 선택 (cafe / brew) |
| `CommonFieldsSection` | recorded_at, companions, memo |
| `CafeFieldsSection` | 카페 관련 필드 전체 |
| `BrewFieldsSection` | 브루 관련 필드 + brew step 조작 포함 |

`ratio` 계산과 brew step 조작 로직(`updateStep`, `addStep`, `moveStep`, `removeStep`)이 `BrewFieldsSection` 안으로 이동했습니다.

---

### 6. 필드 단위 validation UX

**문제:** 서버 validation 에러가 상단 에러 박스로만 표시되어, 어떤 필드가 잘못됐는지 알 수 없음

**백엔드 변경:**

- `errorResponse`에 `Field *string` 추가 (`json:"field,omitempty"`)
- `writeServiceError`에서 `ValidationError` 발생 시 `error` (메시지)와 `field` (필드 경로)를 분리하여 응답

```go
// 이전
writeError(w, http.StatusBadRequest, err.Error()) // "cafe.cafe_name: 필수값입니다"

// 이후
writeJSON(w, http.StatusBadRequest, errorResponse{Error: ve.Message, Field: &field})
// { "error": "필수값입니다", "field": "cafe.cafe_name" }
```

**openapi.yml 변경:**

`ErrorResponse` 스키마에 `field` 프로퍼티 추가 후 `npm run generate`로 타입 재생성.

**프론트엔드 변경:**

- `ApiError`에 `field?: string` 추가
- `client.ts`에서 응답 body의 `field`를 파싱하여 `ApiError`에 포함
- `LogFormPage`에 `fieldErrors: Record<string, string>` state 추가
- submit 실패 시 `ApiError.field`가 있으면 해당 필드 아래 인라인 에러 표시
- `Field` 컴포넌트에 `error?: string` prop 추가

---

### 7. `recorded_at` 입력 정책 통일

**문제:** 백엔드는 RFC3339와 `YYYY-MM-DD` 두 형식을 허용하지만, 프론트는 `datetime-local`만 사용 (항상 RFC3339 전송)

**결정:** datetime-only로 고정. 백엔드를 프론트 실제 동작에 맞춤.

**백엔드 변경:**

```go
// 이전
for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} { ... }

// 이후 — date-only 제거
for _, layout := range []string{time.RFC3339Nano, time.RFC3339} { ... }
```

에러 메시지도 `"RFC3339 datetime 또는 YYYY-MM-DD 형식이어야 합니다"` → `"RFC3339 datetime 형식이어야 합니다"` 로 수정.

서비스 테스트에서 date-only 형식을 사용하던 3곳을 RFC3339로 일괄 수정.

---

### 8. UI 언어 통일

**문제:** 내비게이션 버튼과 페이지 제목이 한국어/영어 혼재

**결정:** 한국어 우선. 기술 용어(필드 레이블)는 영어 유지(AGENTS.md 주석 정책 준수).

| 이전 | 이후 | 위치 |
|---|---|---|
| `Coffee logs that stay readable` | `커피 기록` | HomePage 제목 |
| `Write today's log` | `오늘의 기록 추가` | HomePage 버튼 |
| `Quick add` | `빠른 추가` | HomePage 버튼 |
| `Log detail` | `기록 상세` | LogDetailPage 제목 |
| `Back to list` | `목록으로` | LogDetailPage, LogFormPage |
| `Edit log` | `수정` | LogDetailPage 버튼 |
| `Cafe details` | `카페 상세` | LogDetailPage 섹션 |
| `Brew details` | `브루 상세` | LogDetailPage 섹션 |
| `Delete log` / `Deleting...` | `삭제` / `삭제 중...` | LogDetailPage 버튼 |
| `Capture a coffee moment` | `커피 기록 추가` | LogFormPage 제목 |
| `Refine the cup` | `기록 수정` | LogFormPage 제목 |
| `Back to detail` | `상세로` | LogFormPage 버튼 |
| `Saving...` / `Save changes` / `Create log` | `저장 중...` / `변경 저장` / `기록 추가` | LogFormPage 버튼 |
| `Log type` / `Common fields` / `Cafe section` / `Brew section` | `로그 유형` / `공통 필드` / `카페 정보` / `브루 정보` | LogFormPage 섹션 제목 |

---

## 다음으로 미룬 항목

### Phase 4에서 해결 예정

**인증/사용자 위조 문제**

현재 `X-User-Id` 헤더를 그대로 신뢰하는 구조는 POC 전제에 묶인 임시 구현입니다.
Phase 4 JWT/cookie 기반 인증 도입 시 해결 예정이며, 그 전까지는 로컬/개발 환경 한정으로 운영해야 합니다.

---

### Phase 2에서 해결 예정

**필터 부재와 목록 탐색 부족**

타입/날짜 필터, 무한 스크롤 고도화는 Phase 2 탐색 기능에 포함됩니다.

---

### 별도 스케줄 미정

**E2E 자동화**

"생성 → 목록 반영 → 상세 → 수정 → 삭제" 핵심 플로우에 대한 Playwright 또는 Cypress 기반 자동화가 아직 없습니다.
특정 phase에 묶여 있지 않지만, phase가 늘어날수록 필요성이 커집니다.
Phase 2 기능 구현과 병행하여 최소 happy-path E2E를 추가하는 것을 권장합니다.

---

## 결과

Phase 2 진입 전 체크리스트:

- [x] SQLite foreign key 활성화
- [x] 요청 body size 제한 + unknown field 차단
- [x] 상세 누락을 데이터 손상으로 간주 (repository invariant)
- [x] 목록 N+1 → 최대 3쿼리로 개선
- [x] `LogFormPage` 4개 섹션 컴포넌트로 분리
- [x] 필드 단위 validation UX (backend field 응답 + frontend inline error)
- [x] `recorded_at` 형식 RFC3339 단일 정책으로 통일
- [x] UI 언어 한국어 우선으로 통일
- [ ] 인증 (Phase 4 예정)
- [ ] E2E 자동화 (별도 스케줄)
