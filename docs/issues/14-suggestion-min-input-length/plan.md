# Issue #14 — Suggestion endpoint 최소 입력 길이 강제

## 목표

`GET /suggestions/tags`와 `GET /suggestions/companions` 엔드포인트에서 `q` 파라미터가 빈 문자열이거나 누락된 경우, 서비스 레이어를 호출하지 않고 즉시 빈 배열 `[]`을 반환하도록 한다. 이를 통해 단일 요청으로 사용자 데이터를 전량 조회하는 것을 방지한다.

---

## 현재 동작 분석

두 handler 메서드(`GetTagSuggestions`, `GetCompanionSuggestions`)는 `r.URL.Query().Get("q")`로 검색어를 받아 그대로 service로 전달한다. `q`가 빈 문자열이면 service가 전체 데이터 상위 10건을 반환하는 구조이다.

- 파일: `backend/internal/handler/suggestion_handler.go`
- 기존 테스트 `TestGetTagSuggestions_EmptyQ_ReturnsAll` (58-79줄)은 빈 `q`가 service로 전달되어 결과가 반환되는 동작을 검증한다. 이 테스트는 새로운 동작에 맞게 수정해야 한다.

---

## 변경 설계

### Handler 레이어 가드 추가

`GetTagSuggestions`와 `GetCompanionSuggestions` 각각에서 `q` 값을 가져온 직후, `strings.TrimSpace(q)`의 길이가 1자 미만이면 service 호출 없이 `suggestionsResponse{Suggestions: []string{}}`를 반환한다.

`strings.TrimSpace`를 사용하는 이유: 공백만으로 이루어진 입력(`q=  `)도 빈 입력과 동일하게 처리하기 위함이다. `q` 파라미터가 아예 누락된 경우 `r.URL.Query().Get("q")`는 빈 문자열을 반환하므로 동일한 가드에 걸린다.

패턴:

```go
q := r.URL.Query().Get("q")
if len(strings.TrimSpace(q)) < 1 {
    writeJSON(w, http.StatusOK, suggestionsResponse{Suggestions: []string{}})
    return
}
```

`import`에 `"strings"` 추가가 필요하다.

### 응답 코드는 200 유지

빈 입력은 클라이언트 에러가 아니라 "결과 없음"으로 처리한다. 프론트엔드에서 입력 중 자동완성을 요청하는 흐름에서 에러 처리 없이 빈 드롭다운을 표시하면 되므로, 400이 아닌 200 + 빈 배열이 적절하다.

### OpenAPI 스펙 업데이트

`docs/openapi.yml`의 두 endpoint에서 `q` 파라미터의 `description`을 변경한다:
- 현재: `검색어 (빈 문자열이면 전체 반환)`
- 변경: `검색어 (최소 1자 이상, 빈 문자열이면 빈 배열 반환)`

`minLength: 1`을 schema에 추가한다.

### 기존 테스트 수정

`TestGetTagSuggestions_EmptyQ_ReturnsAll`은 빈 `q`가 service를 호출하여 3건을 반환하는 것을 검증한다. 이 테스트를 삭제하고, 빈 `q`가 service를 호출하지 않고 빈 배열을 반환하는 새 테스트로 대체한다.

---

## 수정하지 않는 것

- `backend/internal/service/` — service 레이어는 변경하지 않는다.
- `backend/internal/repository/` — repository 레이어는 변경하지 않는다.
- `frontend/` — 프론트엔드는 이미 빈 배열 응답을 처리할 수 있는 구조이므로 변경 불필요.
- `docs/spec.md` — planner가 이미 업데이트 완료.

---

## 테스트 전략

Handler unit test로 충분하다. 변경 범위가 handler 가드 1줄이므로 integration test는 불필요.

**추가할 테스트 케이스:**
1. `GET /suggestions/tags?q=` (빈 문자열) -> 200, 빈 배열, service 미호출
2. `GET /suggestions/tags` (q 파라미터 누락) -> 200, 빈 배열, service 미호출
3. `GET /suggestions/companions?q=` (빈 문자열) -> 200, 빈 배열, service 미호출
4. `GET /suggestions/companions` (q 파라미터 누락) -> 200, 빈 배열, service 미호출

**수정할 테스트:**
- `TestGetTagSuggestions_EmptyQ_ReturnsAll` -> 삭제 후 위 1, 2번 케이스로 대체

**service 미호출 검증 방법:** stub의 `tagsFunc`/`companionsFunc`에 `t.Fatal("service가 호출되면 안 됨")`을 설정하여, handler가 service를 호출하면 테스트가 실패하도록 한다.

**기존 테스트 전체 통과 확인:**
```bash
cd /Users/junyoung/workspace/coffee-of-the-day/backend && go test ./internal/handler/...
```
