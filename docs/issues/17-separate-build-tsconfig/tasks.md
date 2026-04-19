# Tasks — Issue #17 빌드용 tsconfig 분리 및 CI 타입 체크 추가

> 본 이슈는 frontend 빌드 설정 및 CI 워크플로우 변경이 전부이다. 백엔드/도메인 코드 변경 없음.
> 상세 설계 의도는 `plan.md` 참조.
> 태스크 1~4는 강한 순서 의존성을 가진다 (1 → 2 → 3 → 4 순). 태스크 5는 1~4 완료 후 실행.

---

## 1. `tsconfig.app.json` 에서 테스트 파일 제외

- [x] **`exclude` 추가 및 `types` 정리**
  - Target: `frontend/tsconfig.app.json`
  - 기존 객체 끝의 `"include": ["src"]` 다음 라인에 아래 `exclude` 배열을 추가한다.
    ```json
    "exclude": [
      "src/**/*.test.ts",
      "src/**/*.test.tsx",
      "src/**/*.spec.ts",
      "src/**/*.spec.tsx",
      "src/test"
    ]
    ```
  - `compilerOptions.types`를 `["vite/client", "@testing-library/jest-dom"]` → `["vite/client"]`로 변경한다. (테스트 전용 타입은 `tsconfig.test.json`으로 이관)
  - 그 외 옵션(`strict`, `noUnusedLocals`, `noUnusedParameters` 등)은 변경하지 않는다.

---

## 2. `tsconfig.test.json` 신규 작성

- [x] **테스트 전용 tsconfig 파일 생성**
  - Target: `frontend/tsconfig.test.json` (신규)
  - 내용:
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
  - 주의: `extends`는 `tsconfig.app.json`의 `exclude`를 함께 상속하지 않는다(컴파일러 옵션만 상속). 따라서 위 `include`는 정확히 테스트 파일만 가리키게 된다.
  - `vitest/globals`는 `vitest` 패키지에 포함되어 있어 별도 설치 불필요.
  - **`references` 필드는 두지 않는다**. test → app 의 명시적 dependency reference는 `composite: true`를 요구하는데, app은 Vite 컨벤션상 `noEmit: true`라 충돌한다. 루트 `tsconfig.json`의 솔루션 스타일 references로 이미 두 프로젝트가 함께 type-check되므로 별도 명시가 불필요하다. (상세 근거는 plan.md 설계 결정 3 참조.)

---

## 3. 루트 `tsconfig.json` 의 references 갱신

- [x] **`tsconfig.test.json`을 references에 추가**
  - Target: `frontend/tsconfig.json`
  - `references` 배열에 `{ "path": "./tsconfig.test.json" }` 한 줄 추가.
  - 변경 후 최종 형태:
    ```json
    {
      "files": [],
      "references": [
        { "path": "./tsconfig.app.json" },
        { "path": "./tsconfig.node.json" },
        { "path": "./tsconfig.test.json" }
      ]
    }
    ```

---

## 4. `package.json` 스크립트 조정

- [ ] **`build` 스크립트를 production tsconfig만 빌드하도록 변경**
  - Target: `frontend/package.json`
  - 현재: `"build": "tsc -b && vite build"`
  - 변경: `"build": "tsc -b tsconfig.app.json && vite build"`
  - 의도: 빌드 시점에 테스트 파일을 검사하지 않도록 명시적으로 app 프로젝트만 빌드.
  - 주의: `tsc --build` 모드는 `-p` 플래그를 받지 않는다. tsconfig 경로는 positional argument로 전달해야 한다.

- [ ] **`type-check` 스크립트 신규 추가**
  - Target: `frontend/package.json`
  - `scripts` 객체 내 `lint` 다음 위치에 추가: `"type-check": "tsc -b"`
  - 이 스크립트는 루트 `tsconfig.json`을 통해 모든 references(app, node, test)를 빌드한다.

---

## 5. CI 워크플로우에 type-check job 추가

- [x] **`.github/workflows/ci.yml` 에 `frontend-type-check` job 추가**
  - Target: `.github/workflows/ci.yml`
  - 기존 `frontend-unit-test` job 하단에 아래 job을 추가한다 (들여쓰기는 기존과 동일하게 2 spaces).
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
  - `needs` 키는 추가하지 않는다 — `backend-test`, `frontend-unit-test`와 병렬 실행.
  - `on:` 트리거 섹션은 그대로 유지 (PR 및 `workflow_call`).
  - 결과적으로 `deploy.yml`의 `deploy` job은 `needs: ci`를 통해 type-check 통과를 기다리게 된다 (별도 변경 불필요).

---

## 6. 검증

- [x] **로컬 type-check 통과 확인**
  - 명령어: `cd /Users/junyoung/workspace/coffee-of-the-day/frontend && npm run type-check`
  - 종료 코드 0이어야 함.
  - 만약 실제 타입 에러가 있다면 plan.md 의 "회귀 테스트 케이스" 항목 참고.

- [x] **로컬 build 통과 확인**
  - 명령어: `cd /Users/junyoung/workspace/coffee-of-the-day/frontend && npm run build`
  - 종료 코드 0이어야 함.
  - 변경 전에는 `src/pages/logFormState.test.ts(120,7): error TS6133: 'cafeLog' ...` 로 실패했음. 변경 후에는 해당 에러가 사라져야 한다.

- [x] **로컬 unit test 회귀 없음 확인**
  - 명령어: `cd /Users/junyoung/workspace/coffee-of-the-day/frontend && npm test -- --run`
  - 변경 전과 동일한 결과(모두 통과)여야 함.

- [x] **수동 회귀 테스트 — production 파일 미사용 변수 시나리오**
  - 임시로 `src/pages/HomePage.tsx` 등 production 파일에 `const unused = 1` 같은 미사용 선언 추가.
  - `npm run type-check`, `npm run build` 모두 실패해야 함을 확인 후 변경 되돌림.

- [x] **수동 회귀 테스트 — 테스트 파일 미사용 변수 시나리오**
  - 임시로 임의 테스트 파일(`src/api/logs.test.ts` 등)에 `const unused = 1` 추가.
  - `npm run build`는 성공해야 한다 (테스트 파일이 빌드에서 제외됨).
  - `npm run type-check`도 성공해야 한다 (`noUnusedLocals: false`).
  - 확인 후 변경 되돌림.

- [x] **CI 통과 확인**
  - 본 브랜치(`feat/17-separate-build-tsconfig`)에서 PR을 열고 GitHub Actions에서 `Frontend Type Check` job이 등장하고 성공하는지 확인.
  - main 머지 후 `Deploy` 워크플로우 성공 확인.
