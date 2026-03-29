# Phase 1 코드 리뷰

## 범위

이 문서는 `Phase 1 — 프로젝트 기반 + 기록 CRUD` 전체 구현을 기준으로 작성했습니다.

- Backend: 마이그레이션, repository, service, handler, middleware, 서버 부트스트랩
- Frontend: API 클라이언트, TanStack Query 훅, 목록/상세/폼 페이지, 공용 컴포넌트
- 검증 기준: `go test ./...`, `npm exec vitest run`, `npm run lint`

## 한 줄 총평

Phase 1 결과물은 **"작동하는 첫 vertical slice"로서는 충분히 좋은 수준**입니다.  
백엔드 레이어 분리가 명확하고, 프론트도 타입 생성과 폼 변환 계층을 따로 둔 점이 좋습니다. 다만 **보안/데이터 무결성 측면의 POC 한계**와 **Phase 2부터 체감될 성능/구조 리스크**가 남아 있습니다.

## 잘된 점

### 1. 백엔드 레이어 경계가 비교적 명확합니다

- `handler -> service -> repository` 흐름이 분명하고, 각 레이어 책임이 크게 섞이지 않았습니다.
- 특히 service 계층에서 유효성 검사와 정규화를 담당하고, repository 계층에서는 DB 작업과 트랜잭션에 집중하고 있습니다.

관련 파일:
- `backend/internal/service/log_service.go`
- `backend/internal/repository/log_repository.go`
- `backend/internal/handler/log_handler.go`

### 2. OpenAPI와 프론트 타입 생성 흐름이 잘 잡혀 있습니다

- 프론트 타입을 백엔드 코드에서 손으로 복제하지 않고 `openapi.yml -> schema.ts -> app types` 흐름으로 가져간 점이 좋습니다.
- 이 구조는 이후 API 변경 시 프론트 타입 drift를 줄이는 데 효과적입니다.

관련 파일:
- `openapi.yml`
- `frontend/src/types/schema.ts`
- `frontend/src/types/log.ts`

### 3. 프론트 폼 상태와 API payload 변환을 분리한 판단이 좋습니다

- `frontend/src/pages/logFormState.ts`에서 화면 입력 상태와 실제 API 요청 본문을 분리한 것은 Phase 1 기준에서 특히 좋은 선택입니다.
- 이 덕분에 `LogFormPage`가 전부 mapper 역할까지 떠안지 않고, 테스트 가능한 순수 함수가 생겼습니다.

관련 파일:
- `frontend/src/pages/logFormState.ts`
- `frontend/src/pages/logFormState.test.ts`

### 4. 기본 검증 세트가 이미 갖춰져 있습니다

- backend unit/integration test
- frontend API/hook/component/form-state test
- frontend lint

Phase 1에서 이 정도 검증 습관이 잡혀 있다는 점은 이후 phase 확장에 유리합니다.

## 주요 취약점 및 리스크

### 높음 1. SQLite 외래키 제약이 실제로는 보장되지 않을 가능성이 큽니다

문제:
- 스키마에는 FK와 `ON DELETE CASCADE`가 선언되어 있지만, SQLite는 연결 시 `PRAGMA foreign_keys = ON`이 활성화되지 않으면 이를 강제하지 않습니다.
- 현재 서버 부트스트랩은 plain `sql.Open("sqlite3", cfg.DBPath)`만 수행하고 있습니다.

근거:
- `backend/cmd/server/main.go:25`
- `backend/db/migrations/002_create_coffee_logs.up.sql:1`
- `backend/db/migrations/003_create_cafe_logs.up.sql:1`
- `backend/db/migrations/004_create_brew_logs.up.sql:1`

영향:
- 존재하지 않는 `user_id`로 로그가 생성될 수 있습니다.
- FK가 꺼져 있으면 `CASCADE` 기대가 깨집니다.
- 결과적으로 "DB 제약이 있으니 안전하다"는 가정이 흔들립니다.

평가:
- 이 문제는 단순 성능 이슈가 아니라 **데이터 무결성 리스크**입니다.
- Phase 1 코드 중 우선순위가 가장 높습니다.

개선:
- SQLite 연결 직후 `PRAGMA foreign_keys = ON`을 강제해야 합니다.
- 가능하면 DSN 수준에서도 외래키 활성화를 명시하는 편이 안전합니다.

### 높음 2. `X-User-Id` 헤더는 누구나 위조할 수 있습니다

문제:
- 현재 사용자 식별은 `X-User-Id` 헤더 값을 그대로 신뢰합니다.
- 프론트도 기본값 `dev-user`를 전역 상태로 들고 있고, 누구든 헤더를 바꿔 다른 사용자처럼 요청할 수 있습니다.

근거:
- `backend/internal/handler/middleware.go:12-24`
- `frontend/src/api/client.ts:15-27`

영향:
- 멀티유저 구조를 이미 갖고 있지만, 실제 보호는 없습니다.
- 외부 공개 환경에서는 타 사용자 데이터 접근/수정이 가능한 구조입니다.

평가:
- 이건 설계 실수라기보다 **POC 전제에 묶인 임시 구현**입니다.
- 다만 그 전제가 README나 운영 방식에서 분명히 드러나야 하고, 외부 공개 환경에 그대로 두면 안 됩니다.

후속 phase 여부:
- 이 항목은 **Phase 4 인증/JWT 도입에서 해결 예정**입니다.
- 따라서 "아직 안 한 것" 자체는 계획과 일치합니다.
- 다만 그 전까지는 반드시 로컬/개발 환경 한정으로 취급해야 합니다.

### 중간 1. 요청 본문 검증이 느슨해서 오타와 과도한 payload를 조용히 받아들입니다

문제:
- `json.Decoder`는 사용하지만 `DisallowUnknownFields()`가 없습니다.
- 요청 바디 크기 제한도 없습니다.

근거:
- `backend/internal/handler/log_handler.go:108-121`
- `backend/internal/handler/log_handler.go:196-209`

영향:
- 클라이언트가 `coffee_nmae`처럼 잘못된 필드를 보내도 일부는 조용히 무시될 수 있습니다.
- 큰 요청 본문을 방어하는 장치가 없어 API 경계가 느슨합니다.

개선:
- `http.MaxBytesReader`로 body size 제한
- `decoder.DisallowUnknownFields()` 적용
- 필요하다면 validation error를 field 단위로 더 구조화

평가:
- 지금 당장 서비스가 무너지지는 않지만, API 신뢰도를 떨어뜨리는 지점입니다.

### 중간 2. 목록 조회가 N+1 쿼리 구조입니다

문제:
- 목록 쿼리로 공통 로그를 읽은 뒤, 각 항목마다 `loadDetail`을 다시 호출합니다.
- 즉 목록 20건이면 최소 21개 쿼리 패턴입니다.

근거:
- `backend/internal/repository/log_repository.go:140-194`
- `backend/internal/repository/log_repository.go:286-336`

영향:
- Phase 1 규모에서는 버틸 수 있습니다.
- 하지만 Phase 2에서 필터, 무한 스크롤, 데이터 증가가 본격화되면 응답 시간이 눈에 띄게 나빠질 수 있습니다.

개선:
- 목록 전용 projection query를 분리
- `LEFT JOIN` 기반 조회 또는 `UNION`/타입별 projection 전략 검토
- "목록에 필요한 필드만" 읽는 DTO를 따로 두는 것도 방법입니다

후속 phase 여부:
- 이 문제는 **Phase 2 탐색/목록 고도화 전에 해결하는 것이 좋습니다.**
- 아직 Phase 2 기능이 본격 추가되기 전이라, 지금은 "허용 가능한 부채"에 가깝습니다.

### 중간 3. 도메인 invariant가 일부 느슨합니다

문제:
- `loadDetail()`은 상세 레코드가 없어도 정상으로 처리합니다.
- 하지만 현재 도메인 규칙상 `cafe` 로그에는 `cafe_detail`, `brew` 로그에는 `brew_detail`이 사실상 필수입니다.

근거:
- `backend/internal/repository/log_repository.go:283-336`

영향:
- DB가 수동 변경되거나 partial failure가 생기면, API 응답이 `log_type`과 detail 존재 여부가 어긋난 상태가 될 수 있습니다.
- 프론트는 discriminated union을 기대하므로, 실제 런타임에서는 `log.cafe`/`log.brew` 접근 시 깨질 위험이 있습니다.

개선:
- 상세 데이터 누락은 정상 상태가 아니라 데이터 손상으로 간주하는 것이 더 일관적입니다.
- 최소한 `GetLogByID`와 `ListLogs`에서는 mismatch를 에러로 승격하는 편이 안전합니다.

## 아쉬운 점

### 1. `LogFormPage`가 이미 꽤 큰 단일 컴포넌트입니다

문제:
- 페이지 안에 공통 필드, 카페 섹션, 브루 섹션, 스텝 조작, 저장 흐름이 모두 들어 있습니다.
- 현재도 400라인 이상이고, Phase 2에서 브루 폼이 더 고도화되면 유지보수가 빠르게 어려워질 가능성이 높습니다.

근거:
- `frontend/src/pages/LogFormPage.tsx:108-420`

개선:
- `CommonFieldsSection`
- `CafeFieldsSection`
- `BrewFieldsSection`
- `BrewStepsEditor`

정도로 분리하는 것이 좋습니다.

후속 phase 여부:
- 이 항목은 **Phase 2 브루 폼 고도화 전에 리팩터링하는 것이 가장 자연스럽습니다.**

### 2. 날짜 입력 계약이 프론트와 백엔드 사이에서 완전히 일치하지 않습니다

문제:
- 백엔드는 `recorded_at`에 `RFC3339`와 `YYYY-MM-DD`를 모두 허용합니다.
- 프론트는 `datetime-local` 기반 입력만 제공하고, `Date` 객체 변환을 통해 ISO 문자열로 다시 직렬화합니다.

근거:
- `backend/internal/service/log_service.go:579-587`
- `frontend/src/pages/LogFormPage.tsx:281-289`
- `frontend/src/pages/logFormState.ts:80-100`

영향:
- 스펙상 허용하는 date-only 입력이 UI에서는 자연스럽게 드러나지 않습니다.
- 시간대 처리에 민감한 케이스에서는 예상과 다른 시각으로 재직렬화될 여지가 있습니다.

개선:
- 정책을 하나로 통일하는 편이 좋습니다.
- 선택지는 둘 중 하나입니다.
1. Phase 1 범위에서는 `recorded_at`를 datetime-only로 고정
2. 실제로 date-only까지 지원할 것이라면 UI 입력 모델도 분리

### 3. UI 언어 톤이 아직 일관되지 않습니다

문제:
- 한국어 설명과 영어 버튼/섹션명이 혼재되어 있습니다.
- 예: `Write today's log`, `Log detail`, `Capture a coffee moment`, `Back to list`

근거:
- `frontend/src/pages/HomePage.tsx:54-69`
- `frontend/src/pages/LogDetailPage.tsx:73-89`
- `frontend/src/pages/LogFormPage.tsx:204-220`

영향:
- 기능상 문제는 아니지만, 공개용 앱에서는 완성도가 낮아 보일 수 있습니다.

개선:
- 언어 정책을 한 번 정하면 전체 화면에 통일하는 것이 좋습니다.
- 현재 README 방향을 보면 한국어 우선이 더 자연스럽습니다.

### 4. 필드 단위 오류 피드백이 부족합니다

문제:
- 서버 validation error는 있지만, 프론트에서는 대부분 상단 에러 박스로만 보여줍니다.
- 어떤 필드가 잘못됐는지 사용자가 다시 추측해야 합니다.

근거:
- `backend/internal/service/log_service.go`의 `ValidationError`
- `frontend/src/pages/LogFormPage.tsx:231-235`

영향:
- 폼이 커질수록 UX가 빠르게 나빠집니다.

개선:
- `field -> message` 매핑
- 입력 필드 아래 inline error 표시
- submit 직후 첫 invalid 필드 focus 이동

후속 phase 여부:
- 브루 폼이 더 복잡해지는 **Phase 2 전에 착수하면 효과가 큽니다.**

### 5. E2E 자동화가 아직 없습니다

문제:
- 현재는 unit test와 hook/component test는 있지만, 실제 사용자 흐름 전체를 검증하는 자동화가 없습니다.

영향:
- "생성 -> 목록 반영 -> 상세 -> 수정 -> 삭제"처럼 Phase 1의 핵심 흐름은 수동 테스트 의존도가 남아 있습니다.

개선:
- Playwright 또는 Cypress 기반의 최소 happy-path E2E 추가

평가:
- 코드 품질을 한 단계 올리려면 이 부분이 중요합니다.
- 이 항목은 특정 phase 계획에 직접 묶여 있지는 않지만, 앞으로 phase가 늘어날수록 필요성이 커집니다.

## 개선 우선순위 제안

### 바로 수정 권장

1. SQLite foreign key 활성화
2. 요청 body size 제한 + unknown field 차단
3. 상세 누락을 정상 상태로 두는 repository invariant 정리

### Phase 2 시작 전 권장

1. 목록 N+1 조회 구조 개선
2. `LogFormPage` 컴포넌트 분리
3. 필드 단위 validation UX 개선
4. `recorded_at` 입력 정책 정리

### 계획상 후속 phase에서 해결 가능

1. 인증/사용자 위조 문제  
   `Phase 4`에서 JWT/cookie 기반 인증으로 해결 예정

2. 필터 부재와 목록 탐색 부족  
   `Phase 2`에서 타입/날짜 필터 및 무한 스크롤 고도화 예정

3. 태그/동행인 입력 편의성 부족  
   `Phase 3` 자동완성에서 개선 예정

## 결론

Phase 1 코드는 **"다음 phase로 넘어갈 수 있을 정도의 기반"은 충분히 갖추고 있습니다.**  
특히 레이어 분리, 타입 흐름, 기본 테스트 습관은 좋은 출발점입니다.

다만 현재 상태를 더 오래 유지하면 문제가 커질 가능성이 높은 부분도 명확합니다.

- SQLite 외래키 활성화 누락 가능성
- POC 인증 방식의 명백한 한계
- 목록 조회 N+1 구조
- 커지는 폼 페이지 구조

정리하면, **지금 당장 가장 먼저 손봐야 할 것은 데이터 무결성과 API 경계**, 그리고 **Phase 2 전에 선제적으로 줄여야 할 것은 성능 부채와 폼 복잡도**입니다.

---

이 문서는 Codex가 작성했습니다.
