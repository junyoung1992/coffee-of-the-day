# Tasks — Issue #19 GitHub Actions Node.js 20 런타임 deprecation 대응

> 본 이슈는 `.github/workflows/*.yml`의 액션 메이저 핀 버전만 교체하는 인프라 변경이다.
> 백엔드/프론트엔드 코드, 도메인 스펙, OpenAPI 스펙 모두 변경 없음.
> 상세 설계 의도와 버전 선정 근거는 `plan.md` 참조.
> 작업 브랜치는 이미 `feat/19-actions-node24-upgrade`로 생성되어 있다.
> 태스크 1, 2는 독립적이며 순서 무관. 태스크 3은 1, 2 완료 후 PR 단계에서 수행.

---

## 1. `.github/workflows/ci.yml` 액션 핀 업그레이드

- [ ] **`actions/checkout@v4` → `actions/checkout@v5` 일괄 치환**
  - Target: `.github/workflows/ci.yml`
  - 대상 라인: 14, 30, 51 (총 3곳).
  - 각 라인의 `- uses: actions/checkout@v4`를 `- uses: actions/checkout@v5`로 변경.
  - 들여쓰기, 라인 위치, 주변 빈 줄은 그대로 유지.
  - `with:` 블록이 없는 형태이므로 추가 입력 변경 없음.

- [ ] **`actions/setup-node@v4` → `actions/setup-node@v5` 일괄 치환**
  - Target: `.github/workflows/ci.yml`
  - 대상 라인: 32, 53 (총 2곳).
  - 각 라인의 `- uses: actions/setup-node@v4`를 `- uses: actions/setup-node@v5`로 변경.
  - 바로 아래 `with:` 블록(`node-version: 22`, `cache: npm`, `cache-dependency-path: frontend/package-lock.json`)은 그대로 유지. v5에서도 동일하게 지원된다.

- [ ] **`actions/setup-go@v5` → `actions/setup-go@v6` 치환**
  - Target: `.github/workflows/ci.yml`
  - 대상 라인: 16 (1곳).
  - `- uses: actions/setup-go@v5`를 `- uses: actions/setup-go@v6`로 변경.
  - 바로 아래 `with:` 블록(`go-version-file: go.mod`)은 그대로 유지. v6에서도 동일하게 지원된다.

---

## 2. `.github/workflows/deploy.yml` 액션 핀 업그레이드

- [ ] **`actions/checkout@v4` → `actions/checkout@v5` 치환**
  - Target: `.github/workflows/deploy.yml`
  - 대상 라인: 22 (1곳, deploy job 내부).
  - `- uses: actions/checkout@v4`를 `- uses: actions/checkout@v5`로 변경.
  - **변경하지 말 것**: 24행 `superfly/flyctl-actions/setup-flyctl@master`는 본 이슈 범위 외.
  - **변경하지 말 것**: 14-15행 `uses: ./.github/workflows/ci.yml` (로컬 워크플로우 재사용 — 액션 핀 아님).
  - `concurrency`, `needs: ci`, `env: FLY_API_TOKEN` 등 다른 모든 설정은 그대로 유지.

---

## 3. 검증 (PR 생성 및 실 환경 실행)

- [ ] **로컬 YAML 정합성 빠른 점검 (선택)**
  - 명령어 (actionlint 설치 시): `actionlint .github/workflows/ci.yml .github/workflows/deploy.yml`
  - 미설치 시 생략. PR CI에서 어차피 잡힌다.

- [ ] **변경 사항 커밋 및 PR 생성**
  - 사용자 승인 후 `feat/19-actions-node24-upgrade` 브랜치에 변경을 커밋하고 push.
  - PR 본문에 plan.md 핵심 요점(Node 20 deprecation 대응, 업그레이드 매트릭스, 검증 절차) 요약 포함.
  - PR 트리거(`pull_request: branches: [main]`)에 의해 `ci.yml`이 자동 실행됨.

- [ ] **PR CI job 전체 통과 확인**
  - GitHub Actions UI에서 다음 3개 job이 모두 success여야 함.
    - `Backend Test` (backend-test)
    - `Frontend Unit Test` (frontend-unit-test)
    - `Frontend Type Check` (frontend-type-check)
  - 어느 하나라도 실패 시 plan.md "실패 시 롤백" 절차 참조.

- [ ] **deprecation 경고 소거 확인**
  - 각 job의 raw 로그를 열어 다음 step 시작 부분에서 `Node.js 20 actions are deprecated` 문자열이 더 이상 등장하지 않음을 확인.
    - `backend-test` → `Set up Go` step (setup-go@v6)
    - `frontend-unit-test` → `Set up Node` step (setup-node@v5)
    - `frontend-type-check` → `Set up Node` step (setup-node@v5)
    - 모든 job → `Run actions/checkout@v5` step
  - 빠른 검색: 로그에서 "deprecated" 키워드로 grep해 0건이어야 함.

- [ ] **main 머지 후 Deploy 워크플로우 검증**
  - PR squash-merge 후 자동 트리거되는 `Deploy` 워크플로우 실행 결과 확인.
  - 다음 모두 만족해야 함:
    - 재사용된 ci 단계 통과 (3개 job 모두 success).
    - deploy job의 `Run actions/checkout@v5` step에서 deprecation 경고 없음.
    - `flyctl deploy --remote-only`가 정상 완료.
    - fly.io 사이트가 평소대로 응답(루트 페이지 200 OK 등 간단 확인).
  - 이상 시 즉시 revert 커밋으로 롤백 후 재조사.

- [ ] **머지 후 정리**
  - 작업 브랜치 `feat/19-actions-node24-upgrade` 원격/로컬 삭제.
  - 이슈 #19 close (PR 머지 시 자동 close되도록 PR 본문에 `Closes #19` 포함).
