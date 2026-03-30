# Go 학습 가이드 — 프로젝트 코드 따라가기

> 대상 독자: Java/Spring 개발자
> 목표: Coffee of the Day 코드를 읽을 때 필요한 Go 문법과 사고방식을 빠르게 익힌다.

---

## 1. 먼저 감 잡기: Go를 Java와 어떻게 대응해서 보면 되나

이 프로젝트의 Go 코드는 대략 아래처럼 대응해서 보면 이해가 빠릅니다.

| Go | Java/Spring에서 비슷한 개념 |
|------|------|
| `package` | Java의 package |
| `struct` | 필드만 있는 class / record / DTO |
| `interface` | Java interface |
| `func` | 메서드 또는 static 함수 |
| 메서드 receiver | 인스턴스 메서드 |
| `context.Context` | 요청 스코프 메타데이터, 취소 신호 |
| `error` 반환 | checked exception을 값으로 돌려주는 스타일 |
| `defer` | `try-finally`의 finally 쪽 정리 코드 |
| `[]T` | `List<T>` 비슷한 동적 배열 |
| `map[K]V` | `Map<K,V>` |
| `*string`, `*int`, `*float64` | nullable wrapper처럼 쓰는 optional 값 |

중요한 차이는 하나입니다.

**Go는 객체지향 문법보다 "단순한 데이터 + 함수 조합"을 선호합니다.**

그래서 Java처럼 클래스 안에 모든 것을 몰아넣기보다:

- 데이터는 `struct`
- 동작은 `func`
- 다형성은 `interface`

방식으로 더 평평하게 작성하는 편입니다.

---

## 2. `package`와 파일 구조

Go는 파일이 속한 디렉토리가 곧 패키지입니다.

예:

```go
package service
```

즉 `backend/internal/service/` 아래 파일들은 모두 `service` 패키지입니다.

이 프로젝트에서 중요한 패키지:

- `internal/domain`: 도메인 타입
- `internal/repository`: DB 접근
- `internal/service`: 비즈니스 규칙
- `internal/handler`: HTTP 입출력

Spring의 `controller`, `service`, `repository` 패키지 나누기와 거의 같은 의미로 보면 됩니다.

---

## 3. `struct` — Go의 기본 데이터 타입

Go의 `struct`는 Java의 DTO나 단순 class에 가깝습니다.

```go
type CoffeeLog struct {
    ID         string
    UserID     string
    RecordedAt string
    Companions []string
    LogType    LogType
    Memo       *string
    CreatedAt  string
    UpdatedAt  string
}
```

이 프로젝트에서는 `struct`를 주로 아래 용도로 씁니다.

- 도메인 모델
- 요청/응답 모델
- DB 매핑 모델
- 설정 객체

Java처럼 getter/setter를 두지 않고, 보통 필드를 직접 씁니다.

### 대문자로 시작하는 필드가 중요한 이유

Go는 대문자로 시작하면 **exported(public)**, 소문자로 시작하면 **package-private 비슷한 private** 입니다.

예:

```go
type Config struct {
    Port string   // 외부 패키지에서 접근 가능
}

type defaultLogService struct {
    repo LogRepository // 외부 패키지에서 직접 접근 불가
}
```

---

## 4. `interface` — 의존성 분리의 핵심

Java 인터페이스와 거의 같습니다.

```go
type LogRepository interface {
    CreateLog(ctx context.Context, log domain.CoffeeLogFull) error
    GetLogByID(ctx context.Context, logID, userID string) (domain.CoffeeLogFull, error)
}
```

이 프로젝트는 다음 식으로 의존성을 뒤집습니다.

- handler는 `service.LogService`에 의존
- service는 `repository.LogRepository`에 의존

Spring에서는 보통:

```java
public interface LogRepository { ... }
public class LogService {
    private final LogRepository repository;
}
```

처럼 쓰는데, Go도 개념은 동일합니다.

차이는 Go는 `implements`를 명시하지 않는다는 점입니다.

```go
type SQLiteLogRepository struct { ... }
```

이 구조체가 `LogRepository` 메서드를 모두 구현하면, 자동으로 그 인터페이스를 만족합니다.

이걸 **implicit implementation**이라고 생각하면 됩니다.

---

## 5. 메서드와 receiver

Go는 class 문법은 없지만, `receiver`를 붙여 메서드를 만듭니다.

```go
func (s *DefaultLogService) CreateLog(ctx context.Context, userID string, req CreateLogRequest) (domain.CoffeeLogFull, error) {
    ...
}
```

Java로 치면:

```java
class DefaultLogService {
    CoffeeLogFull createLog(...) { ... }
}
```

와 같습니다.

### `*DefaultLogService`의 `*`는 무엇인가

포인터 receiver입니다.

의미:
- 원본 인스턴스를 공유해 사용
- 큰 struct 복사를 피함
- 내부 상태가 있으면 수정 가능

실무에서는 service, repository 같은 구조체는 거의 포인터 receiver를 쓴다고 보면 됩니다.

---

## 6. `error` 반환 — Go의 예외 처리 방식

Go는 예외를 던지지 않고, 대부분 함수가 `error`를 마지막 반환값으로 돌려줍니다.

```go
log, err := h.svc.GetLog(r.Context(), userID, logID)
if err != nil {
    writeServiceError(w, err)
    return
}
```

Java에서는:

```java
try {
    return service.getLog(...);
} catch (NotFoundException e) {
    ...
}
```

처럼 하던 흐름을 Go에서는 `if err != nil`로 분기합니다.

### 왜 이런 방식이 중요한가

이 프로젝트는 오류를 이렇게 층별로 전달합니다.

- repository: DB 오류 또는 `ErrNotFound`
- service: validation error, business error
- handler: HTTP status code로 변환

즉, 예외 스택 대신 **에러 값을 리턴하며 위로 올리는 방식**입니다.

---

## 7. sentinel error, wrapping, `errors.Is`, `errors.As`

Go는 오류 분류를 위해 sentinel error를 자주 씁니다.

```go
var ErrNotFound = errors.New("log not found")
```

그리고 더 자세한 맥락을 붙일 때는 wrapping합니다.

```go
return fmt.Errorf("get log: %w", err)
```

Java에서 `new RuntimeException("get log", cause)`와 비슷하다고 보면 됩니다.

### `errors.Is`

원인 체인 안에 특정 에러가 있는지 검사합니다.

```go
if errors.Is(err, service.ErrNotFound) {
    ...
}
```

### `errors.As`

구체 에러 타입으로 꺼냅니다.

```go
var ve *service.ValidationError
if errors.As(err, &ve) {
    ...
}
```

Java의 `instanceof ValidationError`와 비슷하지만, wrapping chain까지 따라간다고 이해하면 됩니다.

---

## 8. `defer` — 정리 코드를 마지막에 실행

Go의 `defer`는 함수가 끝날 때 실행됩니다.

```go
tx, err := r.sqlDB.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()
```

Java로 치면:

```java
try {
    ...
} finally {
    rollback();
}
```

와 비슷합니다.

이 프로젝트에서는 특히 트랜잭션 처리에서 많이 쓰입니다.

패턴은 보통 이렇습니다.

1. `BeginTx`
2. `defer Rollback`
3. 중간 작업 수행
4. 마지막에 `Commit`

이미 commit이 성공했으면 rollback은 실질적으로 무시됩니다.

---

## 9. `context.Context` — 요청 스코프 전달

Go 웹 코드에서 거의 항상 등장합니다.

```go
func (h *LogHandler) GetLog(w http.ResponseWriter, r *http.Request) {
    log, err := h.svc.GetLog(r.Context(), userID, logID)
}
```

Spring에서 대응되는 개념은 한 가지로 딱 맞지는 않지만, 대략:

- 요청 생명주기
- 취소 신호
- request-scoped 메타데이터

를 함께 담는 객체라고 보면 됩니다.

이 프로젝트에서는 `X-User-Id`를 context에 담아 내려보냅니다.

```go
ctx := context.WithValue(r.Context(), userIDKey, userID)
next.ServeHTTP(w, r.WithContext(ctx))
```

Java/Spring처럼 메서드 파라미터에 자동 주입되는 느낌은 아니고, 명시적으로 계속 전달합니다.

---

## 10. 포인터로 optional 표현하기

Go에는 Java의 `Integer`, `Double`, `Optional<String>` 같은 nullable wrapper가 없습니다.

그래서 optional 값은 보통 포인터로 표현합니다.

```go
Memo       *string
Rating     *float64
BrewTimeSec *int
```

의미:
- `nil`이면 값 없음
- 값이 있으면 포인터가 실제 값을 가리킴

이 방식은 이 프로젝트에서 특히 JSON/DB optional 필드에 많이 쓰입니다.

Java로 보면:

- `String?`
- `Double?`
- `Integer?`

처럼 쓰는 느낌에 가깝습니다.

### 왜 primitive가 아니라 포인터인가

예를 들어 `rating`이 0이면 "0점"인지 "입력 안 함"인지 구분이 어렵습니다.  
포인터를 쓰면:

- `nil` = 입력 안 함
- `&4.5` = 실제 값 4.5

처럼 구분할 수 있습니다.

---

## 11. slice (`[]T`) — Go의 리스트

```go
Companions []string
TastingTags []string
BrewSteps []string
```

Java의 `List<String>`와 거의 같은 역할입니다.

차이:
- 인터페이스가 아니라 언어 기본 타입
- `append`로 요소 추가
- `len(slice)`로 길이 확인

예:

```go
normalized := make([]string, 0, len(values))
normalized = append(normalized, trimmed)
```

Java로 치면:

```java
List<String> normalized = new ArrayList<>();
normalized.add(trimmed);
```

과 같습니다.

---

## 12. `switch` — 타입/상태 분기

Go의 `switch`는 Java보다 더 자주 쓰이고, 더 간결합니다.

```go
switch logType {
case domain.LogTypeCafe:
    ...
case domain.LogTypeBrew:
    ...
}
```

이 프로젝트에서 `switch`는 다음 상황에 자주 나옵니다.

- `log_type`별 처리 분기
- `brew_method`, `roast_level` 유효성 검사
- 에러 타입 분기

특히 Java의 `if-else` 연쇄보다 읽기 좋게 쓰는 경우가 많습니다.

---

## 13. JSON 디코딩과 struct tag

handler의 요청/응답 타입을 보면 JSON tag가 붙어 있습니다.

```go
type createLogRequest struct {
    RecordedAt string `json:"recorded_at"`
    LogType    string `json:"log_type"`
}
```

Spring에서:

```java
@JsonProperty("recorded_at")
private String recordedAt;
```

와 같은 역할입니다.

### 디코딩

```go
var req createLogRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    ...
}
```

즉, JSON body를 struct에 바인딩합니다.

Spring MVC의 `@RequestBody CreateLogRequest req`와 비슷합니다.

---

## 14. 함수도 값이다

Go에서는 함수를 필드에 넣을 수 있습니다.

```go
type DefaultLogService struct {
    repo  repository.LogRepository
    now   func() time.Time
    newID func() (string, error)
}
```

이 프로젝트에서는 테스트를 쉽게 하려고 현재 시각 생성 함수, ID 생성 함수를 주입 가능한 형태로 열어뒀습니다.

Java로 치면:

- `Clock`
- `IdGenerator`

같은 의존성을 인터페이스로 주입하는 것과 비슷한 목적입니다.

---

## 15. Go 테스트에서 자주 보이는 패턴

### stub 구현체

Go에서는 Mockito 같은 프레임워크 없이 인터페이스를 직접 구현한 stub을 많이 씁니다.

```go
type stubLogRepository struct {
    createFunc func(ctx context.Context, log domain.CoffeeLogFull) error
}
```

이 방식은 처음에는 수동처럼 보이지만, 인터페이스가 작을수록 꽤 읽기 쉽습니다.

### table test

Go는 반복 시나리오 검증에 table-driven test를 자주 씁니다.  
이 프로젝트에서는 table test 비중이 아주 높진 않지만, Go 문화에서는 매우 흔한 패턴입니다.

---

## 16. `time` 패키지와 레이아웃 문자열

Phase 2부터는 날짜 필터와 `recorded_at` 검증 때문에 `time` 패키지를 읽을 줄 알아야 합니다.

Go는 Java의 `DateTimeFormatter`처럼 패턴 문자를 쓰지 않습니다.  
대신 **기준 시각 `2006-01-02 15:04:05` 자체를 레이아웃으로 사용**합니다.

```go
time.Parse("2006-01-02", "2026-03-29")
time.Parse(time.RFC3339, "2026-03-29T09:30:00Z")
```

Java로 치면:

```java
LocalDate.parse("2026-03-29", DateTimeFormatter.ofPattern("yyyy-MM-dd"))
OffsetDateTime.parse("2026-03-29T09:30:00Z", DateTimeFormatter.ISO_OFFSET_DATE_TIME)
```

와 비슷합니다.

이 프로젝트에서 특히 중요한 포인트:

- `time.RFC3339`, `time.RFC3339Nano` 같은 상수 레이아웃이 있다
- 날짜만 있는 입력(`YYYY-MM-DD`)과 datetime(`RFC3339`)은 구분해서 처리한다
- 같은 시각이라도 문자열 표현이 다르면 SQLite 문자열 비교 결과가 달라질 수 있다

---

## 17. 날짜 필터 정규화: 날짜를 경계값으로 확장하기

Phase 2 백엔드는 `date_from=2026-03-29`, `date_to=2026-03-29` 같은 입력을 그대로 비교하지 않습니다.  
하루의 시작/끝 시각으로 바꾼 뒤 SQL에 넘깁니다.

```go
if d, err := time.Parse("2006-01-02", trimmed); err == nil {
    if endOfDay {
        normalized = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, time.UTC)
    } else {
        normalized = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
    }
}
```

이건 Spring Data에서 `LocalDate`를 `atStartOfDay()`나 `LocalTime.MAX`로 확장하는 것과 같은 사고입니다.

왜 필요한가:

- DB에는 `recorded_at`이 datetime 문자열로 저장됨
- 사용자는 날짜만 입력함
- 날짜 문자열을 그대로 비교하면 당일 데이터가 누락될 수 있음

즉, **날짜 필터는 항상 비교 가능한 datetime 경계값으로 정규화해야 한다**는 점이 핵심입니다.

---

## 18. Opaque cursor — struct를 JSON/base64로 직렬화하기

Phase 2 목록 조회는 커서 기반 페이지네이션을 사용합니다.

```go
type Cursor struct {
    SortBy    string `json:"sort_by"`
    Order     string `json:"order"`
    SortValue string `json:"sort_value"`
    ID        string `json:"id"`
}
```

그리고 이 값을:

1. `json.Marshal`
2. `base64.URLEncoding.EncodeToString`

순서로 문자열화합니다.

```go
raw, _ := json.Marshal(c)
return base64.URLEncoding.EncodeToString(raw)
```

Java로 보면 DTO를 JSON 문자열로 직렬화한 뒤 URL-safe Base64로 감싸는 것과 비슷합니다.

이 방식을 쓰는 이유:

- 클라이언트는 커서 내부 구조를 몰라도 됨
- 정렬 기준이 늘어나도 API 파라미터는 `cursor` 하나로 유지 가능
- offset 기반 페이지네이션보다 삽입/삭제에 덜 취약함

---

## 19. 동적 SQL 문자열 조립과 `[]any`

Phase 2 목록 API는 필터 유무에 따라 WHERE 절이 달라집니다.  
그래서 repository에서 SQL 문자열과 인자 배열을 함께 조립합니다.

```go
query := `SELECT ... FROM coffee_logs WHERE user_id = ?`
args := []any{userID}

if filter.LogType != nil {
    query += ` AND log_type = ?`
    args = append(args, *filter.LogType)
}
```

여기서 `[]any`는 Java의 `List<Object>`처럼 "타입이 서로 다른 SQL 바인딩 값 목록"이라고 보면 됩니다.

중요한 점은 SQL 문자열과 인자 순서가 항상 함께 움직여야 한다는 것입니다.  
문자열만 늘리고 `args`를 빼먹으면 바인딩이 틀어지고, 반대도 마찬가지입니다.

---

## 20. 문자열 정렬에 기대는 코드의 전제

이 프로젝트의 목록 정렬은 `recorded_at DESC, id DESC`이고, SQLite에서는 이를 문자열 비교로 처리합니다.

```go
query += ` ORDER BY recorded_at DESC, id DESC LIMIT ?`
```

즉, 이 설계가 안전하려면 **문자열 정렬 결과가 시간 순서와 일치해야 합니다.**

그래서 Phase 2 코드를 읽을 때는 단순히 "시간 파싱"만 볼 것이 아니라:

- 어떤 포맷으로 저장하는가
- 필터 값도 같은 포맷으로 맞추는가
- 커서의 `sort_value`도 같은 비교 규칙을 따르는가

를 함께 봐야 합니다.

---

## 21. `database/sql`과 raw SQL 병행 읽기

Phase 3 자동완성에서는 sqlc만으로 처리하지 않고, 일부 쿼리를 `database/sql`로 직접 실행합니다.

```go
rows, err := r.db.QueryContext(ctx, tagSuggestionsQuery, userID, userID, q, q)
if err != nil {
    return nil, fmt.Errorf("get tag suggestions: %w", err)
}
defer rows.Close()
```

이 코드는 "sqlc를 우회한 예외 처리"가 아니라, **sqlc도 내부적으로 쓰는 표준 라이브러리 API를 직접 호출한 것**입니다.

Spring/JdbcTemplate으로 비유하면:

- 평소에는 생성된 repository 메서드를 쓰다가
- 특수한 집계 쿼리만 `jdbcTemplate.query(...)`를 직접 쓰는 상황

과 비슷합니다.

여기서 읽어야 할 포인트는 세 가지입니다.

- `QueryContext`는 여러 행을 반환하는 SELECT에 사용한다
- 바인딩 값은 `?` 개수와 순서를 정확히 맞춰야 한다
- 반환된 `rows`는 반드시 `Close()`해야 한다

즉, Phase 3부터는 "Go 문법"만이 아니라 **표준 라이브러리 DB API를 직접 읽는 눈**도 필요합니다.

---

## 22. `sql.Rows` 순회와 스캔 패턴

raw SQL을 직접 실행하면 결과를 수동으로 읽어야 합니다.

```go
func scanSuggestions(rows *sql.Rows) ([]string, error) {
    var suggestions []string
    for rows.Next() {
        var value string
        var cnt int64
        if err := rows.Scan(&value, &cnt); err != nil {
            return nil, fmt.Errorf("scan suggestion row: %w", err)
        }
        suggestions = append(suggestions, value)
    }
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("iterate suggestion rows: %w", err)
    }
    if suggestions == nil {
        return []string{}, nil
    }
    return suggestions, nil
}
```

이 패턴은 거의 Go의 정석입니다.

1. `for rows.Next()`로 한 행씩 순회
2. `rows.Scan(...)`으로 현재 행 값을 변수에 복사
3. 루프가 끝난 뒤 `rows.Err()`로 순회 중 오류 확인

Java의 `ResultSet`와 매우 비슷하지만, Go는 보통 이 흐름을 helper 함수로 감싸 재사용합니다.

특히 이 프로젝트에서는 `[]string(nil)` 대신 `[]string{}`를 반환해 JSON 응답이 `null`이 아니라 빈 배열이 되도록 맞춥니다.
즉, DB 계층에서부터 API 응답 모양까지 의식하고 있다는 뜻입니다.

---

## 23. 입력 정규화 함수와 validation helper 재사용

Phase 3 서비스 코드는 새 검증 로직을 전부 다시 쓰지 않고, 기존 helper를 재사용합니다.

```go
normalizedUserID, err := validateIdentifier("user_id", userID)
if err != nil {
    return nil, err
}

normalizedQ, err := normalizeQ(q)
if err != nil {
    return nil, err
}
```

그리고 `normalizeQ`는 아주 작은 함수로 분리되어 있습니다.

```go
func normalizeQ(q string) (string, error) {
    trimmed := strings.TrimSpace(q)
    if len(trimmed) > 100 {
        return "", newValidationError("q", "검색어는 100자 이하여야 합니다")
    }
    return trimmed, nil
}
```

이런 코드는 Java/Spring의:

- `@Validated`로 다 해결되지 않는 입력 정리
- 서비스 진입 직후 수행하는 `trim`, 길이 제한, 공통 예외 생성

과 비슷한 역할입니다.

핵심은 다음입니다.

- HTTP 레이어는 문자열을 꺼내오기만 한다
- service는 "이 문자열을 실제 비즈니스 입력으로 써도 되는가"를 판단한다
- 정규화와 검증을 작은 함수로 분리하면 새 기능에서도 재사용하기 쉽다

Phase 3 코드를 읽을 때는 "기능이 단순한데 왜 service를 거치지?"가 아니라, **입력 정규화와 에러 규약을 한 곳에 모으기 위해서**라고 이해하면 됩니다.

---

## 24. raw string literal과 긴 SQL 상수

Go는 여러 줄 문자열을 backtick으로 표현할 수 있습니다.

```go
const companionSuggestionsQuery = `
SELECT j.value AS companion, COUNT(*) AS cnt
FROM coffee_logs l
JOIN json_each(l.companions) j
WHERE l.user_id = ?
  AND (? = '' OR LOWER(j.value) LIKE '%' || LOWER(?) || '%')
GROUP BY companion
ORDER BY cnt DESC, companion ASC
LIMIT 10
`
```

Java 15+ text block과 비슷하지만 더 단순합니다.

이 방식을 쓰는 이유:

- SQL 줄바꿈을 거의 그대로 유지할 수 있다
- `\n` 연결 없이 읽기 쉽다
- DB에서 실제 실행되는 쿼리와 코드 상 문자열이 거의 동일하다

Phase 3 repository를 읽을 때는 Go 문법보다도, **SQL을 코드 안에 어떻게 안전하게 고정하는지**를 함께 보는 것이 중요합니다.

---

## 25. 이 프로젝트에서 꼭 익혀야 할 Go 문법 우선순위

프로젝트 코드를 리뷰하려면 아래 순서로 익히는 것이 가장 효율적입니다.

1. `struct`
2. `interface`
3. 포인터 optional (`*string`, `*int`, `*float64`)
4. `error` 반환 + `if err != nil`
5. `defer`
6. `context.Context`
7. `switch`
8. `time.Parse`, `time.Date`, `RFC3339`
9. JSON tag와 디코딩
10. slice (`[]T`)
11. `errors.Is`, `errors.As`
12. `encoding/json`, `encoding/base64`
13. `database/sql`의 `QueryContext`, `Rows`, `Scan`
14. `strings.TrimSpace`, `len` 같은 입력 정규화 함수
15. raw string literal로 작성된 멀티라인 SQL
16. `//go:embed`와 `embed.FS` (파일을 바이너리에 포함)
17. `os/signal`과 `http.Server.Shutdown` (graceful shutdown)

이 정도만 익혀도 Phase 3 자동완성 흐름 + 배포 파이프라인 코드까지 대부분 읽힙니다.

---

## 26. 실제 코드 읽기 추천 순서

처음부터 repository 구현을 파고들기보다 아래 순서가 좋습니다.

1. `backend/internal/domain/log.go`
   도메인 데이터 형태 파악
2. `backend/internal/handler/log_handler.go`
   HTTP 입출력 흐름 파악
3. `backend/internal/service/log_service.go`
   비즈니스 규칙 이해
4. `backend/internal/repository/log_repository.go`
   DB 저장 방식 확인
5. `backend/internal/repository/cursor.go`
   커서 직렬화 방식 확인
6. `backend/internal/service/suggestion_service.go`
   입력 정규화와 에러 전파 방식 확인
7. `backend/internal/repository/suggestion_repository.go`
   raw SQL + `sql.Rows` 패턴 확인
8. `backend/internal/handler/suggestion_handler.go`
   query string → service → JSON 응답 흐름 확인
9. `backend/cmd/server/main.go`
   전체 조립 구조 확인

배포 파이프라인 코드를 읽으려면 여기에 추가합니다.

10. `web/fs.go`
    embed와 SPA fallback 방식 확인
11. `backend/db/embed.go`
    마이그레이션 embed 방식 확인
12. `backend/cmd/server/main.go` (graceful shutdown 부분)
    signal 수신 + Shutdown 호출 흐름 확인

이 순서가 Spring 개발자가 읽을 때 가장 자연스럽습니다.
Controller → Service → Repository 순으로 내려가고, 마지막에 인프라 코드를 확인하는 흐름입니다.

---

## 27. 마지막 정리

이 프로젝트의 Go 코드는 "Go다운 저수준 최적화"보다 **웹 서비스 레이어 구조를 명확히 나누는 방식**에 더 가깝습니다.
그래서 Java/Spring 경험이 있다면 문법만 익숙해지면 구조 자체는 오히려 낯설지 않습니다.

핵심은 다음 세 가지입니다.

- Go의 `struct`는 class보다 DTO에 가깝다
- 예외 대신 `error`를 반환한다
- optional은 포인터로 표현한다

Phase 2부터는 여기에 두 가지가 더 중요해집니다.

- 시간 값을 비교 가능한 문자열로 정규화하는 사고
- 커서/필터 같은 인프라 규칙을 코드로 직접 다루는 방식

Phase 3부터는 여기에 세 가지가 더 붙습니다.

- sqlc와 `database/sql`을 상황에 맞게 병행하는 방식
- `Rows`를 직접 순회하며 응답 형태를 조립하는 방식
- 작은 정규화 함수로 입력 규약을 재사용하는 방식

Issue #1 배포 파이프라인에서는 다음이 추가됩니다.

- `//go:embed`로 정적 파일과 마이그레이션을 바이너리에 포함하는 방식
- `os/signal`과 `http.Server.Shutdown`으로 컨테이너 환경에서 안전하게 종료하는 방식
- 멀티스테이지 Docker 빌드에서 Go 바이너리가 만들어지는 과정

이 흐름까지 이해하면 프로젝트 전체 코드를 읽는 데 어려움이 없습니다.

---

## 28. `//go:embed` — 파일을 바이너리에 포함하기

Issue #1에서 프론트엔드 빌드 결과물과 마이그레이션 파일을 Go 바이너리에 포함하기 위해 `embed` 패키지를 사용합니다.

```go
import "embed"

//go:embed all:static
var staticFS embed.FS
```

이 한 줄로 `static/` 디렉토리 전체가 컴파일 시 바이너리에 포함됩니다.

Java로 비유하면 `src/main/resources/`에 넣은 파일이 JAR에 포함되어 `ClassLoader.getResource()`로 접근하는 것과 비슷합니다. 차이는 Go에서는 컴파일러 지시자(`//go:embed`)로 명시적으로 선언한다는 점입니다.

주의할 점:
- `//go:embed`는 선언 파일 기준 **상대 경로만** 허용하고 `..`은 불가합니다
- `all:` prefix는 `.`으로 시작하는 숨김 파일도 포함합니다 (예: `.gitkeep`)
- embed한 파일은 `io/fs.FS` 인터페이스를 만족하므로 표준 라이브러리와 자연스럽게 조합됩니다

```go
// 서브 디렉토리를 기준으로 새 FS 생성
sub, err := fs.Sub(staticFS, "static")

// HTTP 파일 서버로 사용
http.FileServer(http.FS(sub))
```

이 프로젝트에서 embed가 사용되는 두 곳:
- `web/fs.go`: 프론트엔드 빌드 결과물 (`web/static/`)
- `backend/db/embed.go`: 마이그레이션 SQL 파일 (`backend/db/migrations/`)

---

## 29. `os/signal` — 시그널 기반 Graceful Shutdown

컨테이너 환경(Docker, Fly.io)에서는 배포 교체 시 SIGTERM이 전송됩니다. 이를 수신해 진행 중인 요청을 완료한 후 서버를 종료해야 데이터 손실을 방지할 수 있습니다.

```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
<-quit  // 시그널이 올 때까지 블로킹

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

Java/Spring Boot에서는 `@PreDestroy`나 shutdown hook이 비슷한 역할을 합니다. 차이는 Go에서는 `http.Server`를 직접 생성하고 `Shutdown(ctx)`를 명시적으로 호출한다는 점입니다.

패턴 요약:
1. 별도 고루틴에서 `srv.ListenAndServe()` 실행
2. 메인 고루틴에서 시그널 채널 대기
3. 시그널 수신 → `Shutdown(ctx)` 호출 (타임아웃 내 진행 중 요청 완료)
4. DB 연결 등 리소스 정리 후 종료

---

## 30. 멀티스테이지 Docker 빌드와 Go

이 프로젝트의 Dockerfile은 3단계로 구성됩니다.

```
Stage 1 (node):     프론트엔드 빌드 → web/static/
Stage 2 (golang):   Stage 1 결과물 COPY 후 go build (embed 포함)
Stage 3 (runtime):  바이너리만 복사해 실행
```

Go의 특성상 Stage 2에서 빌드한 바이너리 하나에 모든 것(API 서버, 정적 파일, 마이그레이션)이 포함됩니다. Runtime 이미지에는 Go 툴체인이 필요 없습니다.

단, `go-sqlite3`가 CGO를 사용하므로 `CGO_ENABLED=1`과 gcc가 포함된 이미지(`golang:1.25-bookworm`)가 필요합니다. 순수 Go 드라이버를 쓰면 `alpine`이나 `scratch` 이미지도 가능하지만, `go-sqlite3`의 성숙도와 성능을 우선했습니다.

Java로 비유하면 "Spring Boot fat JAR를 빌드한 뒤, JRE만 있는 경량 이미지에 복사"하는 패턴과 같습니다. Go는 JRE조차 필요 없으므로 더 작은 이미지가 가능합니다.

---

이 문서는 Codex가 작성하고, Issue #1 배포 파이프라인 작업 시 Claude가 보강했습니다.
