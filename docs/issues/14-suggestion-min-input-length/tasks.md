# Tasks — Issue #14 Suggestion endpoint 최소 입력 길이 강제

> handler 레이어에서만 변경. service, repository는 수정하지 않는다.
> 상세 설계는 `plan.md` 참조.

---

## 1. Handler에 최소 입력 길이 가드 추가

- [ ] **`GetTagSuggestions`에 빈 입력 가드 추가**
  - Target: `backend/internal/handler/suggestion_handler.go`
  - `import`에 `"strings"` 추가
  - `GetTagSuggestions` 메서드에서 `q := r.URL.Query().Get("q")` 직후에 가드 삽입:
    ```go
    if len(strings.TrimSpace(q)) < 1 {
        writeJSON(w, http.StatusOK, suggestionsResponse{Suggestions: []string{}})
        return
    }
    ```

- [ ] **`GetCompanionSuggestions`에 빈 입력 가드 추가**
  - Target: `backend/internal/handler/suggestion_handler.go`
  - `GetCompanionSuggestions` 메서드에서 동일한 가드 삽입 (패턴은 위와 동일)

---

## 2. 기존 테스트 수정 및 새 테스트 추가

- [ ] **`TestGetTagSuggestions_EmptyQ_ReturnsAll` 삭제**
  - Target: `backend/internal/handler/suggestion_handler_test.go` (58-79줄)
  - 이 테스트는 빈 `q`가 service로 전달되어 결과를 반환하는 기존 동작을 검증한다. 새로운 동작과 충돌하므로 삭제.

- [ ] **`TestGetTagSuggestions_EmptyQ_ReturnsEmptyArray` 추가**
  - Target: `backend/internal/handler/suggestion_handler_test.go`
  - `q=` (빈 문자열)로 요청, 200 응답, 빈 배열 반환 검증
  - stub의 `tagsFunc`에 `t.Fatal("service should not be called")` 설정하여 service 미호출 확인

- [ ] **`TestGetTagSuggestions_MissingQ_ReturnsEmptyArray` 추가**
  - Target: `backend/internal/handler/suggestion_handler_test.go`
  - q 파라미터 없이 `/api/v1/suggestions/tags`로 요청
  - 200 응답, 빈 배열 반환, service 미호출 검증

- [ ] **`TestGetCompanionSuggestions_EmptyQ_ReturnsEmptyArray` 추가**
  - Target: `backend/internal/handler/suggestion_handler_test.go`
  - `q=` (빈 문자열)로 요청, 200 응답, 빈 배열 반환 검증
  - stub의 `companionsFunc`에 `t.Fatal("service should not be called")` 설정

- [ ] **`TestGetCompanionSuggestions_MissingQ_ReturnsEmptyArray` 추가**
  - Target: `backend/internal/handler/suggestion_handler_test.go`
  - q 파라미터 없이 `/api/v1/suggestions/companions`로 요청
  - 200 응답, 빈 배열 반환, service 미호출 검증

---

## 3. OpenAPI 스펙 업데이트

- [ ] **tags endpoint의 q 파라미터 설명 및 schema 수정**
  - Target: `docs/openapi.yml` (338-342줄 부근)
  - `description`을 `검색어 (최소 1자 이상, 빈 문자열이면 빈 배열 반환)`으로 변경
  - `schema`에 `minLength: 1` 추가

- [ ] **companions endpoint의 q 파라미터 설명 및 schema 수정**
  - Target: `docs/openapi.yml` (358-362줄 부근)
  - 동일한 변경 적용

---

## 4. 검증

- [ ] **handler 테스트 실행**
  - 명령어: `cd /Users/junyoung/workspace/coffee-of-the-day/backend && go test ./internal/handler/... -v -run TestGet.*Suggestions`
  - 새로 추가한 4개 테스트 + 기존 테스트 모두 통과 확인

- [ ] **전체 백엔드 테스트 실행**
  - 명령어: `cd /Users/junyoung/workspace/coffee-of-the-day/backend && go test ./...`
  - 기존 테스트 전체 통과 확인

- [ ] **수동 검증 항목**
  - `GET /suggestions/companions?q=` -> `{"suggestions":[]}` 반환
  - `GET /suggestions/tags?q=` -> `{"suggestions":[]}` 반환
  - `GET /suggestions/tags` (q 누락) -> `{"suggestions":[]}` 반환
  - `GET /suggestions/tags?q=초` -> 기존과 동일하게 제안 목록 반환
