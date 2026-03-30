# Phase 2-2 Backend — 날짜 필터 버그픽스

## 무엇을 고쳤나

Phase 2-2에서 날짜 범위 필터(`date_from`, `date_to`)를 추가했지만, 백엔드가 `YYYY-MM-DD` 형식을 실제로 파싱하지 못하는 버그가 있었습니다.

- `openapi.yml`: `date_from`, `date_to`는 "YYYY-MM-DD 또는 RFC3339" 형식으로 명세
- 백엔드 에러 메시지: "RFC3339 datetime 또는 YYYY-MM-DD 형식이어야 합니다"
- 실제 구현: `parseDateTime`이 RFC3339만 파싱 → `YYYY-MM-DD` 입력 시 400 에러

---

## 왜 프론트엔드가 아닌 백엔드에서 고쳐야 하는가

프론트엔드에서 `<input type="date">`는 `YYYY-MM-DD` 문자열을 반환합니다. 이를 RFC3339로 변환하려면 시간대를 가정해야 합니다.

```ts
// ❌ 프론트엔드 변환 — 타임존 가정이 필요하고 명세와 어긋남
const from = new Date("2026-03-29").toISOString() // "2026-03-28T15:00:00.000Z" (KST 기준)
```

API가 날짜 단위 필터를 공개적으로 약속(`openapi.yml`, 에러 메시지)했으므로, 이를 지키는 것은 백엔드의 책임입니다.

---

## 핵심 이슈: SQLite 문자열 비교

`recorded_at`은 SQLite에 RFC3339 문자열로 저장됩니다 (예: `"2026-03-29T15:00:00Z"`).

SQL 필터는 문자열 비교입니다.

```sql
recorded_at >= ?   -- date_from
recorded_at <= ?   -- date_to
```

`date_to = "2026-03-29"`를 그대로 SQL에 넘기면:

```
"2026-03-29T15:00:00Z" <= "2026-03-29"  →  false
```

SQLite 문자열 비교에서 `"2026-03-29T..."` > `"2026-03-29"`이므로 당일 레코드가 모두 누락됩니다. 단순히 파싱만 추가해서는 안 되고, 반드시 **경계값 정규화**가 필요합니다.

---

## 해결 방법: `validateDateFilter`에 `endOfDay` 파라미터 추가

```go
func validateDateFilter(field, value string, endOfDay bool) (string, time.Time, error) {
    // YYYY-MM-DD 형식인 경우 하루의 시작 또는 끝 시각으로 정규화
    if d, err := time.Parse("2006-01-02", trimmed); err == nil {
        var normalized time.Time
        if endOfDay {
            normalized = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, time.UTC)
        } else {
            normalized = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
        }
        return normalized.Format(time.RFC3339), normalized, nil
    }
    // RFC3339 그대로 통과
    ...
}
```

| 입력 | `endOfDay` | 정규화 결과 | SQL 조건 |
|------|-----------|------------|---------|
| `date_from=2026-03-29` | `false` | `2026-03-29T00:00:00Z` | `recorded_at >= "2026-03-29T00:00:00Z"` ✓ |
| `date_to=2026-03-29` | `true` | `2026-03-29T23:59:59Z` | `recorded_at <= "2026-03-29T23:59:59Z"` ✓ |
| `date_from=2026-03-29T09:00:00Z` | `false` | 그대로 통과 | `recorded_at >= "2026-03-29T09:00:00Z"` ✓ |

---

## Java/Spring 비유

Spring Data JPA에서 날짜 범위 조회 시 `LocalDate`를 `LocalDateTime`으로 변환하는 패턴과 동일합니다.

```java
// Spring에서 날짜 경계값 변환
LocalDateTime from = localDate.atStartOfDay();               // 00:00:00
LocalDateTime to   = localDate.atTime(LocalTime.MAX);        // 23:59:59.999...
repository.findByCreatedAtBetween(from, to);
```

Go에서도 같은 원칙입니다. 날짜 필터는 항상 datetime 경계로 변환한 뒤 비교해야 합니다.
