# Phase 3 코드 리뷰

## 범위

이 문서는 `Phase 3 — 자동완성` 구현을 기준으로 작성했습니다.

- Backend: `GET /api/v1/suggestions/tags`, `GET /api/v1/suggestions/companions`
- Frontend: `TagInput`, `useSuggestions`, `LogFormPage` 자동완성 연결

## 검증

- `cd backend && GOCACHE=/tmp/coffee-of-the-day-go-build go test ./...`
- `cd frontend && npm exec vitest run`
- `cd frontend && npm run lint`

모든 명령은 통과했습니다. 다만 Phase 3 전용 경계 조건은 테스트가 아직 충분히 덮고 있지 않습니다.

## Findings

### 높음 1. 자동완성 API가 `X-User-Id` 위조만으로 과거 동반자/태그 히스토리를 열람할 수 있게 만들어서 개인정보 노출 면적을 크게 넓힙니다

근거:

- 현재 인증은 `X-User-Id` 헤더가 비어 있지 않은지만 확인하고 그대로 신뢰합니다.
  - `backend/internal/handler/middleware.go:17-29`
- 자동완성 엔드포인트도 같은 미들웨어만 거친 뒤 바로 열려 있습니다.
  - `backend/cmd/server/main.go:67-84`
- 프론트 요청도 전역 `userId` 값을 그대로 헤더에 실어 보냅니다.
  - `frontend/src/api/client.ts:17-31`
- 검색어는 빈 문자열도 허용되고, 빈 문자열이면 전체 상위 10개를 반환합니다.
  - `backend/internal/service/suggestion_service.go:63-69`
  - `backend/internal/repository/suggestion_repository.go:42-47`
  - `backend/internal/repository/suggestion_repository.go:60-68`
  - `openapi.yml:148-175`

영향:

- 지금 구조에서는 임의의 `X-User-Id`만 알거나 추측하면 특정 사용자의 동반자 이름, 취향 태그를 쉽게 수집할 수 있습니다.
- CRUD API도 같은 한계가 있지만, 자동완성은 "짧은 GET 요청 + 검색어 반복"만으로 히스토리를 빠르게 훑을 수 있어서 노출 면적이 더 큽니다.
- 특히 `companions`는 취향 데이터보다 민감도가 높을 수 있어, Phase 4 전에 외부에 노출되면 곧바로 개인정보 이슈가 됩니다.

권장:

- 우선순위 1: Phase 4 인증이 들어가기 전까지는 자동완성 API를 로컬 개발 전용으로 제한하거나, 최소한 production/dev-shared 환경에서는 비활성화하는 것이 안전합니다.
- 우선순위 2: 인증 전환 후에도 자동완성 엔드포인트에는 rate limit, access log, 최소 검색어 길이 같은 방어선을 추가하는 편이 좋습니다.
- 우선순위 3: `companions` 자동완성은 특히 민감하므로, 빈 검색어 전체 반환은 막고 prefix 기반 최소 2자 이상에서만 열도록 줄이는 편이 안전합니다.

### 중간 2. `TagInput`이 한국어 IME 입력과 키보드 자동완성 상호작용을 제대로 처리하지 못해서 실제 사용 중 오입력 가능성이 있습니다

근거:

- `Enter` 또는 `,` 입력 시 현재 입력값을 즉시 태그로 확정하는데, IME composition 여부를 확인하지 않습니다.
  - `frontend/src/components/TagInput.tsx:39-49`
- 드롭다운은 마우스 선택만 고려되어 있고, active option 상태, 화살표 이동, `aria-activedescendant` 같은 combobox 동작이 없습니다.
  - `frontend/src/components/TagInput.tsx:89-120`

영향:

- 한국어 입력 중 `Enter`가 조합 확정이 아니라 태그 추가로 처리되면, 미완성 문자열이나 의도치 않은 조각이 저장될 수 있습니다.
- 키보드만 사용하는 사용자는 추천 목록을 사실상 탐색할 수 없고, 스크린리더 관점에서도 현재 마크업은 완전한 combobox로 보기 어렵습니다.
- 이 컴포넌트는 `companions`, `tasting_tags` 둘 다 공용이라 문제 발생 범위가 넓습니다.

권장:

- 우선순위 1: `e.nativeEvent.isComposing` 또는 composition 이벤트를 사용해 IME 조합 중 `Enter`/`,` 처리 로직을 무시해야 합니다.
- 우선순위 2: active option state를 두고 `ArrowUp/ArrowDown/Enter/Escape`를 지원하는 실제 combobox 패턴으로 올리는 편이 좋습니다.
- 우선순위 3: 이 동작은 회귀가 잦은 영역이므로 `TagInput` 컴포넌트 테스트를 별도로 추가하는 것이 맞습니다.

### 중간 3. 검색 정책이 계획과 어긋나 있고, 현재 구현은 타이핑마다 전체 스캔을 유발해서 앞으로 성능과 운영 비용이 빨리 나빠질 수 있습니다

근거:

- Phase 계획은 `prefix 필터링`을 명시하고 있습니다.
  - `plan.md:234-236`
- 실제 구현은 `%q%` 부분 일치입니다.
  - `backend/internal/repository/suggestion_repository.go:42-47`
  - `backend/internal/repository/suggestion_repository.go:60-68`
- 프론트는 입력값이 1글자만 있어도 즉시 요청하고, debounce가 없습니다.
  - `frontend/src/hooks/useSuggestions.ts:8-15`
  - `frontend/src/components/TagInput.tsx:32-37`

영향:

- `json_each()`로 배열을 펼친 뒤 `LOWER(...) LIKE '%q%'`를 적용하는 방식은 입력마다 사실상 전체 후보를 훑는 쿼리입니다.
- prefix가 아니라 contains라서 나중에 인덱싱이나 집계 최적화를 붙이기도 더 어렵습니다.
- 현재는 개인용 POC라 버틸 수 있어도, 로그가 쌓이거나 모바일 네트워크에서 쓰기 시작하면 자동완성이 "반응형"이 아니라 "키 입력마다 네트워크 호출"로 체감될 가능성이 큽니다.
- 공백만 입력해도 프론트는 요청을 보내고, 백엔드는 trim 후 빈 검색어로 해석해 상위 제안을 반환합니다. 즉 의도하지 않은 전체 조회가 쉽게 발생합니다.

권장:

- 우선순위 1: 계획대로 prefix 검색으로 되돌릴지, 아니면 contains를 유지할지 먼저 제품 정책을 명확히 정해야 합니다. 지금 상태는 계획/구현/학습문서가 서로 다릅니다.
- 우선순위 2: 프론트는 `trim()` 기준 최소 1~2자 이상일 때만 요청하고, 150~250ms debounce를 넣는 편이 좋습니다.
- 우선순위 3: 데이터가 더 늘면 suggestion 전용 정규화 테이블이나 집계 캐시를 두는 방향이 낫습니다. JSON 배열을 실시간 집계하는 방식은 초기엔 단순하지만 오래 가기 어렵습니다.

## 테스트 공백

- 백엔드는 `service` 레벨 테스트만 있고, `SuggestionHandler`와 `SQLiteSuggestionRepository`에 대한 테스트는 없습니다.
  - 현재 확인된 파일: `backend/internal/service/suggestion_service_test.go`
- 프론트는 `TagInput`, `useSuggestions`, `api/suggestions`에 대한 테스트가 없습니다.
  - 현재 통과한 테스트 스위트에는 Phase 3 전용 테스트 파일이 보이지 않습니다.

이 상태에서는 다음 경계 조건이 회귀로 빠질 가능성이 큽니다.

- 빈 검색어 / 공백 검색어 처리
- suggestion API의 에러 응답 매핑
- IME 입력 중 `Enter` 처리
- 추천 목록의 열림/닫힘/선택 동작
- 중복 태그와 이미 선택된 suggestion 제외 동작

## 개선 우선순위 제안

### 바로 수정 권장

1. 자동완성 API의 노출 범위를 인증 상태에 맞게 줄이기
2. `TagInput`의 IME 조합 중 `Enter` 오동작 방지
3. 검색 정책(prefix vs contains) 확정 후, 최소 검색 길이 + debounce 적용

### 다음 phase 전에 권장

1. `SuggestionHandler` / `SuggestionRepository` 테스트 추가
2. `TagInput` 키보드/접근성 테스트 추가
3. 공백 입력과 빈 검색어 정책을 프론트/백엔드에서 일관되게 정리

### 이후 구조 개선으로 권장

1. 자동완성 후보를 JSON 실시간 집계 대신 정규화/캐시 가능한 구조로 옮기기
2. `companions`처럼 민감한 suggestion 데이터에 대한 제품 정책 정리

## 결론

Phase 3는 기능 자체는 잘 붙었습니다. 레이어 분리도 무난하고, 프론트 연결도 간단하게 닫혀 있습니다. 다만 이번 phase는 CRUD보다 더 민감한 "과거 입력 히스토리 재노출" 기능이라서, 보안과 입력 UX 쪽 기준을 한 단계 높게 잡아야 합니다.

정리하면 가장 먼저 손봐야 할 것은 다음 세 가지입니다.

1. 인증 전 단계에서 자동완성 데이터가 너무 쉽게 열람되는 문제
2. 한국어 입력 환경에서 `TagInput`이 오동작할 수 있는 문제
3. 매 키입력마다 전체 스캔이 발생하는 현재 검색 정책

---

이 문서는 Codex가 작성했습니다.
