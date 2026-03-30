# Backlog

security, testing, 기술 부채, 기능 개선에 걸친 미완성 작업을 추적합니다.
AI agent가 추가 조사 없이 바로 작업을 시작할 수 있도록, 각 항목에 충분한 맥락을 포함합니다.

**상태:** `[ ]` 미시작 · `[~]` 진행 중 · `[x]` 완료
**우선순위:** `P0` 긴급 · `P1` 높음 · `P2` 보통 · `P3` 낮음

**항목 ID prefix:**
| Prefix | 카테고리 | 설명 |
|--------|---------|------|
| `SEC` | Security | 보안 취약점, 인증/인가, 데이터 노출 관련 |
| `TEST` | Testing | 테스트 커버리지 누락, 회귀 테스트, E2E 시나리오 |
| `DEBT` | 기술 부채 | 아키텍처 개선, 성능, 코드 품질 — 기능 변경 없이 내부 구조를 개선하는 작업 |
| `FEAT` | 기능 | 새로운 사용자 기능 또는 기획 요구사항 |

---

## Security

### [SEC-1] Suggestion endpoint 최소 입력 길이 강제
- **상태:** [ ]
- **우선순위:** P1
- **맥락:** `GET /suggestions/companions`에 빈 문자열을 전달하면 사용자의 전체 기록 상위 10건이 반환된다. `GET /suggestions/tags`도 동일한 문제가 있다. 단 한 번의 요청으로 개인 데이터를 덤프할 수 있는 상태다.
- **할 일:** 최소 1자 이상(companions는 2자 이상도 검토)이 입력되지 않으면 쿼리를 실행하지 않고 빈 배열을 반환한다. 오류가 아닌 빈 배열로 응답해야 한다. handler 레이어에서 처리한다.
- **완료 기준:**
  - `GET /suggestions/companions?q=` → `[]` 반환
  - `GET /suggestions/tags?q=` → `[]` 반환
  - 빈 입력 및 최소 길이 미만 케이스를 커버하는 unit test 추가
- **참조:** `docs/issues/initial/review/code_review_phase_3.md` (P3-3)

---

### [SEC-2] Suggestion endpoint rate limiting 및 access logging 추가
- **상태:** [ ]
- **우선순위:** P2
- **맥락:** auth endpoint에는 rate limiting(분당 20회/IP)이 적용되어 있지만 suggestion endpoint에는 없다. suggestion은 개인 기록 데이터를 반환하므로 rate limit이 없으면 저비용 열람 공격에 노출된다.
- **할 일:** `GET /suggestions/tags`, `GET /suggestions/companions`에 rate limiter를 적용한다(auth와 별도 limit, 예: 사용자당 분당 60회). suggestion 요청에 대한 structured access log를 추가한다.
- **완료 기준:**
  - rate limit 초과 시 `429` 반환
  - rate limit 값이 설정 파일에서 관리됨 (하드코딩 금지)
  - access log에 user ID, query string, 응답 건수 포함
- **참조:** `docs/issues/initial/review/code_review_phase_3.md` (P3-2)

---

### [SEC-3] 이메일 unique 제약을 DB 레벨에서 강제
- **상태:** [ ]
- **우선순위:** P2
- **맥락:** 현재 이메일 중복 검사는 application 레이어에서만 수행된다. 동시에 두 요청이 들어오면 application 레벨 검사를 통과하고 중복 row가 삽입될 수 있다.
- **할 일:** `users.email` 컬럼에 `UNIQUE` 제약을 추가하는 migration을 작성한다. SQLite의 `UNIQUE`는 대소문자를 구분하므로, service 레이어에서 이미 lowercase 정규화를 하고 있다면 그대로 `UNIQUE`만 추가하면 된다. DB constraint 위반 시 service 레이어에서 `409 Conflict`로 매핑한다.
- **완료 기준:**
  - 동일 이메일 중복 삽입 시 DB constraint 오류 발생
  - service 레이어가 해당 오류를 `409 Conflict`로 변환
  - 기존 이메일 정규화 로직(lowercase + trim) 유지
- **참조:** `docs/issues/initial/review/code_review_phase_4.md` (P4-4)

---

### [SEC-4] Refresh token rotation 구현
- **상태:** [ ]
- **우선순위:** P2
- **맥락:** 현재 토큰 무효화는 `token_version` 증가로만 처리되며, 이는 해당 사용자의 모든 세션을 일괄 무효화한다. per-session 추적이 없고 `jti`도 없다. refresh token이 탈취되어도 재사용 여부를 감지할 수 없다.
- **할 일:** rotation 전략을 결정한다. 최소 구현: `/auth/refresh` 호출마다 새 refresh token을 발급하고 이전 토큰을 무효화한다(단일 사용 토큰, `refresh_tokens` 테이블 + `jti` 기반). 완전 구현: 이미 rotate된 토큰이 재사용되면 해당 사용자의 모든 세션을 revoke한다.
- **완료 기준:**
  - refresh 성공 시 이전 토큰 무효화, 새 토큰 발급
  - 이미 사용된 refresh token 재사용 시 `401` 반환
  - `token_version`은 일괄 revoke 수단으로 유지하거나 제거 (구현 시 결정)
- **참조:** `docs/issues/initial/review/code_review_phase_4.md` (P4-1, P4-2)

---

### [SEC-5] 비밀번호 강도 검증 추가
- **상태:** [ ]
- **우선순위:** P3
- **맥락:** 현재 회원가입 시 비어있지 않은 문자열이면 비밀번호로 허용된다. 최소 길이나 복잡도 제약이 없다.
- **할 일:** service 레이어에서 최소 길이(예: 8자) 검증을 추가한다. 위반 시 명확한 오류 메시지와 함께 `400` 반환. `docs/openapi.yml`에 제약 조건을 문서화한다.
- **완료 기준:**
  - 최소 길이 미만 비밀번호 → `400` 반환 및 명확한 메시지
  - `docs/openapi.yml`에 제약 조건 반영
  - 경계값을 커버하는 unit test 추가

---

## Testing

### [TEST-1] E2E test: 전체 CRUD happy path
- **상태:** [ ]
- **우선순위:** P1
- **맥락:** Playwright가 설정되어 있고 auth E2E 테스트는 존재하지만, 핵심 사용자 흐름(로그 생성 → 목록 확인 → 상세 조회 → 수정 → 삭제)을 커버하는 E2E 테스트가 없다.
- **할 일:** 인증된 상태에서 cafe log 전체 CRUD를 커버하는 Playwright 테스트를 추가한다. brew log는 보너스지만 필수는 아니다.
- **완료 기준:**
  - `npm run test:e2e`로 실행 가능
  - 흐름: 로그인 → cafe log 생성 → 목록에 노출 확인 → 상세 페이지에서 데이터 확인 → 수정 후 반영 확인 → 삭제 후 목록에서 제거 확인
  - 테스트가 독립적으로 실행 가능 (자체 사용자 생성, 상태 정리)
- **참조:** `docs/issues/initial/review/code_review_phase_1.md` (deferred item)

---

### [TEST-2] Backend 회귀 테스트: 날짜 필터 + cursor + recorded_at 포맷 일관성
- **상태:** [ ]
- **우선순위:** P2
- **맥락:** 날짜 필터는 KST 기준 UTC boundary 계산을 사용하고, `recorded_at`은 저장 시 UTC RFC3339Nano로 정규화된다. 이 두 동작이 cursor pagination과 맞물릴 때 자정 근방 데이터에서 잘못된 결과가 나올 수 있다.
- **할 일:** KST 자정 근방 `recorded_at`을 가진 로그를 여러 건 저장하고, 날짜 필터 + cursor로 페이지네이션하며 결과가 올바르고 페이지 간 일관성이 있는지 검증하는 backend integration test를 작성한다.
- **완료 기준:**
  - KST 23:59 vs 익일 00:00 경계 케이스 커버
  - cursor가 날짜 경계를 넘을 때도 결과 누락/중복 없음
  - `go test ./...`로 실행 가능
- **참조:** `docs/issues/initial/review/code_review_phase_2.md` (P2-1)

---

## 기술 부채

### [DEBT-1] Suggestion 데이터 저장 구조 정규화
- **상태:** [ ]
- **우선순위:** P3
- **맥락:** tasting tag와 companion은 `coffee_logs` 테이블의 JSON 배열 컬럼에 저장되며, suggestion 쿼리 시 `json_each`로 분해해 집계한다. 단일 사용자·소량 데이터에서는 동작하지만, 로그 수가 늘어나면 suggestion 쿼리마다 전체 테이블 scan + JSON 분해가 발생한다.
- **할 일:** `log_tags`, `log_companions` 정규화 테이블을 도입하거나, 쓰기 시점에 갱신되는 별도 집계 테이블을 구성한다. 기존 데이터를 migration한다. suggestion repository를 새 구조 기반으로 교체한다.
- **완료 기준:**
  - suggestion 쿼리에서 `json_each` 제거
  - 기존 tag/companion 데이터 손실 없이 migration 완료
  - 로그 수 증가에 따른 suggestion 응답 시간 선형 악화 없음
- **주의:** `docs/spec.md` 변경이 필요할 수 있으므로 구현 전 사용자와 검토 필요.
- **참조:** `docs/issues/initial/review/code_review_phase_3.md` (P3-4)

---

### [DEBT-2] 트랜잭션 경계를 service 레이어로 이동 (트리거 발생 시)
- **상태:** [ ]
- **우선순위:** P3
- **맥락:** 현재 트랜잭션은 repository 레이어에서 관리되며, 단일 aggregate 저장에는 적합하다. 아래 트리거 중 하나가 발생하면 service 레이어로 이동이 필요하다: (1) 하나의 use case에서 두 개 이상의 repository를 atomic하게 쓰는 경우, (2) audit log나 outbox event를 같은 트랜잭션에 포함해야 하는 경우, (3) service 레이어에서 여러 repository 호출을 조율해야 하는 경우.
- **할 일:** 트리거 발생 시 service 레이어에 `TxManager` 추상화를 도입한다. repository는 자체 트랜잭션 대신 context 또는 인자로 전달받은 `*sql.Tx`를 사용한다.
- **완료 기준:**
  - 트리거 미발생 → 작업 불필요
  - 리팩토링 시: 기존 integration test 전체 통과
- **참조:** `docs/issues/initial/review/code_review_transaction_boundary.md`

---

### [DEBT-3] Backend의 동시 refresh 요청 처리
- **상태:** [ ]
- **우선순위:** P3
- **맥락:** frontend는 refresh 중복 호출을 single-flight으로 방지하지만, backend는 그렇지 않다. 브라우저 탭 여러 개가 동시에 동일한 refresh token으로 요청을 보내면 모두 성공하고 각각 새 token을 발급할 수 있다.
- **할 일:** [SEC-4]를 먼저 구현한다. 단일 사용 토큰 기반 rotation이 완료되면 두 번째 요청은 자동으로 `401`이 되므로, 이 항목은 [SEC-4] 완료 후 재평가한다.
- **완료 기준:** [SEC-4] 완료 후 동시 요청 케이스가 해당 테스트에서 커버되면 이 항목을 닫는다.
- **참조:** `docs/issues/initial/review/code_review_phase_4.md` (P4-3)

---

## 기능

### [FEAT-1] 비밀번호 재설정 플로우
- **상태:** [ ]
- **우선순위:** P2
- **맥락:** 비밀번호를 잊으면 계정을 복구할 방법이 없다. 현재는 단일 사용자 POC이므로 이메일 기반 외에도 간소한 방식을 검토할 수 있다.
- **할 일:** 재설정 방식(이메일 토큰 방식 vs 관리자 직접 방식 등)을 사용자와 먼저 논의한다. 이메일 발송 인프라가 필요한 경우 해당 의사결정도 포함된다.
- **완료 기준:** 구현 방식 결정 후 별도 정의.
- **주의:** 구현 전 `docs/spec.md` 검토 및 사용자 승인 필요.

---

## 작업 가이드

1. **항목 선택:** 우선순위(`P0` → `P1` → …) 순으로 진행한다. 같은 우선순위 내에서는 security 항목을 먼저 처리한다.
2. **시작 전:** GitHub issue를 생성하고 feature branch를 만든다. *"사용자와 검토 필요"* 또는 *"spec 검토 필요"* 표시가 있는 항목은 반드시 먼저 승인을 받는다.
3. **상태 갱신:** 작업 시작 시 `[ ]` → `[~]`, 완료 기준 충족 시 `[~]` → `[x]`로 변경한다.
4. **의존 관계:** [DEBT-3]은 [SEC-4]에 의존한다. 작업 중 발견한 새 의존 관계는 해당 항목에 기록한다.
5. **완료 기준:** acceptance criteria 충족 + 테스트 통과 + architecture 변경 시 `docs/arch/` 갱신.
