# Phase 1-3 Backend 학습 문서

> sqlc 쿼리 작성, 도메인 타입 설계, SQLite 배열 직렬화 헬퍼를 설명합니다.

---

## 1. sqlc 쿼리 파일 작성 방식

**Spring에서의 대응**: MyBatis Mapper XML / `@Query` 애노테이션

sqlc는 SQL 파일에 주석으로 쿼리 이름과 반환 방식을 지정합니다:

```sql
-- name: GetLogByID :one
SELECT * FROM coffee_logs WHERE id = ? AND user_id = ?;
```

- `:one` — 단건 반환 (`SELECT ... LIMIT 1`과 같은 의미)
- `:many` — 여러 행 반환 (`List<T>`)
- `:exec` — 반환값 없음 (`INSERT`, `UPDATE`, `DELETE`)

sqlc가 이 SQL로부터 타입 안전한 Go 함수를 자동 생성합니다:

```go
// 자동 생성된 코드 (internal/db/coffee_logs.sql.go)
func (q *Queries) GetLogByID(ctx context.Context, arg GetLogByIDParams) (CoffeeLog, error)
func (q *Queries) ListLogs(ctx context.Context, arg ListLogsParams) ([]CoffeeLog, error)
func (q *Queries) InsertLog(ctx context.Context, arg InsertLogParams) error
```

SQL을 직접 제어하면서도 컴파일 타임 타입 검사를 받습니다.

---

## 2. sqlc.narg — nullable 파라미터

`ListLogs` 쿼리에서 선택적 필터를 처리하는 방법:

```sql
WHERE user_id = ?
  AND (sqlc.narg('log_type') IS NULL OR log_type = sqlc.narg('log_type'))
  AND (sqlc.narg('date_from') IS NULL OR recorded_at >= sqlc.narg('date_from'))
```

`sqlc.narg('name')`은 nullable 파라미터를 선언합니다. Go에서는 `*string` (포인터)로 생성됩니다.

```go
// nil이면 필터 무시, 값이 있으면 WHERE 조건 적용
type ListLogsParams struct {
    UserID      string
    LogType     *string  // nil = 필터 없음
    DateFrom    *string  // nil = 필터 없음
    // ...
}
```

**Spring에서의 대응**: MyBatis의 `<if test="logType != null">` 동적 쿼리

---

## 3. Cursor-based Pagination 쿼리

**Spring에서의 대응**: `LIMIT/OFFSET` 페이지네이션 (단, 커서 방식이 더 안전)

```sql
AND (
  sqlc.narg('cursor_recorded_at') IS NULL
  OR recorded_at < sqlc.narg('cursor_recorded_at')
  OR (recorded_at = sqlc.narg('cursor_recorded_at') AND id < sqlc.narg('cursor_id'))
)
ORDER BY recorded_at DESC, id DESC
LIMIT ?;
```

**왜 OFFSET 대신 커서인가?**

`LIMIT 20 OFFSET 40`은 40번째 행을 찾기 위해 DB가 처음 40개를 스캔합니다. 페이지가 뒤로 갈수록 느려지고, 중간에 새 항목이 추가되면 같은 항목이 두 번 나오거나 누락됩니다.

커서 방식은 마지막으로 본 항목(`recorded_at`, `id`)을 기준으로 "그 다음부터" 가져옵니다. 항상 인덱스를 직접 탐색하므로 빠르고 일관됩니다.

---

## 4. sqlc generate 결과물

```
internal/db/
├── db.go           # Queries 구조체, DB 연결 관리
├── models.go       # 테이블 구조에 대응하는 Go 구조체
├── coffee_logs.sql.go  # coffee_logs 쿼리 함수들
├── cafe_logs.sql.go
└── brew_logs.sql.go
```

`models.go`에는 DB 테이블과 1:1 대응하는 구조체가 생성됩니다:

```go
// 자동 생성 — 수정하지 않음
type CoffeeLog struct {
    ID         string
    UserID     string
    RecordedAt string
    Companions string  // JSON 배열 문자열 그대로
    LogType    string
    // ...
}
```

이 구조체는 DB 레이어 전용입니다. 비즈니스 로직에서는 `internal/domain/` 의 타입을 사용합니다.

---

## 5. 도메인 타입 설계 (`internal/domain/log.go`)

**Spring에서의 대응**: Entity + DTO 분리

sqlc가 생성한 DB 모델과 별개로 도메인 타입을 정의하는 이유:

- DB 모델(`internal/db`)은 DB 구조를 그대로 반영 (배열이 JSON 문자열)
- 도메인 타입(`internal/domain`)은 비즈니스 로직에서 쓰기 편한 형태 (배열이 `[]string`)
- 핸들러가 클라이언트에게 반환하는 JSON 구조와도 일치

```go
// DB 모델 (sqlc 생성)
type CoffeeLog struct {
    Companions string // "[\"지수\",\"민준\"]"
}

// 도메인 타입 (직접 정의)
type CoffeeLog struct {
    Companions []string // ["지수", "민준"]
}
```

`CoffeeLogFull`은 공통 필드(`CoffeeLog`)에 타입별 서브 객체를 포함하는 응답 구조입니다:

```go
type CoffeeLogFull struct {
    CoffeeLog              // 공통 필드 임베딩
    Cafe *CafeDetail       // cafe면 존재, brew면 nil
    Brew *BrewDetail       // brew면 존재, cafe면 nil
}
```

**Spring에서의 대응**: `@Inheritance(JOINED)` 엔티티를 응답 DTO로 변환한 것

---

## 6. Go 임베딩 (Embedding)

```go
type CoffeeLogFull struct {
    CoffeeLog   // 임베딩
    Cafe *CafeDetail
    Brew *BrewDetail
}
```

`CoffeeLog`의 모든 필드가 `CoffeeLogFull`에서 직접 접근 가능합니다:

```go
full := CoffeeLogFull{...}
fmt.Println(full.ID)        // full.CoffeeLog.ID와 동일
fmt.Println(full.RecordedAt)
```

**Spring에서의 대응**: 상속(`extends`)과 비슷하지만, Go는 상속이 없고 컴포지션(포함)으로 같은 효과를 냅니다.

---

## 7. StringsToJSON / JSONToStrings 헬퍼 (TDD로 구현)

SQLite는 배열 타입이 없어서 `["초콜릿","체리"]` 같은 배열을 TEXT로 저장합니다. 변환을 담당하는 두 함수를 TDD로 구현했습니다.

```go
func StringsToJSON(ss []string) string {
    if ss == nil {
        return "[]"
    }
    b, _ := json.Marshal(ss)
    return string(b)
}

func JSONToStrings(s string) []string {
    var out []string
    json.Unmarshal([]byte(s), &out)
    if out == nil {
        out = []string{}
    }
    return out
}
```

**왜 에러를 무시하는가?**
`json.Marshal([]string)`은 실패할 수 없습니다 (문자열 슬라이스는 항상 유효한 JSON). `json.Unmarshal` 실패(잘못된 JSON)는 빈 슬라이스를 반환하는 것이 더 안전한 기본값입니다.

### TDD로 작성된 테스트 케이스

| 테스트 | 검증 내용 |
|--------|-----------|
| `TestStringsToJSON_EmptySlice` | `nil` → `"[]"` |
| `TestStringsToJSON_SingleElement` | `["a"]` → `"[\"a\"]"` |
| `TestStringsToJSON_MultipleElements` | 복수 원소 |
| `TestStringsToJSON_SpecialCharacters` | 특수문자 이스케이프 |
| `TestJSONToStrings_ValidJSONArray` | `"[\"a\"]"` → `["a"]` |
| `TestJSONToStrings_EmptyJSONArray` | `"[]"` → `[]string{}` |
| `TestJSONToStrings_NullLiteral` | `"null"` → `[]string{}` |
| `TestJSONToStrings_InvalidJSON` | 잘못된 JSON → `[]string{}` |
| `TestJSONToStrings_EmptyString` | `""` → `[]string{}` |

**Spring에서의 대응**: `@Converter` (`AttributeConverter<List<String>, String>`)와 동일한 역할

---

## 8. 테스트 파일이 소스 옆에 있는 이유

`strings.go` 옆에 `strings_test.go`가 있는 것이 어색하게 느껴진다면, Java/Spring에 익숙해서 그런 거야. 사실 **Java/Spring이 오히려 특이한 케이스**야.

### Java/Spring — `src/main` vs `src/test` 분리

```
src/
├── main/java/com/example/domain/StringHelper.java
└── test/java/com/example/domain/StringHelperTest.java
```

Maven/Gradle의 빌드 관례에서 비롯된 구조. 빌드 도구가 두 디렉토리를 별도로 처리해서 `test` 코드가 운영 JAR에 포함되지 않는다.

### Go — 소스 옆에 `_test.go`

```
internal/domain/
├── strings.go
└── strings_test.go
```

Go 컴파일러가 `_test.go`로 끝나는 파일을 언어 수준에서 직접 처리한다.
- `go build` — `_test.go` 파일 제외 (운영 바이너리에 미포함)
- `go test` — `_test.go` 파일 포함하여 컴파일

빌드 도구 관례가 아니라 **언어 자체가 이 패턴을 지원**하기 때문에 별도 디렉토리가 필요 없다. 테스트 대상 코드와 테스트 코드가 바로 옆에 있어서 탐색도 쉽다.

### TypeScript/React — 소스 옆이 기본

```
src/api/
├── client.ts
└── client.test.ts      ← 소스 옆 (일반적)
```

또는 `__tests__/` 서브디렉토리에 모으기도 하지만, 소스 옆이 요즘 React 생태계의 기본값이다.

### 언어별 비교

| 언어 | 테스트 파일 위치 | 근거 |
|------|----------------|------|
| Java/Spring | `src/test/` 분리 | Maven/Gradle 빌드 관례 |
| Go | 소스 옆 (`_test.go`) | 언어 자체 지원 |
| TypeScript/React | 소스 옆 or `__tests__/` | 도구가 유연하게 지원 |
| Python, Rust 등 | 소스 옆이 기본 | Go와 같은 흐름 |

Java가 특이한 케이스이고, 현대 언어 대부분은 소스 옆에 테스트를 두는 것을 기본으로 한다.
