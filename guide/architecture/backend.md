# Backend 아키텍처 결정 문서

> 각 기술 선택의 이유와 트레이드오프를 설명합니다.
> "왜 이걸 썼는가"를 기록해두면 나중에 교체 또는 유지 결정을 내릴 때 기준이 됩니다.

---

## 언어: Go

**결정**: 사용자가 선택.

Go가 이 프로젝트에 잘 맞는 이유:
- 컴파일 바이너리 하나로 배포 — 나중에 서버에 올릴 때 런타임 설치 불필요
- 표준 라이브러리의 `net/http`가 강력해서 외부 의존성을 최소화할 수 있음
- 정적 타입으로 DB 쿼리 결과와 API 응답 타입을 컴파일 타임에 검증 가능

---

## HTTP 라우터: chi

**결정**: `net/http` 표준 라이브러리 + `chi` 라우터

**왜 chi인가**

Go의 HTTP 라우터 생태계에서 주요 선택지는 세 가지입니다.

| 라우터 | 특징 | 이 프로젝트와의 맞음 |
|--------|------|---------------------|
| `net/http` 표준만 사용 | 의존성 0, 기능 제한 | URL 파라미터(`/logs/:id`) 처리가 불편 |
| **chi** | 경량, `net/http` 100% 호환 | ✅ 표준 호환 유지하면서 라우팅 편의성 확보 |
| Gin / Echo | 풀프레임워크, 독자적인 Context | 표준 `http.Handler`와 호환 안 됨, 이 규모에서 과함 |

chi의 핵심 장점은 **`net/http`의 `http.Handler` 인터페이스를 그대로 사용**한다는 점입니다. 미들웨어를 표준 방식으로 작성할 수 있고, 나중에 chi를 제거해도 핸들러 코드를 거의 바꿀 필요가 없습니다.

```go
// chi 미들웨어는 그냥 net/http 미들웨어
func UserIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ...
        next.ServeHTTP(w, r)
    })
}
```

---

## DB: SQLite

**결정**: POC 단계에서 SQLite 사용.

**왜 SQLite인가**

| | SQLite | PostgreSQL |
|--|--------|------------|
| 설치 | 파일 하나 | 서버 설치·실행 필요 |
| 로컬 실행 | `./coffee.db` 파일만 있으면 됨 | Docker 또는 로컬 설치 필요 |
| 동시성 | 쓰기 락 있음 (POC엔 무관) | 높은 동시성 지원 |
| 이식성 | DB 파일을 그대로 복사해 이동 가능 | — |

POC에서 단일 사용자 / 로컬 실행이라는 전제 조건에서 SQLite는 "설정 없이 바로 시작"할 수 있는 최선의 선택입니다. 이후 사용자가 늘어나거나 서버 배포가 필요해지면 PostgreSQL로 교체합니다.

**SQLite의 외래키 비활성화 기본값**

SQLite는 연결을 열어도 외래키 강제가 기본으로 꺼져 있습니다. `PRAGMA foreign_keys = ON`을 명시적으로 실행하지 않으면 스키마에 `REFERENCES`와 `ON DELETE CASCADE`를 선언해도 아무 효과가 없습니다.

이 프로젝트에서는 DSN 수준에서 활성화합니다.

```go
// connection pool의 모든 연결에 일괄 적용된다
db, err := sql.Open("sqlite3", cfg.DBPath+"?_foreign_keys=on")
```

코드로 `PRAGMA foreign_keys = ON`을 직접 실행하면 pool에서 해당 연결 하나에만 적용되어 다른 연결에서는 여전히 꺼진 상태가 될 수 있습니다. DSN 파라미터 방식이 안전합니다.

PostgreSQL로 교체하면 이 문제는 사라집니다. PostgreSQL은 외래키를 항상 강제합니다.

**SQLite에서 배열 저장 방식**

SQLite는 배열 타입이 없어서 `companions`, `tasting_tags`, `brew_steps`를 JSON 텍스트로 저장합니다.

```
companions TEXT NOT NULL DEFAULT '[]'  →  '["지수","민준"]'
```

Go 레이어에서 `encoding/json`으로 `[]string` ↔ `string` 직렬화를 처리합니다. PostgreSQL로 교체하면 이 부분만 `TEXT[]` 또는 `JSONB`로 바꾸면 됩니다.

---

## 쿼리 레이어: sqlc

**결정**: ORM 대신 **sqlc를 기본으로 사용하고, 필요한 곳에 raw SQL을 병행**.

**왜 sqlc인가**

sqlc는 SQL 쿼리 파일을 읽어서 타입이 완전히 맞는 Go 코드를 자동 생성합니다.

```sql
-- db/queries/logs.sql
-- name: GetLogByID :one
SELECT * FROM coffee_logs WHERE id = ? AND user_id = ?;
```

위 SQL에서 아래 Go 코드가 자동 생성됩니다.

```go
func (q *Queries) GetLogByID(ctx context.Context, id, userID string) (CoffeeLog, error)
```

주요 선택지와 비교:

| | GORM (ORM) | database/sql (raw) | **sqlc** |
|--|------------|---------------------|----------|
| 타입 안전성 | 런타임 오류 가능 | 없음 | ✅ 컴파일 타임 보장 |
| SQL 제어 | ORM이 SQL 생성 (최적화 어려움) | 직접 작성 | ✅ 직접 작성한 SQL 그대로 실행 |
| 코드량 | 적음 | 많음 | ✅ 자동 생성으로 적음 |
| N+1 문제 | 발생하기 쉬움 | 직접 제어 | ✅ 직접 제어 |

이 프로젝트의 기본 CRUD 쿼리 패턴(JOIN이 있는 1:1 서브 테이블 조회 등)은 ORM보다 직접 SQL이 더 명확합니다.

다만 Phase 3 자동완성에서 SQLite `json_each()` 가상 테이블을 사용하는 집계 쿼리가 추가되면서, **모든 쿼리를 sqlc로만 처리하지는 않게 되었습니다.**

```go
// suggestion_repository.go
rows, err := r.db.QueryContext(ctx, tagSuggestionsQuery, userID, userID, q, q)
```

현재 전략은 다음과 같습니다.

- 기본 CRUD, 단순 조회: sqlc 사용
- sqlc가 정적으로 분석하기 어려운 쿼리(`json_each`, 특수 집계): `database/sql`로 직접 실행

즉, 이 프로젝트의 쿼리 레이어 결정은 "sqlc만 고집"이 아니라 **sqlc 우선 + raw SQL 보완**입니다.

---

## 마이그레이션: golang-migrate

**결정**: golang-migrate 사용.

SQL 파일을 버전 번호로 관리합니다.

```
db/migrations/
  001_create_users.up.sql
  001_create_users.down.sql
  002_create_coffee_logs.up.sql
  ...
```

- `.up.sql`: 스키마 변경 적용
- `.down.sql`: 롤백

대안으로 `goose`도 있지만, golang-migrate는 SQL 파일을 그대로 사용하고 Go 코드에 의존하지 않아서 DB 도구로 직접 SQL을 실행하는 것과 동일한 결과를 보장합니다.

---

## 아키텍처 패턴: Layered Architecture

**결정**: handler → service → repository 3계층 구조.

```
HTTP 요청
    ↓
handler      : 요청 파싱, 응답 직렬화, HTTP 상태 코드 결정
    ↓
service      : 비즈니스 로직, 입력 정규화, 유효성 검사
    ↓
repository   : DB 쿼리 실행 (sqlc + 필요 시 raw SQL)
    ↓
SQLite
```

**각 레이어의 책임**

- `handler`: HTTP를 알고, 비즈니스 로직을 모른다
- `service`: 비즈니스 규칙과 입력 정규화를 알고, HTTP를 모른다
- `repository`: SQL과 DB 트랜잭션을 알고, 비즈니스 규칙을 모른다

이 구조의 실용적 이유:
1. `service`를 테스트할 때 HTTP 요청 없이 함수 직접 호출 가능
2. SQLite → PostgreSQL 교체 시 `repository`만 수정
3. 인증 방식(POC 헤더 → JWT) 변경 시 `handler` 미들웨어만 수정

**현재 구현에서 트랜잭션은 repository가 연다**

일반론으로는 Spring에서 하던 것처럼 service/application layer가 유스케이스 단위 트랜잭션을 제어하는 편이 더 보편적입니다. 여러 repository 호출을 하나의 비즈니스 작업으로 묶어야 할 때는 그 방식이 더 잘 맞습니다.

하지만 현재 구현은 `coffee_logs`와 서브 테이블 삽입/수정을 **하나의 repository가 aggregate 단위로 캡슐화**하고 있으므로, 트랜잭션 경계도 repository에 있습니다.

```go
tx, err := r.sqlDB.BeginTx(ctx, nil)
defer tx.Rollback()
qtx := r.queries.WithTx(tx)
```

현재 이 선택이 허용되는 이유:

- 현재 유스케이스는 "로그 저장"이 곧 하나의 영속성 작업 묶음이다
- service는 입력 정규화와 비즈니스 규칙 검증에 집중한다
- repository가 `sqlc.WithTx(...)`와 raw SQL을 함께 다루는 쪽이 구현 복잡도를 낮춘다

즉, 이 프로젝트의 layered architecture는 교과서적으로 "service가 tx를 잡는다"가 아니라, **현재 aggregate 저장 경계에 맞춰 repository가 tx를 소유하는 구조**입니다.

다만 이것이 장기 고정 원칙은 아닙니다. 아래와 같은 상황이 생기면 service/application layer로 트랜잭션 경계를 올리는 쪽이 더 적절합니다.

- 하나의 유스케이스가 여러 repository 호출을 하나의 원자적 작업으로 묶어야 할 때
- 로그 저장 외에 audit, outbox, 통계 갱신 같은 후속 영속성 작업이 같은 tx에 묶여야 할 때
- repository 메서드 재사용 조합이 늘어나면서 "repository마다 자기 tx를 연다"는 구조가 service orchestration을 방해할 때

그 시점에는 service가 `sql.Tx`를 직접 다루기보다, application layer의 transaction runner 또는 `WithinTransaction(...)` 형태 추상화를 두는 편이 더 낫습니다.

**Phase 3 이후 레이어 확장 방식**

자동완성 기능이 추가되면서 새 레이어 조합도 생겼습니다.

- `SuggestionHandler`
- `SuggestionService`
- `SuggestionRepository`

기존 CRUD 흐름을 건드리지 않고 새 vertical slice를 옆으로 확장한 것입니다. Layered architecture를 선택한 이유가 Phase 3에서 실제로 검증된 셈입니다.

---

## 인증: JWT + httpOnly 쿠키

**결정**: Phase 4에서 `X-User-Id` 헤더(POC)를 JWT + httpOnly 쿠키 방식으로 교체.

Phase 1~3은 인증 구현 비용을 절약하면서 멀티유저 DB 구조를 검증하기 위해 헤더 방식을 사용했습니다. `service` / `repository` 레이어가 이미 `userID string` 파라미터를 받는 구조였기 때문에, Phase 4에서 `handler` 미들웨어만 바꿔서 교체가 완료됐습니다.

```go
// POC (Phase 1~3): 헤더에서 읽음
userID := r.Header.Get("X-User-Id")

// Phase 4~: JWT 클레임에서 읽음 (같은 인터페이스)
userID := claims.Subject  // context에서 추출
```

**왜 httpOnly 쿠키인가**

| 방식 | XSS 취약 | CSRF 취약 |
|---|---|---|
| `localStorage` | O (JS로 직접 탈취) | X |
| `httpOnly` 쿠키 | X (JS 접근 불가) | O (SameSite로 완화) |

`httpOnly + SameSite=Strict` 조합으로 두 공격 벡터를 모두 완화합니다.

**Access Token + Refresh Token 분리**

- **Access token** (15분): 짧은 수명 → 탈취돼도 피해 최소화
- **Refresh token** (7일): 긴 수명, httpOnly 쿠키 전용 → 자동 갱신 UX 유지

두 토큰 모두 동일한 비밀키로 서명되므로 `token_type` 클레임으로 혼용을 방지합니다. 리프레시 토큰을 액세스 토큰 자리에 사용하는 **토큰 혼용 공격(token confusion attack)**을 차단합니다.

```go
// JWTMiddleware에서 token_type 검증
if claims.TokenType != "access" {
    return "", fmt.Errorf("expected access token, got %q", claims.TokenType)
}
```

**레이어 구조**

```
handler/auth_handler.go        ← HTTP 요청 파싱, 쿠키 설정/만료
handler/middleware.go          ← JWTMiddleware: 쿠키 검증 → context에 userID 주입
service/auth_service.go        ← 비밀번호 검증(bcrypt), JWT 생성/파싱
repository/user_repository.go  ← users 테이블 CRUD (sqlc)
db/migrations/005_*.sql        ← email, password_hash 컬럼 추가
```

**CORS와 쿠키**

쿠키를 cross-origin 요청에서 전송하려면 `Access-Control-Allow-Credentials: true`가 필요하며, 이때 `*` 와일드카드 origin은 사용 불가합니다. 기존에 이미 specific origin(`localhost:5173`)을 허용하던 CORS 설정이 그대로 호환됩니다.

---

*Last updated: 2026-03-30 (Phase 4 JWT 인증 반영)*
