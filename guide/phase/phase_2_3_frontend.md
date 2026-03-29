# Phase 2-3: 무한 스크롤 고도화

## 무엇을 만들었나

Phase 2-3에서는 무한 스크롤 경험을 세 가지 측면에서 완성했습니다.

1. **`LogCardSkeleton` 컴포넌트** — 초기 로딩과 추가 페이지 로딩 중 카드 자리를 채우는 플레이스홀더
2. **초기 로딩 스켈레톤** — `isLoading === true`일 때 실제 카드 대신 6개의 스켈레톤 표시
3. **추가 페이지 스켈레톤** — `isFetchingNextPage === true`일 때 그리드 하단에 3개의 스켈레톤 추가
4. **종료 메시지** — `hasNextPage === false`이고 기록이 존재할 때 "더 이상 기록이 없습니다" 표시

---

## IntersectionObserver sentinel 재설계

기존 코드는 sentinel `div`를 "Load more" 버튼과 같은 조건부 블록(`hasNextPage ? ...`) 안에 묶어두었습니다.

```tsx
// 이전
{hasNextPage ? (
  <div className="space-y-3">
    <div ref={sentinelRef} className="h-4" aria-hidden="true" />
    <button ...>Load more</button>
  </div>
) : null}
```

이 구조의 문제는 버튼의 UI 변경이 sentinel의 DOM 위치에 영향을 준다는 것입니다. Phase 2-3에서는 sentinel을 독립적인 요소로 분리했습니다.

```tsx
// 이후
{hasNextPage ? <div ref={sentinelRef} className="h-1" aria-hidden="true" /> : null}
```

Java/Spring 관점에서 비유하자면, sentinel은 "다음 페이지가 있을 때만 활성화되는 트리거"입니다. 비즈니스 로직(페이지 로드)과 UI(버튼 레이블, 스켈레톤)가 분리되어야 하는 것처럼, sentinel도 분리된 DOM 요소로 관리하는 것이 바람직합니다.

---

## 스켈레톤 설계 원칙

### co-location

`LogCardSkeleton`을 `LogCard.tsx`에 함께 배치했습니다. 동일 파일에 두면 실제 카드 레이아웃이 변경될 때 스켈레톤도 함께 업데이트할 가능성이 높아집니다.

### CLS(Cumulative Layout Shift) 방지

스켈레톤의 목적은 단순히 "로딩 중"을 알리는 것이 아닙니다. **실제 콘텐츠가 렌더링될 때 페이지가 갑자기 재배치되는 현상(CLS)** 을 방지하는 것입니다.

이를 위해 스켈레톤은 실제 카드와 동일한 컨테이너 클래스(`rounded-[1.75rem] border ...`)를 사용하고, 내부 요소들도 실제 카드의 높이에 맞춰 설계되었습니다.

Java/Spring의 DTO placeholder 패턴과 비슷합니다. 실제 데이터가 도착하기 전에 동일한 형태의 빈 객체를 먼저 보여주는 것입니다.

### `animate-pulse`

Tailwind CSS의 `animate-pulse`는 `opacity: 1 → 0.5 → 1`을 반복하는 단순한 CSS 애니메이션입니다. 별도의 JavaScript 없이 CSS만으로 "로딩 중" 상태를 시각적으로 표현합니다.

---

## 스켈레톤 배치 전략

추가 페이지 스켈레톤을 **그리드 내부**에 배치한 것이 중요합니다.

```tsx
<div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
  {logs.map((log) => <LogCard key={log.id} log={log} />)}
  {isFetchingNextPage
    ? Array.from({ length: 3 }).map((_, i) => <LogCardSkeleton key={`skeleton-next-${i}`} />)
    : null}
</div>
```

만약 스켈레톤을 그리드 바깥 별도 `div`에 넣었다면, 두 그리드 사이의 `gap`이 두 번 적용되어 불필요한 공백이 생겼을 것입니다. 그리드 안에 직접 삽입함으로써 실제 카드와 동일한 간격과 열 배치를 유지합니다.

---

## 상태 분기 정리

무한 스크롤 화면의 렌더링 분기를 정리하면 다음과 같습니다.

| 상태 | 렌더링 |
|------|--------|
| `isLoading` | 스켈레톤 6개 (초기 로드) |
| `isError` | 에러 메시지 |
| `logs.length === 0` (로드 완료 후) | 빈 상태 안내 |
| `logs.length > 0` + `isFetchingNextPage` | 카드 목록 + 하단 스켈레톤 3개 |
| `logs.length > 0` + `!hasNextPage` | 카드 목록 + "더 이상 기록이 없습니다" |

이 분기들은 서로 배타적이며, 각 상태에서 사용자가 기대하는 피드백을 명확하게 제공합니다.
