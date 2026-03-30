# Phase 1-2 DB 스키마 및 마이그레이션 학습 문서

> Java/Spring 경험을 기반으로, DB 스키마 설계와 마이그레이션 구성을 설명합니다.

---

## 1. golang-migrate — 마이그레이션 도구

**Spring에서의 대응**: Flyway / Liquibase

Flyway처럼 SQL 파일을 버전 순서대로 실행합니다. 한 번 실행된 파일은 다시 실행하지 않습니다.

```
db/migrations/
├── 001_create_users.up.sql       # 적용
├── 001_create_users.down.sql     # 롤백
├── 002_create_coffee_logs.up.sql
├── 002_create_coffee_logs.down.sql
...
```

Flyway와의 차이점:
- 파일명 규칙: Flyway는 `V1__create_users.sql`, migrate는 `001_create_users.up.sql`
- `up.sql` / `down.sql` 쌍으로 롤백을 명시적으로 지원

---

## 2. SQLite — 왜 파일 기반 DB인가

**Spring에서의 대응**: H2 인메모리 DB (개발/테스트용)

POC 단계에서 PostgreSQL, MySQL 같은 서버형 DB 대신 SQLite를 선택한 이유:
- 별도 DB 서버 설치·실행 없이 파일 하나로 동작
- 로컬 개발 환경 설정이 단순
- Phase 4 이후 실제 서비스로 전환할 때 마이그레이션만 교체하면 됨

Spring 개발 시 H2를 로컬에서 쓰고 운영에서 PostgreSQL로 바꾸는 패턴과 동일합니다.

---

## 3. 테이블 구조 설계 — 왜 3개 테이블인가

카페 기록과 브루 기록을 하나의 테이블에 모두 넣는 방법도 있었습니다. 하지만 두 기록의 필드가 너무 달라서 단일 테이블로 만들면 한쪽은 항상 NULL 컬럼이 많아집니다.

선택한 전략: **공통 + 서브타입 분리 (1:1 관계)**

```
coffee_logs (공통 필드)
    ↓ 1:1
cafe_logs   (카페 전용 필드)
brew_logs   (브루 전용 필드)
```

**Spring/JPA에서의 대응**: `@Inheritance(strategy = JOINED)`

JPA의 `JOINED` 상속 전략과 동일한 구조입니다.

```java
// JPA에서의 동일한 설계
@Entity
@Inheritance(strategy = InheritanceType.JOINED)
public abstract class CoffeeLog { ... }

@Entity
public class CafeLog extends CoffeeLog { ... }

@Entity
public class BrewLog extends CoffeeLog { ... }
```

---

## 4. SQLite에서 UUID와 날짜를 TEXT로 저장하는 이유

SQLite는 PostgreSQL, MySQL과 달리 `UUID` 타입과 `TIMESTAMP` 타입이 없습니다. TEXT로 저장하고 애플리케이션 레이어에서 처리합니다.

```sql
id          TEXT PRIMARY KEY,   -- UUID 문자열로 저장
recorded_at TEXT NOT NULL,       -- ISO8601 문자열로 저장 ("2024-03-28T14:30:00+09:00")
created_at  TEXT NOT NULL,
updated_at  TEXT NOT NULL
```

Go 코드에서 UUID 생성과 ISO8601 변환을 담당합니다. SQLite의 트레이드오프입니다.

---

## 5. JSON 배열을 TEXT로 저장하는 이유

```sql
companions  TEXT NOT NULL DEFAULT '[]',  -- ["지수", "민준"]
tasting_tags TEXT NOT NULL DEFAULT '[]', -- ["초콜릿", "체리"]
brew_steps   TEXT NOT NULL DEFAULT '[]'  -- ["포터필터 예열", ...]
```

PostgreSQL은 `jsonb`, `TEXT[]` 같은 배열 타입을 지원하지만 SQLite는 지원하지 않습니다. JSON 문자열로 저장하고 Go 레이어에서 `[]string` ↔ JSON 직렬화를 처리합니다.

이 처리는 1-3 단계에서 도메인 타입과 함께 구현합니다.

---

## 6. ON DELETE CASCADE

```sql
cafe_logs (
    log_id TEXT PRIMARY KEY REFERENCES coffee_logs(id) ON DELETE CASCADE,
    ...
)
```

`coffee_logs`의 레코드가 삭제될 때 연결된 `cafe_logs`나 `brew_logs` 레코드도 자동으로 삭제됩니다.

**Spring에서의 대응**: JPA `@OneToOne(cascade = CascadeType.REMOVE)` 또는 `orphanRemoval = true`

데이터베이스 레벨에서 처리하므로 Go 코드에서 서브 테이블을 별도로 삭제하는 로직이 필요 없습니다.

---

## 7. CHECK 제약조건

```sql
log_type TEXT NOT NULL CHECK(log_type IN ('cafe', 'brew')),
roast_level TEXT CHECK(roast_level IN ('light', 'medium', 'dark')),
rating REAL CHECK(rating >= 0.5 AND rating <= 5.0)
```

**Spring에서의 대응**: `@Enumerated(EnumType.STRING)` + `@Min` / `@Max` Bean Validation

데이터베이스 레벨에서 잘못된 값이 들어오는 것을 막습니다. Go 서비스 레이어의 유효성 검사와 이중으로 보호합니다.

---

## 8. sqlc.yaml 설정

```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "db/queries"    # SQL 쿼리 파일 위치
    schema: "db/migrations"  # 스키마 파일 위치 (마이그레이션 파일을 스키마로 사용)
    gen:
      go:
        package: "db"
        out: "internal/db"   # 생성된 Go 코드 출력 위치
        emit_json_tags: true  # JSON 직렬화 태그 자동 생성
        emit_pointers_for_null_fields: true  # NULL 가능 필드는 포인터 타입으로 생성
```

`emit_pointers_for_null_fields: true`는 중요한 설정입니다. SQL에서 `NULL` 가능한 컬럼(예: `location TEXT`)을 Go에서 `*string`(포인터)으로 생성합니다. Java의 `Optional<String>`과 유사한 개념입니다. `null`과 빈 문자열을 명확히 구분할 수 있습니다.
