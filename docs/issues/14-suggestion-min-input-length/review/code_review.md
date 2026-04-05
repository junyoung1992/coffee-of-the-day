# 코드 리뷰

## 리뷰 범위

- **브랜치**: `feat/14-suggestion-min-input-length`
- **비교 기준**: `main...feat/14-suggestion-min-input-length`
- **변경 파일**:
  - `backend/internal/handler/suggestion_handler.go`
  - `backend/internal/handler/suggestion_handler_test.go`
  - `docs/openapi.yml`
  - `docs/spec.md`
  - `docs/issues/14-suggestion-min-input-length/plan.md` (신규)
  - `docs/issues/14-suggestion-min-input-length/tasks.md` (신규)

## 요약

handler 레이어에 최소 입력 길이 가드를 추가하여 `q` 파라미터가 빈 문자열이거나 공백만인 경우 service를 호출하지 않고 빈 배열을 반환하도록 했다. 구현 자체는 단순하고 올바르나, service 레이어의 `normalizeQ`에 빈 문자열을 허용하는 로직이 handler 가드와 계층별 책임 측면에서 일관성을 잃는 상태가 되었다는 점을 주목해야 한다.

## 발견 사항

### [Medium] service 레이어의 빈 q 허용 주석이 handler 변경과 불일치

- **파일**: `backend/internal/service/suggestion_service.go:63`
- **카테고리**: Architecture
- **현재**: `normalizeQ` 함수의 주석이 `// 빈 문자열은 전체 조회를 의미한다.`라고 명시되어 있다. handler에서 빈 q를 차단하는 가드가 추가되었으나, service는 여전히 빈 q를 유효한 입력으로 받아들이고 repository로 전달한다. service 레이어의 계약이 실제 시스템 동작과 괴리된 채로 남아있다.
- **제안**: service 레이어는 handler에 의존하지 않으므로, service의 `normalizeQ`가 빈 q를 받을 경우의 동작을 명확히 해야 한다. 두 가지 접근이 가능하다.
  1. (권장) 주석을 현재 사실 기반으로 수정한다: `// handler에서 빈 문자열을 차단하므로 이 함수에는 비어 있지 않은 값만 전달된다. 빈 문자열이 전달되더라도 그대로 repo에 위임한다.`
  2. service에서도 빈 q를 ValidationError로 처리하여 두 레이어 모두 동일한 불변식을 시행한다. 단, 이 경우 `TestGetTagSuggestions_EmptyQ_ReturnsAll`(service 테스트)도 수정 필요.
- **근거**: `docs/arch/backend.md`에 따르면 service는 비즈니스 규칙과 입력 정규화를 담당한다. 현재는 handler가 입력 길이 규칙을 단독으로 시행하고 service의 주석은 이를 반영하지 않아, 향후 service를 직접 호출하는 경로(예: 배치, 내부 테스트)에서 의도치 않게 전체 조회가 발생할 수 있다.

### [Medium] 공백만 포함된 q 입력에 대한 테스트 누락

- **파일**: `backend/internal/handler/suggestion_handler_test.go`
- **카테고리**: Quality
- **현재**: 빈 문자열(`q=`)과 파라미터 누락(`q` 없음) 케이스는 테스트하지만, 공백만 포함된 입력(`q=%20%20`)은 테스트하지 않는다. 가드 조건이 `strings.TrimSpace(q)` 기반이므로 이 케이스도 가드에 걸리는 것이 의도된 동작이며, 해당 동작을 보장하는 테스트가 없다.
- **제안**: `TestGetTagSuggestions_WhitespaceQ_ReturnsEmptyArray` 테스트를 추가한다.
  ```go
  func TestGetTagSuggestions_WhitespaceQ_ReturnsEmptyArray(t *testing.T) {
      svc := &stubSuggestionService{
          tagsFunc: func(_ context.Context, _, _ string) ([]string, error) {
              t.Fatal("공백만 있는 q일 때 서비스가 호출되면 안 된다")
              return nil, nil
          },
      }
      h := NewSuggestionHandler(svc)

      r := httptest.NewRequest(http.MethodGet, "/api/v1/suggestions/tags?q=++++", nil)
      r = withUserID(r, "user-1")
      w := httptest.NewRecorder()

      h.GetTagSuggestions(w, r)

      assert.Equal(t, http.StatusOK, w.Code)
      var resp suggestionsResponse
      require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
      assert.NotNil(t, resp.Suggestions)
      assert.Empty(t, resp.Suggestions)
  }
  ```
  `GetCompanionSuggestions`에도 동일한 케이스를 추가한다.
- **근거**: `plan.md`에서 `strings.TrimSpace`를 사용하는 이유를 "공백만으로 이루어진 입력(`q=  `)도 빈 입력과 동일하게 처리"라고 명시했으나 이를 검증하는 테스트가 없다. 명세에서 의도한 동작임에도 테스트로 보장되지 않는다.

### [Low] OpenAPI 스펙의 `minLength`가 실제 구현과 의미상 불일치

- **파일**: `docs/openapi.yml:342`, `docs/openapi.yml:363`
- **카테고리**: Quality
- **현재**: `q` 파라미터 schema에 `minLength: 1`을 추가했으나, 실제 구현은 `q`가 없거나 빈 문자열일 때 400을 반환하지 않고 200과 빈 배열을 반환한다. OpenAPI에서 `minLength: 1`은 통상 해당 제약 위반 시 에러를 반환함을 암시하지만, 이 API는 허용 후 빈 결과를 돌려준다.
- **제안**: 파라미터 설명으로 동작을 보완하는 것은 이미 되어 있으나, `minLength` 제약을 서버 시행 제약으로 오해하지 않도록 description을 더 명확히 한다: `검색어. 빈 문자열이거나 누락된 경우 400을 반환하지 않으며 빈 배열을 반환한다. 서버는 최소 1자를 권장하지만 강제하지 않는다.` 혹은 `minLength` 제약을 제거하고 description만으로 명세를 표현한다.
- **근거**: API 소비자(프론트엔드 또는 외부 클라이언트)가 OpenAPI 스펙을 읽을 때 `minLength: 1`로부터 validation error 응답을 기대할 수 있다. 실제로는 빈 배열이 반환되므로 스펙이 오해를 유발할 수 있다.

## 액션 아이템

1. [Medium] `backend/internal/service/suggestion_service.go` 63번째 줄의 `normalizeQ` 주석을 현재 동작 기반으로 수정한다. `// 빈 문자열은 전체 조회를 의미한다.`를 handler 가드 이후의 실제 상황을 반영하는 내용으로 교체한다.
2. [Medium] `backend/internal/handler/suggestion_handler_test.go`에 공백만 포함된 q(`q=++++` 또는 URL 인코딩된 공백)에 대한 테스트를 `GetTagSuggestions`와 `GetCompanionSuggestions` 각각에 추가하여 `strings.TrimSpace` 가드가 공백 입력에도 동작함을 검증한다.
3. [Low] `docs/openapi.yml`의 `/api/v1/suggestions/tags`와 `/api/v1/suggestions/companions` 파라미터에서 `minLength: 1`을 제거하거나, 빈 입력 시 400이 아닌 200 + 빈 배열을 반환한다는 사실을 description에 명시하여 스펙 소비자의 오해를 방지한다.
