# Issue #17 — 빌드용 tsconfig 분리 및 CI 타입 체크 추가

## 목표

`tsc -b && vite build` 빌드 흐름에서 테스트 파일이 컴파일 대상에 포함되어 발생한 CD 실패를 해결한다.
빌드 대상(`src/**/*.{ts,tsx}` 中 production 코드)과 테스트 대상(`*.test.{ts,tsx}`, 테스트 setup)의 TypeScript 검사 범위를 분리하고,
PR 단계에서 빌드 타입 체크가 사전에 수행되도록 CI 워크플로우에 type-check 단계를 추가한다.

---

## 배경 및 원인 정리

- `frontend/tsconfig.app.json`의 `include`는 `["src"]`이므로 `src/**/*.test.{ts,tsx}` 와 `src/test/setup.ts` 도 컴파일 대상에 포함된다.
- 동시에 `noUnusedLocals: true`, `noUnusedParameters: true`가 활성화되어 있어 테스트 파일 내 미사용 변수가 컴파일 에러로 직결된다.
- `npm run build` = `tsc -b && vite build`이므로 Docker 이미지 빌드(Deploy 워크플로우) 시 위 조건이 그대로 적용되어 `src/pages/logFormState.test.ts` 의 미사용 변수로 빌드가 실패했다.
- 기존 CI(`.github/workflows/ci.yml`)는 `npm test -- --run`(Vitest)만 실행하므로 `tsc` 타입 체크를 수행하지 않아 PR 단계에서 사전 검출이 불가능했다.

---

## 설계 결정 1: 별도 `tsconfig.test.json` 도입 (project references)

### 채택안

`tsconfig.app.json`은 production 소스만 포함하도록 좁히고, 테스트 파일은 `tsconfig.test.json` 으로 분리한 뒤 루트 `tsconfig.json`의 `references`에 등록한다.

| 파일 | 역할 | include / exclude |
|---|---|---|
| `tsconfig.json` | project references 진입점 | `references`: app, node, test |
| `tsconfig.app.json` | 빌드 대상 (`tsc -b && vite build`) | `include: ["src"]`, `exclude: ["src/**/*.test.ts", "src/**/*.test.tsx", "src/**/*.spec.ts", "src/**/*.spec.tsx", "src/test"]` |
| `tsconfig.test.json` | 테스트 코드 타입 체크 (신규) | `include: ["src/**/*.test.ts", "src/**/*.test.tsx", "src/test/**/*.ts", "e2e/**/*.ts"]` |
| `tsconfig.node.json` | Vite/도구 설정 (기존) | 기존 유지 |

### 채택 이유

- 단순히 `tsconfig.app.json`에 `exclude`만 추가하면 IDE(VS Code 등)에서 테스트 파일이 "어떤 tsconfig에도 속하지 않는 파일"로 처리되어 자동완성/타입 추론이 끊긴다.
- project references로 분리하면 `tsc -b`가 두 프로젝트를 모두 빌드하여 production/테스트 코드 양쪽의 타입 안전성을 유지한다. 동시에 `vite build`는 `tsconfig.app.json`만 따르므로 테스트 파일이 번들/타입 체크 대상에서 제외된다.
- 기존 `tsconfig.node.json`이 이미 `references` 패턴으로 통합되어 있어 같은 컨벤션을 유지할 수 있다.

### 단순 `exclude`만 추가하는 안을 택하지 않은 이유

- 위 IDE 문제 외에도, 테스트 파일에 대한 타입 체크가 사실상 Vitest 런타임(트랜스파일만 수행)에만 의존하게 되어, 시그니처 변경 시 테스트 파일의 타입 오류가 누락될 수 있다.
- Vitest의 `typecheck` 옵션은 vue-tsc 등 별도 도구에 의존하며, 본 프로젝트는 React + TS 표준 환경이므로 굳이 도입할 이유가 없다. `tsc -b` 한 번으로 모두 검사하는 편이 단순하다.

---

## 설계 결정 2: `tsconfig.app.json` 의 include/exclude 확정

### include

`["src"]` 유지. (디렉토리 단위 include 후 exclude로 테스트 파일을 제거하는 패턴이 가장 익숙하고 안정적)

### exclude (신규 추가)

```json
"exclude": [
  "src/**/*.test.ts",
  "src/**/*.test.tsx",
  "src/**/*.spec.ts",
  "src/**/*.spec.tsx",
  "src/test"
]
```

이유:
- `*.test.ts`, `*.test.tsx`: 현재 존재하는 테스트 파일 패턴 (15개 확인됨)
- `*.spec.ts`, `*.spec.tsx`: 현재는 존재하지 않지만 `vite.config.ts`의 Vitest `include` 패턴(`src/**/*.{test,spec}.{ts,tsx}`)과 동일하게 맞춰 미래에 추가될 가능성에 대비
- `src/test`: 현재 `setup.ts`만 존재하는 테스트 셋업 디렉토리

### types 옵션 정리

`tsconfig.app.json` 의 `types: ["vite/client", "@testing-library/jest-dom"]` 중 `@testing-library/jest-dom`은 production 빌드에는 필요 없는 테스트 전용 타입 보강이다.
production 컴파일 그래프에서는 제거하고, `tsconfig.test.json`으로 옮긴다.

- 변경 전 `tsconfig.app.json`: `"types": ["vite/client", "@testing-library/jest-dom"]`
- 변경 후 `tsconfig.app.json`: `"types": ["vite/client"]`
- `tsconfig.test.json`: `"types": ["vite/client", "@testing-library/jest-dom", "vitest/globals", "node"]`

`vitest/globals`를 추가하는 이유: `vite.config.ts`에서 `globals: true`이므로 `describe`/`it`/`expect`가 전역으로 사용된다. 테스트 파일에서 별도 import 없이 사용하려면 이 ambient 타입이 필요하다.
`node`를 추가하는 이유: `e2e/` 의 Playwright 테스트와 일부 setup에서 Node 환경 타입을 참조할 가능성에 대비.

---

## 설계 결정 3: `tsconfig.test.json` 상세 설계

```jsonc
{
  "extends": "./tsconfig.app.json",
  "compilerOptions": {
    "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.test.tsbuildinfo",
    "types": ["vite/client", "@testing-library/jest-dom", "vitest/globals", "node"],
    "noUnusedLocals": false,
    "noUnusedParameters": false
  },
  "include": [
    "src/**/*.test.ts",
    "src/**/*.test.tsx",
    "src/test/**/*.ts",
    "e2e/**/*.ts"
  ]
}
```

핵심 포인트:
- `extends: "./tsconfig.app.json"`로 컴파일러 옵션의 중복을 최소화한다.
- `noUnusedLocals` / `noUnusedParameters`를 `false`로 완화한다. 이슈에서 발견된 `cafeLog` 미사용 변수처럼 테스트 코드에서는 의도적인 미사용 패턴(setup에서 fixture만 만들고 검증은 다른 케이스에서 수행 등)이 흔히 발생한다. 다만 `strict: true`는 상속받아 유지한다.
- `tsBuildInfoFile`은 app/node와 충돌하지 않도록 별도 경로로 분리.

### `references` 필드를 두지 않은 이유 (구현 중 발견)

초기 설계에서는 `tsconfig.test.json`에 `references: [{ "path": "./tsconfig.app.json" }]`을 두어 "테스트가 production 코드를 빌드 의존성으로 인식"하도록 의도했다. 그러나 실제 `tsc -b` 실행 시 다음 에러가 발생한다.

```
error TS6306: Referenced project 'tsconfig.app.json' must have setting "composite": true.
error TS6310: Referenced project 'tsconfig.app.json' may not disable emit.
```

TypeScript는 한 프로젝트가 다른 프로젝트를 `references`로 명시 참조할 경우 대상에 `composite: true` 를 요구한다. 그러나 `composite: true`는 `declaration: true` + emit을 강제하므로 Vite 컨벤션인 `tsconfig.app.json`의 `noEmit: true` 와 정면 충돌한다. (Vite는 자체 트랜스파일을 수행하므로 `tsc`는 타입 체크 전용이고 emit이 불필요하다.)

따라서 다음 두 모드 중 후자를 채택한다:
- **명시적 dependency reference 모드**: 한 프로젝트가 다른 프로젝트를 `references`로 가리킴 → `composite` 필수 → Vite와 충돌, 미채택.
- **솔루션 스타일 references 모드** (채택): 루트 `tsconfig.json`이 `files: []` + `references`로 여러 독립 프로젝트를 묶어 `tsc -b` 한 번에 모두 type-check. composite 불필요. 본 프로젝트 기존 패턴(app + node)도 이 방식이다.

`tsconfig.test.json`은 `extends`로 컴파일러 옵션을 공유하고, 테스트 파일 안의 `import` 는 일반 모듈 해석으로 `src/`의 production 타입을 끌어온다. 결과적으로 production 타입 변경이 테스트 파일의 타입 오류로 노출되는 의도는 그대로 보존된다.

루트 `tsconfig.json`도 references에 추가:

```jsonc
{
  "files": [],
  "references": [
    { "path": "./tsconfig.app.json" },
    { "path": "./tsconfig.node.json" },
    { "path": "./tsconfig.test.json" }
  ]
}
```

이렇게 하면 `tsc -b` (인자 없이 실행) 시 세 프로젝트 전부 빌드된다. `npm run build`는 빌드 시점에 테스트 파일을 검사하지 않는 것이 명확한 요구사항이므로 `tsc -b tsconfig.app.json && vite build`로 변경한다 — app 프로젝트만 명시적으로 지정.

> **CLI 주의 (구현 중 발견)**: `tsc --build` 모드에서는 `-p` 플래그가 유효하지 않다. tsconfig 경로는 positional argument로만 받는다. (`-p`는 단일 컴파일 모드의 옵션이다.)

---

## 설계 결정 4: package.json 스크립트 조정

| 스크립트 | 변경 전 | 변경 후 | 비고 |
|---|---|---|---|
| `build` | `tsc -b && vite build` | `tsc -b -p tsconfig.app.json && vite build` | 빌드 시 production 코드만 검사 |
| `type-check` | (없음) | `tsc -b` | 신규. 모든 references 검사 (app + node + test) |

`type-check` 스크립트를 별도로 두는 이유:
- 개발 중 한 번에 모든 타입 체크를 돌릴 수 있는 진입점 제공.
- CI에서 `npm run type-check`로 호출하기 위함 (CLI 인자 노출 최소화).
- `vite build`를 거치지 않아 빠르다.

---

## 설계 결정 5: CI 워크플로우에 type-check job 추가

### 변경 위치

`.github/workflows/ci.yml`에 새 job `frontend-type-check` 추가.

### Job 정의

기존 `frontend-unit-test` job과 거의 동일한 환경(actions/checkout, setup-node, npm ci)을 사용하되, 마지막 step만 `npm run type-check`로 교체한다.

```yaml
frontend-type-check:
  name: Frontend Type Check
  runs-on: ubuntu-latest
  defaults:
    run:
      working-directory: frontend
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-node@v4
      with:
        node-version: 22
        cache: npm
        cache-dependency-path: frontend/package-lock.json
    - name: Install dependencies
      run: npm ci
    - name: Type check
      run: npm run type-check
```

### 별도 job 으로 분리하는 이유

- `frontend-unit-test`와 병렬로 실행되어 전체 CI 시간 영향 최소화.
- 실패 원인을 GitHub Actions UI에서 한눈에 구분할 수 있다 (타입 에러 vs 테스트 실패).
- `deploy.yml`이 `ci.yml`을 `workflow_call`로 재사용하므로, type-check가 실패하면 자동으로 배포가 차단된다.

### needs 의존성 추가는 하지 않는다

`backend-test`, `frontend-unit-test`, `frontend-type-check` 모두 독립적이므로 `needs`를 두지 않고 모두 병렬 실행한다. `deploy.yml`의 `deploy` job이 `needs: ci`를 통해 모든 job 통과를 기다린다.

---

## 수정하지 않는 것

- `vite.config.ts` — Vitest의 `include` 패턴은 그대로 유지. (테스트 발견 패턴은 변경 의도 없음)
- `playwright.config.ts` — Playwright 자체는 별도 트랜스파일을 수행하므로 tsconfig 영향 미미. 다만 `tsconfig.test.json`의 include에 `e2e/**/*.ts`를 포함시켜 IDE 타입 체크는 보장한다.
- `eslint.config.js` — ESLint는 자체 parser로 동작하며 본 변경과 무관.
- `frontend/Dockerfile` (있다면) — 빌드 명령은 `npm run build`를 호출하므로 변경 불필요.
- 백엔드 코드/설정 일체.
- `docs/spec.md`, `docs/openapi.yml` — 도메인/API 변경 없음.
- 기존 테스트 파일의 `cafeLog` 미사용 변수 자체 — `tsconfig.test.json`에서 `noUnusedLocals: false`로 완화하므로 그대로 두어도 무방. (정리는 별도 리팩터링 영역)

---

## 테스트 전략

본 이슈는 빌드/CI 인프라 변경이므로 자동화된 단위 테스트 추가 대상은 아니다. 다음 4단계 검증으로 충분하다.

1. **로컬 빌드 성공 확인**
   - `cd frontend && npm run build` 가 종료 코드 0으로 끝나는지.
   - 변경 전(테스트 파일 미수정 상태)에는 실패하던 것이 통과해야 한다.

2. **로컬 type-check 성공 확인**
   - `cd frontend && npm run type-check` 가 종료 코드 0으로 끝나는지.
   - 만약 production/테스트 코드에 실제 타입 에러가 있다면 여기서 노출되어야 한다.

3. **테스트 실행 영향 없음 확인**
   - `cd frontend && npm test -- --run` 결과가 변경 전과 동일한지.
   - Vitest는 자체 트랜스파일을 사용하므로 영향이 없어야 한다.

4. **CI 시뮬레이션**
   - 본 브랜치(`feat/17-separate-build-tsconfig`)에서 PR을 열어 GitHub Actions에서 `Frontend Type Check` job이 실행되고 통과하는지 확인.
   - main 머지 후 `Deploy` 워크플로우가 통과하는지 확인.

5. **회귀 테스트 케이스 (수동)**
   - 일부러 production 파일(`src/pages/HomePage.tsx` 등) 에 미사용 변수를 추가 → `npm run type-check` / `npm run build` 모두 실패해야 함.
   - 일부러 테스트 파일(`src/api/logs.test.ts` 등) 에 미사용 변수를 추가 → `npm run build`는 성공, `npm run type-check`도 성공(완화 옵션 적용). 단, **테스트 파일에서 실제 타입 오류**(예: 잘못된 인자 전달)는 `npm run type-check`에서 검출되어야 함 — `strict: true`는 상속하므로 보장됨.
