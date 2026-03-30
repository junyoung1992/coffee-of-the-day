# Go 학습 가이드 — 프로젝트 코드 따라가기

> 대상 독자: Java/Spring 개발자
> 목표: Coffee of the Day 백엔드 코드를 읽을 때 필요한 Go 문법과 패턴을 빠르게 익힌다.

---

## 먼저 감 잡기: Go를 Java와 어떻게 대응해서 보면 되나

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

- 데이터는 `struct`
- 동작은 `func`
- 다형성은 `interface`

---

# Part 1: Go 언어 기초

## `struct` — Go의 기본 데이터 타입

Go의 `struct`는 Java의 DTO나 단순 class에 가깝습니다. getter/setter 없이 필드를 직접 씁니다.

```go
type CoffeeLog struct {
    ID         string
    UserID     string
    RecordedAt string
    Companions []string
    LogType    LogType
    Memo       *string
}
```

대문자로 시작하면 **exported(public)**, 소문자로 시작하면 **package-private** 입니다.

→ `internal/domain/log.go`

---

## `interface` — 의존성 분리의 핵심

Java 인터페이스와 거의 같습니다. 차이는 Go는 `implements`를 명시하지 않는다는 점입니다. 메서드를 모두 구현하면 자동으로 인터페이스를 만족합니다 (implicit implementation).

```go
type LogRepository interface {
    CreateLog(ctx context.Context, log domain.CoffeeLogFull) error
    GetLogByID(ctx context.Context, logID, userID string) (domain.CoffeeLogFull, error)
}
```

→ `internal/repository/log_repository.go`

---

## 메서드와 receiver

Go는 class 문법은 없지만, `receiver`를 붙여 메서드를 만듭니다. `*DefaultLogService`의 `*`는 포인터 receiver — 원본 인스턴스를 공유해 사용하며, 큰 struct 복사를 피합니다.

```go
func (s *DefaultLogService) CreateLog(ctx context.Context, userID string, req CreateLogRequest) (domain.CoffeeLogFull, error) {
    ...
}
```

실무에서 service, repository 같은 구조체는 거의 포인터 receiver를 씁니다.

---

## `error` 반환 — Go의 예외 처리 방식

Go는 예외를 던지지 않고, 함수가 `error`를 마지막 반환값으로 돌려줍니다. Java의 `try-catch` 대신 `if err != nil`로 분기합니다.

```go
log, err := h.svc.GetLog(r.Context(), userID, logID)
if err != nil {
    writeServiceError(w, err)
    return
}
```

이 프로젝트는 에러를 층별로 전달합니다: repository(DB 오류, `ErrNotFound`) → service(validation, business error) → handler(HTTP status code 변환).

### sentinel error, wrapping, `errors.Is`, `errors.As`

오류 분류를 위해 sentinel error를 쓰고, 맥락을 붙일 때는 wrapping합니다.

```go
var ErrNotFound = errors.New("log not found")
return fmt.Errorf("get log: %w", err)  // Java의 new RuntimeException("get log", cause) 비슷
```

- `errors.Is(err, service.ErrNotFound)` — 원인 체인 안에 특정 에러가 있는지 검사
- `errors.As(err, &ve)` — 구체 에러 타입으로 꺼냄. wrapping chain까지 따라감

→ `internal/domain/errors.go`, `internal/handler/response.go`

---

## `defer` — 정리 코드를 마지막에 실행

함수가 끝날 때 실행됩니다. Java의 `try-finally`와 비슷합니다. 트랜잭션 처리에서 많이 쓰입니다.

```go
tx, err := r.sqlDB.BeginTx(ctx, nil)
defer tx.Rollback()
// ... 작업 수행 ...
tx.Commit()  // commit이 성공했으면 rollback은 무시됨
```

---

## `context.Context` — 요청 스코프 전달

Go 웹 코드에서 거의 항상 등장합니다. 요청 생명주기, 취소 신호, request-scoped 메타데이터를 함께 담는 객체입니다.

```go
ctx := context.WithValue(r.Context(), userIDKey, userID)
next.ServeHTTP(w, r.WithContext(ctx))
```

Java/Spring처럼 자동 주입이 아니라, 명시적으로 계속 전달합니다.

---

## 포인터로 optional 표현하기

Go에는 `Optional<T>`이 없습니다. optional 값은 포인터로 표현합니다. `nil`이면 값 없음, 값이 있으면 포인터가 실제 값을 가리킵니다.

```go
Memo       *string
Rating     *float64
BrewTimeSec *int
```

`rating`이 0이면 "0점"인지 "입력 안 함"인지 구분이 어렵기 때문에, 포인터를 써서 `nil` = 입력 안 함, `&4.5` = 실제 값으로 구분합니다.

---

## 그 외 자주 쓰이는 문법

### slice (`[]T`)

Java의 `List<T>`와 거의 같은 역할. `append`로 요소 추가, `len(slice)`로 길이 확인.

### `switch`

Java보다 더 자주 쓰이고 더 간결합니다. `log_type`별 처리 분기, 유효성 검사에 자주 나옵니다.

### JSON tag

```go
type createLogRequest struct {
    RecordedAt string `json:"recorded_at"`
}
```

Spring의 `@JsonProperty("recorded_at")`와 같은 역할입니다.

### 함수도 값이다

```go
type DefaultLogService struct {
    now   func() time.Time
    newID func() (string, error)
}
```

테스트를 위해 현재 시각/ID 생성 함수를 주입 가능한 형태로 열어둡니다. Java의 `Clock`, `IdGenerator` 의존성 주입과 같은 목적입니다.

### goroutine과 channel

goroutine은 `go` 키워드로 시작하는 경량 스레드입니다. Java의 `new Thread(() -> ...).start()`보다 훨씬 가볍습니다.

```go
go func() {
    srv.ListenAndServe()
}()
```

channel은 goroutine 간 값을 주고받는 통로입니다. Java의 `BlockingQueue`와 비슷하지만 언어 기본 타입입니다.

```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
<-quit  // 시그널이 올 때까지 블로킹
```

이 프로젝트에서는 graceful shutdown에서 주로 쓰입니다: 메인 goroutine이 시그널 channel을 기다리고, 별도 goroutine이 서버를 실행합니다.

→ `cmd/server/main.go`

### raw string literal

backtick으로 여러 줄 문자열을 표현합니다. SQL을 코드 안에 읽기 쉽게 고정하는 데 사용합니다.

### `time` 패키지

Go는 `DateTimeFormatter` 패턴 문자 대신 **기준 시각 `2006-01-02 15:04:05` 자체를 레이아웃으로 사용**합니다.

```go
time.Parse("2006-01-02", "2026-03-29")
time.Parse(time.RFC3339, "2026-03-29T09:30:00Z")
```

---

# Part 2: 백엔드 패턴

이 프로젝트의 레이어 구조(handler → service → repository)와 결합되는 패턴들입니다.

## 날짜 필터 정규화

날짜 입력(`2026-03-29`)을 그대로 비교하지 않고, 하루의 시작/끝 시각으로 확장한 뒤 SQL에 넘깁니다. 날짜 문자열을 그대로 비교하면 당일 데이터가 누락될 수 있기 때문입니다.

Spring Data에서 `LocalDate`를 `atStartOfDay()`로 확장하는 것과 같은 사고입니다.

→ `internal/service/log_service.go`

---

## Opaque cursor 페이지네이션

커서 struct를 `json.Marshal` → `base64.URLEncoding.EncodeToString` 순서로 문자열화합니다. 클라이언트는 커서 내부 구조를 몰라도 되고, 정렬 기준이 늘어나도 API 파라미터는 `cursor` 하나로 유지됩니다.

→ `internal/repository/cursor.go`

---

## 동적 SQL 조립

필터 유무에 따라 WHERE 절이 달라지므로, SQL 문자열과 `[]any` 인자 배열을 함께 조립합니다. SQL 문자열과 인자 순서가 항상 함께 움직여야 합니다.

```go
query := `SELECT ... FROM coffee_logs WHERE user_id = ?`
args := []any{userID}

if filter.LogType != nil {
    query += ` AND log_type = ?`
    args = append(args, *filter.LogType)
}
```

---

## 문자열 정렬에 기대는 전제

목록 정렬이 `recorded_at DESC, id DESC`이고, SQLite에서는 문자열 비교로 처리됩니다. 이 설계가 안전하려면 **문자열 정렬 결과가 시간 순서와 일치해야** 합니다. 코드를 읽을 때 저장 포맷, 필터 포맷, 커서 `sort_value`가 같은 비교 규칙을 따르는지 함께 봐야 합니다.

---

## `database/sql`과 raw SQL

sqlc만으로 처리하지 않는 쿼리(`json_each` 가상 테이블 등)는 `database/sql`로 직접 실행합니다.

```go
rows, err := r.db.QueryContext(ctx, tagSuggestionsQuery, userID, userID, q, q)
defer rows.Close()
```

`Rows` 순회 패턴은 Go의 정석입니다: `for rows.Next()` → `rows.Scan(...)` → 루프 후 `rows.Err()` 확인.

특히 `[]string(nil)` 대신 `[]string{}`를 반환해 JSON 응답이 `null`이 아니라 빈 배열이 되도록 합니다.

→ `internal/repository/suggestion_repository.go`

---

## 입력 정규화와 validation helper

HTTP 레이어는 문자열을 꺼내오기만 하고, service가 비즈니스 입력으로 써도 되는지 판단합니다. 작은 정규화 함수로 분리하면 새 기능에서도 재사용하기 쉽습니다.

→ `internal/service/suggestion_service.go`

---

## 테스트 패턴

Go에서는 Mockito 없이 인터페이스를 직접 구현한 stub을 많이 씁니다.

```go
type stubLogRepository struct {
    createFunc func(ctx context.Context, log domain.CoffeeLogFull) error
}
```

인터페이스가 작을수록 이 방식이 읽기 쉽습니다.

---

# Part 3: 배포 인프라

## `//go:embed` — 파일을 바이너리에 포함

Java의 `src/main/resources/` → JAR → `ClassLoader.getResource()`와 비슷합니다. 차이는 컴파일러 지시자로 명시적으로 선언한다는 점입니다.

```go
//go:embed all:static
var staticFS embed.FS
```

주의: 선언 파일 기준 상대 경로만 허용, `..` 불가. `all:` prefix는 숨김 파일도 포함.

이 프로젝트에서 embed가 사용되는 두 곳:
- `web/fs.go`: 프론트엔드 빌드 결과물
- `backend/db/embed.go`: 마이그레이션 SQL 파일

---

## Graceful Shutdown

컨테이너 환경에서 SIGTERM/SIGINT를 수신해, 진행 중인 요청을 완료한 후 서버를 종료합니다.

패턴:
1. 별도 고루틴에서 `srv.ListenAndServe()` 실행
2. 메인 고루틴에서 시그널 채널 대기
3. 시그널 수신 → `Shutdown(ctx)` 호출 (타임아웃 30초)
4. DB 연결 등 리소스 정리 후 종료

Java의 `@PreDestroy` / shutdown hook과 비슷하지만, Go에서는 `http.Server`를 직접 생성하고 `Shutdown(ctx)`를 명시적으로 호출합니다.

→ `cmd/server/main.go`

---

## 멀티스테이지 Docker 빌드

```
Stage 1 (node):     프론트엔드 빌드 → web/static/
Stage 2 (golang):   Stage 1 결과물 COPY 후 go build (embed 포함)
Stage 3 (runtime):  바이너리만 복사해 실행
```

`go-sqlite3`가 CGO를 사용하므로 `CGO_ENABLED=1`과 gcc가 포함된 이미지가 필요합니다.

→ `Dockerfile`

---

# 학습 순서 가이드

## Go 문법 우선순위

1. `struct`, `interface`, 메서드 receiver
2. 포인터 optional (`*string`, `*int`, `*float64`)
3. `error` 반환 + `if err != nil` + `errors.Is`, `errors.As`
4. `defer`, `context.Context`
5. `switch`, slice (`[]T`), JSON tag
6. `time.Parse`, `time.Date`, `RFC3339`
7. 함수를 값으로 쓰기, raw string literal
8. `database/sql`의 `QueryContext`, `Rows`, `Scan`
9. `//go:embed`, `os/signal`

## 코드 읽기 추천 순서

처음부터 repository 구현을 파고들기보다 아래 순서가 좋습니다.
Controller → Service → Repository 순으로 내려가고, 마지막에 인프라 코드를 확인하는 흐름입니다.

1. `internal/domain/log.go` — 도메인 데이터 형태 파악
2. `internal/handler/log_handler.go` — HTTP 입출력 흐름 파악
3. `internal/service/log_service.go` — 비즈니스 규칙 이해
4. `internal/repository/log_repository.go` — DB 저장 방식 확인
5. `internal/repository/cursor.go` — 커서 직렬화 방식 확인
6. `internal/service/suggestion_service.go` — 입력 정규화와 에러 전파
7. `internal/repository/suggestion_repository.go` — raw SQL + `sql.Rows` 패턴
8. `cmd/server/main.go` — 전체 조립 구조 확인
9. `web/fs.go`, `backend/db/embed.go` — embed 방식 확인

---

*이 문서는 Codex가 작성하고, 이후 보강되었습니다.*
