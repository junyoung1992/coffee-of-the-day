# Go 학습 가이드 — Phase 1 코드 따라가기

> 대상 독자: Java/Spring 개발자  
> 목표: Coffee of the Day Phase 1 코드를 읽을 때 필요한 Go 문법과 사고방식을 빠르게 익힌다.

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

## 16. 이 프로젝트에서 꼭 익혀야 할 Go 문법 우선순위

Phase 1 코드를 리뷰하려면 아래 순서로 익히는 것이 가장 효율적입니다.

1. `struct`
2. `interface`
3. 포인터 optional (`*string`, `*int`, `*float64`)
4. `error` 반환 + `if err != nil`
5. `defer`
6. `context.Context`
7. `switch`
8. JSON tag와 디코딩
9. slice (`[]T`)
10. `errors.Is`, `errors.As`

이 정도만 익혀도 repository/service/handler 흐름은 대부분 읽힙니다.

---

## 17. 실제 코드 읽기 추천 순서

처음부터 repository 구현을 파고들기보다 아래 순서가 좋습니다.

1. `backend/internal/domain/log.go`
   도메인 데이터 형태 파악
2. `backend/internal/handler/log_handler.go`
   HTTP 입출력 흐름 파악
3. `backend/internal/service/log_service.go`
   비즈니스 규칙 이해
4. `backend/internal/repository/log_repository.go`
   DB 저장 방식 확인
5. `backend/cmd/server/main.go`
   전체 조립 구조 확인

이 순서가 Spring 개발자가 읽을 때 가장 자연스럽습니다.  
Controller → Service → Repository 순으로 내려가는 느낌이기 때문입니다.

---

## 18. 마지막 정리

이 프로젝트의 Go 코드는 "Go다운 저수준 최적화"보다 **웹 서비스 레이어 구조를 명확히 나누는 방식**에 더 가깝습니다.  
그래서 Java/Spring 경험이 있다면 문법만 익숙해지면 구조 자체는 오히려 낯설지 않습니다.

핵심은 다음 세 가지입니다.

- Go의 `struct`는 class보다 DTO에 가깝다
- 예외 대신 `error`를 반환한다
- optional은 포인터로 표현한다

이 세 가지를 먼저 이해하면 Phase 1 백엔드 코드는 훨씬 읽기 쉬워집니다.

---

이 문서는 Codex가 작성했습니다.
