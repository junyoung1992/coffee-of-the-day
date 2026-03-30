# Phase 2 리팩토링

## 범위

`code_review_phase_2.md`에서 제기된 세 가지 버그와 문서 drift를 수정했습니다.

- **높음**: 날짜 필터가 UTC 기준으로 동작해 KST 오전 기록이 누락되는 문제
- **중간**: `recorded_at`을 원문 문자열 그대로 저장해 커서 페이지네이션 정렬이 불안정한 문제
- **중간**: 필터 결과 0건일 때도 첫 방문 빈 상태를 보여주는 UX 문제
- **낮음**: `openapi.yml`의 `recorded_at` 설명이 실제 서버 계약과 다른 문서 drift

## 검증

- `cd backend && go test ./...` — 통과
- `cd frontend && npm exec vitest run` — 통과

---

## 수정 1. 날짜 필터 타임존 — KST 고정, 글로벌 확장 포인트 확보

### 문제

`YYYY-MM-DD` 필터를 받으면 백엔드가 `00:00:00Z ~ 23:59:59Z`, 즉 UTC 하루 경계로 확장했습니다. 그런데 `recorded_at`은 사용자의 로컬 시간 기반이므로, 한국 사용자가 오전에 기록한 항목은 UTC 기준으로 전날에 해당합니다.

예를 들어 `2026-03-29 08:30 KST`는 `2026-03-28T23:30:00Z`로 저장됩니다. 이 상태에서 필터 `date_from=2026-03-29&date_to=2026-03-29`를 보내면 서버는 `2026-03-29T00:00:00Z ~ 2026-03-29T23:59:59Z` 범위를 조회하고, 해당 기록은 범위 밖이라 누락됩니다.

### 수정 내용

`validateDateFilter`에 `*time.Location` 인자를 추가하고, `YYYY-MM-DD` 입력을 해당 타임존 기준 하루 경계로 계산한 뒤 UTC로 변환하도록 변경했습니다.

```go
// backend/internal/service/log_service.go

const defaultTimezone = "Asia/Seoul"

type ListLogsFilter struct {
    // ...
    // 빈 문자열이면 defaultTimezone(Asia/Seoul)을 사용한다.
    Timezone string
}
```

`normalizeListFilter` 안에서 `filter.Timezone`을 읽어 `time.LoadLocation`으로 변환한 뒤 `validateDateFilter`에 전달합니다. `Timezone`이 비어 있으면 `defaultTimezone`이 fallback으로 쓰입니다.

```go
// 2026-03-29T00:00:00+09:00 = 2026-03-28T15:00:00Z
normalized = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
utc := normalized.UTC()
return utc.Format(time.RFC3339Nano), utc, nil
```

### 글로벌 확장 포인트

현재는 모든 호출자가 `Timezone`을 비워두므로 자동으로 `Asia/Seoul`이 적용됩니다. 향후 다국어 지원이 필요하면 핸들러나 서비스 레이어에서 사용자 설정 또는 JWT 클레임으로부터 읽은 타임존 문자열을 `ListLogsFilter.Timezone`에 채우면 됩니다. 서비스 내부 로직은 바꿀 필요가 없습니다.

### date_to 경계값

기존에는 `23:59:59`까지만 잡았습니다. 밀리초 단위까지 기록될 수 있으므로 `23:59:59.999`로 변경했습니다.

```go
// 변경 전
normalized = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, loc)

// 변경 후
normalized = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 999_000_000, loc)
```

---

## 수정 2. `recorded_at` UTC 정규화 — 커서 페이지네이션 정렬 안정화

### 문제

`validateRecordedAt`은 RFC3339를 검증만 하고 원문 문자열을 그대로 저장했습니다. SQLite는 타임존 개념 없이 문자열 대소 비교로 정렬하므로, 오프셋 포함 값(`+09:00`)과 UTC 값(`Z`)이 섞이면 사전순 정렬이 시각 순서와 달라집니다.

```
SQLite 문자열 비교:
'2026-03-29T10:00:00+09:00' < '2026-03-29T02:00:00Z' → false

실제 시각 비교:
 01:00Z                     <  02:00Z                → true (반대 결과)
```

커서 페이지네이션은 `recorded_at DESC, id DESC` 정렬과 `recorded_at < ?` 비교를 함께 사용하므로, 정렬이 깨지면 다음 페이지에서 기록이 누락되거나 중복될 수 있습니다.

### 수정 내용

`validateRecordedAt`에서 저장 전에 `parsed.UTC().Format(time.RFC3339Nano)`로 정규화합니다.

```go
// 변경 전
return trimmed, nil

// 변경 후
return parsed.UTC().Format(time.RFC3339Nano), nil
```

같은 이유로 `validateDateFilter`의 RFC3339 입력 경로도 UTC로 정규화합니다. 저장된 `recorded_at`과 필터 경계값이 모두 UTC RFC3339Nano 포맷이므로 SQLite 문자열 비교가 시각 순서와 일치합니다.

### `time.RFC3339Nano`를 쓰는 이유

`time.RFC3339`는 초 단위까지만 표현하지만, `time.RFC3339Nano`는 나노초까지 표현하면서 소수점 이하 trailing zero는 생략합니다. 나노초가 없는 시간은 `2026-03-29T09:00:00Z`처럼 깔끔하게 출력되고, 소수점이 있는 시간만 `2026-03-29T14:59:59.999Z`처럼 출력됩니다. 포맷이 일관되므로 기존 테스트 기댓값도 그대로 유지됩니다.

---

## 수정 3. 필터 빈 상태 UX 분리

### 문제

`HomePage.tsx`는 `logs.length === 0`만 보고 무조건 "첫 커피 기록을 남길 차례입니다"와 `Create first log` 버튼을 렌더링했습니다. 필터가 활성화된 상태에서 결과가 0건이어도 같은 화면이 나오므로, 사용자가 "기록이 사라졌다"고 오해할 수 있습니다.

### 수정 내용

`hasActiveFilter`로 분기합니다.

```tsx
const hasActiveFilter = !!logType || !!dateFrom || !!dateTo

{!isLoading && !isError && logs.length === 0 ? (
  hasActiveFilter ? (
    // 필터 결과 0건: 데이터가 없는 것이 아니라 조건에 맞는 항목이 없음
    <div>
      <p>조건에 맞는 기록이 없습니다.</p>
      <button onClick={handleClearFilters}>필터 초기화</button>
    </div>
  ) : (
    // 실제 첫 방문 상태
    <div>
      <p>첫 커피 기록을 남길 차례입니다.</p>
      <Link to="/logs/new">Create first log</Link>
    </div>
  )
) : null}
```

---

## 수정 4. `openapi.yml` 문서 drift 정정

`CreateLogRequest`와 `UpdateLogRequest`의 `recorded_at` 설명이 `RFC3339 또는 YYYY-MM-DD`로 되어 있었지만, 서버는 RFC3339만 허용합니다. 설명을 실제 계약에 맞게 수정했습니다.

```yaml
# 변경 전
description: RFC3339 또는 YYYY-MM-DD

# 변경 후
description: RFC3339 datetime (예: 2026-03-29T10:00:00Z 또는 2026-03-29T10:00:00+09:00)
```

---

## 추가된 테스트

| 테스트 | 내용 |
|--------|------|
| `TestValidateRecordedAt_NormalizesToUTC` | UTC 입력은 그대로 통과, `+09:00` 오프셋은 UTC로 정규화, 빈 값/잘못된 형식은 검증 에러 |
| `TestValidateDateFilter_YYYYMMDDNormalization` 업데이트 | KST 기준 경계값으로 기댓값 수정, RFC3339 오프셋 정규화 케이스 추가 |
| `TestListLogs_DateFilterNormalizesYYYYMMDD` 업데이트 | KST 기준 UTC 변환 결과(`2026-02-28T15:00:00Z`, `2026-03-29T14:59:59.999Z`)로 기댓값 수정 |

---

## 결론

세 가지 수정이 모두 "시간"이라는 공통 주제에 묶여 있습니다.

- **저장**: `recorded_at`을 UTC RFC3339Nano 단일 포맷으로 강제하여 SQLite 정렬 안정성 확보
- **필터**: 날짜 경계를 KST 기준 UTC instant로 계산하여 저장 포맷과 일치
- **UX**: 필터와 빈 상태의 의미를 분리하여 사용자 오해 방지

이로써 커서 페이지네이션의 전제(정렬 안정성)와 날짜 필터의 의미(사용자 기준 날짜)가 모두 일관되게 정리되었습니다.
