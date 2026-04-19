# Postmortem — 빌드 tsconfig 미분리로 인한 CD 실패

- **일시**: 2026-04-19 (Issue #17 발견 및 대응 시점 기준)
- **작업**: Issue #17 — 빌드용 tsconfig 분리하여 테스트 파일을 tsc 컴파일 대상에서 제외
- **심각도**: 중간 (CD 워크플로우 실패로 main 머지분 배포가 차단됨, 운영 서비스 영향은 단발성)

---

## 요약

main 머지 후 Deploy 워크플로우의 Docker 빌드(`npm run build` = `tsc -b && vite build`)가 다음 에러로 실패했다.

```
src/pages/logFormState.test.ts(120,7): error TS6133: 'cafeLog' is declared but its value is never read.
```

- 코드 자체의 회귀가 아니라 **빌드 설정의 결함**이었다.
- `tsconfig.app.json`의 `include: ["src"]`가 테스트 파일까지 컴파일 대상으로 끌어왔고, 동일 설정의 `noUnusedLocals: true`가 테스트 파일의 의도적 미사용 변수를 에러로 승격시켰다.
- 동일 결함이 PR CI에서 검출되지 않은 이유는 CI가 `npm test`(Vitest)만 실행하고 `tsc` 검사를 거치지 않았기 때문이다. Vitest는 자체 트랜스파일을 사용해 타입 체크를 건너뛴다.

---

## 원인

### 1차 원인 — 빌드 검사 범위와 테스트 검사 범위가 분리되지 않음

`tsconfig.app.json`은 production 빌드 진입점이면서 동시에 테스트 파일까지 포함하는 광역 include 를 가지고 있었다. 동일 파일에 다음 두 옵션이 공존했다:

- `include: ["src"]` — 테스트 파일도 자동 포함
- `noUnusedLocals: true`, `noUnusedParameters: true` — 테스트 코드의 흔한 패턴(setup에서 fixture만 만들고 검증은 다른 케이스에서 수행 등)과 충돌

두 옵션 어느 쪽도 단독으로는 문제가 아니지만, **동일 컴파일 그래프에 묶인 순간 production 빌드가 테스트 코드의 미사용 변수에 좌우되는 구조**가 된다.

### 2차 원인 — CI가 production 빌드와 동일한 검사를 수행하지 않음

`.github/workflows/ci.yml`에는 `npm test -- --run`만 등록되어 있었다. 결과적으로 다음 비대칭이 생겼다:

| 단계 | TS 검사 도구 | `noUnusedLocals` 적용 |
|------|-------------|----------------------|
| 로컬 `npm test` / CI Vitest | esbuild (트랜스파일) | ✗ |
| 로컬 `npm run build` / CD Docker | `tsc -b` | ✓ |

PR에서는 항상 좌측 경로만 통과되어 우측 경로의 결함이 main 머지 후 CD 단계까지 노출이 미뤄졌다.

---

## 해결

### 빌드/테스트 검사 범위 분리

`tsconfig.app.json`을 좁히고 테스트 코드는 별도 `tsconfig.test.json`으로 분리한 뒤, 루트 `tsconfig.json`에서 솔루션 스타일 references로 묶었다.

- `tsconfig.app.json`: `*.test.{ts,tsx}`, `*.spec.{ts,tsx}`, `src/test/` exclude. `@testing-library/jest-dom` 타입은 production 그래프에서 제거.
- `tsconfig.test.json` (신규): app config를 `extends`하되 `noUnusedLocals`/`noUnusedParameters`를 `false`로 완화. test 전용 ambient 타입(`vitest/globals`, `@testing-library/jest-dom`, `node`)을 여기서만 적재.
- `npm run build`를 `tsc -b tsconfig.app.json && vite build`로 좁혀 빌드 시점에 테스트가 검사되지 않음을 명시적으로 보장.

### CI에 빌드 타입 체크 단계 추가

`.github/workflows/ci.yml`에 `frontend-type-check` job을 추가하고 `npm run type-check`(=`tsc -b`)를 실행한다. `deploy.yml`이 `ci.yml`을 `workflow_call`로 재사용하므로 type-check 실패는 자동으로 배포를 차단한다.

---

## 구현 중 발견한 사항

### 1. Project references의 `composite` 요구사항이 Vite 컨벤션과 충돌

`tsconfig.test.json`이 `tsconfig.app.json`을 명시적 dependency reference로 가리키게 하려 했으나 다음 에러가 발생했다.

```
error TS6306: Referenced project 'tsconfig.app.json' must have setting "composite": true.
error TS6310: Referenced project 'tsconfig.app.json' may not disable emit.
```

`composite: true`는 declaration 파일 emit을 강제하는데, Vite 컨벤션상 `tsc`는 타입 체크 전용이므로 `noEmit: true`이다. 두 옵션은 양립 불가하다.

**채택안**: 명시적 reference를 포기하고 솔루션 스타일(루트 `tsconfig.json`이 독립 프로젝트들을 묶는 방식)만 사용. test config는 `extends`로 컴파일러 옵션을 공유하고, 모듈 해석으로 `src/`의 production 타입을 끌어온다. 결과적으로 production 타입 변경이 테스트 파일의 타입 오류로 노출되는 의도는 그대로 유지된다.

### 2. `tsc --build` 모드는 `-p` 플래그를 받지 않음

빌드 스크립트를 `tsc -b -p tsconfig.app.json`로 작성했더니 `error TS5072: Unknown build option '-p'`가 발생했다. `-p`는 단일 컴파일 모드(`tsc -p`) 전용이고, build 모드에서는 tsconfig 경로를 positional argument로 전달해야 한다(`tsc -b tsconfig.app.json`).

---

## 교훈

1. **CI는 production 빌드와 동일한 도구로 동일한 옵션을 적용해야 한다.** Vitest 같은 트랜스파일러 기반 테스트 러너는 `tsc` 의 lint성 옵션(`noUnusedLocals` 등)을 강제하지 않는다. PR 단계에서 production 빌드의 타입 검사를 따로 수행하지 않으면, 동일 옵션이 다른 단계에 적용된다는 사실이 머지 이후에야 드러난다.

2. **하나의 tsconfig가 여러 목적을 겸하면 옵션 간 의도치 않은 결합이 생긴다.** "production 코드만 검사" 와 "테스트 코드도 검사" 는 다른 use case이고, `noUnusedLocals` 같은 옵션은 두 코드 카테고리에서 의미가 다르다. 검사 범위를 명시적으로 분리하면 옵션의 적용 범위도 자연스럽게 분리된다.

3. **TypeScript의 빌드 인프라 옵션은 서로 강한 제약을 가지므로, 설계 단계에서 가정한 references 구성이 실제로 컴파일되는지 빠르게 검증하는 편이 안전하다.** 본 건의 `composite` vs `noEmit` 충돌, `tsc -b` 의 `-p` 미지원 모두 plan 단계에서는 발견되지 않았다.

---

## 재발 방지

- CI 워크플로우에 `frontend-type-check` job 상시 가동 (Issue #17 반영분).
- 향후 frontend tsconfig 변경 시 다음 4종을 모두 통과하는지 확인:
  - `npm run type-check` (모든 references)
  - `npm run build` (production만)
  - `npm test -- --run`
  - 의도적 회귀 테스트: production 파일에 미사용 변수 추가 → build 실패 / 테스트 파일에 미사용 변수 추가 → build 성공
