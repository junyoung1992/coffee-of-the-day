# Tasks — Issue #1 Fly.io 배포 파이프라인

> 구현 순서대로 나열. 각 Phase는 독립적으로 완료 가능하다.

---

## Phase 1 — 백엔드 안정성

- [x] **WAL 모드**: `main.go` DB 연결 문자열에 `_journal_mode=WAL` 추가
- [x] **Graceful Shutdown**: `main.go`에 SIGTERM/SIGINT 핸들링 추가
  - `http.Server` 명시적 생성으로 전환
  - signal 채널 수신 후 `server.Shutdown(ctx)` 호출 (타임아웃 30초)
  - `db.Close()` graceful 종료 포함

---

## Phase 2 — 프론트엔드 embed

- [x] **Go 모듈 루트 이동**
  - `backend/go.mod`, `backend/go.sum` → 프로젝트 루트로 이동
  - `module coffee-of-the-day/backend` → `module coffee-of-the-day` 로 변경
  - 내부 import path는 변경 불필요 (`coffee-of-the-day/backend/...` 형태 유지)
- [x] **환경 변수 정리**
  - `frontend/.env` 생성 — `VITE_API_BASE_URL=/api/v1` (모든 환경 기본값, 커밋됨)
  - `.env.local`은 `frontend/.gitignore`의 `*.local` 패턴으로 무시되므로 커밋하지 않음
  - `frontend/src/api/client.ts` BASE_URL fallback을 `/api/v1`로 수정 (`??` → `||`)
- [x] **Vite 프록시 설정**: `vite.config.ts`에 `/api` → `http://localhost:8080` 프록시 추가
- [x] **마이그레이션 소스 `iofs` 전환**
  - `backend/db/embed.go` 생성 (`//go:embed all:migrations`, package db)
  - `runMigrations`를 `iofs` 소스로 교체
  - `golang-migrate` iofs는 v4.19.1에 포함 (별도 의존성 불필요)
- [x] **`web` 패키지 생성** — embed 진입점
  - `web/fs.go` 생성 (`//go:embed all:static`)
  - Vite outDir을 `../web/static`으로 설정해 빌드 결과물이 embed 대상 경로에 출력
  - `web/static/.gitkeep` 추가 (빌드 전 컴파일 오류 방지)
  - `.gitignore` 생성: `web/static/*`, `!web/static/.gitkeep`, `frontend/dist/`
- [x] **SPA 핸들러 등록**
  - `backend/cmd/server/main.go`에서 `coffee-of-the-day/web` import
  - `/api/v1` 이외 경로 SPA fallback (`index.html`) 처리
- [x] **로컬 동작 확인**: `npm run build` 후 `go build ./...` 및 `go test` 전체 통과

---

## Phase 3 — 컨테이너화

- [x] **Dockerfile 작성** (멀티스테이지)
  - Stage 1 (`node`): `frontend/` 빌드 → `web/static/`
  - Stage 2 (`golang`): Stage 1 `web/static/` COPY 후 `go build` (embed 포함)
  - Stage 3 (`debian:bookworm-slim`): 바이너리 + Litestream 복사, 실행
- [x] **Litestream 설정**
  - `litestream.yml` 작성 (복제 대상 DB 경로, 오브젝트 스토리지 설정)
  - Dockerfile에서 Litestream `-exec` 래퍼로 앱 실행
- [x] **docker-compose.yml 작성** (로컬 검증용, `command: ["./server"]`로 Litestream 없이 실행)
- [x] **컨테이너 로컬 동작 확인**: `docker compose up` 후 전체 기능 확인
  - GET `/` → 200 (React SPA), GET `/health` → 200, GET `/api/v1/auth/me` → 401, OPTIONS → 204

---

## Phase 4 — Fly.io 배포

- [x] **fly.toml 작성**
  - 앱 이름, 리전(`nrt`) 설정
  - SQLite 영구 볼륨 마운트 (`/data`)
  - health check (`GET /health`)
  - 환경변수: `GO_ENV=production`, `DB_PATH=/data/coffee.db`
- [x] **Fly.io 앱 생성 및 볼륨 생성**: `fly launch` / `fly volumes create`
- [x] **Fly.io Secrets 등록**: `JWT_SECRET` (Litestream은 첫 배포에서 제외, 별도 활성화 예정)
- [x] **초기 수동 배포 및 동작 확인**: `fly deploy` 후 `/health` 200 OK 및 브라우저 접속 확인

---

## Phase 5 — GitHub Actions CI/CD

- [ ] **CI 워크플로우** (`.github/workflows/ci.yml`)
  - 트리거: PR
  - `backend-test`: `go test ./...`
  - `frontend-test`: `npm test` (unit) + `npm run test:e2e` (E2E)
    - E2E 실행 전 `npx playwright install --with-deps chromium` 단계 포함
- [ ] **배포 워크플로우** (`.github/workflows/deploy.yml`)
  - 트리거: `main` 브랜치 푸시
  - CI 통과 후 `fly deploy --remote-only`
- [ ] **GitHub Secrets 등록**: `FLY_API_TOKEN`
- [ ] **자동 배포 확인**: `main` 푸시 → CI → 배포 전 과정 확인
