# Fly.io 배포 가이드

## 개요

Coffee of the Day를 Fly.io에 배포하는 전체 과정을 다룬다.
Go 바이너리 하나가 API + React SPA를 서빙하는 단일 앱 구조이며, SQLite 데이터는 Fly.io 볼륨에 영속된다.

### 배포 아키텍처

```
인터넷 → Fly.io Anycast Edge → Fly Proxy → Machine (Go 서버)
                                              ├── API        (/api/v1/*)
                                              ├── SPA        (/* → index.html)
                                              └── Health     (/health)
                                              │
                                         Volume (/data)
                                              └── coffee.db  (SQLite + WAL)
```

- **리전**: `nrt` (도쿄, NRT — 한국에서 가장 가까운 Fly.io 리전)
- **머신**: shared-cpu-1x, 256MB RAM
- **볼륨**: 1GB (`coffee_data` → `/data`)
- **URL**: https://coffee-of-the-day.fly.dev

---

## 사전 조건

| 항목 | 설명 |
|------|------|
| Fly.io 계정 | https://fly.io 가입, 결제 수단 등록 필요 (무료 허용량 사용을 위해) |
| Fly CLI (`flyctl`) | `brew install flyctl` 또는 [공식 설치 가이드](https://fly.io/docs/flyctl/install/) |
| Docker | 로컬에서 이미지 빌드 확인 시 필요 (Fly.io 리모트 빌드 사용 시 선택사항) |

---

## 핵심 설정 파일 설명

### fly.toml

Fly.io 앱의 전체 구성을 정의하는 파일이다. 프로젝트 루트에 위치한다.

```toml
app = 'coffee-of-the-day'       # Fly.io 앱 이름 (고유해야 함)
primary_region = 'nrt'           # 머신이 생성될 기본 리전

[build]                          # 빌드 설정 (Dockerfile 자동 감지)

[env]                            # 환경변수 (평문, 비밀 아닌 값만)
  DB_PATH = '/data/coffee.db'    # SQLite DB 파일 경로 (볼륨 내부여야 영속됨)
  GO_ENV = 'production'          # 운영 모드 플래그

[processes]                      # 프로세스 그룹 정의
  app = './server'               # 'app' 그룹: Go 바이너리 직접 실행
                                 # Litestream 활성화 시 Dockerfile CMD로 전환

[[mounts]]                       # 영구 볼륨 마운트
  source = 'coffee_data'         # 볼륨 이름 (fly volumes create 시 지정한 이름과 일치해야 함)
  destination = '/data'          # 컨테이너 내 마운트 경로

[http_service]                   # HTTP 서비스 설정
  internal_port = 8080           # Go 서버가 리슨하는 포트 (서버와 반드시 일치)
  force_https = true             # HTTP → HTTPS 리다이렉트
  auto_stop_machines = 'stop'    # 트래픽 없으면 머신 정지 (비용 절감 핵심)
  auto_start_machines = true     # 요청 들어오면 자동 시작
  min_machines_running = 0       # 전부 정지 허용 (항시 가동 불필요)
  processes = ['app']            # 이 서비스가 연결된 프로세스 그룹

[[http_service.checks]]          # 헬스체크 설정
  interval = '30s'               # 30초마다 체크
  timeout = '5s'                 # 5초 내 응답 없으면 실패
  grace_period = '10s'           # 시작 후 10초간 실패 무시 (콜드스타트 허용)
  method = 'GET'
  path = '/health'

[[vm]]                           # 머신 스펙
  memory = '256mb'
  cpu_kind = 'shared'            # 공유 CPU (전용 CPU 대비 저렴)
  cpus = 1
```

### Dockerfile (관련 부분)

멀티스테이지 빌드로 프론트엔드 빌드 → Go 바이너리 빌드 → 런타임 이미지를 생성한다.
`fly deploy` 실행 시 Fly.io의 리모트 빌더가 이 Dockerfile을 사용해 이미지를 빌드한다.

### litestream.yml

Litestream WAL 복제 설정이다. 현재는 Litestream 없이 배포 중이므로 비활성 상태.
활성화하려면 `fly.toml`의 `[processes]` 섹션을 제거하면 Dockerfile의 기본 CMD(`litestream replicate -exec ./server`)가 적용된다.

---

## 배포 절차

### 1. CLI 인증

```bash
fly auth login
```

브라우저가 열리며 Fly.io 계정으로 로그인한다.

### 2. 앱 생성

```bash
fly launch --no-deploy
```

- `fly.toml`이 이미 있으므로 기존 설정을 사용할지 묻는다 → 기존 설정 유지
- `--no-deploy`: 앱 리소스만 생성하고 실제 배포는 하지 않는다
- "Do you want to tweak these settings?" → **N** (이미 `fly.toml`에 원하는 설정이 있으므로)
- GitHub Actions 워크플로우 덮어쓰기 → **N** (Phase 5에서 직접 작성)

### 3. 볼륨 생성

```bash
fly volumes create coffee_data --region nrt --size 1
```

- `coffee_data`: `fly.toml`의 `[[mounts]].source`와 반드시 일치해야 한다
- `--size 1`: 1GB (SQLite 커피 저널에 충분)
- HA 경고가 뜨지만, 단일 사용자 앱이므로 1개로 충분하다

### 4. Secrets 등록

```bash
fly secrets set JWT_SECRET=$(openssl rand -base64 48)
```

- JWT 서명에 사용되는 비밀키를 랜덤 생성 후 등록한다
- Fly.io가 암호화해서 저장하며, 머신 실행 시 환경변수로 주입된다
- 값은 등록 후 다시 조회할 수 없다 (필요 시 재생성)
- `fly secrets list`로 키 목록만 확인 가능

### 5. 배포

```bash
fly deploy
```

- Fly.io 리모트 빌더가 Dockerfile을 빌드하고 머신에 배포한다
- 빌드 로그가 터미널에 실시간 출력된다
- 헬스체크 통과 후 배포 완료

### 6. 검증

```bash
fly status                                          # 머신 상태 확인
fly logs                                            # 실시간 로그
curl https://coffee-of-the-day.fly.dev/health       # 헬스체크
```

**검증 체크리스트:**

- [ ] `/health` → 200 OK
- [ ] 브라우저에서 SPA 정상 로드
- [ ] 로그인 및 전체 기능 동작
- [ ] 재배포 후 SQLite 데이터 유지

---

## 비용 구조

### 예상 월 비용 (단일 사용자, scale-to-zero)

| 항목 | 단가 | 예상 비용 | 비고 |
|------|------|-----------|------|
| Compute (shared-cpu-1x, 256MB) | ~$1.94/월 (24시간 가동 시) | **~$0** | `auto_stop_machines`로 미사용 시 정지, 정지 중 과금 없음 |
| Volume (1GB) | $0.15/GB/월 | **$0.15** | 머신 정지 중에도 과금됨 (스토리지이므로) |
| Bandwidth (아웃바운드) | $0.02/GB | **~$0** | 월 100GB 무료 허용량, 개인 앱으로 초과 가능성 없음 |
| **합계** | | **~$0.15/월** | 무료 허용량 범위 내에서 운영 가능 |

### Fly.io 무료 허용량 (결제 수단 등록 필요)

- shared-cpu-1x 256MB 머신 3대 (24시간 가동 기준)
- 영구 볼륨 3GB
- 아웃바운드 대역폭 100GB/월

### 비용 절감 핵심 설정

```toml
auto_stop_machines = 'stop'    # 트래픽 없으면 머신 정지 → 과금 중단
auto_start_machines = true     # 요청 시 자동 시작 (콜드스타트 1~3초)
min_machines_running = 0       # 모든 머신 정지 허용
```

이 설정 조합으로 사용하지 않는 시간에는 compute 비용이 0이 된다.
트레이드오프로 첫 요청 시 1~3초의 콜드스타트 지연이 발생한다 (Go 바이너리는 빠르므로 체감이 크지 않다).

---

## 보안 및 비용 위험 대응

### 위험 1: DDoS / 대량 트래픽 공격

**위험도: 중간**

Fly.io는 L3/L4 수준의 기본적인 DDoS 방어를 제공하지만, 애플리케이션 레벨(L7) 공격에 대한 보호는 제한적이다.
공격 트래픽으로 인한 아웃바운드 대역폭과 compute 비용이 청구될 수 있다.

**Fly.io에 하드 지출 상한(hard spending cap)이 없다**는 점이 핵심 우려사항이다.

**대응 방안:**

| 우선순위 | 방안 | 설명 |
|----------|------|------|
| 1 | **Fly.io 지출 알림 설정** | Dashboard > Organization > Billing에서 알림 금액 설정 (예: $5, $10). 하드캡은 아니지만 이상 징후를 조기에 감지할 수 있다. |
| 2 | **Cloudflare 프록시 (무료)** | Cloudflare 무료 플랜을 프론트에 두면 L7 DDoS 방어, 봇 차단, Under Attack 모드를 사용할 수 있다. **비용 보호에 가장 효과적인 단일 조치**이다. |
| 3 | **애플리케이션 레벨 Rate Limiting** | Go chi 미들웨어로 IP당 요청 수를 제한한다 (예: `go-chi/httprate`). 서버까지 도달한 요청에 대한 방어선이다. |
| 4 | **머신 수 제한** | `fly scale count 1`로 머신이 자동 확장되지 않도록 고정한다. |

### 위험 2: 볼륨 데이터 손실

**위험도: 낮음 (현재 Litestream 미활성)**

Fly.io 볼륨은 단일 물리 호스트에 고정된다. 해당 호스트에 장애가 발생하면 데이터를 잃을 수 있다.

**대응 방안:**

- Litestream 활성화로 오브젝트 스토리지(S3 호환)에 실시간 복제 (Phase 4 이후 별도 진행)
- 주기적으로 `fly ssh console`로 접속해 DB 파일 존재 여부 확인

### 위험 3: Secret 노출

**위험도: 낮음**

**현재 보호 상태:**

- `JWT_SECRET`은 `fly secrets`로 암호화 저장되며 코드에 포함되지 않음
- `fly.toml`에는 비밀 값이 없음 (커밋해도 안전)
- 환경변수는 머신 실행 시에만 복호화되어 주입됨

**주의사항:**

- `fly secrets list`로 키 목록은 보이지만 값은 조회 불가
- 시크릿 변경이 필요하면 `fly secrets set`으로 덮어쓰기
- GitHub Actions에서 사용할 `FLY_API_TOKEN`도 GitHub Secrets에만 저장할 것

### 위험 4: 예상치 못한 머신 가동

**위험도: 낮음**

`auto_start_machines = true`이므로 봇 크롤러나 헬스체크 모니터링 서비스가 주기적으로 요청을 보내면 머신이 계속 깨어날 수 있다.

**대응 방안:**

- `robots.txt`를 서빙해 검색 엔진 크롤러 차단
- Fly.io 대시보드에서 머신 가동 시간 모니터링
- 비정상적으로 긴 가동 시간 발견 시 `fly logs`로 원인 파악

---

## 유용한 CLI 명령어

| 명령어 | 설명 |
|--------|------|
| `fly status` | 앱, 머신, 볼륨 상태 확인 |
| `fly logs` | 실시간 로그 스트리밍 |
| `fly deploy` | 빌드 및 배포 |
| `fly secrets list` | 등록된 시크릿 키 목록 |
| `fly secrets set KEY=VALUE` | 시크릿 등록/변경 |
| `fly ssh console` | 머신에 SSH 접속 |
| `fly volumes list` | 볼륨 목록 및 상태 |
| `fly scale show` | 현재 머신 스펙 확인 |
| `fly machine list` | 머신 목록 및 상태 |
| `fly apps destroy coffee-of-the-day` | 앱 완전 삭제 (모든 리소스 제거) |

---

## 트러블슈팅

### 머신이 시작되지 않을 때

```bash
fly logs                    # 에러 로그 확인
fly machine list            # 머신 상태 확인 (created, started, stopped, failed)
```

### 흔한 원인

| 증상 | 원인 | 해결 |
|------|------|------|
| `connection refused` | 서버가 `localhost`에만 바인딩 | Go 서버가 `:8080` (= `0.0.0.0:8080`)에 리슨하는지 확인 |
| 헬스체크 실패 | `internal_port` 불일치 | `fly.toml`의 `internal_port`와 서버 포트가 같은지 확인 |
| DB 초기화됨 | 볼륨 미마운트 | `fly volumes list`로 볼륨 존재 확인, `fly.toml`의 `source` 이름 일치 확인 |
| `JWT_SECRET` 에러 | 시크릿 미등록 | `fly secrets list`로 확인 후 `fly secrets set`으로 등록 |
| 빌드 실패 | Docker 이미지 빌드 에러 | 로컬에서 `docker build .`로 먼저 확인 |

### 앱 완전 초기화가 필요할 때

```bash
fly apps destroy coffee-of-the-day    # 앱 + 머신 + 볼륨 모두 삭제
fly launch --no-deploy                 # 다시 생성
fly volumes create coffee_data --region nrt --size 1
fly secrets set JWT_SECRET=$(openssl rand -base64 48)
fly deploy
```
