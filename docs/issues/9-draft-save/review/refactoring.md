# Refactoring — Issue #9 Code Review Follow-up

코드 리뷰(`code_review.md`)에서 지적된 7개 항목을 모두 처리했다. 백로그로 넘긴 항목은 없다.

## 처리 결과 요약

| # | 우선순위 | 카테고리 | 결과 | 커밋 |
|---|---------|---------|------|------|
| 1 | High | 테스트 픽스처 | 처리 | `5c823a3` |
| 2 | High | 테스트 누락 | 처리 | `5d0a38b` |
| 3 | Medium | UX 깜빡임 | 처리 | `5d0a38b` |
| 4 | Medium | Dead code | 처리 | `5c823a3` |
| 5 | Medium | UX 라벨 불일치 | 처리 | `5d0a38b` |
| 6 | Low | 의도 명시 | 처리 | `5d0a38b` |
| 7 | Low | 주석 보강 | 처리 | `5d0a38b` |

## 커밋 분리 기준

리뷰 응답을 두 커밋으로 나눴다.

- `5c823a3 refactor(backend): address review feedback on tests and dead code`
  - 백엔드 단독 변경(suggestion 테스트 픽스처, `normalizeListFilter` dead code 제거).
  - 백엔드 회귀 테스트만 영향을 받으므로 프론트엔드 변경과 분리해 리뷰 부하를 낮췄다.

- `5d0a38b refactor(frontend): address review feedback on draft UX and tests`
  - 프론트엔드 변경 일괄(LogDetailPage 렌더 가드, LogCard 라벨, RecipePickerModal status 명시, logFormState 주석, LogFormPage 임시 저장 테스트 7건).
  - 모두 draft UX의 일관성 강화 또는 테스트 보강으로 묶어 의미 단위가 같다.

## 변경 상세

### [#1, High] suggestion 테스트 픽스처 status 명시

**파일:** `backend/internal/repository/suggestion_repository_test.go`

`INSERT INTO coffee_logs` 9개 구문이 `status` 컬럼 없이 DEFAULT('published')에 의존하고 있었다. 같은 파일의 새 draft 격리 테스트(`TestGetTagSuggestions_DraftLog_Excluded` 등)는 `status`를 명시하므로 코드 스타일이 분기되어 있었고, 후속 기여자가 기존 패턴을 복사해 draft 픽스처를 작성하면 silent failure로 이어질 수 있다.

`Edit replace_all`로 컬럼 목록과 VALUES placeholder를 한 번에 동기화하고 데이터 라인의 log_type 다음에 `"published"`를 삽입했다. 9개 구문 모두 명시적으로 status 값을 갖도록 통일되어 픽스처 의도가 SQL 레벨에서 즉시 보인다.

### [#2, High] LogFormPage 임시 저장 버튼 테스트 7건

**파일:** `frontend/src/pages/LogFormPage.test.tsx`

plan.md 테스트 전략 섹션에 명시된 케이스가 모두 미구현 상태였다. 추가한 케이스:

1. 신규 cafe 폼에서 "임시 저장" 버튼 표시
2. 빈 cafe 폼 → disabled
3. cafe_name 입력 → enabled로 전환
4. 클릭 → `createLog` payload에 `status: 'draft'` 포함
5. published 수정 모드 → 버튼 미표시 (회귀 방지)
6. draft 수정 모드 → 버튼 표시
7. draft 수정 모드에서 "변경 저장" 클릭 → `updateLog` payload에 `status: 'published'` 포함

기존 mock 구조는 module-level `vi.mock` 안에서 반환값을 고정해 두어 케이스마다 `useLog`가 다른 로그를 돌려주도록 하기 어려웠다. **`vi.hoisted`로 mutable `mocks` 객체를 만들어** factory가 그 변수를 참조하게 했고, 각 테스트가 `mocks.logData = ...`로 픽스처를 심을 수 있게 했다. mock factory의 호이스팅 제약을 우회하면서도 mock 자체는 그대로 유지되는 패턴이다.

### [#3, Medium] LogDetailPage draft redirect 깜빡임 제거

**파일:** `frontend/src/pages/LogDetailPage.tsx`

`useEffect`에서 redirect를 트리거하기 전에 `{log ? <div>...</div> : null}` 블록이 한 프레임 렌더되어, draft 로그의 빈 상세 내용이 순간 노출됐다. 렌더 게이트를 `log && log.status !== 'draft'`로 강화해 draft는 곧바로 redirect만 발생하게 했다. published는 기존 동작 그대로 유지된다.

### [#4, Medium] normalizeListFilter dead code 제거 + 검증 순서 정리

**파일:** `backend/internal/service/log_service.go`

`if limit == 0` 검사가 두 번 등장했다. 첫 번째 블록에서 0을 `defaultListLimit`으로 치환하므로 두 번째는 절대 실행되지 않는 dead branch였다. 동시에 검증 순서를 **"입력 검증 → 기본값 채우기 → 상한 검사"** 순으로 정리했다.

```go
// Before
if limit == 0 { limit = defaultListLimit }
if limit < 0 { return ValidationError }
if limit > maxListLimit { return ValidationError }
if limit == 0 { return ValidationError }  // dead

// After
if limit < 0 { return ValidationError }    // 음수 거부 먼저
if limit == 0 { limit = defaultListLimit } // 그 다음 기본값
if limit > maxListLimit { return ValidationError }
```

순서를 바꾼 이유: "0은 defaultLimit과 동일하므로 의도적으로 0을 허용하는 클라이언트는 기본값을 받는다"는 의미가 더 명확해진다. 음수는 명백히 잘못된 입력이므로 가장 먼저 차단한다.

### [#5, Medium] LogCard footer 라벨 분기

**파일:** `frontend/src/components/LogCard.tsx`

draft 카드는 `/logs/{id}/edit`로 이동하는데 footer는 항상 "View log"로 표시되어 사용자가 click affordance와 실제 동작을 매칭하기 어려웠다. `isDraft ? '이어서 작성' : 'View log'` 분기로 명확화. plan.md "드래프트 카드의 to 분기" 의도와 일치하게 됐다.

### [#6, Low] RecipePickerModal status 명시

**파일:** `frontend/src/pages/LogFormPage.tsx`

`useLogList({ log_type: 'brew' })`는 backend default가 published이므로 안전했지만, 프론트엔드 코드만 보면 draft brew 로그가 레시피 후보에 섞일 가능성을 코드로는 배제할 수 없었다. `status: 'published'`를 명시하고 한국어 주석으로 의도를 표기했다. 향후 `ListLogsParams` 기본값이 바뀌어도 회귀가 차단된다.

### [#7, Low] buildLogPayload cafe 블록 주석

**파일:** `frontend/src/pages/logFormState.ts`

`cafe_name`/`coffee_name`은 status와 무관하게 trim 후 그대로 전달하지만, 같은 함수 내 옵셔널 필드는 `normalizeText`로 빈 값을 `undefined`로 정규화한다. 이 비대칭이 의도(draft validator에 위임)임을 명시하지 않으면 후속 기여자가 "draft에서 빈 값 제거"로 잘못 수정할 수 있다. 한국어 주석으로 plan.md 6항을 참조하도록 추가했다.

## 검증

```
backend  : go test ./...               → 모든 패키지 통과
frontend : npm test                    → 128/128 (이전 120 + 임시 저장 7 + 신규 helper 1)
frontend : npm run type-check          → 에러 없음
frontend : npm run build               → 성공 (396.34 kB / 116.91 kB gzip)
```

## 백로그 이전 항목

없음. 리뷰에서 지적된 7개 항목 모두 이번 PR 범위 내에서 해소했다.

## 후속 고려 사항(이번 범위 밖)

리뷰에서 직접 지적되지는 않았으나 작업 중 인지한 항목들. 별도 이슈로 다룰 만한 가치가 있다고 판단되면 backlog.md에 등재한다(현재는 등재 보류).

- **`vi.hoisted` 패턴의 다른 페이지 적용**: 이번에 LogFormPage.test.tsx에 도입한 mutable mock 패턴은 LogDetailPage, HomePage 같은 다른 페이지 테스트에서도 useful할 수 있다. 동일 보일러플레이트가 반복되면 공통 helper로 추출 검토.
- **`canSaveAsDraft`의 brew 분기**: brew 폼은 `brewMethod` 기본값('pour_over') 때문에 사실상 항상 `true`를 반환한다. 이미 backlog.md `[DEBT-8]`로 등재됨.
- **드래프트 자동 만료 정책**: spec 6.5의 미정 항목, backlog.md `[DEBT-10]` 등재됨.

---

*작성일: 2026-05-10 · 처리 커밋: `5c823a3`, `5d0a38b`*
