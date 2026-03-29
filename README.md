# Coffee of the Day

**Coffee of the Day**는 오늘 마신 커피를 기록하는 개인용 커피 저널입니다.

카페에서 마신 한 잔과 직접 추출한 한 잔을 같은 앱 안에서 기록하고,  
그날의 맛, 분위기, 레시피, 메모를 다시 읽기 좋은 형태로 남기는 데 초점을 맞췄습니다.

메모 앱처럼 가볍게 시작할 수 있지만,  
나중에 돌아봤을 때는 단순한 한 줄 감상보다 더 많은 정보가 남도록 만들었습니다.

## Overview

커피 기록 앱은 많지만, 오래 쓰게 되는 앱은 생각보다 드뭅니다.  
무엇을 남겨야 할지 애매하거나, 입력이 번거롭거나, 다시 봤을 때 남는 정보가 적기 때문입니다.

Coffee of the Day는 그 사이를 목표로 합니다.

- 카페에서 마신 커피는 `cafe` 로그로 기록
- 직접 내린 커피는 `brew` 로그로 기록
- 공통 정보와 타입별 정보를 한 번에 정리
- 최근 기록을 빠르게 훑고 다시 찾아볼 수 있는 흐름 제공

## Current Features

- 회원가입 / 로그인 / 로그아웃
- JWT + httpOnly cookie 기반 인증
- `cafe` / `brew` 로그 생성, 조회, 수정, 삭제
- 별점, 메모, 동행인, 테이스팅 태그 기록
- 브루 레시피와 추출 스텝 저장
- 날짜 / 로그 타입 필터
- 무한 스크롤 기반 목록 탐색
- 이전 입력 기반 태그 / 동행인 자동완성

## Preview

메인 화면에서는 최근 커피 기록을 카드 단위로 탐색할 수 있습니다.  
필터를 적용해 특정 기간이나 로그 타입만 좁혀 볼 수 있고, 아래로 스크롤하면 다음 기록이 이어집니다.

<p align="center">
  <img src="./image/main.png" width="780" alt="Coffee of the Day 메인 화면" />
</p>

## Tech Stack

| Layer | Stack |
|------|-------|
| Backend | Go, chi, sqlc, SQLite, golang-migrate |
| Frontend | React, TypeScript, Vite, TanStack Query, React Router, Tailwind CSS v4 |
| Auth | JWT, httpOnly Cookie |
| API | OpenAPI 3.0 |
| Test | Go testing, Vitest, Playwright |

## Run Locally

### Backend

```bash
cd backend
go run ./cmd/server
```

기본 주소: `http://localhost:8080`

### Frontend

```bash
cd frontend
npm install
npm run dev
```

기본 주소: `http://localhost:5173`

## Test

```bash
cd backend
go test ./...
```

```bash
cd frontend
npm test
```

```bash
cd frontend
npm run test:e2e:install
npm run test:e2e
```

## Documents

- [`spec.md`](./spec.md): 기능 명세
- [`plan.md`](./plan.md): 개발 계획
- [`tasks.md`](./tasks.md): 구현 체크리스트
- [`guide/`](./guide): 학습 문서와 아키텍처 문서
- [`review/`](./review): 코드 리뷰와 리팩터링 관련 문서
- [`openapi.yml`](./openapi.yml): API 명세

## Built With AI

이 프로젝트는 구현, 테스트, 문서화까지 AI 에이전트 중심으로 진행한 애플리케이션입니다.  
사람은 방향을 정하고 결과를 검수했고, AI는 그 방향을 코드와 문서로 구체화했습니다.

---

이 문서는 Claude Code와 Codex가 작성했습니다.
