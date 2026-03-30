# Phase 4 코드 리뷰

## 범위

이 문서는 `Phase 4 — 계정 및 인증` 종료 시점의 구현을 기준으로 작성했다.

- Backend: JWT 인증, 쿠키 기반 세션, 인증 미들웨어, 사용자 조회
- Frontend: 로그인/회원가입 UI, 보호 라우트, 자동 토큰 갱신, 로그아웃

검토 대상 핵심 파일:

- `backend/config/config.go`
- `backend/cmd/server/main.go`
- `backend/internal/service/auth_service.go`
- `backend/internal/handler/auth_handler.go`
- `backend/internal/handler/middleware.go`
- `backend/internal/repository/user_repository.go`
- `backend/db/queries/users.sql`
- `frontend/src/api/client.ts`
- `frontend/src/hooks/useAuth.ts`
- `frontend/src/router.tsx`
- `openapi.yml`

## 요약

Phase 4는 기능적으로는 닫혀 있다. 인증 흐름 자체도 테스트에서 잘 동작하고, 백엔드 단위 테스트와 프론트 단위 테스트, 프론트 빌드도 통과한다.

다만 "작동한다"와 "운영 가능한 인증 경계다"는 다르다. 현재 상태는 POC로는 충분하지만, 인증 영역 기준으로 보면 아직 다음 보완이 필요하다.

1. 토큰 수명 관리가 너무 낙관적이다.
2. 운영 환경 안전장치가 부족하다.
3. 계정 식별자 정규화와 스펙/툴링 정합성이 아직 거칠다.

## 검증

- `cd backend && go test ./...` : 통과
- `cd frontend && npx vitest run` : 통과
- `cd frontend && npm run build` : 통과
- `cd frontend && npm run lint` : 실패
  - `frontend/src/router.tsx:17`에서 `react-refresh/only-export-components` 위반
- E2E는 이번 리뷰에서 다시 돌리지 않았다

## Claude Code용 실행 우선순위

1. `P1-01` JWT 시크릿 강제
2. `P1-02` refresh token revocation / rotation 설계 반영
3. `P2-01` 로그인/회원가입 rate limit 추가
4. `P2-02` 이메일 정규화 및 case-insensitive unique 처리
5. `P3-01` refresh single-flight 처리
6. `P3-02`, `P3-03` 툴링/스펙 정리

정렬 기준:

- 인증 우회나 세션 탈취로 이어질 수 있는 문제를 최우선으로 둔다
- 그 다음은 계정 데이터 정합성과 운영 장애 가능성
- 마지막은 툴링/문서/정리성 이슈

## Findings

### P1-01. `JWT_SECRET` 기본값이 하드코딩되어 있어서 운영 설정 실수 시 토큰 위조가 가능하다

- `Priority`: P1
- `Severity`: 높음
- `Category`: Security
- `왜 문제인가`: 현재는 환경변수가 없으면 서버가 안전하게 실패하지 않고, 예측 가능한 문자열로 토큰 서명/검증을 계속한다. 운영 배포에서 환경변수 누락이 한 번만 발생해도 액세스 토큰과 리프레시 토큰을 외부에서 위조할 수 있다.
- `근거`:
  - `backend/config/config.go:29-33`에서 `JWT_SECRET`이 없으면 `dev-secret-change-in-production`을 사용한다.
  - `backend/cmd/server/main.go:46-47`과 `backend/cmd/server/main.go:75`에서 그 값을 그대로 `AuthService`와 `JWTMiddleware`에 연결한다.
- `영향`:
  - 운영 환경 실수 한 번이 곧바로 인증 전체 붕괴로 이어진다.
  - 이 문제는 코드 버그라기보다 "운영 실수 허용 설계"라서 더 위험하다.
- `권장 수정`:
  - 개발 환경이 아니면 `JWT_SECRET` 미설정 시 서버가 시작하지 않게 바꾼다.
  - 가능하면 길이/엔트로피 최소 기준도 검증한다.
  - `config.Load()`에서 `IsProduction` 분기만 두지 말고, 명시적 `APP_ENV` 또는 유사 설정으로 환경 판별을 안정화한다.
- `Done when`:
  - 운영 모드에서 `JWT_SECRET`이 없거나 너무 짧으면 프로세스가 즉시 실패한다.
  - 테스트/로컬 개발에서만 안전한 dev secret fallback이 허용되거나, dev `.env`를 강제한다.
- `권장 테스트`:
  - config 단위 테스트: production + empty secret => 에러
  - server bootstrap 테스트: insecure secret로는 startup 실패

### P1-02. refresh token이 서버에서 추적되지 않아서 탈취 후에도 logout과 무관하게 재사용할 수 있다

- `Priority`: P1
- `Severity`: 높음
- `Category`: Security
- `왜 문제인가`: 현재 refresh token은 "서명만 맞으면" 계속 유효하다. 서버에 세션 저장소도 없고 `jti`, `token_version`, revoke list도 없다. 그래서 refresh token이 한 번 유출되면 사용자는 로그아웃해도 그 토큰을 서버가 막을 방법이 없다.
- `근거`:
  - `backend/internal/service/auth_service.go:138-150`에서 refresh token을 파싱한 뒤 사용자 존재만 확인하고 새 토큰을 다시 발급한다.
  - `backend/internal/service/auth_service.go:170-180`의 토큰 생성에는 세션 식별자나 회전 추적 정보가 없다.
  - `backend/internal/handler/auth_handler.go:119-123`의 logout은 서버 상태를 바꾸지 않고 쿠키만 만료시킨다.
- `영향`:
  - 탈취된 refresh token은 만료 시점까지 새 access token을 계속 발급받을 수 있다.
  - "로그아웃했는데도 다른 곳에서는 계속 로그인된다"는 보안상 가장 설명하기 어려운 상황이 생긴다.
- `권장 수정`:
  - 최소안: `users`에 `token_version`을 두고 토큰 클레임에 넣어 logout 시 증가시킨다.
  - 권장안: refresh token을 DB에 세션 단위로 저장하고 rotation + reuse detection까지 구현한다.
  - access/refresh 둘 다 `jti`를 넣고 감사 로그를 남길 수 있게 설계한다.
- `Done when`:
  - logout 이후 이전 refresh token으로 `/auth/refresh`가 실패한다.
  - refresh rotation 후 이전 refresh token 재사용 시 거부된다.
  - 서버가 "현재 유효한 세션"을 설명할 수 있다.
- `권장 테스트`:
  - refresh 후 이전 refresh token 재사용 거부 테스트
  - logout 후 refresh 실패 테스트
  - 동일 refresh token replay 시도 테스트

### P2-01. 로그인/회원가입 엔드포인트에 rate limit이 없어서 brute-force와 계정 열거 시도를 막기 어렵다

- `Priority`: P2
- `Severity`: 중간 이상
- `Category`: Security / Abuse prevention
- `왜 문제인가`: 인증 API는 인터넷에 가장 먼저 노출되는 엔드포인트인데, 현재는 시도 횟수 제한이 전혀 없다. 비밀번호 대입, credential stuffing, 이메일 대량 조회 시도를 서비스 레벨에서 방어하지 못한다.
- `근거`:
  - `backend/cmd/server/main.go:65-69`에서 `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`이 별도 방어 미들웨어 없이 직접 노출된다.
  - `backend/internal/handler/auth_handler.go:43-85`는 요청 본문 파싱 후 바로 서비스 호출만 한다.
- `영향`:
  - 단일 사용자 POC라도 공개 배포 시 무차별 로그인 시도가 바로 가능하다.
  - `ErrInvalidCredentials`로 사용자 존재 노출은 줄였지만, 시도 횟수를 막지 않으면 방어선이 약하다.
- `권장 수정`:
  - 최소안: IP 기준 sliding window rate limit 추가
  - 권장안: IP + email 조합 기준 제한, 실패 누적 시 backoff
  - 로그인 실패/refresh 실패에 대한 구조화 로그를 남긴다.
- `Done when`:
  - 짧은 시간 내 반복 로그인 시도가 429로 차단된다.
  - 정상 사용자의 UX를 과하게 해치지 않는 범위의 임계치가 문서화된다.
- `권장 테스트`:
  - 같은 IP에서 임계치 초과 시 429 테스트
  - 다른 계정/다른 IP 조합에서 정상 요청이 통과하는 테스트

### P2-02. 이메일 정규화가 없어서 같은 사용자를 서로 다른 계정처럼 저장하거나 로그인 실패를 만들 수 있다

- `Priority`: P2
- `Severity`: 중간
- `Category`: Data integrity / Auth correctness
- `왜 문제인가`: 회원가입 유효성 검사는 `TrimSpace`를 보지만, 저장과 조회에는 정규화된 값을 사용하지 않는다. 그래서 공백, 대소문자 차이로 사실상 같은 이메일이 다른 계정으로 들어가거나, 가입 후 로그인 입력값에 따라 조회가 실패할 수 있다.
- `근거`:
  - `backend/internal/service/auth_service.go:67-68`은 검증만 하고 정규화된 값을 다시 요청 객체에 반영하지 않는다.
  - `backend/internal/service/auth_service.go:88-94`는 `req.Email` 원본을 그대로 저장한다.
  - `backend/internal/service/auth_service.go:111-118`은 로그인 시에도 입력값을 정규화하지 않고 그대로 조회한다.
  - `backend/internal/repository/user_repository.go:78-88`과 `backend/db/queries/users.sql:6-7`은 `email = ?` 정확 일치 비교만 수행한다.
- `영향`:
  - `User@example.com`과 `user@example.com`을 다른 계정으로 저장할 수 있다.
  - `" user@example.com "`으로 가입한 뒤 `"user@example.com"`으로 로그인하면 실패할 수 있다.
  - 이후 password reset, 이메일 변경, 외부 인증 연동으로 갈수록 정합성 비용이 커진다.
- `권장 수정`:
  - 서비스 진입 직후 이메일을 `trim + lowercase` 정규화한다.
  - DB도 가능하면 정규화 컬럼 기준으로 unique를 건다.
  - SQLite에서는 `COLLATE NOCASE` 또는 정규화된 별도 컬럼/인덱스를 검토한다.
- `Done when`:
  - 가입/로그인 모두 동일한 정규화 규칙을 사용한다.
  - 대소문자/앞뒤 공백 차이는 동일 이메일로 취급된다.
  - unique 제약이 제품 정책과 일치한다.
- `권장 테스트`:
  - mixed-case email 중복 가입 거부 테스트
  - 앞뒤 공백이 있는 email 로그인 허용 테스트

### P3-01. 프론트의 자동 refresh가 single-flight가 아니라서 만료 시점에 병렬 요청이 몰리면 refresh 경쟁이 발생한다

- `Priority`: P3
- `Severity`: 중간
- `Category`: Frontend resilience
- `왜 문제인가`: 액세스 토큰이 만료된 상태에서 화면이 여러 데이터를 동시에 요청하면, 각 요청이 모두 `/auth/refresh`를 따로 호출한다. 지금은 동작할 수도 있지만, refresh rotation을 도입하면 이 구조가 바로 race condition으로 바뀐다.
- `근거`:
  - `frontend/src/api/client.ts:27-37`에서 각 401 응답이 독립적으로 `/auth/refresh`를 호출한다.
  - refresh 중복 실행을 막는 shared promise 또는 mutex가 없다.
- `영향`:
  - 네트워크 낭비가 생긴다.
  - 서버가 refresh rotation을 도입한 뒤에는 "먼저 재발급된 토큰 때문에 뒤 요청이 실패"하는 식의 간헐 오류가 생길 수 있다.
  - 탭 여러 개를 동시에 쓸 때도 같은 문제가 커진다.
- `권장 수정`:
  - 클라이언트에 refresh single-flight를 넣는다.
  - refresh 진행 중에는 다른 요청이 같은 promise를 await하도록 바꾼다.
  - refresh 실패 시 auth 캐시 정리와 리다이렉트 흐름을 한 곳으로 모은다.
- `Done when`:
  - 동시 401 다발 상황에서도 refresh 요청은 1회만 발생한다.
  - refresh 성공/실패 후 후속 요청 동작이 일관된다.
- `권장 테스트`:
  - 병렬 401 상황에서 `/auth/refresh` 1회만 호출되는 API client 테스트
  - refresh 실패 시 모든 대기 요청이 동일하게 실패하는 테스트

### P3-02. 현재 프론트 린트가 깨져 있어서 리팩토링 이후 회귀 감지가 약하다

- `Priority`: P3
- `Severity`: 낮음
- `Category`: Tooling / Maintainability
- `왜 문제인가`: 기능은 동작하지만, 기본 정적 검증이 이미 깨진 상태다. 이후 Claude Code나 다른 agent가 리팩토링하면서 "기존부터 실패하던 lint" 때문에 실제 회귀와 기존 노이즈를 구분하기 어려워진다.
- `근거`:
  - `npm run lint`가 실패한다.
  - 에러 위치는 `frontend/src/router.tsx:17`이며 `ProtectedRoute`와 `router` export가 같은 파일에 있어 `react-refresh/only-export-components` 규칙을 위반한다.
- `영향`:
  - CI 도입 시 바로 실패한다.
  - 프론트 리팩토링 PR에서 lint 신뢰도가 떨어진다.
- `권장 수정`:
  - `ProtectedRoute`를 별도 파일로 분리하거나, router 정의와 컴포넌트 export 구조를 정리한다.
- `Done when`:
  - `npm run lint`가 clean pass 한다.

### P3-03. OpenAPI와 실제 라우팅 정책이 어긋나 있어서 이후 자동 생성 클라이언트/문서가 잘못될 수 있다

- `Priority`: P3
- `Severity`: 낮음
- `Category`: Spec consistency
- `왜 문제인가`: 실제 서버는 logout을 비인증 라우트로 열어두고 있는데, OpenAPI에서는 전역 `cookieAuth`가 걸린 상태에서 `/auth/logout`만 예외 처리하지 않았다. 지금은 수동 클라이언트를 써서 크게 티가 안 나지만, 이후 generated client나 문서화 단계에서 혼선을 만든다.
- `근거`:
  - `backend/cmd/server/main.go:64-69`에서 `/auth/logout`은 인증 불필요 라우트다.
  - `openapi.yml:79-85`의 `/api/v1/auth/logout`에는 `security: []`가 없다.
- `영향`:
  - 스펙상으로는 로그아웃에 access token이 필요하다고 읽힐 수 있다.
  - generated SDK나 테스트 도구가 실제 서버 정책과 다르게 동작할 수 있다.
- `권장 수정`:
  - `openapi.yml`의 `/auth/logout`에 `security: []`를 명시한다.
  - 스펙 변경 후 `cd frontend && npm run generate`를 다시 실행한다.
- `Done when`:
  - OpenAPI와 실제 라우팅 정책이 일치한다.
  - 생성 타입이 최신 스펙 기준으로 갱신된다.

## 테스트 공백

현재 테스트는 "기능 정상 동작"은 잘 덮고 있지만, 인증 경계의 운영 리스크는 거의 덮지 못한다.

- refresh replay / logout 이후 refresh 거부 테스트가 없다
- 이메일 정규화 정책 테스트가 없다
- rate limit 테스트가 없다
- 프론트 API client의 동시 401 / refresh 경쟁 테스트가 없다
- 스펙 정합성(`openapi.yml` vs 실제 라우터) 검증이 없다

## 리팩토링 순서 제안

### 1단계: 보안 경계 먼저

1. `JWT_SECRET` 강제 설정
2. refresh token revocation/rotation 도입
3. auth rate limit 추가

### 2단계: 계정 정합성

1. 이메일 정규화 정책 확정
2. DB unique 정책과 로그인 정책 일치
3. 관련 테스트 추가

### 3단계: 프론트 안정화 및 정리

1. refresh single-flight 도입
2. lint 복구
3. OpenAPI 정합성 복구

## 결론

Phase 4는 "JWT + cookie 기반 인증이 동작한다"는 목표는 달성했다. 하지만 현재 상태를 인증 기능 완성으로 보기는 어렵고, 정확히는 "보안 하드닝 전 단계의 작동하는 POC"에 가깝다.

가장 먼저 손봐야 할 것은 두 가지다.

1. 운영에서 안전하지 않은 `JWT_SECRET` fallback 제거
2. refresh token을 서버가 실제로 통제할 수 있도록 세션 수명 모델을 다시 설계하는 것

이 두 가지를 먼저 정리한 뒤에 rate limit, 이메일 정규화, 프론트 refresh 안정화를 붙이는 순서가 가장 효율적이다.

---

이 문서는 Codex가 작성했습니다.
