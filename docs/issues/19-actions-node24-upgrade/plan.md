# Issue #19 — GitHub Actions Node.js 20 런타임 deprecation 대응

## 목표

`.github/workflows/ci.yml`, `.github/workflows/deploy.yml`에서 사용 중인 GitHub-maintained 액션 3종(`actions/checkout`, `actions/setup-node`, `actions/setup-go`)을 Node 24 런타임을 사용하는 최신 메이저 버전으로 업그레이드한다.
작업 종료 시점 기준으로 main 브랜치의 CI/Deploy 워크플로우 실행 로그에서 `Node.js 20 actions are deprecated.` 경고가 사라지고, 모든 기존 step(backend test, frontend unit test, frontend type check, fly deploy)이 회귀 없이 통과해야 한다.

---

## 배경 및 일정 압박

GitHub은 2025-09-19 공지로 Actions runner의 Node 20 런타임 단계적 폐기를 선언했다. 본 이슈가 작성된 2026-04-19 시점에는 강제 전환까지 약 5개월 여유가 있었으나, 작업 시작 시점인 2026-05-10 기준으로는 다음과 같이 마감이 가까워졌다.

- **2026-06-02 (3주 뒤)**: 기본 동작이 Node 24 강제 전환으로 바뀜. 이전 메이저(`@v4` 등)에 묶여 있는 액션은 의도와 다른 런타임에서 실행되며 경고가 에러로 격상될 가능성이 있음.
- **2026-09-16 (~4개월 뒤)**: Node 20 런타임이 runner에서 완전히 제거됨. 이 시점 이후 미대응 액션은 실행 자체가 실패할 수 있음.

따라서 본 이슈는 2026-06-02 이전에 main에 머지되어야 안전하다. 임시 회피책(`FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true`)은 본질적 해법이 아니며, 액션 메이저 버전 자체를 올리는 것이 정공법이다.

---

## 영향 범위 (현재 사용 위치)

`grep`으로 사전 확인한 결과, GitHub-maintained 액션의 사용 위치는 다음과 같다.

| 파일 | 라인 | 액션 | 현재 핀 | 입력 사용 |
|---|---|---|---|---|
| `.github/workflows/ci.yml` | 14 | `actions/checkout` | `@v4` | (없음) |
| `.github/workflows/ci.yml` | 16 | `actions/setup-go` | `@v5` | `go-version-file: go.mod` |
| `.github/workflows/ci.yml` | 30 | `actions/checkout` | `@v4` | (없음) |
| `.github/workflows/ci.yml` | 32 | `actions/setup-node` | `@v4` | `node-version: 22`, `cache: npm`, `cache-dependency-path: frontend/package-lock.json` |
| `.github/workflows/ci.yml` | 51 | `actions/checkout` | `@v4` | (없음) |
| `.github/workflows/ci.yml` | 53 | `actions/setup-node` | `@v4` | `node-version: 22`, `cache: npm`, `cache-dependency-path: frontend/package-lock.json` |
| `.github/workflows/deploy.yml` | 22 | `actions/checkout` | `@v4` | (없음) |

`deploy.yml` 24행의 `superfly/flyctl-actions/setup-flyctl@master`는 GitHub-maintained가 아니므로 본 이슈 범위에서 제외한다(별도 backlog 후보).

`ci.yml`은 `workflow_call` 트리거를 노출하고 `deploy.yml`이 `uses: ./.github/workflows/ci.yml`로 재사용한다(`deploy.yml:14-15`). 즉 PR에서 `ci.yml` 자체가 검증되면 deploy 흐름까지 자연스럽게 같은 액션 버전으로 검증되며, deploy job 자체가 사용하는 `actions/checkout`도 함께 업그레이드해야 deploy 실행 로그에서 경고가 완전히 사라진다.

---

## 설계 결정 1: 업그레이드 목표 버전

각 액션의 GitHub Releases를 조사하여 Node 24 런타임을 채택한 최초 메이저와 최신 메이저를 확인했다.

| 액션 | 현재 | 업그레이드 목표 | Node 24 도입 시점 | 비고 |
|---|---|---|---|---|
| `actions/checkout` | `@v4` | `@v5` | v5.0.0 (2024-08-11) — "Update actions checkout to use node 24" | v6도 존재(v6.0.x)하지만 v6는 자격증명 저장 방식 변경(`persist creds to a separate file`)으로 runner v2.329.0+ 요구. self-hosted runner 사용 시 호환성 점검이 필요. 본 프로젝트는 `ubuntu-latest`(GitHub-hosted)만 사용하므로 v6도 무방하지만, 변화 폭을 최소화하고 입력 schema가 동일한 v5를 채택한다. |
| `actions/setup-node` | `@v4` | `@v5` | v5.0.0 (2025-09-04 무렵, "Upgrade action to use node24") | 현재 사용 중인 입력(`node-version`, `cache`, `cache-dependency-path`)은 v5/v6에서도 그대로 지원. v6에서 자동 캐싱이 npm 한정으로 축소되었으나, 본 프로젝트는 어차피 npm + 명시적 `cache: npm`이므로 v6도 안전. v5 채택. |
| `actions/setup-go` | `@v5` | `@v6` | v6.0.0 ("Upgrade Nodejs runtime from node20 to node 24") | 본 프로젝트는 `go-version-file: go.mod` 사용 — v6에서도 지원 유지. 툴체인 처리 로직이 개선되었으나 `go-version-file`만 사용하는 본 프로젝트의 동작에는 영향 없음. |

### v5 vs v6 (`actions/checkout`) 선택 근거

- v6.0.0 changelog의 핵심 변경은 "credentials를 git config 대신 별도 파일로 분리 저장"이며 runner 최소 버전 v2.329.0을 요구한다. `ubuntu-latest`는 충분히 이 요건을 만족한다.
- 그럼에도 v5를 채택하는 이유: (1) 본 이슈의 목적은 deprecation 경고 해소이므로 Node 24 도입 최초 메이저인 v5로 충분하다. (2) v5는 입력/동작이 v4와 사실상 동일해 회귀 위험이 가장 낮다. (3) 추가 안정화가 필요하면 후속 이슈에서 v6로 올리면 된다.

### v5 vs v6 (`actions/setup-node`) 선택 근거

- v6.0.0의 breaking change는 "자동 캐싱이 npm 한정으로 좁혀짐"이다. 본 프로젝트는 명시적으로 `cache: npm`을 지정하므로 영향이 없으며 v6도 안전하다.
- 그러나 v5와 v6 모두 Node 24를 사용하므로 deprecation 해소 측면에서는 동일하며, 변경 폭을 최소화하기 위해 v5 채택. 추후 자동 캐싱 동작 변화가 필요해지면 v6로 올린다.

### `actions/setup-go` v6 채택

- 본 프로젝트의 입력은 `go-version-file: go.mod` 한 가지이며, v6에서도 그대로 지원된다.
- v6는 Node 24 런타임 + 툴체인 처리 개선이 핵심이고, 입력 schema 호환성이 보장되므로 최신 메이저로 업그레이드해도 회귀 위험이 낮다.

---

## 설계 결정 2: 변경 형태 — 핀 문자열 단순 치환

워크플로우 YAML의 `uses:` 라인의 메이저 버전 핀 문자열만 교체한다. 다음 규칙으로 일관되게 변경한다.

- `actions/checkout@v4` → `actions/checkout@v5`
- `actions/setup-node@v4` → `actions/setup-node@v5`
- `actions/setup-go@v5` → `actions/setup-go@v6`

### 패치 수준 핀(`@v5.0.1` 등) 대신 메이저 핀을 유지하는 이유

- 기존 워크플로우가 메이저 핀(`@v4`, `@v5`) 컨벤션을 따르고 있어 일관성 유지가 자연스럽다.
- GitHub Actions 메이저 태그는 같은 메이저 내 패치/마이너 업데이트를 자동 추적해 보안 패치 수신에 유리하다.
- 본 프로젝트는 단일 사용자 POC이며 supply-chain 공격 모델에 대한 SHA 핀(`@<commit-sha>`)까지 도입할 필요가 낮다(향후 별도 backlog로 검토 가능).

### 입력(`with:` 블록)은 변경하지 않는다

위 표에서 확인한 대로 현재 사용 중인 모든 입력(`node-version`, `cache`, `cache-dependency-path`, `go-version-file`)이 목표 메이저에서 그대로 호환된다. 따라서 `with:` 블록은 손대지 않으며, 기존 들여쓰기/포맷도 그대로 유지한다.

---

## 설계 결정 3: 검증 전략 — PR CI 실행으로 단일 확인

### 로컬 재현은 시도하지 않는다

- `act` 등 도구로 GitHub Actions를 로컬 재현하는 방법이 있으나, runner 이미지 차이로 인해 deprecation 경고 발생/소거 여부를 정확히 검증하기 어렵다.
- 본 변경은 액션 핀 6개를 치환하는 텍스트 수준의 변경이며, 실 환경에서 단 한 번 실행해 검증하는 비용이 가장 낮다.

### 검증 절차

1. **PR 생성**: `feat/19-actions-node24-upgrade` 브랜치에서 PR을 연다. PR 트리거(`pull_request: branches: [main]`)에 의해 `ci.yml`이 자동 실행된다.
2. **CI job 통과 확인**: GitHub Actions UI에서 `backend-test`, `frontend-unit-test`, `frontend-type-check` 세 job이 모두 성공해야 한다.
3. **deprecation 경고 소거 확인**: 각 job의 step별 raw 로그에서 `Node.js 20 actions are deprecated` 문자열이 더 이상 출력되지 않아야 한다. 특히 다음 step의 시작 부분 로그를 확인한다.
   - `backend-test`: `Set up Go` (setup-go@v6)
   - `frontend-unit-test`, `frontend-type-check`: `Set up Node` (setup-node@v5)
   - 모든 job: `Run actions/checkout@v5`
4. **main 머지 후 Deploy 검증**: PR 머지 후 자동 트리거되는 `Deploy` 워크플로우가 ci 단계 + deploy job(`actions/checkout@v5` 사용) 모두 통과하고, deprecation 경고가 deploy 로그에서도 사라졌는지 확인한다. fly.io 배포 자체가 정상 완료되어 사이트가 살아 있어야 한다.

### 실패 시 롤백

- PR 단계에서 어느 job이든 실패하면 즉시 변경을 되돌리고 실패 원인을 분석한다.
- 가장 가능성 높은 실패 케이스는 `actions/setup-go@v6`의 툴체인 동작 변화로 추정된다. 이 경우 `setup-go@v6` → `setup-go@v5`로만 되돌리고(나머지 두 액션은 v5 유지), 별도 후속 이슈로 분리해 처리한다.
- main에 머지된 후 deploy에서만 실패가 발견되는 시나리오는 ci.yml이 `workflow_call`로 deploy.yml에서 재사용되므로 가능성이 낮다. 그래도 발생 시 `Revert "feat: ..."` 커밋으로 즉시 롤백 후 재조사한다.

---

## 수정하지 않는 것

- `superfly/flyctl-actions/setup-flyctl@master` (`.github/workflows/deploy.yml:24`) — GitHub-maintained가 아니며, 별도 deprecation 영향도 다르다. 필요 시 별도 backlog로 분리.
- 워크플로우의 트리거 섹션(`on:`) — `pull_request`, `workflow_call`, `push` 트리거 정의는 본 이슈와 무관.
- job 구성, step 순서, `needs`, `concurrency`, `defaults` 등 모든 워크플로우 토폴로지 — 핀 버전 외 어떤 것도 손대지 않는다.
- 액션 입력(`with:` 블록) — 호환성이 확인되었으므로 그대로 유지.
- 백엔드/프론트엔드 코드, `docs/spec.md`, `docs/openapi.yml`, `docs/arch/*` — 본 이슈는 인프라 메타데이터 변경이며 애플리케이션 동작/스펙에 영향 없음.
- `docs/backlog.md` — 본 이슈는 backlog 항목으로 등재되어 있지 않으며, 신규 후속 작업도 본 이슈 범위 밖이다.

---

## 테스트 전략

본 이슈는 GitHub Actions 워크플로우 메타데이터 변경이며, 자동화된 단위/통합 테스트 추가 대상은 아니다. 다음 두 단계로 충분하다.

1. **YAML 정합성 빠른 점검 (선택, 로컬)**
   - `actionlint`가 설치되어 있다면 `actionlint .github/workflows/ci.yml .github/workflows/deploy.yml` 실행. 미설치 시 생략 가능 (PR CI에서 어차피 잡힌다).

2. **PR 단위 실 환경 검증 (필수)**
   - 위 "설계 결정 3: 검증 전략"의 1~4 절차를 그대로 수행.
   - 핵심 통과 조건:
     - `ci.yml` 모든 job (`backend-test`, `frontend-unit-test`, `frontend-type-check`) 성공.
     - 각 job 로그에 `Node.js 20 actions are deprecated` 메시지가 0회 출현.
     - main 머지 후 `Deploy` 워크플로우 성공 + fly.io 사이트 정상 응답.

회귀 위험이 낮은 변경이지만, deploy까지 통과 확인이 끝나야 본 이슈 종료로 간주한다.
