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

**결정**: ORM 대신 sqlc 사용.

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

이 프로젝트의 쿼리 패턴(JOIN이 있는 1:1 서브 테이블 조회 등)은 ORM보다 직접 SQL이 더 명확합니다.

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
service      : 비즈니스 로직, 트랜잭션 제어, 유효성 검사
    ↓
repository   : DB 쿼리 실행 (sqlc 생성 코드 래핑)
    ↓
SQLite
```

**각 레이어의 책임**

- `handler`: HTTP를 알고, 비즈니스 로직을 모른다
- `service`: 비즈니스 규칙을 알고, HTTP와 DB를 모른다
- `repository`: SQL을 알고, 비즈니스 규칙을 모른다

이 구조의 실용적 이유:
1. `service`를 테스트할 때 HTTP 요청 없이 함수 직접 호출 가능
2. SQLite → PostgreSQL 교체 시 `repository`만 수정
3. 인증 방식(POC 헤더 → JWT) 변경 시 `handler` 미들웨어만 수정

---

## POC 사용자 식별: X-User-Id 헤더

**결정**: 인증 없이 `X-User-Id` 헤더로 user_id를 전달.

Phase 1에서 인증 구현 비용을 절약하면서 멀티유저 DB 구조를 검증하기 위한 임시 방편입니다. Phase 4에서 JWT 미들웨어로 교체할 때 `handler` 레이어의 미들웨어만 바꾸면 되고, `service` / `repository` 레이어는 이미 `userID string`을 파라미터로 받고 있어서 변경이 없습니다.

```go
// POC: 헤더에서 읽음
userID := r.Header.Get("X-User-Id")

// Phase 4: JWT 클레임에서 읽음 (같은 인터페이스)
userID := claims.UserID
```

---

*Last updated: 2026-03-29 (SQLite foreign key 기본값 비활성화 및 DSN 활성화 방식 추가)*
