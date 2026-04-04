# 코드 리뷰

## 리뷰 범위

- **브랜치**: feat/7-favorite-presets
- **비교 기준**: main...feat/7-favorite-presets
- **변경 파일**:
  - `backend/cmd/server/main.go`
  - `backend/db/migrations/007_create_presets.up.sql`, `007_create_presets.down.sql`
  - `backend/db/queries/presets.sql`, `brew_presets.sql`, `cafe_presets.sql`
  - `backend/internal/db/brew_presets.sql.go`, `cafe_presets.sql.go`, `models.go`, `presets.sql.go`
  - `backend/internal/domain/preset.go`
  - `backend/internal/handler/preset_handler.go`, `preset_handler_test.go`
  - `backend/internal/repository/preset_repository.go`, `preset_repository_test.go`
  - `backend/internal/service/preset_service.go`, `preset_service_test.go`
  - `docs/arch/backend.md`, `docs/openapi.yml`, `docs/spec.md`
  - `frontend/src/api/presets.ts`
  - `frontend/src/hooks/usePresets.ts`, `usePresets.test.tsx`
  - `frontend/src/pages/HomePage.tsx`, `LogDetailPage.tsx`, `LogFormPage.tsx`, `PresetsPage.tsx`
  - `frontend/src/pages/logFormState.ts`, `logFormState.test.ts`
  - `frontend/src/router.tsx`
  - `frontend/src/types/preset.ts`, `schema.ts`

## 요약

즐겨찾기 프리셋 기능(CRUD + 사용 기록)을 backend/frontend 전체 vertical slice로 구현했다. 아키텍처 레이어 분리, N+1 방지 배치 로딩, discriminated union 타입 설계 등 기존 프로젝트 패턴을 잘 따르고 있다. 다만 데이터 유실 가능성이 있는 `recipe_detail` 누락 이슈와, 일부 repository 메서드의 rows affected 미확인이 수정 필요하다.

## 발견 사항

### [High] `UpdatePreset` / `UpdateLastUsedAt`에서 rows affected 미확인

- **파일**: `backend/internal/repository/preset_repository.go:161-208`, `225-236`
- **카테고리**: Quality
- **현재**: `UpdatePreset`과 `UpdateLastUsedAt`은 sqlc의 `:exec` 쿼리를 사용하므로 대상 row가 없어도 에러 없이 성공한다. 반면 `DeletePreset`(210-223)은 raw SQL + `RowsAffected()` 체크로 `ErrNotFound`를 반환한다.
- **제안**: 두 가지 방법 중 택일: (1) sqlc 쿼리를 `:execresult`로 변경하여 `RowsAffected()` 확인, (2) `DeletePreset`처럼 raw SQL로 전환하여 `RowsAffected() == 0` 시 `ErrNotFound` 반환. 현재 service layer에서 `UpdatePreset` 전에 `GetPresetByID`로 존재 확인하므로 실질적 위험은 낮지만, `UpdateLastUsedAt`은 service에서 존재 확인과 업데이트 사이 race condition이 있을 수 있다.
- **근거**: `DeletePreset`과 패턴을 통일해야 repository layer의 동작 예측이 일관된다. defense-in-depth 관점에서도 DB 반영 실패를 감지해야 한다.

### [High] `buildPresetBody`에서 brew의 `recipe_detail` 누락

- **파일**: `frontend/src/pages/LogDetailPage.tsx:56-75`
- **카테고리**: Quality
- **현재**: `buildPresetBody` 함수가 brew 프리셋 생성 시 `recipe_detail` 필드를 포함하지 않는다. `log.brew`에 `recipe_detail` 정보가 있어도 프리셋에 저장되지 않는다.
- **제안**: brew 분기에 `recipe_detail: log.brew.recipe_detail ?? undefined`를 추가한다. `PresetsPage.tsx`의 `EditModal.buildUpdateBody()`(52-68)에도 동일한 누락이 있으므로 함께 수정한다: `recipe_detail: preset.brew.recipe_detail ?? undefined`.
- **근거**: `brew_presets` 테이블에 `recipe_detail` 컬럼이 존재하고, backend API가 이 필드를 지원한다. 프리셋 저장/수정 시 데이터가 유실된다.

### [High] `handleSavePreset` 후 mutation 상태 미리셋

- **파일**: `frontend/src/pages/LogDetailPage.tsx:100-105`
- **카테고리**: Quality
- **현재**: `handleSavePreset` 성공 후 `showPresetInput`을 false로 설정하고 `presetName`을 초기화하지만, `createPresetMutation`의 상태(isSuccess/isError)를 리셋하지 않는다. "프리셋 저장" 버튼을 다시 눌러 입력란을 열면 이전 성공/에러 메시지가 그대로 표시된다.
- **제안**: `handleSavePreset` 성공 후 `createPresetMutation.reset()`을 호출하거나, "프리셋 저장" 버튼의 `onClick` 핸들러에서 `setShowPresetInput(true)` 전에 `createPresetMutation.reset()`을 호출한다.
- **근거**: TanStack Query mutation은 명시적으로 `reset()`하지 않으면 이전 상태를 유지한다. 사용자가 동일 세션에서 반복 저장 시 혼란을 줄 수 있다.

### [Medium] `CreatePreset` repository switch문에 default case 없음

- **파일**: `backend/internal/repository/preset_repository.go:61-79`
- **카테고리**: Quality
- **현재**: `CreatePreset`의 switch문(62행)에 default case가 없다. service layer에서 `normalizeCreatePresetRequest`로 `log_type`을 검증하므로 실제로 도달할 수 없지만, repository 단독 사용 시 보호가 없다.
- **제안**: `default: return fmt.Errorf("create preset: unsupported log_type: %s", preset.LogType)` 추가. `UpdatePreset`(181행)도 동일하게 적용.
- **근거**: defense-in-depth. repository는 service와 독립적으로 호출될 수 있으므로 자체 검증이 필요하다.

### [Medium] 프리셋 선택 시 확인 없이 폼 데이터 덮어쓰기

- **파일**: `frontend/src/pages/LogFormPage.tsx:922-930`
- **카테고리**: Quality
- **현재**: `PresetSection`에서 프리셋을 선택하면 `presetToFormState(preset)`로 폼 전체를 즉시 덮어쓴다. 사용자가 이미 입력한 데이터가 경고 없이 사라진다.
- **제안**: 폼에 이미 값이 입력된 경우 `window.confirm('현재 입력한 내용이 초기화됩니다. 계속하시겠습니까?')`로 확인을 받는다. 폼이 초기 상태인 경우에는 확인 없이 바로 적용한다.
- **근거**: 사용자가 필수 필드를 채운 후 실수로 프리셋을 클릭하면 데이터가 유실된다.

### [Medium] clone 모드에서 `PresetSection` 노출

- **파일**: `frontend/src/pages/LogFormPage.tsx:922-931`
- **카테고리**: Quality
- **현재**: `{!isEditMode ? (<PresetSection .../>)}`로 렌더링하므로, clone 모드(`isEditMode=false`, `isCloneMode=true`)에서도 프리셋 섹션이 표시된다. 복제 데이터가 프리셋 선택으로 덮어써질 수 있다.
- **제안**: 조건을 `{!isEditMode && !isCloneMode ? (<PresetSection .../>)}`로 변경한다.
- **근거**: clone 모드는 이전 기록을 기반으로 새 기록을 작성하는 흐름이다. 프리셋으로 덮어쓰기는 의도하지 않은 동작이다.

### [Medium] `PresetCard` 삭제 실패 시 에러 UI 없음

- **파일**: `frontend/src/pages/PresetsPage.tsx:120-186`
- **카테고리**: Quality
- **현재**: `PresetCard` 컴포넌트에서 `deleteMutation.isError` 상태를 체크하지 않는다. 삭제 실패 시 사용자에게 피드백이 없다.
- **제안**: `PresetCard` 반환 JSX 내부에 에러 표시를 추가한다:
  ```tsx
  {deleteMutation.isError ? (
    <p className="mt-2 text-xs text-rose-600">{getErrorMessage(deleteMutation.error)}</p>
  ) : null}
  ```
- **근거**: `EditModal`은 `updateMutation.isError`를 체크하고 있어 패턴이 불일치한다.

### [Medium] `EditModal`에서도 brew `recipe_detail` 누락

- **파일**: `frontend/src/pages/PresetsPage.tsx:52-68`
- **카테고리**: Quality
- **현재**: `EditModal.buildUpdateBody()`가 brew 프리셋 수정 시 `recipe_detail`을 포함하지 않는다. 이름만 변경하는 수정에서도 기존 `recipe_detail`이 `undefined`로 전송되어 서버에서 null로 덮어쓰일 수 있다.
- **제안**: brew 분기에 `recipe_detail: preset.brew.recipe_detail ?? undefined`를 추가한다.
- **근거**: `buildPresetBody`(LogDetailPage.tsx)와 동일한 누락. 두 곳 모두 수정해야 한다.

### [Low] sqlc `DeletePreset` 쿼리가 dead code

- **파일**: `backend/db/queries/presets.sql:26-28`
- **카테고리**: Quality
- **현재**: `presets.sql`에 `DeletePreset` 쿼리가 정의되어 있고 `presets.sql.go`에 코드가 생성되지만, repository에서는 raw SQL로 직접 `DELETE`를 실행한다(`RowsAffected` 확인을 위해). sqlc 생성 `DeletePreset` 함수는 어디에서도 호출되지 않는다.
- **제안**: (1) sqlc 쿼리를 `:execresult`로 변경하여 repository에서 사용하거나, (2) 쿼리를 제거하여 dead code를 정리한다.
- **근거**: dead code는 유지보수 혼란을 유발한다. 특히 쿼리 수정 시 어느 경로가 실제 실행되는지 파악하기 어렵다.

### [Low] `EditModal`에 접근성 속성 부족

- **파일**: `frontend/src/pages/PresetsPage.tsx:76-116`
- **카테고리**: Quality
- **현재**: 모달 오버레이 `<div>`에 `role="dialog"`, `aria-modal="true"`, `aria-labelledby` 속성이 없다. 포커스 트랩도 구현되어 있지 않아 Tab 키로 모달 뒤의 요소에 접근할 수 있다.
- **제안**: 최외곽 `<div>`에 `role="dialog"` `aria-modal="true"`를 추가하고, 제목에 id를 부여하여 `aria-labelledby`로 연결한다. POC 단계이므로 focus trap은 후순위로 할 수 있지만 ARIA 속성은 추가한다.
- **근거**: 스크린 리더가 모달 컨텍스트를 인식할 수 없다.

### [Low] `useUpdatePreset`에서 중복 invalidation

- **파일**: `frontend/src/hooks/usePresets.ts:53-62`
- **카테고리**: Quality
- **현재**: `onSuccess`에서 `PRESET_KEYS.detail(id)`와 `PRESET_KEYS.all`을 모두 무효화한다. `PRESET_KEYS.all`은 `['presets']`이므로 prefix 매칭으로 `['presets', 'detail', id]`도 포함하여 이미 무효화된다.
- **제안**: `PRESET_KEYS.detail(id)` 무효화를 제거하고 `PRESET_KEYS.all`만 남긴다. 또는 의도적으로 detail을 먼저 즉시 무효화하려면 `detail` 무효화 후 `list`만 추가 무효화한다.
- **근거**: TanStack Query의 `invalidateQueries`는 queryKey prefix 매칭으로 동작한다. `['presets']`는 모든 `['presets', ...]` 쿼리를 무효화하므로 detail 무효화는 중복이다.

## 액션 아이템

1. [High] `backend/internal/repository/preset_repository.go`에서 `UpdatePreset`과 `UpdateLastUsedAt`이 rows affected를 확인하도록 수정한다. `DeletePreset`과 동일하게 raw SQL + `RowsAffected() == 0` 시 `ErrNotFound` 반환 패턴을 적용하거나, sqlc 쿼리를 `:execresult`로 변경한다.
2. [High] `frontend/src/pages/LogDetailPage.tsx`의 `buildPresetBody` brew 분기에 `recipe_detail: log.brew.recipe_detail ?? undefined`를 추가한다.
3. [High] `frontend/src/pages/PresetsPage.tsx`의 `EditModal.buildUpdateBody` brew 분기에 `recipe_detail: preset.brew.recipe_detail ?? undefined`를 추가한다.
4. [High] `frontend/src/pages/LogDetailPage.tsx`의 `handleSavePreset` 성공 후 `createPresetMutation.reset()`을 호출하거나, "프리셋 저장" 버튼 클릭 시 mutation을 리셋한다.
5. [Medium] `backend/internal/repository/preset_repository.go`의 `CreatePreset`(62행)과 `UpdatePreset`(181행) switch문에 `default: return fmt.Errorf("...: unsupported log_type: %s", preset.LogType)` case를 추가한다.
6. [Medium] `frontend/src/pages/LogFormPage.tsx`의 922행 조건을 `{!isEditMode && !isCloneMode ? (<PresetSection .../>)}`로 변경한다.
7. [Medium] `frontend/src/pages/LogFormPage.tsx`의 `PresetSection.onSelect` 콜백에서, 폼에 값이 있으면 `window.confirm()`으로 확인 후 적용한다.
8. [Medium] `frontend/src/pages/PresetsPage.tsx`의 `PresetCard` 컴포넌트에 `deleteMutation.isError` 시 에러 메시지를 표시하는 JSX를 추가한다.
9. [Low] `backend/db/queries/presets.sql`에서 사용하지 않는 `DeletePreset` 쿼리를 제거하거나, repository에서 sqlc 생성 함수를 사용하도록 전환한다.
10. [Low] `frontend/src/pages/PresetsPage.tsx`의 `EditModal` 최외곽 `<div>`에 `role="dialog"` `aria-modal="true"` 속성을 추가한다.
11. [Low] `frontend/src/hooks/usePresets.ts`의 `useUpdatePreset`에서 `PRESET_KEYS.detail(id)` 무효화를 제거하고 `PRESET_KEYS.all`만 남긴다.
