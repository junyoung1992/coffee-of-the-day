# Issue #1 — 로컬/운영 환경 분리 및 Fly.io 배포 파이프라인

## 목표

- 로컬 개발 환경은 기존과 동일하게 유지한다
- Go 바이너리 하나로 API + 프론트엔드 정적 파일을 서빙한다
- `main` 브랜치 푸시 시 CI → 자동 배포가 완료된다
- SQLite 데이터가 Fly.io 볼륨에 영속되고, Litestream으로 오브젝트 스토리지에 복제된다

---

## 아키텍처

```
로컬 개발
  브라우저 → http://localhost:5173         # Vite dev server (React)
           → http://localhost:5173/api/v1  # Vite proxy → localhost:8080

운영 (Fly.io)
  브라우저 → https://coffee.fly.dev/        # React 앱 (Go embed)
           → https://coffee.fly.dev/api/v1  # API (동일 Go 서버)
```

단일 origin 구조이므로 운영 환경에서는 CORS 설정이 불필요하다.
기존 `CORSMiddleware`는 로컬 개발(`localhost:5173 ↔ :8080` cross-origin)을 위해 유지한다.

---

## 구현 계획

### Phase 1 — 백엔드 안정성

운영 환경에서 필요한 안정성 기반을 먼저 확보한다.

#### 1-1. SQLite WAL 모드

- `main.go`의 DB 연결 문자열에 `_journal_mode=WAL` 추가
- WAL 모드는 읽기/쓰기 동시성을 높이고, Litestream이 WAL 파일을 tail하는 방식으로 복제하므로 Litestream과 반드시 함께 사용해야 한다

```go
// 변경 전
cfg.DBPath + "?_foreign_keys=on"

// 변경 후
cfg.DBPath + "?_foreign_keys=on&_journal_mode=WAL"
```

#### 1-2. Graceful Shutdown

- `main.go`에 `os/signal`로 SIGTERM/SIGINT를 수신해 진행 중인 요청을 완료하고 종료
- `http.Server`를 명시적으로 생성하고 `Shutdown(ctx)` 호출
- 컨테이너 환경에서 SIGTERM은 `docker stop` 및 Fly.io 배포 교체 시 발생한다

### Phase 2 — 프론트엔드 embed

Go 바이너리에 React 빌드 결과물을 포함하는 구조로 전환한다.

#### 2-1. 환경 변수 분리

`frontend/.env` 하나로 모든 환경의 기본값을 관리한다:

```
VITE_API_BASE_URL=/api/v1
```

`.env.local`은 `frontend/.gitignore`의 `*.local` 패턴으로 git에서 무시되므로 커밋할 수 없다.
Vite 프록시 덕분에 로컬에서도 상대경로가 동작해 환경별 파일이 불필요하다.

`frontend/src/api/client.ts` BASE_URL 처리:
```ts
// VITE_API_BASE_URL이 빈 문자열이면 상대경로 사용 (?? 대신 ||)
const BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'
```

#### 2-2. Vite 프록시 설정 (로컬 개발)

로컬에서 Vite dev server(`localhost:5173`)가 `/api`를 백엔드(`localhost:8080`)로 프록시하도록 설정한다.
이렇게 하면 개발 시에도 브라우저 입장에서는 동일 origin처럼 동작한다.

```ts
// vite.config.ts
server: {
  proxy: {
    '/api': 'http://localhost:8080',
  },
},
```

#### 2-3. Go embed 적용

`//go:embed`는 선언 파일 기준 상대경로만 허용하고 `..`가 불가하다.
`frontend/dist`를 embed하려면 선언 파일이 `frontend/`와 같은 레벨에 있어야 한다.

해결책: Vite `outDir`을 `../web/static`으로 변경해 빌드 결과물을 `web/static/`에 출력하고,
루트의 `web` 패키지에서 embed한다:

```go
// web/fs.go
//go:embed all:static
var staticFS embed.FS
```

`web/static/.gitkeep`을 커밋해 프론트엔드 빌드 전에도 컴파일이 가능하도록 한다.

#### 2-4. SPA Fallback 라우터

`/api/v1` 이외의 모든 경로를 `index.html`로 fallback한다.
React Router가 클라이언트 사이드 라우팅을 처리한다.

```go
// web/fs.go — Handler()가 파일 존재 여부를 확인해 분기
r.Handle("/*", web.Handler())
```

`web.Handler()`는 `fs.Sub(staticFS, "static")`으로 서브 FS를 만든 뒤,
파일이 존재하면 그대로 서빙하고 없으면 `index.html`로 fallback한다.

#### 2-5. 마이그레이션 소스 변경

현재 `file://db/migrations` 경로는 실행 위치에 의존한다.
embed와 컨테이너 환경을 고려해 `iofs` 드라이버로 교체한다.

embed 선언은 `backend/db/` 레벨에서만 `migrations/` 서브디렉토리에 접근 가능하므로
별도 파일로 분리한다:

```go
// backend/db/embed.go
//go:embed all:migrations
var MigrationsFS embed.FS
```

```go
// backend/cmd/server/main.go
src, err := iofs.New(coffeedb.MigrationsFS, "migrations")
m, err := migrate.NewWithInstance("iofs", src, "sqlite3", driver)
```

`iofs`는 `golang-migrate/v4`에 포함되어 별도 의존성 추가가 불필요하다.

### Phase 3 — 컨테이너화

#### 3-1. 멀티스테이지 Dockerfile

```
Stage 1 (node): 프론트엔드 빌드 → web/static/ (Vite outDir)
Stage 2 (go-builder): Stage 1의 web/static/ COPY 후 go build (embed 포함)
Stage 3 (runtime): 최소 이미지 (debian:bookworm-slim)
  + Litestream 바이너리 포함
```

Stage 1 빌드 결과물(`web/static/`)을 Stage 2에 COPY하면 `go build` 시 embed가 자동으로 포함된다.

Litestream은 Go 바이너리의 래퍼로 실행된다 (`litestream replicate -exec "./server"`).

#### 3-2. docker-compose.yml

로컬 컨테이너 환경 검증용. 프론트엔드 embed 포함 전체 스택을 단일 명령으로 실행한다.

```yaml
services:
  app:
    build: .
    ports: ["8080:8080"]
    environment:
      - GO_ENV=production
      - JWT_SECRET=...
    volumes:
      - ./data:/data
```

### Phase 4 — Fly.io 배포

#### 4-1. fly.toml

- 앱 이름, 리전 설정
- SQLite용 영구 볼륨 마운트 (`/data`)
- health check (`GET /health`)
- 환경변수: `GO_ENV=production`, `DB_PATH=/data/coffee.db`

#### 4-2. Fly.io Secrets

```
fly secrets set JWT_SECRET=<값>
fly secrets set LITESTREAM_...=<값>
```

#### 4-3. 초기 수동 배포

`fly deploy`로 동작 확인 후 CI/CD를 연결한다.

### Phase 5 — GitHub Actions CI/CD

#### 5-1. PR 테스트 워크플로우 (`.github/workflows/ci.yml`)

트리거: PR 오픈/업데이트

```
jobs:
  backend-test: go test ./...
  frontend-test: npm test (unit), npm run test:e2e (E2E)
```

#### 5-2. 자동 배포 워크플로우 (`.github/workflows/deploy.yml`)

트리거: `main` 브랜치 푸시

```
jobs:
  deploy:
    - CI 통과 확인 (needs: test)
    - fly deploy --remote-only
```

`FLY_API_TOKEN`은 GitHub Secrets에 등록한다.

---

## 결정 사항

| 항목 | 결정 | 이유 |
|------|------|------|
| 프론트엔드 서빙 | Go embed | 단일 바이너리 배포, 동일 origin으로 CORS/쿠키 단순화 |
| Go 모듈 위치 | 프로젝트 루트로 이동 | `//go:embed`가 `..` 경로를 허용하지 않아 `frontend/dist`를 embed하려면 모듈 루트가 그 위에 있어야 함 |
| embed 진입점 | 루트 `web/fs.go` 패키지 | `//go:embed`는 선언 파일 기준 상대경로만 허용. `backend/cmd/server/`에서 `frontend/dist`에 접근 불가 → 루트에 얇은 `web` 패키지를 두고 `main.go`에서 import |
| FE/BE 결합 수준 | 배포 방식만 결합, 아키텍처는 분리 유지 | embed는 서빙 방식이지 API 계약이 아님. 분리 시 `web/fs.go` 삭제 + SPA 핸들러 제거 + CORS 재활성화로 충분. 디렉터리 구조(`backend/`, `frontend/`)는 그대로 유지 |
| import path | `coffee-of-the-day/backend/...` 유지 | 모듈명을 `coffee-of-the-day`로 변경해도 코드가 `backend/` 하위에 있으면 경로가 동일하게 유지됨 → 내부 import 수정 불필요 |
| SQLite 백업 | Litestream | WAL 기반 실시간 복제, Fly.io 볼륨 장애 대비 |
| Litestream 실행 방식 | `-exec` 래퍼 | 앱 프로세스 수명을 Litestream이 관리, PID 1 문제 회피 |
| 런타임 이미지 | debian:bookworm-slim | Litestream 바이너리 실행에 libc 필요 (distroless 불가) |
| 마이그레이션 소스 | embed `iofs` | 컨테이너 내 파일 경로 의존성 제거 |

---

## 완료 기준

- [ ] `main` 푸시 시 CI 통과 후 Fly.io 자동 배포
- [ ] 배포 URL에서 로그인 및 전체 기능 정상 동작
- [ ] 로컬 개발 환경 기존과 동일 (`npm run dev` + `go run`)
- [ ] 배포 간 SQLite 데이터 유지, 오브젝트 스토리지 복제 확인
- [ ] SIGTERM 수신 시 진행 중 요청 완료 후 종료
