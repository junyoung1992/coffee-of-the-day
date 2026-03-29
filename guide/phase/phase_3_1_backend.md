# Phase 3-1 Backend 학습 문서

> 자동완성 API 구현을 통해 배운 내용을 정리합니다.
> 주요 주제: sqlc의 한계와 raw SQL 병행 사용, SQLite json_each() 가상 테이블, 레이어드 아키텍처의 확장 방식.

---

## 1. 기능 개요

유저가 과거에 입력한 `tasting_tags`와 `companions`를 빈도순으로 집계해 자동완성 제안으로 반환한다.

```
GET /api/v1/suggestions/tags?q=초       → ["초콜릿", "초록사과", ...]
GET /api/v1/suggestions/companions?q=지  → ["지수", "지원", ...]
```

- `q`가 없거나 빈 문자열이면 전체 목록을 빈도순으로 반환한다.
- 결과는 최대 10개로 제한한다.
- 대소문자 무시 부분 일치(`LIKE '%q%'`)로 검색한다.

---

## 2. SQLite json_each() — 배열 컬럼을 행으로 펼치기

`tasting_tags`와 `companions`는 DB에 JSON 배열 텍스트로 저장된다 (`["초콜릿","체리"]`). 개별 항목을 집계하려면 배열을 행으로 "펼쳐야" 한다.

SQLite는 이를 위해 **가상 테이블 함수(table-valued function)** `json_each()`를 제공한다.

```sql
-- companions 컬럼이 '["지수","민준"]'인 행에서 개별 이름을 추출
SELECT j.value AS companion
FROM coffee_logs l
JOIN json_each(l.companions) j
WHERE l.user_id = 'user-1';

-- 결과:
-- companion
-- ---------
-- 지수
-- 민준
```

`json_each(column)`은 마치 테이블처럼 동작하므로 `JOIN`으로 연결한다. `j.value`가 배열의 각 원소에 해당한다.

**Spring/JPA에서의 대응**: JPA에는 배열 컬럼을 직접 집계하는 기능이 없다. 보통 별도 테이블(`tasting_tags` 정규화 테이블)로 관계를 분리한 뒤 `GROUP BY`로 집계한다. SQLite의 `json_each()`는 정규화 없이 이와 동일한 결과를 얻을 수 있는 방법이다.

---

## 3. sqlc가 json_each를 처리하지 못하는 이유

이 프로젝트는 SQL 쿼리 → Go 타입 코드 자동 생성을 위해 sqlc를 사용한다. 하지만 `json_each()`에는 sqlc를 적용할 수 없었다.

### sqlc의 동작 방식

sqlc는 SQL을 **정적으로 분석**해 쿼리 결과의 컬럼 타입을 Go 타입으로 변환한다. 이를 위해 스키마(DDL)를 읽어 각 테이블의 컬럼 타입을 파악한다.

```sql
-- 이 경우 sqlc는 coffee_logs.companions가 TEXT임을 알고 있다.
SELECT companions FROM coffee_logs WHERE id = ?;
-- → 생성 타입: string
```

### json_each의 문제

`json_each()`는 가상 테이블이다. 실제 테이블이 아니므로 스키마에 컬럼 정보가 없다. sqlc의 정적 분석기는 `j.value`의 타입을 추론하지 못한다.

```
db/queries/suggestions.sql:5:12: column "value" does not exist
```

이것은 sqlc의 **의도적인 트레이드오프**다. sqlc는 타입 안전성을 최우선으로 하기 때문에, 타입을 확신할 수 없는 동적 결과는 지원하지 않는다. 복잡한 동적 쿼리 지원보다 단순하고 안전한 코드 생성에 집중한다.

### 해결: 해당 쿼리만 raw SQL로 구현

sqlc를 전면 포기할 필요는 없다. 기존 CRUD 쿼리는 sqlc를 그대로 사용하고, `json_each()`가 필요한 집계 쿼리만 `database/sql`의 `db.QueryContext()`로 직접 실행했다.

```go
// internal/repository/suggestion_repository.go
const tagSuggestionsQuery = `
WITH all_tags AS (
    SELECT j.value AS tag
    FROM cafe_logs cl
    JOIN coffee_logs l ON l.id = cl.log_id
    JOIN json_each(cl.tasting_tags) j
    WHERE l.user_id = ?
    UNION ALL
    ...
)
SELECT tag, COUNT(*) AS cnt FROM all_tags ...
`

func (r *SQLiteSuggestionRepository) GetTagSuggestions(ctx context.Context, userID, q string) ([]string, error) {
    rows, err := r.db.QueryContext(ctx, tagSuggestionsQuery, userID, userID, q, q)
    ...
}
```

`sql.DB`는 sqlc가 감싸고 있는 표준 라이브러리다. sqlc가 생성한 코드도 내부적으로 `db.QueryContext()`를 호출하므로 완전히 동일한 방식이다.

---

## 4. WITH ... UNION ALL — CTE로 두 테이블 합치기

`tasting_tags`는 `cafe_logs`와 `brew_logs` 두 테이블에 각각 존재한다. 두 테이블의 태그를 하나의 집계로 합산하려면 CTE(Common Table Expression)와 `UNION ALL`을 사용한다.

```sql
WITH all_tags AS (
    -- 카페 기록의 태그
    SELECT j.value AS tag
    FROM cafe_logs cl
    JOIN coffee_logs l ON l.id = cl.log_id
    JOIN json_each(cl.tasting_tags) j
    WHERE l.user_id = ?
    UNION ALL
    -- 브루 기록의 태그 (동일한 태그가 있어도 별도 카운트)
    SELECT j.value AS tag
    FROM brew_logs bl
    JOIN coffee_logs l ON l.id = bl.log_id
    JOIN json_each(bl.tasting_tags) j
    WHERE l.user_id = ?
)
SELECT tag, COUNT(*) AS cnt
FROM all_tags
WHERE ? = '' OR LOWER(tag) LIKE '%' || LOWER(?) || '%'
GROUP BY tag
ORDER BY cnt DESC, tag ASC
LIMIT 10;
```

**`UNION` vs `UNION ALL`**: `UNION`은 중복 행을 제거한다. `UNION ALL`은 중복을 그대로 유지한다. 여기서는 카페에서도, 브루에서도 "초콜릿"을 쓴 기록이 있다면 그 횟수가 모두 카운트에 반영되어야 하므로 `UNION ALL`이 올바르다.

**파라미터가 4개인 이유**: SQLite의 `?`는 위치 기반 파라미터다. `user_id`가 CTE의 두 브랜치에, `q`가 WHERE 조건에 두 번 등장하므로 Go에서 같은 값을 두 번 전달한다.

```go
rows, err := r.db.QueryContext(ctx, query,
    userID, // CTE cafe 브랜치의 user_id
    userID, // CTE brew 브랜치의 user_id
    q,      // WHERE의 첫 번째 q (= '' 체크)
    q,      // WHERE의 두 번째 q (LIKE 검색)
)
```

---

## 5. 새 기능 추가 시 레이어드 아키텍처 확장 패턴

이 프로젝트는 handler → service → repository의 3계층 구조다. 새 기능을 추가할 때 각 계층의 역할이 명확하게 분리된다.

```
[새 의존성 연결]
main.go
  └─ NewSQLiteSuggestionRepository(db)   ← DB 직접 접근
       └─ NewSuggestionService(repo)     ← 유효성 검사, 비즈니스 로직
            └─ NewSuggestionHandler(svc) ← HTTP 요청/응답 변환
```

각 계층이 **인터페이스**에만 의존하기 때문에:
- `SuggestionService`는 `SuggestionRepository` 인터페이스만 알고, SQLite인지 PostgreSQL인지 모른다.
- 테스트에서 `SuggestionRepository` 인터페이스를 mock으로 교체하면 DB 없이 서비스 로직만 검증할 수 있다.

**Spring에서의 대응**:

```java
// Spring의 계층 분리
@RestController  → Handler
@Service         → Service
@Repository      → Repository
```

Go는 애노테이션 대신 인터페이스와 생성자 주입으로 동일한 구조를 만든다. Spring의 DI 컨테이너가 자동으로 연결해주는 부분을 `main.go`에서 명시적으로 연결한다. 더 verbose하지만 의존 관계가 눈에 바로 보인다.

---

## 6. 단위 테스트 — mock 없는 언어에서 mock 만들기

**Spring에서의 대응**: `@MockBean`, Mockito의 `mock()` / `when().thenReturn()`

Go에는 Mockito 같은 mock 프레임워크가 기본 없다. 대신 인터페이스를 직접 구현하는 구조체로 mock을 만든다.

```go
// 테스트 파일 안에 mock 선언
type mockSuggestionRepo struct {
    tags  []string
    err   error
    lastQ string // 서비스가 실제로 어떤 값을 넘겼는지 검증용
}

func (m *mockSuggestionRepo) GetTagSuggestions(_ context.Context, userID, q string) ([]string, error) {
    m.lastQ = q
    return m.tags, m.err
}

func (m *mockSuggestionRepo) GetCompanionSuggestions(...) ([]string, error) { ... }
```

테스트에서는 이 구조체를 서비스에 주입한다:

```go
func TestGetTagSuggestions_QWithWhitespace_IsTrimmed(t *testing.T) {
    repo := &mockSuggestionRepo{tags: []string{"초콜릿"}}
    svc := NewSuggestionService(repo)  // 실제 DB 대신 mock 주입

    _, _ = svc.GetTagSuggestions(context.Background(), "user-1", "  초콜릿  ")

    // 서비스가 공백을 trim해서 repo에 전달했는지 검증
    if repo.lastQ != "초콜릿" {
        t.Errorf("expected trimmed q, got %q", repo.lastQ)
    }
}
```

**`lastQ` 필드 패턴**: mock이 받은 인자를 필드에 기록해두면, "서비스가 어떤 값을 repo에 전달했는가"를 검증할 수 있다. Spring의 `verify(repo).getTagSuggestions(eq("초콜릿"))`와 같은 역할이다.

---

## 7. 검색어 유효성 검사

`SuggestionService`는 `log_service.go`에 이미 정의된 `validateIdentifier`, `newValidationError` 헬퍼를 재사용한다. 같은 패키지에 있으므로 별도 import 없이 호출할 수 있다.

새로 추가한 `normalizeQ`는 검색어 전용 정규화다:

```go
func normalizeQ(q string) (string, error) {
    trimmed := strings.TrimSpace(q)
    if len(trimmed) > 100 {
        return "", newValidationError("q", "검색어는 100자 이하여야 합니다")
    }
    return trimmed, nil
}
```

`q`가 빈 문자열이면 전체 조회를 의미하므로 에러로 처리하지 않는다. SQL의 `WHERE ? = '' OR LOWER(tag) LIKE ...` 조건이 이 경우를 자연스럽게 처리한다.

---

## 8. 응답 형식 결정

자동완성 제안은 단순 배열로 충분하지만, 최상위를 객체로 감쌌다:

```json
{ "suggestions": ["초콜릿", "체리", "플로럴"] }
```

배열을 직접 반환(`["초콜릿", ...]`)하지 않고 객체로 감싼 이유:
- **확장성**: 나중에 `total`, `has_more` 같은 메타데이터를 추가하려면 루트가 객체여야 한다.
- **JSON 파싱 일관성**: 루트가 배열인 JSON은 일부 클라이언트에서 파싱 이슈가 있다.
- 이미 `ListLogsResponse`도 `{ "items": [...] }` 형식을 따르고 있어 일관성이 유지된다.
