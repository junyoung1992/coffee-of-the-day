# Postmortem — Docker 컨테이너 GET `/` 405 오류

**일시**: 2026-03-30
**작업**: Issue #1 Phase 3 — 컨테이너 로컬 동작 확인
**심각도**: 낮음 (운영 영향 없음, 로컬 개발 환경)

---

## 요약

`docker compose up` 후 `curl http://localhost:8080/`이 `405 Method Not Allowed`를 반환했다.
Colima VM 내부에서 동일하게 요청했을 때는 `200 OK`가 반환됐다.
원인은 이전 테스트에서 종료되지 않은 **로컬 Go 서버 프로세스가 8080 포트를 선점**하고 있었기 때문이었다.
macOS의 curl은 Docker 컨테이너가 아닌 이 프로세스에 도달하고 있었다.

---

## 타임라인

| 시각 | 상황 |
|------|------|
| 14:43 | `docker compose build` 및 `docker compose up -d` 실행 |
| 14:44 | `curl http://localhost:8080/` → `405 Method Not Allowed` (`Allow: OPTIONS`) |
| 14:44 | `r.Options("/*", ...)` + `r.Handle("/*", ...)` 라우팅 충돌로 추정 → 코드 수정 반복 |
| 14:53 | 로컬 포트 9090으로 서버를 직접 실행해 `200 OK` 확인 → 코드 자체는 정상 |
| 14:54 | `docker compose build --no-cache` 후에도 동일하게 405 |
| 15:03 | Colima VM 내부(`colima ssh`)에서 직접 요청 → `200 OK` 확인 |
| 15:05 | `lsof -i :8080` 으로 로컬 프로세스(PID 27487) 발견 |
| 15:05 | `kill 27487` 후 `docker compose restart` → 모든 엔드포인트 정상 |

---

## 근본 원인

디버깅 과정 중 실행한 아래 명령이 문제의 시작이었다:

```bash
/tmp/coffee-server &
SERVER_PID=$!
...
kill $SERVER_PID
```

백그라운드로 실행한 서버를 `kill $SERVER_PID`로 종료하려 했으나, 서버 시작 직후 `:8080 address already in use` 에러로 즉시 종료됐기 때문에 `$SERVER_PID`가 실제 서버 프로세스의 PID와 달랐다. 그 결과 kill이 엉뚱한 프로세스(또는 아무것도 없는 PID)를 대상으로 했고, 실제 서버 프로세스는 살아남았다.

- **macOS의 curl** → PID 27487 로컬 프로세스 → 405 (해당 프로세스는 구버전 코드 실행 중)
- **Colima VM의 curl** → Docker 컨테이너 → 200 (신버전 코드, 정상)

Docker 포트 포워딩(`0.0.0.0:8080 → container:8080`)은 macOS에서 소켓을 통해 동작한다.
로컬 프로세스가 먼저 8080 소켓을 점유하면 Docker 포워딩보다 우선한다.

---

## 혼선이 생긴 이유

1. **증상이 코드 버그처럼 보였다**: `Allow: OPTIONS` 헤더는 chi 라우터가 등록된 메서드를 반환할 때 사용하는 형식이라, 실제로 chi 라우팅 문제처럼 보였다.

2. **구버전 코드가 동일한 증상을 유발했다**: 기존 코드에는 `r.Options("/*", ...)` + `r.Handle("/*", ...)` 조합이 있어 chi가 GET을 OPTIONS 전용 패턴으로 인식할 수 있었다. 이 잠재적 버그가 실제 장애와 동일한 405를 반환했다.

3. **Docker 재빌드가 아무것도 바꾸지 않았다**: Docker 이미지를 아무리 새로 빌드해도, curl이 로컬 프로세스에 닿는 한 결과는 변하지 않았다.

---

## 조치 사항

### 즉각 조치 (근본 원인 해결)
- `kill 27487`로 8080을 점유한 로컬 프로세스 종료
- `docker compose restart`로 포트 포워딩 재확인

### 부가 개선 (디버깅 중 발견한 실제 코드 문제)

구버전 코드의 `r.Options("/*", ...)` 등록이 불필요했다:

```go
// 제거 전
r.Options("/*", func(w http.ResponseWriter, r *http.Request) {})
r.Handle("/*", web.Handler())

// 제거 후
r.Handle("/*", web.Handler())
```

`CORSMiddleware`가 이미 OPTIONS 요청을 미들웨어 단계에서 차단하고 204를 반환한다.
라우터에 별도로 OPTIONS 핸들러를 등록하면 chi가 해당 패턴을 OPTIONS 전용으로 인식해 같은 패턴의 `r.Handle` 등록에 영향을 줄 수 있다.

---

## 확인된 최종 동작

```
GET /               → 200 (React SPA index.html)
GET /health         → 200
GET /some-spa-route → 200 (SPA fallback)
GET /api/v1/auth/me → 401 (인증 필요)
OPTIONS /           → 204 (CORSMiddleware 처리)
```

---

## 재발 방지

1. **테스트 서버는 포어그라운드로 실행하거나, 종료 확인 후 다음 단계 진행**
   ```bash
   # 백그라운드 서버를 사용할 경우
   PORT=9090 ./server &
   SERVER_PID=$!
   # ... 테스트 ...
   kill $SERVER_PID && wait $SERVER_PID 2>/dev/null  # wait으로 종료 확인
   ```

2. **컨테이너 동작 확인 전 포트 점유 여부 먼저 확인**
   ```bash
   lsof -i :8080
   ```

3. **로컬 vs Docker 동작이 다를 때 VM 내부에서 직접 확인**
   ```bash
   colima ssh -- curl http://localhost:<port>/<path>
   ```
   macOS 네트워킹을 우회해 컨테이너에 직접 닿는다. 결과가 다르면 네트워크/포트 레이어 문제다.
