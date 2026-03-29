# Phase 1-6 Backend — Handler 및 미들웨어

> 대상 독자: Java/Spring 경험이 있는 개발자. Go와 HTTP 핸들러 패턴을 Spring과 비교하여 설명합니다.

---

## 무엇을 만들었나

이 단계에서는 Service 계층까지 완성된 비즈니스 로직을 HTTP로 노출시켰습니다.

| 파일 | 역할 |
|------|------|
| `internal/handler/middleware.go` | `X-User-Id` 헤더 파싱 미들웨어, CORS 미들웨어 |
| `internal/handler/log_handler.go` | 5개 엔드포인트 HTTP 핸들러 + JSON 타입 |
| `internal/handler/log_handler_test.go` | 핸들러 단위 테스트 |
| `cmd/server/main.go` | Repository → Service → Handler 연결, 라우터 구성 |
| `openapi.yml` | API 명세 (OpenAPI 3.0) |

---

## Spring @RestController vs Go HTTP Handler

Spring에서는 이렇게 씁니다:

```java
@RestController
@RequestMapping("/api/v1/logs")
public class LogController {
    private final LogService logService;

    @PostMapping
    public ResponseEntity<LogResponse> create(
        @RequestHeader("X-User-Id") String userId,
        @RequestBody CreateLogRequest req
    ) {
        ...
    }
}
```

Go chi에서는 이렇게 씁니다:

```go
type LogHandler struct {
    svc service.LogService
}

func (h *LogHandler) CreateLog(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(r)  // context에서 꺼냄 (미들웨어가 주입)
    var req createLogRequest
    json.NewDecoder(r.Body).Decode(&req)
    ...
    writeJSON(w, http.StatusCreated, resp)
}
```

핵심 차이:
- Spring은 어노테이션으로 메타데이터를 선언하면 프레임워크가 처리
- Go는 `http.ResponseWriter`와 `*http.Request`를 직접 다루며 명시적으로 처리

---

## 미들웨어 패턴

Spring의 `Filter` / `HandlerInterceptor`에 해당하는 것이 Go에서는 **미들웨어**입니다.

```go
// 미들웨어 시그니처: http.Handler를 받아 http.Handler를 반환하는 함수
func UserIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := r.Header.Get("X-User-Id")
        if userID == "" {
            writeError(w, http.StatusUnauthorized, "...")
            return  // next를 호출하지 않으면 체인 중단
        }
        ctx := context.WithValue(r.Context(), userIDKey, userID)
        next.ServeHTTP(w, r.WithContext(ctx))  // 다음 핸들러로 전달
    })
}
```

이는 **데코레이터 패턴**입니다. Spring의 `doFilter(chain.doFilter(...))` 구조와 동일합니다.

### context를 통한 데이터 전달

Spring에서 `Filter`가 처리한 데이터를 `Controller`로 전달할 때 `HttpServletRequest.setAttribute()`를 쓰듯, Go에서는 `context.WithValue()`를 씁니다.

```go
// 미들웨어에서 주입
ctx := context.WithValue(r.Context(), userIDKey, userID)
next.ServeHTTP(w, r.WithContext(ctx))

// 핸들러에서 꺼냄
userID := r.Context().Value(userIDKey).(string)
```

타입 키(`contextKey` 타입)를 사용하는 이유는 **키 충돌 방지**입니다. 문자열 `"userID"` 대신 전용 타입을 쓰면 다른 패키지의 동일한 문자열 키와 구분됩니다.

---

## CORS 미들웨어와 미들웨어 순서

CORS preflight 요청(`OPTIONS`)은 `X-User-Id` 헤더 없이 브라우저가 자동으로 보냅니다. 따라서:

```
CORS 미들웨어 (전역)
    ↓ OPTIONS이면 204 반환, 여기서 종료
UserID 미들웨어 (로그 라우트 그룹)
    ↓
실제 핸들러
```

이 순서가 역전되면 모든 OPTIONS preflight가 `401 Unauthorized`로 거부됩니다.

chi에서의 구현:
```go
r.Use(handler.CORSMiddleware)    // 전역 — OPTIONS를 여기서 처리

r.Route("/api/v1", func(r chi.Router) {
    r.Route("/logs", func(r chi.Router) {
        r.Use(handler.UserIDMiddleware)  // 그룹 — OPTIONS는 여기 도달 안 함
        r.Post("/", ...)
    })
})
```

chi는 라우트가 매칭되어야 미들웨어가 실행됩니다. OPTIONS 요청이 `/api/v1/logs`에 도달하려면 OPTIONS 라우트가 등록되어 있어야 합니다:
```go
r.Options("/*", func(w http.ResponseWriter, r *http.Request) {})
```

---

## JSON 타입 분리 — 왜 도메인 타입에 JSON 태그를 안 붙이나?

도메인 타입(`domain.CoffeeLog`)에 직접 `json:"..."` 태그를 붙이지 않고, 핸들러에 별도의 JSON 타입(`coffeeLogResponse`, `createLogRequest` 등)을 두는 이유:

1. **관심사 분리**: 도메인 모델은 HTTP와 무관합니다. 내일 gRPC나 WebSocket으로 바꿔도 도메인 타입은 그대로입니다.
2. **API 응답 유연성**: 도메인 필드를 그대로 노출하지 않아도 됩니다. 예컨대 `user_id`를 응답에서 제외하거나, 계산된 필드를 추가할 수 있습니다.
3. **버전 관리**: API v2에서 응답 구조가 바뀌어도 도메인이 영향받지 않습니다.

Spring의 DTO 패턴(`LogResponseDto`, `CreateLogRequestDto`)과 동일한 이유입니다.

---

## 오류 매핑 — 서비스 오류 → HTTP 상태 코드

```go
func writeServiceError(w http.ResponseWriter, err error) {
    var ve *service.ValidationError
    switch {
    case errors.Is(err, service.ErrNotFound):
        writeError(w, http.StatusNotFound, err.Error())
    case errors.As(err, &ve):
        writeError(w, http.StatusBadRequest, err.Error())
    default:
        writeError(w, http.StatusInternalServerError, "내부 오류가 발생했습니다")
    }
}
```

- `errors.Is()` → Spring의 `instanceof` / `@ExceptionHandler` 중 타입 체크
- `errors.As()` → 오류 체인에서 특정 타입을 찾음 (`Unwrap()` 체인을 거슬러 올라감)
- default 케이스에서 내부 오류 메시지를 숨기는 것은 보안상 중요합니다.

이 패턴은 Spring의 `@ControllerAdvice` + `@ExceptionHandler`가 하는 역할과 동일합니다.

---

## 의존성 연결 (Wiring)

Spring은 `@Autowired` / `@Bean`으로 의존성을 자동 주입하지만, Go는 `main.go`에서 명시적으로 조립합니다:

```go
// Spring: @Service, @Repository 어노테이션으로 자동 등록
// Go: main.go에서 직접 생성하고 주입
logRepo := repository.NewSQLiteLogRepository(db)
logSvc := service.NewLogService(logRepo)
logHandler := handler.NewLogHandler(logSvc)
```

이것이 Go의 **명시적 의존성 관리** 철학입니다. 코드를 읽으면 전체 의존성 그래프가 한눈에 보입니다.

---

## 커서 페이지네이션 API 설계

`GET /api/v1/logs`는 오프셋 페이지네이션 대신 커서 방식을 사용합니다:

```
GET /api/v1/logs?limit=20
→ { items: [...], next_cursor: "eyJzb3J0...", has_next: true }

GET /api/v1/logs?cursor=eyJzb3J0...&limit=20
→ { items: [...], next_cursor: null, has_next: false }
```

오프셋(`?page=2&size=20`)보다 커서가 나은 이유:
- 목록이 실시간으로 변경되어도 중복·누락 없이 페이지를 이어갈 수 있습니다.
- `next_cursor`는 서버에서 생성한 불투명(opaque) base64 문자열이라 클라이언트가 내부를 파싱할 필요가 없습니다.

---

## 다음 단계

Phase 1-6이 완료되면 백엔드 CRUD API가 완전히 작동합니다. 다음은 **Phase 1-7/1-8 (Frontend)**로, React에서 이 API를 호출하여 실제로 커피 기록을 남길 수 있는 UI를 만듭니다.
