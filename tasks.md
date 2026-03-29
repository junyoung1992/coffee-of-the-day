# Coffee of the Day — Task List

> plan.md 기반 세부 실행 태스크입니다.
> 각 태스크는 완료 여부를 명확히 판단할 수 있는 단위로 작성했습니다.

---

## Phase 1 — 프로젝트 기반 + 기록 CRUD

### 1-1. 프로젝트 초기 설정

**Backend**
- [x] `backend/` 디렉토리 생성 및 Go 모듈 초기화 (`go mod init`)
- [x] 의존성 추가: `go-chi/chi`, `golang-migrate`, `sqlc`, `mattn/go-sqlite3`
- [x] 디렉토리 구조 생성: `cmd/server/`, `internal/{handler,service,repository,domain}/`, `db/{migrations,queries}/`, `config/`
- [x] `config/config.go` — `DB_PATH`, `PORT` 환경변수 로딩
- [x] `cmd/server/main.go` — DB 연결, 라우터 마운트, 서버 시작

**Frontend**
- [x] `frontend/` Vite + React + TypeScript 프로젝트 생성
- [x] 의존성 추가: `tailwindcss`, `react-router-dom`, `@tanstack/react-query`
- [x] Tailwind CSS 설정 (v4: `tailwind.config.ts`/`postcss.config.ts` 대신 `@tailwindcss/vite` 플러그인 방식으로 처리)
- [x] 디렉토리 구조 생성: `src/{pages,components,api,types,hooks}/`
- [x] `src/main.tsx` — `QueryClientProvider`, `RouterProvider` 설정
- [x] `src/api/client.ts` — base URL, `X-User-Id` 헤더, 공통 에러 처리
- [x] 기본 라우터 설정: `/`, `/logs/new`, `/logs/:id`, `/logs/:id/edit`

---

### 1-2. DB 스키마 및 마이그레이션

- [x] `golang-migrate` CLI 설치 및 사용법 확인 (Go 라이브러리로 통합)
- [x] `001_create_users.up.sql` / `.down.sql` 작성
- [x] `002_create_coffee_logs.up.sql` / `.down.sql` 작성
- [x] `003_create_cafe_logs.up.sql` / `.down.sql` 작성
- [x] `004_create_brew_logs.up.sql` / `.down.sql` 작성
- [x] `cmd/server/main.go`에서 서버 시작 시 마이그레이션 자동 실행
- [x] `sqlc.yaml` 설정 파일 작성 (SQLite 드라이버, 쿼리/스키마 경로, 출력 경로)

---

### 1-3. Backend — sqlc 쿼리 및 도메인 타입

**sqlc 쿼리 작성** (`db/queries/`)
- [x] `coffee_logs.sql` — `InsertLog`, `GetLogByID`, `ListLogs`, `UpdateLog`, `DeleteLog`
- [x] `cafe_logs.sql` — `InsertCafeLog`, `GetCafeLogByLogID`, `UpdateCafeLog`
- [x] `brew_logs.sql` — `InsertBrewLog`, `GetBrewLogByLogID`, `UpdateBrewLog`
- [x] `sqlc generate` 실행 및 생성 코드 확인

**도메인 타입** (`internal/domain/`)
- [x] `log.go` — `LogType`, `CoffeeLog`, `CafeDetail`, `BrewDetail`, `CoffeeLogFull` 타입 정의
- [x] `[]string` ↔ JSON 직렬화 헬퍼 (SQLite TEXT 배열 처리용)

---

### 1-4. Backend — Repository

- [x] `internal/repository/log_repository.go` 인터페이스 및 SQLite 구현체 정의
- [x] `CreateLog` — 트랜잭션으로 `coffee_logs` + 서브 테이블 동시 삽입
- [x] `GetLogByID` — `coffee_logs` + 서브 테이블 JOIN 조회, `user_id` 소유권 검증
- [x] `ListLogs` — 필터(`log_type`, `date_from`, `date_to`) + cursor-based 페이지네이션
- [x] `UpdateLog` — 트랜잭션으로 `coffee_logs` + 서브 테이블 동시 수정
- [x] `DeleteLog` — `coffee_logs` 삭제 (CASCADE로 서브 테이블 자동 삭제)

**Cursor 구현**
- [x] 커서 인코딩: `{sort_by, order, sort_value, id}` → base64 opaque 문자열
- [x] 커서 디코딩 및 SQL WHERE 조건 생성 (`recorded_at < ? OR (recorded_at = ? AND id < ?)`)

---

### 1-5. Backend — Service

- [x] `internal/service/log_service.go` — `LogService` 인터페이스 정의
- [x] `CreateLog(userID, req)` — 유효성 검사 (`log_type`에 따라 서브 객체 필수 확인)
- [x] `GetLog(userID, logID)` — 소유권 검증 포함
- [x] `ListLogs(userID, filter)` — 필터·커서 파라미터 전달
- [x] `UpdateLog(userID, logID, req)` — 소유권 검증 후 수정
- [x] `DeleteLog(userID, logID)` — 소유권 검증 후 삭제

---

### 1-6. Backend — Handler 및 미들웨어

- [x] `internal/handler/middleware.go` — `X-User-Id` 헤더 파싱 미들웨어 (POC)
- [x] `internal/handler/middleware.go` — CORS 미들웨어 (`localhost:5173` 허용)
- [x] `internal/handler/log_handler.go` — `POST /api/v1/logs`
- [x] `internal/handler/log_handler.go` — `GET /api/v1/logs`
- [x] `internal/handler/log_handler.go` — `GET /api/v1/logs/:id`
- [x] `internal/handler/log_handler.go` — `PUT /api/v1/logs/:id`
- [x] `internal/handler/log_handler.go` — `DELETE /api/v1/logs/:id`
- [x] `cmd/server/main.go` — 라우터에 핸들러·미들웨어 연결

---

### 1-7. Frontend — 타입 및 API 클라이언트

- [x] `src/types/log.ts` — `CoffeeLogBase`, `CafeDetail`, `BrewDetail`, `CafeLogFull`, `BrewLogFull`, `CoffeeLogFull` (Discriminated Union)
- [x] `src/types/common.ts` — `CursorPage<T>`, 공통 에러 타입
- [x] `src/api/logs.ts` — `getLogs`, `getLog`, `createLog`, `updateLog`, `deleteLog`
- [x] `src/hooks/useLogs.ts` — `useLogList` (`useInfiniteQuery`), `useLog`, `useCreateLog`, `useUpdateLog`, `useDeleteLog`

---

### 1-8. Frontend — 페이지 및 컴포넌트

**공용 컴포넌트**
- [x] `src/components/Layout.tsx` — 공통 레이아웃 (헤더, 컨텐츠 영역)
- [x] `src/components/LogCard.tsx` — 목록에서 보이는 기록 카드 (cafe/brew 분기)
- [x] `src/components/RatingDisplay.tsx` — 0.5 단위 별점 표시
- [x] `src/components/RatingInput.tsx` — 0.5 단위 별점 입력

**페이지**
- [x] `src/pages/HomePage.tsx` — 기록 카드 목록, 무한 스크롤 (`IntersectionObserver`)
- [x] `src/pages/LogDetailPage.tsx` — 기록 상세 보기 (cafe/brew 분기 렌더링)
- [x] `src/pages/LogFormPage.tsx` — 신규 작성 / 수정 통합 폼
  - [x] cafe/brew 탭 전환
  - [x] 공통 필드 섹션 (recorded_at, companions)
  - [x] 카페 전용 섹션
  - [x] 브루 전용 섹션 (brew_method 선택, 레시피 입력)
  - [x] `brew_steps` 동적 입력 (추가/삭제/위아래 버튼)

**Phase 1 완료 기준**
- [x] 브라우저에서 카페 기록 생성 후 목록에서 확인
- [x] 기록 수정 및 삭제 동작
- [x] 브루 기록 생성·조회 동작

---

## Phase 2 — 브루 폼 고도화 + 목록 필터

### 2-1. 브루 기록 전용 폼 UI 고도화

- [x] `brew_method` 선택 UI — 버튼 그룹 (아이콘 또는 라벨)
- [x] `brew_device` 자유 입력 필드
- [x] 레시피 섹션 레이아웃 개선 (원두량/물량 비율 자동 계산 표시)

### 2-2. 목록 필터

- [x] `log_type` 필터 탭 UI (전체 / 카페 / 브루)
- [x] 날짜 범위 필터 UI (`date_from`, `date_to` 날짜 선택)
- [x] 필터 상태를 URL 쿼리 파라미터에 반영 (공유·북마크 가능)
- [x] 필터 변경 시 TanStack Query 캐시 키 갱신

### 2-3. 무한 스크롤 고도화

- [x] 스크롤 하단 도달 감지 (`IntersectionObserver` sentinel 요소)
- [x] 다음 페이지 로딩 중 스켈레톤 UI
- [x] 마지막 페이지 도달 시 "더 이상 기록이 없습니다" 표시

### 2-4. Happy-path E2E 자동화

- [x] Playwright 의존성 및 실행 스크립트 추가
- [x] E2E용 백엔드/프론트엔드 서버 기동 설정 추가
- [x] E2E용 POC 사용자 시드 훅 추가
- [x] 브루 로그 기준 happy-path E2E 추가 (`생성 → 목록 → 상세 → 수정 → 삭제`)

**Phase 2 완료 기준**
- [x] 브루 기록을 레시피 포함 완전히 기록 가능
- [x] 타입/날짜 필터로 원하는 기록만 조회 가능
- [x] 무한 스크롤로 전체 기록 탐색 가능
- [x] 핵심 happy-path가 E2E로 자동 검증됨

---

## Phase 3 — 자동완성

### 3-1. 자동완성 API (Backend)

- [x] `db/queries/suggestions.sql` — 유저의 `tasting_tags` 집계 쿼리 (빈도순)
- [x] `db/queries/suggestions.sql` — 유저의 `companions` 집계 쿼리 (빈도순)
- [x] `GET /api/v1/suggestions/tags?q=` 핸들러
- [x] `GET /api/v1/suggestions/companions?q=` 핸들러

### 3-2. 자동완성 컴포넌트 (Frontend)

- [x] `src/api/suggestions.ts` — `getSuggestions(type, q)`
- [x] `src/hooks/useSuggestions.ts` — `useTagSuggestions`, `useCompanionSuggestions`
- [x] `src/components/TagInput.tsx` — 텍스트 입력 + 드롭다운 추천 + 태그 뱃지
- [x] `LogFormPage`의 `tasting_tags` 필드에 `TagInput` 적용
- [x] `LogFormPage`의 `companions` 필드에 `TagInput` 적용

**Phase 3 완료 기준**
- [x] 태그 입력 시 이전 태그 자동완성 제안
- [x] 동반자 입력 시 이전 이름 자동완성 제안

---

## Phase 4 — 계정 및 인증

### 4-1. DB 마이그레이션

- [ ] `005_add_auth_to_users.up.sql` — `users` 테이블에 `email`, `password_hash` 컬럼 추가
- [ ] `005_add_auth_to_users.down.sql`

### 4-2. Backend — 인증 API

- [ ] `bcrypt` 의존성 추가
- [ ] JWT 라이브러리 추가 (`golang-jwt/jwt`)
- [ ] `POST /api/v1/auth/register` — 회원가입, 비밀번호 bcrypt 해싱
- [ ] `POST /api/v1/auth/login` — 로그인, JWT 발급, httpOnly cookie 설정 (`SameSite=Strict`)
- [ ] `POST /api/v1/auth/refresh` — 토큰 갱신
- [ ] `POST /api/v1/auth/logout` — 쿠키 만료 처리
- [ ] JWT 검증 미들웨어 (`X-User-Id` 헤더 미들웨어 교체)

### 4-3. Frontend — 인증 UI

- [ ] `src/types/auth.ts` — `User`, `LoginRequest`, `RegisterRequest` 타입
- [ ] `src/api/auth.ts` — `login`, `register`, `logout`, `refresh`
- [ ] `src/hooks/useAuth.ts` — 인증 상태 관리
- [ ] `src/pages/LoginPage.tsx`
- [ ] `src/pages/RegisterPage.tsx`
- [ ] 미인증 시 로그인 페이지로 리다이렉트 (Protected Route)
- [ ] `X-User-Id` 헤더 방식 제거, 쿠키 기반으로 전환

**Phase 4 완료 기준**
- [ ] 회원가입 후 로그인하여 본인 기록만 조회·수정·삭제 가능
- [ ] 로그아웃 후 기록에 접근 불가

---

*Last updated: 2026-03-29*
