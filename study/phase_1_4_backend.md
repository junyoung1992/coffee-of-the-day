# Phase 1-4 Backend 학습 문서

> Repository 패턴, Go 트랜잭션, 커서 페이지네이션 구현, 인메모리 SQLite 통합 테스트를 설명합니다.

---

## 1. Repository 패턴 — Go vs Spring

**Spring에서의 대응**: `@Repository` + `JpaRepository<T, ID>` 인터페이스

Spring에서는 JPA가 기본 CRUD를 자동으로 구현해준다. Go에서는 직접 인터페이스를 정의하고 구현체를 작성한다.

```go
// Go: 인터페이스를 직접 정의
type LogRepository interface {
    CreateLog(ctx context.Context, log domain.CoffeeLogFull) error
    GetLogByID(ctx context.Context, logID, userID string) (domain.CoffeeLogFull, error)
    ListLogs(ctx context.Context, userID string, filter ListFilter) ([]domain.CoffeeLogFull, error)
    UpdateLog(ctx context.Context, log domain.CoffeeLogFull) error
    DeleteLog(ctx context.Context, logID, userID string) error
}
```

```java
// Spring: JpaRepository가 기본 CRUD 자동 제공
@Repository
public interface LogRepository extends JpaRepository<CoffeeLog, String> {
    List<CoffeeLog> findByUserId(String userId);
}
```

**Go 방식의 장점**: 인터페이스가 정확히 필요한 메서드만 선언한다. JPA의 `findById`, `save`, `delete` 등 불필요한 메서드를 노출하지 않는다. 이 인터페이스는 테스트에서 mock으로 교체하기도 쉽다.

---

## 2. Go 트랜잭션 처리

**Spring에서의 대응**: `@Transactional` 애노테이션

Spring은 AOP로 트랜잭션을 투명하게 처리한다. Go는 명시적으로 트랜잭션을 관리한다.

```go
func (r *SQLiteLogRepository) CreateLog(ctx context.Context, log domain.CoffeeLogFull) error {
    // 1. 트랜잭션 시작
    tx, err := r.sqlDB.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("create log: begin tx: %w", err)
    }
    // 2. 함수 종료 시 롤백 예약 — Commit 후에는 no-op
    defer tx.Rollback()

    qtx := r.queries.WithTx(tx)

    // 3. coffee_logs 삽입
    err = qtx.InsertLog(ctx, ...)
    if err != nil {
        return fmt.Errorf("create log: insert log: %w", err)  // defer가 롤백
    }

    // 4. 서브 테이블 삽입 (cafe_logs 또는 brew_logs)
    err = qtx.InsertCafeLog(ctx, ...)
    if err != nil {
        return fmt.Errorf("create log: insert detail: %w", err)  // defer가 롤백
    }

    // 5. 명시적 커밋
    return tx.Commit()
}
```

### `defer tx.Rollback()` 패턴

이 패턴이 영리한 이유:
- 함수가 어느 지점에서든 `return err`로 나가면 `defer`가 실행되어 롤백
- `tx.Commit()` 성공 후에 `tx.Rollback()`을 호출하면 `sql.ErrTxDone` 오류가 나지만 이미 커밋된 트랜잭션이므로 무해하게 무시됨
- 에러를 놓칠 걱정 없이 단순한 early return 코드를 작성할 수 있음

```java
// Spring의 선언적 방식
@Transactional
public void createLog(CoffeeLogFull log) {
    logRepo.save(log.toCoffeeLog());
    cafeLogRepo.save(log.getCafe()); // 예외 발생 시 프레임워크가 롤백
}
```

Go는 더 verbose하지만, 트랜잭션이 어디서 시작되고 끝나는지 코드를 읽는 것만으로 파악할 수 있다.

---

## 3. 에러 래핑 — `fmt.Errorf("%w")`

**Spring에서의 대응**: 커스텀 예외 클래스 체인 (`throw new ServiceException("...", cause)`)

Go의 에러 래핑은 컨텍스트를 계층적으로 쌓는다:

```go
return fmt.Errorf("create log: insert detail: %w", err)
// 결과 메시지: "create log: insert detail: SQLITE_CONSTRAINT: ..."
```

`%w` 동사는 원본 에러를 감싸서 `errors.Is()` / `errors.As()`로 언래핑이 가능하다:

```go
// ErrNotFound를 어디서 wrap했든 상위 레이어에서 체크 가능
if errors.Is(err, repository.ErrNotFound) {
    // 404 응답
}
```

---

## 4. ErrNotFound — Sentinel 에러

**Spring에서의 대응**: `@ResponseStatus(HttpStatus.NOT_FOUND)` + `EntityNotFoundException`

```go
// repository 패키지에서 정의
var ErrNotFound = errors.New("log not found")

// 사용처
func (r *SQLiteLogRepository) GetLogByID(...) (domain.CoffeeLogFull, error) {
    row, err := r.queries.GetLogByID(ctx, ...)
    if errors.Is(err, sql.ErrNoRows) {
        return domain.CoffeeLogFull{}, ErrNotFound  // DB 에러를 도메인 에러로 변환
    }
    ...
}
```

`sql.ErrNoRows`는 DB 레이어의 에러다. 이것을 그대로 상위 레이어에 노출하면 상위 레이어가 DB 구현에 의존하게 된다. `ErrNotFound`로 변환함으로써 service 레이어는 `sql` 패키지를 import하지 않아도 된다.

---

## 5. ListLogs — 왜 raw SQL을 썼는가

sqlc는 SQL을 미리 작성해두고 Go 코드를 생성한다. 그런데 `ListLogs`는 WHERE 조건이 런타임에 동적으로 결정된다:

```go
query := `SELECT ... FROM coffee_logs WHERE user_id = ?`

if filter.LogType != nil {
    query += ` AND log_type = ?`
    args = append(args, *filter.LogType)
}
if filter.Cursor != nil {
    query += ` AND (recorded_at < ? OR (recorded_at = ? AND id < ?))`
    args = append(args, filter.Cursor.SortValue, filter.Cursor.SortValue, filter.Cursor.ID)
}

query += ` ORDER BY recorded_at DESC, id DESC LIMIT ?`
```

**sqlc.narg의 한계**: `sqlc.narg()`는 파라미터가 NULL이면 조건을 건너뛰는 방식이다. 하지만 커서 페이지네이션의 `(A < ? OR (A = ? AND B < ?))` 복합 조건은 같은 파라미터를 두 번 참조해야 해서 positional parameter(`?`)로 구현하기 어렵다.

이런 경우 raw SQL이 더 명확하고 안전하다. SQL Injection은 `?` placeholder를 사용하므로 방지된다.

---

## 6. Cursor-based Pagination 구현

커서는 `{sort_by, order, sort_value, id}` → JSON → base64 URL 인코딩으로 만들어진 불투명 문자열이다.

```go
type Cursor struct {
    SortBy    string `json:"sort_by"`
    Order     string `json:"order"`
    SortValue string `json:"sort_value"`  // recorded_at 값
    ID        string `json:"id"`
}

func EncodeCursor(c Cursor) string {
    raw, _ := json.Marshal(c)
    return base64.URLEncoding.EncodeToString(raw)
}
```

클라이언트는 커서의 내부 구조를 알 수 없다 — 단순히 다음 요청에 그대로 돌려보내기만 한다.

**복합 정렬 키 커서**:

```sql
AND (recorded_at < ?                            -- 이전 페이지의 마지막 recorded_at보다 이전
  OR (recorded_at = ? AND id < ?))              -- 같은 시각이면 id로 구분 (tie-breaking)
ORDER BY recorded_at DESC, id DESC
```

`recorded_at`만으로는 동일한 시각에 기록된 항목들을 구분할 수 없다. `id`를 보조 정렬 키로 사용해서 유일성을 보장한다.

---

## 7. 통합 테스트 — 인메모리 SQLite

**Spring에서의 대응**: `@DataJpaTest` + H2 인메모리 DB

Go는 `":memory:"` DSN으로 인메모리 SQLite를 사용한다:

```go
func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)

    // 마이그레이션 직접 실행
    _, err = db.Exec(`CREATE TABLE coffee_logs (...)`)
    require.NoError(t, err)

    t.Cleanup(func() { db.Close() })
    return db
}
```

각 테스트 함수에서 독립된 DB를 새로 만든다. 테스트 간 상태가 공유되지 않는다.

### `testify` 라이브러리

```go
assert.NoError(t, err)      // 실패해도 계속 실행
require.NoError(t, err)     // 실패하면 테스트 즉시 중단

assert.Equal(t, expected, actual)
assert.ErrorIs(t, err, repository.ErrNotFound)
```

**Spring에서의 대응**: JUnit의 `assertEquals` / AssertJ의 `assertThat().isEqualTo()`

`require`는 이후 코드가 해당 값에 의존할 때 사용한다. DB 연결 실패 후 쿼리를 시도하는 것은 무의미하므로 `require.NoError`를 쓴다.

---

## 8. 단위 테스트 vs 통합 테스트 — 이 Repository는 왜 통합 테스트인가

```
Repository 테스트 = 통합 테스트
  → 실제 SQLite + 실제 SQL 쿼리 실행
  → 테이블 생성, INSERT, SELECT, CASCADE 삭제 검증 포함

Service 테스트 = 단위 테스트 (예정)
  → LogRepository 인터페이스를 mock으로 교체
  → DB 없이 비즈니스 로직만 검증
```

Repository 레이어는 SQL 쿼리 자체가 핵심 로직이다. Mock으로 교체하면 쿼리의 정확성을 검증할 수 없다. 인메모리 SQLite를 쓰면 실제 DB와 동일한 동작을 보장하면서도 빠르게 실행된다.

**Spring에서의 대응**: `@DataJpaTest`가 인메모리 H2를 띄우고 JPA Repository만 테스트하는 것과 같은 원리다.
