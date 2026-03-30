# 트랜잭션 경계 리뷰

## 범위

이 문서는 현재 backend가 **repository 레벨에서 트랜잭션을 여는 구조**를 어떻게 볼지 정리한 리뷰 메모입니다.

검토 대상:

- `backend/internal/repository/log_repository.go`
- `backend/internal/service/log_service.go`
- `guide/architecture/backend.md`

## 현재 판단

현재 구조는 **즉시 리팩터링이 필요한 문제는 아닙니다.**

이유는 다음과 같습니다.

1. 현재 쓰기 유스케이스의 중심은 `CoffeeLog` aggregate 저장 하나다.
2. `coffee_logs`와 `cafe_logs`/`brew_logs`를 함께 저장하는 세부 영속성 로직을 repository가 이미 알고 있다.
3. service는 입력 정규화, 검증, 에러 매핑에 집중하고 있고, 여러 repository를 조합하는 orchestration은 아직 거의 없다.

즉, 지금은 "비즈니스 유스케이스 여러 개를 한 tx로 묶는 상황"보다 "하나의 aggregate를 여러 테이블에 저장하는 상황"에 더 가깝다. 이 경우 repository 안에서 tx를 여는 선택은 충분히 설명 가능하다.

## 왜 Spring과 다르게 느껴지는가

Spring에서는 보통 service 메서드가 유스케이스 경계이고, `@Transactional`이 그 경계에 붙는다. 그래서 repository는 개별 DB 작업만 알고, "이 작업들을 어떤 단위로 묶을지"는 service가 정한다.

현재 이 프로젝트는 반대로:

- service가 여러 repository를 조합하지 않고
- repository 하나가 aggregate 저장 전체를 캡슐화하며
- tx가 필요한 작업 묶음도 repository 안에 닫혀 있다

그래서 Spring 습관대로 보면 repository tx가 어색하지만, 현재 코드 구조만 놓고 보면 모순은 아니다.

## 지금 바로 service 레벨로 올리지 않는 이유

지금 리팩터링하면 얻는 실익보다 구조 복잡도가 먼저 늘 가능성이 높습니다.

- service가 `sql.Tx` 또는 tx 추상화에 관여해야 한다
- repository 인터페이스가 tx-aware 형태로 바뀔 수 있다
- 아직 필요하지 않은 orchestration 구조를 미리 도입할 수 있다

현재 요구사항만 보면 repository tx는 단순하고 읽기 쉽다. "미래에 필요할 수도 있다"는 이유만으로 지금 일반화하는 것은 과설계가 될 수 있다.

## 향후 리팩터링 트리거

아래 중 하나라도 생기면 transaction boundary를 service/application layer로 옮기는 리팩터링을 우선 검토하는 편이 좋습니다.

### 1. cross-repository 유스케이스가 생길 때

예:

- 로그 저장 + 다른 repository의 통계 갱신
- 로그 삭제 + 별도 이벤트 저장
- 로그 수정 + 사용자 선호 집계 테이블 업데이트

이때 각 repository가 자기 tx를 열면 전체 유스케이스를 하나의 원자적 작업으로 보장하기 어려워집니다.

### 2. 같은 tx 안에서 후속 영속성 작업이 추가될 때

예:

- outbox event 발행용 테이블 기록
- audit log 저장
- 검색 인덱스 갱신 예약

이런 작업은 "로그 저장 성공 후 부가 작업"이 아니라, 종종 같은 commit 단위에 있어야 합니다.

### 3. service orchestration이 늘어날 때

service가 repository 메서드를 여러 개 조합해 시나리오를 만들기 시작하면, tx 경계가 repository 내부에 숨어 있는 구조는 점점 다루기 어려워집니다.

## 권장 방향

트리거가 발생하면 바로 service가 `sql.Tx`를 직접 다루기보다, application layer의 transaction runner를 두는 편이 낫습니다.

예시 방향:

```go
type TxManager interface {
    WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

그 위에서 service/application layer가 유스케이스 경계를 정의하고, repository는 tx가 포함된 context 또는 tx-bound query executor를 사용하도록 바꾸는 방식이 더 깔끔합니다.

이 접근의 장점:

- service가 유스케이스 단위 경계를 소유한다
- repository는 여전히 SQL 중심으로 남는다
- `sql.Tx`가 service 전체에 새지 않도록 제어할 수 있다

## 결론

현재 repository 레벨 트랜잭션은 **허용 가능**합니다. 다만 이것을 장기 원칙처럼 굳히는 것은 좋지 않습니다.

정확한 정리는 다음과 같습니다.

- 지금: aggregate 저장 경계가 repository 하나에 닫혀 있으므로 repository tx 허용 가능
- 나중: cross-repository 유스케이스가 생기면 service/application layer tx로 이동 검토

따라서 지금은 아키텍처 문서에 현 상태와 전환 조건을 함께 적는 것이 가장 정확하고, 즉시 리팩터링 이슈로 올리기보다는 **향후 구조 변화 시 재검토할 설계 포인트**로 관리하는 편이 맞습니다.

---

이 문서는 Codex가 작성했습니다.
