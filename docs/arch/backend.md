# Backend 아키텍처 결정 문서

> 이 코드베이스에서 작업할 때 알아야 하는 비자명한 규칙과 제약을 설명합니다.

---

## Layered Architecture

handler → service → repository 3계층 구조.

- **handler**: HTTP를 알고, 비즈니스 로직을 모른다 (요청 파싱, 응답 직렬화, 상태 코드 결정)
- **service**: 비즈니스 규칙과 입력 정규화를 알고, HTTP를 모른다
- **repository**: SQL과 DB 트랜잭션을 알고, 비즈니스 규칙을 모른다

새 기능을 추가할 때는 기존 레이어를 건드리지 않고 vertical slice로 확장한다 (예: `SuggestionHandler/Service/Repository`, `PresetHandler/Service/Repository`).

**트랜잭션 소유권**: repository가 aggregate 단위로 tx를 연다. `coffee_logs` + 서브 테이블, `presets` + 서브 테이블 삽입을 각각 하나의 repository 메서드에서 처리. 단, 하나의 유스케이스가 여러 repository를 원자적으로 묶어야 하는 상황이 생기면 service layer로 tx 경계를 올린다.

---

## SQLite

### WAL 모드 + 외래키

DSN에서 WAL 모드와 외래키를 함께 활성화한다. 별도로 `PRAGMA`를 실행하면 connection pool에서 해당 연결 하나에만 적용되므로 DSN 파라미터 방식을 사용한다.

→ `config/config.go`

### 배열 저장

SQLite에 배열 타입이 없으므로 `companions`, `tasting_tags`, `brew_steps`를 JSON 텍스트로 저장한다 (`'["지수","민준"]'`). Go 레이어에서 `[]string` ↔ `string` 직렬화를 처리한다.

---

## sqlc 우선 + raw SQL 보완

- 기본 CRUD, 단순 조회: sqlc 사용 (`db/queries/`)
- sqlc가 정적 분석하기 어려운 쿼리 (`json_each` 가상 테이블, 특수 집계): `database/sql`로 직접 실행

→ sqlc 쿼리: `db/queries/*.sql`, raw SQL 예시: `internal/repository/suggestion_repository.go`

---

## 마이그레이션

golang-migrate + `//go:embed`로 마이그레이션 파일을 바이너리에 포함한다. 컨테이너 환경에서 실행 위치에 의존하지 않기 위함.

→ `db/embed.go`, `db/migrations/`

---

## 인증: JWT + httpOnly 쿠키

### 토큰 구조

- **Access token** (15분): 짧은 수명, 탈취 시 피해 최소화
- **Refresh token** (7일): httpOnly 쿠키 전용, 자동 갱신 UX 유지
- 두 토큰 모두 동일 비밀키로 서명 → `token_type` 클레임으로 혼용 방지

### Revocation

`users.token_version` 컬럼으로 세대 관리. 로그아웃 시 `token_version`을 증가시켜 이전 리프레시 토큰을 전부 무효화한다.

### 제약

- `GO_ENV=production`일 때 `JWT_SECRET`이 없거나 32바이트 미만이면 서버 시작을 차단한다 (fail-fast).
- `/auth/*` 라우트에 IP 기준 rate limit 적용 (1분 20회, `go-chi/httprate`).
- 이메일 정규화 (`ToLower + TrimSpace`)는 서비스 진입 시점에 적용한다.

→ `internal/handler/auth_handler.go`, `internal/handler/middleware.go`, `internal/service/auth_service.go`

---

## 배포: 단일 바이너리 + Fly.io

### 프론트엔드 embed

`//go:embed`로 React 빌드 결과물을 바이너리에 포함. 운영 환경에서는 단일 origin으로 API + SPA를 함께 서빙하므로 CORS 불필요. `web.Handler()`는 파일이 있으면 정적 서빙, 없으면 `index.html` fallback (SPA 라우팅).

`//go:embed`는 선언 파일 기준 상대 경로만 허용하고 `..` 불가. 그래서 `web/` 패키지가 프로젝트 루트에 위치한다.

→ `web/fs.go`, `web/static/`

### Fly.io 인프라

| 항목 | 설정 |
|------|------|
| 리전 | `nrt` (도쿄) |
| 머신 | shared-cpu-1x, 256MB |
| 볼륨 | 1GB (`coffee_data` → `/data`) |
| DB 경로 | `/data/coffee.db` |
| Scale-to-zero | `auto_stop_machines = 'stop'` |
| Health check | `GET /health` (30초 간격) |

### Graceful Shutdown

SIGTERM/SIGINT 수신 시 진행 중인 요청을 완료한 후 서버 종료 (타임아웃 30초).

→ `cmd/server/main.go`

### CI/CD

- **CI** (`.github/workflows/ci.yml`): PR 시 `go test ./...` + `npm test` 실행
- **Deploy** (`.github/workflows/deploy.yml`): `main` 푸시 → CI 통과 → `fly deploy --remote-only`

---

*Last updated: 2026-04-04*
