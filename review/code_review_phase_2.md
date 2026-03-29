# Phase 2 코드 리뷰

## 범위

이 문서는 `Phase 2 — 브루 폼 고도화 + 목록 필터 + 무한 스크롤 + Happy-path E2E` 구현을 기준으로 작성했습니다.

- Backend: 날짜 필터, 커서 기반 페이지네이션, E2E용 seed/server 설정
- Frontend: 브루 폼 UI, 홈 목록 필터, 무한 스크롤, Playwright E2E

## 검증

- `cd backend && go test ./...`
- `cd frontend && npm exec vitest run`
- `cd frontend && npm run lint`
- `cd frontend && npm run test:e2e`

모든 명령은 통과했습니다. 다만 아래 항목들은 현재 테스트가 놓치고 있는 경계 조건입니다.

## Findings

### 높음 1. 날짜 필터가 "사용자 로컬 날짜"가 아니라 UTC 날짜로 동작해서 한국 시간 기준 기록이 빠집니다

근거:

- 프론트는 `datetime-local` 값을 `Date`로 해석한 뒤 `toISOString()`으로 UTC 문자열로 전송합니다. 즉 사용자가 `2026-03-29 08:30`을 입력하면 서버에는 `2026-03-28T23:30:00.000Z`가 저장됩니다.
  - `frontend/src/pages/logFormState.ts:88-99`
  - `frontend/src/pages/logFormState.ts:215-218`
- 반면 날짜 필터 UI는 `type="date"`로 로컬 날짜(`YYYY-MM-DD`)만 보냅니다.
  - `frontend/src/components/FilterBar.tsx:51-67`
- 백엔드는 이 값을 UTC 하루 경계(`00:00:00Z` ~ `23:59:59Z`)로 확장합니다.
  - `backend/internal/service/log_service.go:590-617`

영향:

- 한국 시간 기준으로 `2026-03-29` 오전에 기록한 로그가, 필터 `date_from=2026-03-29&date_to=2026-03-29`에서 누락됩니다.
- 현재 프로젝트의 UI/문구/로케일이 한국어 중심이라 실제 사용자 체감은 "날짜 필터가 가끔 하루씩 어긋난다"는 형태로 나타납니다.

권장:

- 정책을 하나로 정해야 합니다.
- 가장 자연스러운 방법은 프론트가 선택한 로컬 날짜의 시작/끝을 UTC instant로 변환해 보내고, 백엔드는 RFC3339 instant 비교만 하도록 맞추는 것입니다.
- 또는 서버가 명시적으로 타임존을 받아 날짜 경계를 계산해야 합니다. 지금처럼 `YYYY-MM-DD -> UTC 하루`로 고정하면 한국 사용자 기준으로 계속 어긋납니다.

### 중간 2. `recorded_at`를 정규화하지 않고 원문 문자열 그대로 저장/정렬해서 커서 페이지네이션이 RFC3339 전체를 안정적으로 처리하지 못합니다

근거:

- 서버는 `recorded_at`이 RFC3339 형식인지 검증만 하고, 정규화 없이 원문 문자열을 그대로 저장합니다.
  - `backend/internal/service/log_service.go:579-587`
- 목록/필터/커서는 모두 `recorded_at` 문자열의 대소 비교에 의존합니다.
  - `backend/internal/repository/log_repository.go:142-166`
- `parseDateTime`은 `RFC3339Nano`와 `RFC3339`를 모두 허용하므로, 오프셋(`+09:00`)이나 소수초 유무가 다른 값이 함께 들어올 수 있습니다.
  - `backend/internal/service/log_service.go:718-727`

재현 예:

- SQLite 문자열 비교 기준으로 `'2026-03-29T10:00:00+09:00' < '2026-03-29T02:00:00Z'`는 `0(false)`입니다.
- 하지만 실제 시각으로는 `10:00+09:00 == 01:00Z`이므로 `02:00Z`보다 앞서야 합니다.

영향:

- 외부 API 클라이언트가 `Z` 대신 오프셋 포함 RFC3339를 보내면 정렬 순서가 깨질 수 있습니다.
- 그 상태에서 커서(`recorded_at DESC, id DESC`)를 만들면 다음 페이지 누락/중복 가능성이 생깁니다.
- 현재 브라우저 UI는 대부분 `toISOString()`을 보내서 덜 드러나지만, API 계약상 허용한 입력 전체에 대해서는 안전하지 않습니다.

권장:

- 저장 전에 `recorded_at`을 하나의 canonical format으로 강제해야 합니다. 예: `parsed.UTC().Format(time.RFC3339Nano)`.
- 날짜 필터와 커서도 같은 canonical format을 사용해야 문자열 비교가 시간 순서와 일치합니다.

### 중간 3. 필터 결과가 0건일 때도 "아직 로그가 없다"는 첫 방문용 빈 상태를 보여줘서 사용자를 오도합니다

근거:

- 홈 화면은 `logs.length === 0`만 보고 빈 상태를 렌더링합니다.
  - `frontend/src/pages/HomePage.tsx:180-193`
- 현재 화면은 `log_type`, `date_from`, `date_to` 필터를 모두 지원합니다.
  - `frontend/src/pages/HomePage.tsx:27-45`

영향:

- 사용자가 브루 탭이나 날짜 범위를 좁혀서 0건이 된 경우에도, 화면은 "아직 저장된 로그가 없습니다"와 `Create first log` CTA를 보여줍니다.
- 실제로는 데이터가 존재하는데도 사용자는 "기록이 사라졌다"거나 "다시 만들어야 하나?"라고 오해할 수 있습니다.

권장:

- "전체 데이터가 0건"과 "필터 결과가 0건"을 분리해야 합니다.
- 필터가 하나라도 활성화된 상태라면 "조건에 맞는 기록이 없습니다"와 "필터 초기화" 액션을 보여주는 편이 맞습니다.

## 추가 메모

- `openapi.yml`의 `CreateLogRequest.recorded_at`, `UpdateLogRequest.recorded_at` 설명은 아직 `RFC3339 또는 YYYY-MM-DD`로 남아 있는데, 실제 서버는 `YYYY-MM-DD`를 허용하지 않습니다.
  - `openapi.yml:333-363`
  - `backend/internal/service/log_service.go:579-587`
- 이 항목은 Phase 2 신규 버그라기보다 기존 문서 drift에 가깝지만, 프론트 타입 생성의 소스가 `openapi.yml`인 프로젝트라 우선순위를 낮게 두면 이후 혼선을 계속 만듭니다.

## 개선 우선순위 제안

### 바로 수정 권장

1. 날짜 필터의 로컬 날짜/UTC 경계 불일치 수정
2. `recorded_at` canonical format 강제

### Phase 3 시작 전 권장

1. 필터 결과 0건 전용 빈 상태 분리
2. 날짜 필터, 커서, 저장 포맷을 묶은 회귀 테스트 추가

### 함께 정리 권장

1. `openapi.yml`의 `recorded_at` 설명을 실제 서버 계약과 일치시키기

## 결론

Phase 2 결과물은 기능 범위 자체는 잘 닫혔습니다. 브루 폼, URL 기반 필터, 무한 스크롤, happy-path E2E까지 모두 갖춰져 있어서 사용자 관점의 기본 흐름은 안정적으로 보입니다.

다만 이번 phase에서 새로 들어온 핵심 리스크는 거의 모두 "시간"에 묶여 있습니다.

- 로컬 날짜와 UTC 날짜의 의미 차이
- RFC3339 문자열의 표현 차이와 정렬 안정성
- 그 결과로 생기는 필터/커서 경계 오류 가능성

정리하면, **가장 먼저 손봐야 할 것은 날짜 필터의 의미를 사용자 기준 날짜로 바로잡는 것**이고, 그 다음은 **`recorded_at` 저장 형식을 단일화해 커서 페이지네이션의 전제를 단단하게 만드는 것**입니다.

---

이 문서는 Codex가 작성했습니다.
