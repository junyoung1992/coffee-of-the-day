# Issue #7 --- 즐겨찾기 프리셋

## 목표

자주 반복하는 카페+메뉴 또는 원두+추출방식 조합을 프리셋으로 저장하고, 새 로그 작성 시 프리셋을 선택하면 관련 필드가 자동 채움되도록 한다. 이를 통해 간격이 벌어진 반복 기록(예: "매주 금요일 카페")을 빠르게 입력할 수 있다.

---

## 데이터베이스 설계

### 테이블 구조

기존 `coffee_logs` + `cafe_logs` / `brew_logs` 패턴을 따른다. 프리셋도 공통 테이블 + 타입별 서브 테이블로 분리한다.

**`presets` 테이블 (공통):**

```sql
CREATE TABLE presets (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id),
    name         TEXT NOT NULL,
    log_type     TEXT NOT NULL CHECK(log_type IN ('cafe', 'brew')),
    last_used_at TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);
```

**`cafe_presets` 테이블:**

```sql
CREATE TABLE cafe_presets (
    preset_id    TEXT PRIMARY KEY REFERENCES presets(id) ON DELETE CASCADE,
    cafe_name    TEXT NOT NULL,
    coffee_name  TEXT NOT NULL,
    tasting_tags TEXT NOT NULL DEFAULT '[]'
);
```

**`brew_presets` 테이블:**

```sql
CREATE TABLE brew_presets (
    preset_id     TEXT PRIMARY KEY REFERENCES presets(id) ON DELETE CASCADE,
    bean_name     TEXT NOT NULL,
    brew_method   TEXT NOT NULL CHECK(brew_method IN (
                      'pour_over', 'immersion', 'aeropress',
                      'espresso', 'moka_pot', 'siphon', 'cold_brew', 'other'
                  )),
    recipe_detail TEXT,
    brew_steps    TEXT NOT NULL DEFAULT '[]'
);
```

migration 번호: `007` (현재 마지막이 `006`).

### 정렬 로직

프리셋 목록은 `last_used_at DESC NULLS LAST, created_at DESC` 순으로 정렬한다. 한 번도 사용되지 않은 프리셋은 `last_used_at`이 NULL이므로 마지막에 위치하며, 그 안에서는 최신 생성순.

---

## 백엔드 설계

기존 vertical slice 패턴을 따라 `preset` 전용 handler/service/repository를 추가한다.

### Domain

`backend/internal/domain/preset.go` 신규 생성.

```go
type Preset struct {
    ID         string
    UserID     string
    Name       string
    LogType    LogType
    LastUsedAt *string
    CreatedAt  string
    UpdatedAt  string
}

type CafePresetDetail struct {
    CafeName    string
    CoffeeName  string
    TastingTags []string
}

type BrewPresetDetail struct {
    BeanName     string
    BrewMethod   BrewMethod
    RecipeDetail *string
    BrewSteps    []string
}

type PresetFull struct {
    Preset
    Cafe *CafePresetDetail
    Brew *BrewPresetDetail
}
```

### Repository

`backend/internal/repository/preset_repository.go` 신규 생성.

interface:
```go
type PresetRepository interface {
    CreatePreset(ctx context.Context, preset domain.PresetFull) error
    GetPresetByID(ctx context.Context, presetID, userID string) (domain.PresetFull, error)
    ListPresets(ctx context.Context, userID string) ([]domain.PresetFull, error)
    UpdatePreset(ctx context.Context, preset domain.PresetFull) error
    DeletePreset(ctx context.Context, presetID, userID string) error
    UpdateLastUsedAt(ctx context.Context, presetID, userID string, usedAt string) error
}
```

`ListPresets`는 페이지네이션 없이 전체 반환한다. 프리셋은 사용자당 소량(수십 개 이하)이므로 커서 기반 페이지네이션은 과도하다.

트랜잭션 패턴: `CreatePreset`과 `UpdatePreset`에서 공통 테이블 + 서브 테이블을 하나의 tx로 묶는다. 기존 `log_repository.go`의 `CreateLog` 패턴과 동일.

sqlc 쿼리 파일: `backend/db/queries/presets.sql`, `backend/db/queries/cafe_presets.sql`, `backend/db/queries/brew_presets.sql` 신규 생성.

### Service

`backend/internal/service/preset_service.go` 신규 생성.

- `CreatePreset`: 이름 trim, log_type 검증, UUID 생성, 타임스탬프 설정
- `GetPreset`: 조회 + 소유권 확인
- `ListPresets`: userID로 전체 조회
- `UpdatePreset`: 이름/세부 필드 업데이트. log_type 변경 불가 (로그 수정과 동일한 제약)
- `DeletePreset`: 삭제 + 소유권 확인
- `UsePreset`: `last_used_at`을 현재 시각으로 갱신

### Handler

`backend/internal/handler/preset_handler.go` 신규 생성.

JSON 요청/응답 타입은 handler 내부에 정의한다 (기존 `log_handler.go` 패턴).

엔드포인트:
- `POST /api/v1/presets` → `CreatePreset`
- `GET /api/v1/presets` → `ListPresets`
- `GET /api/v1/presets/{id}` → `GetPreset`
- `PUT /api/v1/presets/{id}` → `UpdatePreset`
- `DELETE /api/v1/presets/{id}` → `DeletePreset`
- `POST /api/v1/presets/{id}/use` → `UsePreset`

### 라우터 등록

`cmd/server/main.go`의 인증 필요 라우트 그룹에 추가:

```go
r.Route("/presets", func(r chi.Router) {
    r.Post("/", presetHandler.CreatePreset)
    r.Get("/", presetHandler.ListPresets)
    r.Get("/{id}", presetHandler.GetPreset)
    r.Put("/{id}", presetHandler.UpdatePreset)
    r.Delete("/{id}", presetHandler.DeletePreset)
    r.Post("/{id}/use", presetHandler.UsePreset)
})
```

---

## API 설계 (OpenAPI)

`docs/openapi.yml`에 추가할 주요 스키마:

- `PresetType` (= LogType 재사용 가능, 별도 enum 불필요)
- `CafePresetDetail`: cafe_name, coffee_name, tasting_tags
- `BrewPresetDetail`: bean_name, brew_method, recipe_detail, brew_steps
- `PresetResponse`: id, user_id, name, log_type, last_used_at, created_at, updated_at, cafe?, brew?
- `CreatePresetRequest`: name, log_type, cafe?, brew?
- `UpdatePresetRequest`: name, cafe?, brew?
- `ListPresetsResponse`: items 배열

`/api/v1/presets/{id}/use` 응답은 `204 No Content`로 처리한다 (갱신된 프리셋 데이터를 다시 받을 필요 없음). 프론트엔드에서 optimistic update로 처리.

---

## 프론트엔드 설계

### 타입

`frontend/src/types/preset.ts` 신규 생성. `schema.ts`에서 생성된 타입 기반으로 discriminated union 정의 (기존 `log.ts` 패턴).

### API 함수

`frontend/src/api/presets.ts` 신규 생성.
- `getPresets()`, `getPreset(id)`, `createPreset(body)`, `updatePreset(id, body)`, `deletePreset(id)`, `usePreset(id)`

### Hooks

`frontend/src/hooks/usePresets.ts` 신규 생성.
- `usePresetList()`: 전체 프리셋 목록 조회 (useQuery)
- `usePreset(id)`: 단건 조회 (useQuery)
- `useCreatePreset()`: 생성 mutation
- `useUpdatePreset(id)`: 수정 mutation
- `useDeletePreset()`: 삭제 mutation
- `useUsePreset()`: 사용 기록 mutation (optimistic update로 목록 캐시의 last_used_at 갱신)

### 로그 작성 폼 연동

`LogFormPage.tsx`에 프리셋 선택 UI를 추가한다.

**진입 방식:** 폼 상단(LogTypeSection 바로 아래)에 프리셋 선택 영역을 추가.
- 현재 선택된 `logType`에 해당하는 프리셋만 표시
- 프리셋 선택 시 `presetToFormState()` 함수로 폼 상태를 채움 (기존 `cloneToFormState()`와 유사한 패턴)
- edit 모드에서는 프리셋 선택 영역 미노출

**`logFormState.ts`에 추가:**
- `presetToFormState(preset: PresetFull): LogFormState` 함수

### 로그 상세 화면 연동

`LogDetailPage.tsx`에 "프리셋으로 저장" 버튼을 추가한다.
- 기존 actions 영역(목록으로, 복제, 수정 버튼이 있는 곳)에 배치
- 클릭 시 프리셋 이름 입력을 위한 모달 또는 인라인 입력 표시
- 로그의 관련 필드를 추출하여 `createPreset` API 호출

### 프리셋 관리 화면

`frontend/src/pages/PresetsPage.tsx` 신규 생성.
- 프리셋 목록 표시 (카드 형태, log_type 구분)
- 각 프리셋 카드에 수정/삭제 액션
- 수정 시 인라인 편집 또는 별도 모달
- 라우터에 `/presets` 경로 추가 (ProtectedRoute 내부)

### 네비게이션

프리셋 관리 화면으로의 진입점이 필요하다. Layout 또는 HomePage에 "프리셋 관리" 링크를 추가한다.

---

## 수정하지 않는 것

- 기존 로그 CRUD API 및 테이블 구조 (프리셋은 독립 엔티티)
- 기존 복제 기능 (`cloneToFormState`) 로직 자체는 변경하지 않음
- 인증/권한 체계 (기존 JWT 미들웨어 그대로 사용)
- 페이지네이션 방식 (프리셋 목록은 전체 조회)

---

## 테스트 전략

### 백엔드
- **Unit tests**: `preset_service.go`의 검증 로직 (이름 필수, log_type 검증, 필드 trim 등)
- **Integration tests**: `preset_repository.go`의 CRUD + 정렬 로직 (last_used_at 기준 정렬 검증)
- **Handler tests**: `preset_handler.go`의 HTTP 요청/응답 변환, 에러 매핑

### 프론트엔드
- **Unit tests**: `presetToFormState()` 함수, `usePresets` hooks
- **Component tests**: 프리셋 선택 UI 동작, 프리셋 관리 페이지
- **E2E**: 프리셋 생성 → 프리셋으로 로그 작성 → 필드 자동 채움 확인 (critical user journey)
