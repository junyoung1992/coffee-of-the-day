# Tasks --- Issue #7 즐겨찾기 프리셋

> 선행 조건: P1(최근 기록 복제, Issue #6) 완료 상태.
> 작업 순서: 백엔드 → OpenAPI → 프론트엔드 타입 생성 → 프론트엔드 구현.
> 상세 설계는 `plan.md` 참조.

---

## 1. 데이터베이스 마이그레이션

- [x] **마이그레이션 파일 생성**
  - Target: `backend/db/migrations/007_create_presets.up.sql`
  - `presets` 테이블 생성 (id, user_id, name, log_type, last_used_at, created_at, updated_at)
  - `cafe_presets` 테이블 생성 (preset_id FK, cafe_name, coffee_name, tasting_tags)
  - `brew_presets` 테이블 생성 (preset_id FK, bean_name, brew_method, recipe_detail, brew_steps)
  - CHECK 제약: log_type IN ('cafe', 'brew'), brew_method 값 목록은 기존 `brew_logs`와 동일
  - ON DELETE CASCADE 설정
  - `plan.md`의 테이블 구조 참조

- [x] **down 마이그레이션 파일 생성**
  - Target: `backend/db/migrations/007_create_presets.down.sql`
  - `brew_presets`, `cafe_presets`, `presets` 순서로 DROP TABLE

---

## 2. 백엔드 Domain 모델

- [x] **프리셋 도메인 타입 정의**
  - Target: `backend/internal/domain/preset.go` (신규)
  - `Preset`, `CafePresetDetail`, `BrewPresetDetail`, `PresetFull` 구조체 정의
  - 기존 `domain/log.go`의 `CoffeeLogFull` 패턴 참조
  - `LogType`, `BrewMethod` enum은 기존 것 재사용

---

## 3. 백엔드 Repository

- [x] **sqlc 쿼리 파일 작성**
  - Target: `backend/db/queries/presets.sql` (신규)
  - Target: `backend/db/queries/cafe_presets.sql` (신규)
  - Target: `backend/db/queries/brew_presets.sql` (신규)
  - 각 테이블의 INSERT, SELECT (by id + user_id), SELECT ALL (by user_id), UPDATE, DELETE 쿼리
  - `presets` 목록 조회 시 정렬: `ORDER BY CASE WHEN last_used_at IS NULL THEN 1 ELSE 0 END, last_used_at DESC, created_at DESC`
  - `last_used_at` 갱신 전용 UPDATE 쿼리
  - `sqlc generate` 실행하여 Go 코드 생성

- [x] **PresetRepository 구현**
  - Target: `backend/internal/repository/preset_repository.go` (신규)
  - `PresetRepository` interface 정의
  - `SQLitePresetRepository` 구현
  - `CreatePreset`: tx 내에서 presets + cafe_presets/brew_presets 삽입 (기존 `CreateLog` 패턴 참조)
  - `GetPresetByID`: presets + 서브 테이블 JOIN 조회, user_id 확인
  - `ListPresets`: user_id로 전체 조회, 서브 테이블 JOIN
  - `UpdatePreset`: tx 내에서 presets + 서브 테이블 UPDATE
  - `DeletePreset`: presets 삭제 (CASCADE로 서브 테이블 자동 삭제)
  - `UpdateLastUsedAt`: last_used_at 필드만 갱신
  - JSON 배열 직렬화/역직렬화: 기존 `log_repository.go`의 `tasting_tags`, `brew_steps` 처리 패턴 참조

- [x] **Repository 테스트 작성**
  - Target: `backend/internal/repository/preset_repository_test.go` (신규)
  - CRUD happy path 테스트
  - 정렬 검증: last_used_at이 있는 프리셋이 NULL인 것보다 먼저 나오는지
  - 다른 사용자의 프리셋에 접근 불가 확인
  - 존재하지 않는 프리셋 조회 시 ErrNotFound

---

## 4. 백엔드 Service

- [x] **PresetService 구현**
  - Target: `backend/internal/service/preset_service.go` (신규)
  - `NewPresetService(repo PresetRepository)` 생성자
  - `CreatePreset`: name trim + 빈 문자열 검증, log_type 검증, cafe/brew 일치 검증, UUID 생성 (`crypto/rand`), 타임스탬프 설정
  - `GetPreset`: repository 위임 + 에러 매핑
  - `ListPresets`: repository 위임
  - `UpdatePreset`: name trim, 기존 프리셋 조회 후 log_type 불변 검증, 필드 업데이트
  - `DeletePreset`: repository 위임
  - `UsePreset`: 프리셋 존재 확인 + `UpdateLastUsedAt` 호출 (현재 시각 RFC3339)

- [x] **Service 테스트 작성**
  - Target: `backend/internal/service/preset_service_test.go` (신규)
  - 검증 로직 테스트: 빈 이름, 잘못된 log_type, cafe 프리셋에 brew 데이터 전달 등
  - mock repository 사용 (기존 `log_service_test.go` 패턴 참조)
  - UsePreset 호출 시 last_used_at 갱신 확인

---

## 5. 백엔드 Handler

- [x] **PresetHandler 구현**
  - Target: `backend/internal/handler/preset_handler.go` (신규)
  - JSON 요청/응답 타입 정의 (handler 내부, 기존 `log_handler.go` 패턴)
  - `NewPresetHandler(svc PresetService)` 생성자
  - 6개 엔드포인트 핸들러: CreatePreset, ListPresets, GetPreset, UpdatePreset, DeletePreset, UsePreset
  - UsePreset 응답: `204 No Content`
  - 에러 매핑: `ErrNotFound` → 404, `ValidationError` → 400
  - userID는 `r.Context().Value("user_id").(string)`로 추출 (기존 패턴)

- [x] **Handler 테스트 작성**
  - Target: `backend/internal/handler/preset_handler_test.go` (신규)
  - 요청/응답 직렬화 테스트
  - 에러 케이스 (400, 404) 테스트
  - 기존 `log_handler_test.go` 패턴 참조

- [x] **라우터 등록**
  - Target: `backend/cmd/server/main.go`
  - 의존성 연결: `NewSQLitePresetRepository` → `NewPresetService` → `NewPresetHandler`
  - 인증 필요 라우트 그룹 내에 `/presets` 라우트 추가
  - 위치: 기존 `/logs` 라우트와 `/suggestions` 라우트 사이

---

## 6. OpenAPI 스펙 업데이트

- [x] **프리셋 스키마 및 엔드포인트 추가**
  - Target: `docs/openapi.yml`
  - tags에 `presets` 추가
  - schemas: `CafePresetDetail`, `BrewPresetDetail`, `PresetResponse`, `CreatePresetRequest`, `UpdatePresetRequest`, `ListPresetsResponse`
  - paths: `/api/v1/presets` (POST, GET), `/api/v1/presets/{id}` (GET, PUT, DELETE), `/api/v1/presets/{id}/use` (POST)
  - parameters: `PresetId` (path parameter)
  - 기존 `LogType`, `BrewMethod` 스키마 재사용

---

## 7. 프론트엔드 타입 생성 및 정의

의존: Task 6 완료 후.

- [x] **타입 생성 및 파생 타입 정의**
  - `npm run generate` 실행하여 `src/types/schema.ts` 갱신
  - Target: `frontend/src/types/preset.ts` (신규)
  - `schema.ts`에서 생성된 타입 기반으로 discriminated union 정의
  - `CafePresetFull`, `BrewPresetFull`, `PresetFull` 타입 (기존 `types/log.ts` 패턴)
  - `CreatePresetInput`, `UpdatePresetInput` 타입 alias

---

## 8. 프론트엔드 API 및 Hooks

의존: Task 7 완료 후.

- [x] **API 함수 작성**
  - Target: `frontend/src/api/presets.ts` (신규)
  - `getPresets()`, `getPreset(id)`, `createPreset(body)`, `updatePreset(id, body)`, `deletePreset(id)`, `usePreset(id)` 함수
  - 기존 `api/logs.ts` 패턴 참조

- [x] **커스텀 hooks 작성**
  - Target: `frontend/src/hooks/usePresets.ts` (신규)
  - `PRESET_KEYS` 쿼리 키 상수 정의
  - `usePresetList()`: useQuery로 전체 목록 조회
  - `usePreset(id)`: useQuery로 단건 조회
  - `useCreatePreset()`: useMutation, 성공 시 목록 무효화
  - `useUpdatePreset(id)`: useMutation, 성공 시 detail + 목록 무효화
  - `useDeletePreset()`: useMutation, 성공 시 목록 무효화
  - `useUsePreset()`: useMutation, 성공 시 목록 캐시에서 해당 프리셋의 last_used_at을 optimistic update

- [x] **Hooks 테스트 작성**
  - Target: `frontend/src/hooks/usePresets.test.tsx` (신규)
  - 기존 `hooks/useLogs.test.tsx` 패턴 참조
  - 목록 조회, 생성/수정/삭제 mutation 후 캐시 무효화 확인

---

## 9. 프론트엔드: 로그 작성 폼에 프리셋 선택 연동

의존: Task 8 완료 후.

- [x] **presetToFormState 함수 추가**
  - Target: `frontend/src/pages/logFormState.ts`
  - `presetToFormState(preset: PresetFull, now?: Date): LogFormState` 함수 추가
  - 동작: 프리셋의 log_type과 전용 필드를 폼 상태로 변환
  - Cafe: cafeName, coffeeName, tastingTags 채움
  - Brew: beanName, brewMethod, brewSteps 채움, recipeDetail은 memo에 매핑
  - recorded_at은 현재 시각, rating/memo/companions/impressions는 빈 값
  - 기존 `cloneToFormState()` 패턴 참조

- [x] **presetToFormState 테스트**
  - Target: `frontend/src/pages/logFormState.test.ts`
  - cafe/brew 프리셋 각각에 대해 변환 결과 검증
  - 리셋 필드가 올바르게 초기화되는지 확인

- [x] **프리셋 선택 UI 컴포넌트 추가**
  - Target: `frontend/src/pages/LogFormPage.tsx`
  - LogTypeSection 아래에 프리셋 선택 섹션 추가
  - `usePresetList()` 호출하여 현재 logType에 맞는 프리셋 필터링
  - 프리셋 카드 클릭 시 `presetToFormState()` 호출 → `setForm()` 업데이트
  - 프리셋 사용 시 `useUsePreset()` 호출하여 last_used_at 갱신
  - edit 모드에서는 프리셋 선택 영역 미노출 (`isEditMode` 조건)
  - clone 모드에서도 프리셋 선택 가능 (clone 데이터를 프리셋으로 덮어쓸 수 있음)

---

## 10. 프론트엔드: 로그 상세에서 프리셋 저장

의존: Task 8 완료 후. Task 9와 병렬 가능.

- [x] **"프리셋으로 저장" 버튼 및 이름 입력 UI 추가**
  - Target: `frontend/src/pages/LogDetailPage.tsx`
  - actions 영역에 "프리셋으로 저장" 버튼 추가 (기존 복제 버튼 옆)
  - 클릭 시 프리셋 이름 입력용 모달 또는 인라인 input 표시
  - 확인 시 `useCreatePreset()` 호출
  - 요청 body 구성: 로그 데이터에서 프리셋 필드만 추출
    - Cafe: log.cafe.cafe_name, log.cafe.coffee_name, log.cafe.tasting_tags
    - Brew: log.brew.bean_name, log.brew.brew_method, log.brew.brew_steps
  - 성공 시 토스트 또는 확인 메시지, 실패 시 에러 표시

---

## 11. 프론트엔드: 프리셋 관리 페이지

의존: Task 8 완료 후. Task 9, 10과 병렬 가능.

- [x] **프리셋 관리 페이지 구현**
  - Target: `frontend/src/pages/PresetsPage.tsx` (신규)
  - `usePresetList()` 호출하여 목록 표시
  - 카드 형태로 표시: 이름, log_type 뱃지, 주요 정보 (cafe_name/bean_name 등)
  - last_used_at 표시 (미사용 시 "사용 기록 없음")
  - 각 카드에 수정/삭제 버튼
  - 삭제: confirm dialog 후 `useDeletePreset()` 호출
  - 수정: 모달 형태로 이름 + 세부 필드 편집 가능, `useUpdatePreset()` 호출
  - Layout 컴포넌트 사용 (기존 페이지 패턴)

- [x] **라우터에 프리셋 페이지 등록**
  - Target: `frontend/src/router.tsx`
  - ProtectedRoute children에 `{ path: '/presets', element: <PresetsPage /> }` 추가

- [x] **네비게이션 링크 추가**
  - Target: `frontend/src/components/Layout.tsx` 또는 `frontend/src/pages/HomePage.tsx`
  - 프리셋 관리 화면으로의 링크 추가 (위치는 기존 UI 구조에 맞게 결정)

---

## 12. 검증

- [x] **백엔드 테스트 실행**
  - `cd backend && go test ./...` 전체 통과 확인
  - 특히 preset 관련 테스트: repository, service, handler

- [x] **프론트엔드 테스트 실행**
  - `cd frontend && npm test` 전체 통과 확인
  - 특히 logFormState, usePresets, LogFormPage 테스트

- [x] **수동 검증**
  - 로그 상세 → "프리셋으로 저장" → 프리셋 목록에 추가 확인
  - 새 로그 작성 → 프리셋 선택 → 필드 자동 채움 확인
  - 프리셋 관리 → 수정/삭제 동작 확인
  - 프리셋 목록 정렬: 최근 사용순 확인

- [x] **문서 갱신 확인**
  - `docs/spec.md` 반영 확인 (Task 시작 전에 이미 갱신됨)
  - `docs/openapi.yml` 반영 확인 (Task 6에서 갱신)
  - `docs/arch/backend.md` 업데이트 필요 여부 확인 (새 vertical slice 추가에 대한 언급)
