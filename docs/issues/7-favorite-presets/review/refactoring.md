# Issue #7 리팩토링 체크리스트

> `code_review.md` 발견 사항 중 이번 브랜치에서 수정할 항목.
> 완료 시 체크 표시하고, 수정한 파일과 간단한 설명을 추가한다.

---

## High

- [ ] **`UpdatePreset` / `UpdateLastUsedAt` rows affected 확인**
  - Target: `backend/internal/repository/preset_repository.go`
  - `UpdatePreset`과 `UpdateLastUsedAt`이 대상 row가 없어도 에러 없이 성공하는 문제
  - `DeletePreset`과 동일하게 raw SQL + `RowsAffected() == 0` 시 `ErrNotFound` 반환 패턴 적용
  - 관련 테스트 추가 (존재하지 않는 프리셋 update/use 시 ErrNotFound)

- [ ] **`buildPresetBody`에서 brew `recipe_detail` 누락**
  - Target: `frontend/src/pages/LogDetailPage.tsx`
  - brew 분기에 `recipe_detail: log.memo ?? undefined` 추가
  - 로그의 memo가 프리셋의 recipe_detail로 저장되도록 함 (presetToFormState의 역방향)

- [ ] **`EditModal.buildUpdateBody`에서 brew `recipe_detail` 누락**
  - Target: `frontend/src/pages/PresetsPage.tsx`
  - brew 분기에 `recipe_detail: preset.brew.recipe_detail ?? undefined` 추가
  - 이름만 수정해도 기존 recipe_detail이 null로 덮어쓰이는 문제 방지

- [ ] **`handleSavePreset` 후 mutation 상태 미리셋**
  - Target: `frontend/src/pages/LogDetailPage.tsx`
  - "프리셋 저장" 버튼 클릭 시 `createPresetMutation.reset()` 호출
  - 재진입 시 이전 성공/에러 메시지가 잔류하는 문제 해결

## Medium

- [ ] **`CreatePreset` / `UpdatePreset` switch default case 추가**
  - Target: `backend/internal/repository/preset_repository.go`
  - `CreatePreset`(62행)과 `UpdatePreset`(181행)의 switch문에 default error case 추가
  - `default: return fmt.Errorf("...: unsupported log_type: %s", preset.LogType)`

- [ ] **clone 모드에서 `PresetSection` 숨기기**
  - Target: `frontend/src/pages/LogFormPage.tsx`
  - 조건을 `{!isEditMode && !isCloneMode ? (<PresetSection .../>) : null}`로 변경

- [ ] **`PresetCard` 삭제 실패 시 에러 UI 추가**
  - Target: `frontend/src/pages/PresetsPage.tsx`
  - `deleteMutation.isError` 시 에러 메시지 표시하는 JSX 추가
  - `EditModal`의 에러 UI 패턴과 통일

## Low

- [ ] **`useUpdatePreset` 중복 invalidation 제거**
  - Target: `frontend/src/hooks/usePresets.ts`
  - `PRESET_KEYS.detail(id)` 무효화 제거, `PRESET_KEYS.all`만 유지

- [ ] **sqlc `DeletePreset` dead code 제거**
  - Target: `backend/db/queries/presets.sql`
  - 미사용 `DeletePreset` 쿼리 제거 후 `sqlc generate` 재실행
