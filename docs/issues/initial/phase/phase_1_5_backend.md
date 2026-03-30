# Phase 1-5 Backend 학습 문서

> Service 계층의 책임, 입력 검증, 커서 페이지 응답 조립 방식을 설명합니다.

---

## 1. 왜 Service 계층이 필요한가

**Spring에서의 대응**: `@Service`

Repository가 이미 CRUD를 할 수 있는데도 Service를 두는 이유는 "DB 접근"과 "비즈니스 규칙"을 분리하기 위해서다.

```go
type LogService interface {
    CreateLog(ctx context.Context, userID string, req CreateLogRequest) (domain.CoffeeLogFull, error)
    GetLog(ctx context.Context, userID, logID string) (domain.CoffeeLogFull, error)
    ListLogs(ctx context.Context, userID string, filter ListLogsFilter) (ListLogsResult, error)
    UpdateLog(ctx context.Context, userID, logID string, req UpdateLogRequest) (domain.CoffeeLogFull, error)
    DeleteLog(ctx context.Context, userID, logID string) error
}
```

Service가 담당하는 일:
- 요청값 검증
- 입력 정규화(trim, 빈 문자열 제거)
- 생성 시 ID / timestamp 부여
- 수정 시 기존 로그 소유권 확인
- 목록 조회 시 `next_cursor` 계산

즉, Repository는 "DB에 어떻게 저장할지"만 알고, Service는 "무엇을 저장해도 되는지"를 안다.

```java
@Service
public class LogService {
    public CoffeeLog createLog(String userId, CreateLogRequest req) {
        validate(req);
        return repository.save(...);
    }
}
```

Spring과 개념은 같지만, Go는 인터페이스와 구현을 명시적으로 적는다.

---

## 2. Create/Update 요청 모델을 따로 둔 이유

도메인 타입 `CoffeeLogFull`은 "저장된 결과"에 가깝다. 반면 API 요청은 보통 서버가 채우는 필드(`id`, `created_at`, `updated_at`)를 포함하지 않는다.

그래서 Service 계층에서 요청 전용 타입을 둔다.

```go
type CreateLogRequest struct {
    RecordedAt string
    Companions []string
    LogType    domain.LogType
    Memo       *string
    Cafe       *domain.CafeDetail
    Brew       *domain.BrewDetail
}
```

이렇게 분리하면 handler는 HTTP JSON을 그대로 request 타입에 바인딩하고, service는 그 값을 검증한 뒤 최종 `domain.CoffeeLogFull`로 변환하면 된다.

Spring으로 치면 `CreateLogRequest DTO` → `Service` → `Entity` 변환과 같다.

---

## 3. 입력 검증은 왜 Service에서 하나

검증 규칙 예시:
- `log_type`은 `cafe` 또는 `brew`만 허용
- `cafe` 로그는 `cafe` 상세가 반드시 필요
- `brew` 로그는 `brew_method`가 반드시 유효해야 함
- `rating`은 0.5 단위, 0.5~5.0 범위
- `recorded_at`은 `RFC3339` 또는 `YYYY-MM-DD`

이 규칙을 handler에 넣으면 HTTP 외의 호출 경로(예: CLI, 배치, 테스트용 호출)에서 재사용할 수 없다. Repository에 넣으면 HTTP 요청 검증과 DB 로직이 섞인다.

그래서 Service에서 검증한다.

```go
switch logType {
case domain.LogTypeCafe:
    if req.Brew != nil {
        return CreateLogRequest{}, newValidationError("brew", "cafe 로그에는 brew 상세를 함께 보낼 수 없습니다")
    }
    detail, err := normalizeCafeDetail(req.Cafe)
case domain.LogTypeBrew:
    ...
}
```

### 정규화(normalization)

검증과 함께 아래 정규화도 수행한다.
- `"  블루보틀  "` → `"블루보틀"`
- `[" Alice ", "", " Bob "]` → `["Alice", "Bob"]`
- 빈 optional string → `nil`

이 처리를 Service에 두면 handler는 문자열 다듬기 같은 부수 로직 없이 단순해진다.

---

## 4. Update에서 왜 먼저 GetLog를 하나

Repository의 `UpdateLog`는 SQL `UPDATE`를 실행하지만, 기존 구현만으로는 "정말 내 소유 로그가 있었는지"를 명확하게 표현하기 어렵다. 특히 상세 테이블은 `log_id` 기준으로만 수정된다.

그래서 Service는 먼저 기존 로그를 읽는다.

```go
existing, err := s.repo.GetLogByID(ctx, normalizedLogID, normalizedUserID)
if err != nil {
    return domain.CoffeeLogFull{}, mapRepositoryError("update log", err)
}
```

이 한 번의 조회로 얻는 이점:
- 소유권 검증
- `created_at` 보존
- 기존 `log_type` 보존

이번 단계에서는 **로그 타입 변경을 금지**했다.

이유는 현재 DB 구조가 `coffee_logs` + `cafe_logs`/`brew_logs` 1:1 구조라서, `cafe → brew` 변경은 단순 UPDATE가 아니라
1. 기존 상세 행 삭제
2. 공통 로그 타입 변경
3. 새 상세 행 생성
의 트랜잭션 작업이 필요하기 때문이다.

즉, 지금의 Update는 "같은 타입 안에서 내용 수정"만 책임진다.

---

## 5. ListLogsResult — 페이지 응답 조립

Repository는 "DB에서 몇 개 가져왔는가"만 알고, API에 필요한 `has_next`, `next_cursor`는 모른다. 이 계산은 Service가 한다.

```go
repoFilter.Limit = limit + 1
items, err := s.repo.ListLogs(ctx, normalizedUserID, repoFilter)

if len(items) > limit {
    result.HasNext = true
    result.Items = items[:limit]

    last := result.Items[len(result.Items)-1]
    nextCursor := repository.EncodeCursor(repository.Cursor{
        SortBy:    "recorded_at",
        Order:     "desc",
        SortValue: last.RecordedAt,
        ID:        last.ID,
    })
}
```

### 왜 `limit + 1`로 조회하나

예를 들어 클라이언트가 20개를 요청했을 때 DB에서 21개를 조회해본다.

- 20개 이하가 왔다 → 마지막 페이지
- 21개가 왔다 → 다음 페이지가 있음

이 방식은 `SELECT COUNT(*)`를 추가로 하지 않아도 되어서 단순하고 빠르다.

Spring Data의 `Slice<T>`가 `Page<T>`보다 가벼운 이유와 비슷하다. 전체 개수보다 "다음 페이지 존재 여부"만 알면 될 때 적합하다.

---

## 6. ValidationError — 왜 sentinel + 구체 정보 조합을 썼는가

handler는 보통 "400인지 아닌지"만 빠르게 판단하면 된다. 하지만 디버깅이나 응답 메시지에는 어떤 필드가 문제였는지도 필요하다.

그래서 두 가지를 함께 쓴다.

```go
var ErrInvalidArgument = errors.New("invalid argument")

type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Unwrap() error {
    return ErrInvalidArgument
}
```

효과:
- `errors.Is(err, ErrInvalidArgument)` 로 400 분기 가능
- `err.Error()`에는 `brew.brew_method: 지원하지 않는 추출 방식입니다` 같은 구체 메시지 유지

Spring으로 치면 `BadRequestException` 하나로 묶으면서, 내부 필드 에러 정보를 함께 담는 방식과 비슷하다.

---

## 7. 테스트 전략

이번 단계 테스트는 **unit test**로 작성했다. Repository를 stub으로 대체해서 Service 규칙만 검증한다.

검증한 핵심 시나리오:
- Create 시 ID / timestamp 생성 및 입력 정규화
- 잘못된 상세 조합 차단
- List 시 `limit + 1` 조회와 `next_cursor` 생성
- Update 시 기존 소유권 / `created_at` 보존
- Repository의 `ErrNotFound`를 Service 에러로 매핑

이 방식은 Spring에서 Mockito로 Repository를 mock하고 Service 단위 테스트를 작성하는 것과 같은 목적이다.

---

*Last updated: 2026-03-29*
