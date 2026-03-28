# Phase 1-1 Backend 학습 문서

> Java/Spring 경험을 기반으로, Go 백엔드 초기 설정이 왜 이렇게 구성됐는지 설명합니다.

---

## 1. Go 모듈 시스템 (`go.mod`)

**Spring에서의 대응**: `pom.xml` / `build.gradle`

```
go mod init coffee-of-the-day/backend
```

Go는 모듈 이름이 곧 프로젝트 내 import 경로의 prefix가 됩니다.

```go
// 이 모듈 안에서 다른 패키지를 import할 때
import "coffee-of-the-day/backend/config"
import "coffee-of-the-day/backend/internal/handler"
```

Maven의 `groupId:artifactId`처럼 프로젝트를 식별하는 역할입니다.
외부에 배포할 게 아니라면 GitHub URL 형식일 필요가 없고, 단순한 이름이면 충분합니다.

---

## 2. 의존성 관리

**Spring에서의 대응**: Maven Central / Gradle dependencies

```bash
go get github.com/go-chi/chi/v5
go get github.com/golang-migrate/migrate/v4
go get github.com/mattn/go-sqlite3
```

- `go get`이 `mvn install` 또는 `gradle dependencies`와 같은 역할입니다.
- 의존성은 `go.mod`에 기록되고, 정확한 버전 해시는 `go.sum`에 저장됩니다.
- `go mod tidy`는 실제로 사용하지 않는 의존성을 정리합니다 (`mvn dependency:analyze`와 유사).

---

## 3. 디렉토리 구조와 패키지

**Spring에서의 대응**: `com.example.app.{controller,service,repository,domain}`

```
backend/
├── cmd/server/        # 애플리케이션 진입점 (main 함수)
├── internal/
│   ├── handler/       # Spring @Controller / @RestController
│   ├── service/       # Spring @Service
│   ├── repository/    # Spring @Repository
│   └── domain/        # Spring @Entity, DTO
├── db/
│   ├── migrations/    # Flyway/Liquibase 마이그레이션 파일과 동일한 역할
│   └── queries/       # sqlc가 읽는 SQL 쿼리 파일
└── config/            # Spring @Configuration / application.yml 로딩
```

`internal/`은 Go의 특수한 디렉토리입니다. 이 안의 패키지는 **같은 모듈 내부에서만** import할 수 있습니다. 외부 라이브러리나 다른 프로젝트가 직접 import하는 것을 컴파일러 수준에서 막아줍니다. Spring에서 패키지를 `package-private`으로 만드는 것과 비슷한 개념입니다.

---

## 4. `config/config.go` — 환경변수 로딩

**Spring에서의 대응**: `application.yml` + `@Value` / `@ConfigurationProperties`

```go
type Config struct {
    Port   string
    DBPath string
}

func Load() Config {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"  // 기본값
    }
    // ...
}
```

Spring Boot는 `application.yml`에서 설정을 읽고 자동으로 바인딩해주지만, Go는 프레임워크가 없어서 `os.Getenv()`로 직접 읽습니다. 대신 코드가 무엇을 읽는지 명확하게 보입니다.

---

## 5. `cmd/server/main.go` — 서버 부트스트랩

**Spring에서의 대응**: `@SpringBootApplication` + `main()` + `@Bean` 설정

Spring Boot는 자동 설정(Auto Configuration)이 많아서 `main()` 함수가 단순하지만, Go는 직접 조립합니다.

```go
func main() {
    cfg := config.Load()           // 설정 로딩
    db, _ := sql.Open(...)         // DB 연결 (DataSource)
    runMigrations(db)              // Flyway 실행과 동일
    r := chi.NewRouter()           // DispatcherServlet 설정
    r.Use(middleware.Logger)       // Filter / Interceptor
    http.ListenAndServe(addr, r)   // 서버 시작 (EmbeddedTomcat)
}
```

**왜 이렇게 직접 조립하나?**
Spring Boot의 Auto Configuration은 편리하지만 "마법"처럼 동작해서 내부를 이해하기 어렵습니다. Go는 의존성 주입 컨테이너나 자동 설정이 없어서 모든 연결이 코드로 명시적으로 드러납니다. 이것이 Go 생태계의 철학입니다.

---

## 6. chi 라우터

**Spring에서의 대응**: Spring MVC `@RequestMapping`

```go
r := chi.NewRouter()
r.Use(middleware.Logger)     // @Component Filter
r.Get("/health", handler)    // @GetMapping("/health")
r.Route("/api/v1", func(r chi.Router) {
    r.Get("/logs", ...)       // @GetMapping("/api/v1/logs")
    r.Post("/logs", ...)      // @PostMapping("/api/v1/logs")
})
```

chi를 선택한 이유:
- Go 표준 라이브러리 `net/http`와 100% 호환 — Spring에서 Servlet 표준을 벗어나지 않는 것과 유사
- 경량: 불필요한 기능 없이 라우팅과 미들웨어만 담당
- Gin, Echo 같은 풀스택 프레임워크보다 표준에 가까워 나중에 교체하기 쉬움

---

## 7. golang-migrate를 Go 라이브러리로 사용한 이유

**Spring에서의 대응**: Flyway가 `spring.flyway.enabled=true`로 자동 실행되는 것

```go
func runMigrations(db *sql.DB) error {
    m, _ := migrate.NewWithDatabaseInstance("file://db/migrations", "sqlite3", driver)
    m.Up()  // 서버 시작 시 자동 실행
}
```

CLI 도구로 별도 실행하는 방법도 있지만, 서버 시작 시 자동으로 마이그레이션을 적용하면 배포가 단순해집니다. Flyway의 `spring.flyway.enabled=true`와 동일한 전략입니다.

---

## 8. sqlc란?

**Spring에서의 대응**: MyBatis (SQL 직접 작성 + 타입 매핑 자동 생성)

sqlc는 SQL 쿼리 파일을 읽어서 Go 타입과 함수를 자동 생성하는 도구입니다.

```sql
-- db/queries/coffee_logs.sql
-- name: GetLogByID :one
SELECT * FROM coffee_logs WHERE id = ? AND user_id = ?;
```

이 SQL로부터 sqlc가 자동으로 생성:
```go
func (q *Queries) GetLogByID(ctx context.Context, id, userID string) (CoffeeLog, error) { ... }
```

JPA처럼 ORM이 SQL을 생성하는 방식이 아니라, **내가 SQL을 작성하면 타입 안전한 Go 코드가 생성**됩니다. 복잡한 쿼리가 많은 서비스에서 SQL을 완전히 제어하면서도 타입 안전성을 얻을 수 있습니다.
