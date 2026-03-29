# Coffee of the Day — 개발 계획

> 각 Phase는 독립적으로 동작 가능한 상태(vertical slice)로 완료됩니다.
> 좁은 기능이라도 백엔드 + 프론트엔드가 함께 동작하는 것을 목표로 합니다.
> spec.md가 기준이며, 구현 중 발견된 변경사항은 spec.md에도 반영합니다.

---

## Phase 1 — 프로젝트 기반 + 기록 CRUD (Vertical Slice)

> **목표**: 카페 기록을 생성·조회·수정·삭제하는 최소 동작 흐름을 백+프론트 함께 완성한다.
> 이 Phase가 끝나면 브라우저에서 실제로 커피 기록을 남길 수 있다.

### 1-1. 프로젝트 초기 설정

**Backend**
- `backend/` 디렉토리 Go 모듈 초기화 (`go mod init`)
- 의존성 추가: `chi`, `golang-migrate`, `sqlc`, `mattn/go-sqlite3`
- 디렉토리 구조 생성: `cmd/server`, `internal/{handler,service,repository,domain}`, `db/{migrations,queries}`
- `config/` — 환경변수 로딩 (`DB_PATH`, `PORT`)
- `cmd/server/main.go` — 서버 부트스트랩 (라우터 연결, DB 연결)

**Frontend**
- `frontend/` Vite + React + TypeScript 프로젝트 생성
- 의존성 추가: `tailwindcss`, `react-router-dom`, `@tanstack/react-query`
- 디렉토리 구조 생성: `src/{pages,components,api,types,hooks}`
- 기본 라우터 설정 (`/`, `/logs/new`, `/logs/:id`, `/logs/:id/edit`)
- API 클라이언트 기반 코드 (`src/api/client.ts`) — base URL, 공통 헤더, 에러 처리

### 1-2. DB 스키마 및 마이그레이션

```sql
-- 001_create_users.sql
CREATE TABLE users (
  id          TEXT PRIMARY KEY,   -- UUID
  username    TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  created_at  TEXT NOT NULL       -- ISO8601
);

-- 002_create_coffee_logs.sql
CREATE TABLE coffee_logs (
  id          TEXT PRIMARY KEY,
  user_id     TEXT NOT NULL REFERENCES users(id),
  recorded_at TEXT NOT NULL,
  companions  TEXT NOT NULL DEFAULT '[]',  -- JSON array
  log_type    TEXT NOT NULL CHECK(log_type IN ('cafe','brew')),
  memo        TEXT,
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);

-- 003_create_cafe_logs.sql
CREATE TABLE cafe_logs (
  log_id       TEXT PRIMARY KEY REFERENCES coffee_logs(id) ON DELETE CASCADE,
  cafe_name    TEXT NOT NULL,
  location     TEXT,
  coffee_name  TEXT NOT NULL,
  bean_origin  TEXT,
  bean_process TEXT,
  roast_level  TEXT CHECK(roast_level IN ('light','medium','dark')),
  tasting_tags TEXT NOT NULL DEFAULT '[]',  -- JSON array
  tasting_note TEXT,
  impressions  TEXT,
  rating       REAL CHECK(rating >= 0.5 AND rating <= 5.0)
);

-- 004_create_brew_logs.sql
CREATE TABLE brew_logs (
  log_id           TEXT PRIMARY KEY REFERENCES coffee_logs(id) ON DELETE CASCADE,
  bean_name        TEXT NOT NULL,
  bean_origin      TEXT,
  bean_process     TEXT,
  roast_level      TEXT CHECK(roast_level IN ('light','medium','dark')),
  roast_date       TEXT,
  tasting_tags     TEXT NOT NULL DEFAULT '[]',  -- JSON array
  tasting_note     TEXT,
  brew_method      TEXT NOT NULL CHECK(brew_method IN (
                     'pour_over','immersion','aeropress',
                     'espresso','moka_pot','siphon','cold_brew','other')),
  brew_device      TEXT,
  coffee_amount_g  REAL,
  water_amount_ml  REAL,
  water_temp_c     REAL,
  brew_time_sec    INTEGER,
  grind_size       TEXT,
  brew_steps       TEXT NOT NULL DEFAULT '[]',  -- JSON array
  impressions      TEXT,
  rating           REAL CHECK(rating >= 0.5 AND rating <= 5.0)
);
```

> SQLite는 JSON 배열을 TEXT로 저장합니다. Go 레이어에서 `[]string` ↔ JSON 직렬화를 담당합니다.

### 1-3. Backend — CRUD API

**sqlc 쿼리 작성** (`db/queries/`)
- `coffee_logs`: InsertLog, GetLogByID, ListLogs, UpdateLog, DeleteLog
- `cafe_logs`: InsertCafeLog, GetCafeLogByLogID, UpdateCafeLog
- `brew_logs`: InsertBrewLog, GetBrewLogByLogID, UpdateBrewLog

**도메인 타입** (`internal/domain/`)
```go
type LogType string
const (
    LogTypeCafe LogType = "cafe"
    LogTypeBrew LogType = "brew"
)

type CoffeeLog struct { ... }   // 공통 필드
type CafeDetail struct { ... }  // cafe_logs 필드
type BrewDetail struct { ... }  // brew_logs 필드

// API 응답/요청용 통합 타입
type CoffeeLogFull struct {
    CoffeeLog
    Cafe *CafeDetail `json:"cafe,omitempty"`
    Brew *BrewDetail `json:"brew,omitempty"`
}
```

**Repository** (`internal/repository/`)
- `CoffeeLogRepository`: CRUD + 서브 테이블 JOIN 조회

**Service** (`internal/service/`)
- `CreateLog(userID, req)` — 트랜잭션으로 `coffee_logs` + 서브 테이블 동시 삽입
- `GetLog(userID, logID)` — user_id 소유권 검증 포함
- `ListLogs(userID, filter)` — 날짜·타입 필터
- `UpdateLog(userID, logID, req)`
- `DeleteLog(userID, logID)`

**Handler** (`internal/handler/`)
- `POST /api/v1/logs` — 요청 파싱, log_type에 따라 서브 객체 분기
- `GET /api/v1/logs` — 쿼리 파라미터 파싱 (log_type, date_from, date_to, page, per_page)
- `GET /api/v1/logs/:id`
- `PUT /api/v1/logs/:id`
- `DELETE /api/v1/logs/:id`
- CORS 미들웨어 (`localhost:5173` 허용)
- `X-User-Id` 헤더 파싱 미들웨어 (POC용)

### 1-4. Frontend — 기본 CRUD UI

**타입 정의** (`src/types/log.ts`)
- 백엔드 도메인과 1:1 대응하는 TypeScript 타입
- Discriminated Union: `CafeLogFull`, `BrewLogFull`, `CoffeeLogFull = CafeLogFull | BrewLogFull`

**API 클라이언트** (`src/api/logs.ts`)
- `getLogs(params)`, `getLog(id)`, `createLog(body)`, `updateLog(id, body)`, `deleteLog(id)`

**TanStack Query 훅** (`src/hooks/useLogs.ts`)
- `useLogList`, `useLog`, `useCreateLog`, `useUpdateLog`, `useDeleteLog`

**페이지**
- `HomePage` (`/`) — 기록 카드 목록
- `LogDetailPage` (`/logs/:id`) — 상세 보기
- `LogFormPage` (`/logs/new`, `/logs/:id/edit`) — cafe/brew 탭 전환 폼

**Phase 1 완료 기준**
- 브라우저에서 카페 기록을 생성하고 목록에서 확인할 수 있다
- 기록을 수정하고 삭제할 수 있다
- brew 기록도 생성·조회 가능하다

---

## Phase 2 — 브루 기록 폼 + 목록 필터

> **목표**: 홈브루 기록에 특화된 폼(레시피 입력)을 완성하고, 목록 탐색 기능을 추가한다.

### 2-1. 브루 기록 전용 폼 UI

- `brew_method` 선택 (아이콘 또는 라벨 버튼)
- `brew_device` 자유 입력
- 레시피 섹션: 원두량, 물량, 온도, 시간, 분쇄도
- `brew_steps` 동적 입력 (항목 추가/삭제/위아래 버튼으로 한 칸씩 순서 이동)

### 2-2. 목록 필터 UI

- `log_type` 필터 탭 (전체 / 카페 / 브루)
- 날짜 범위 필터 (date_from, date_to)
- URL 쿼리 파라미터에 필터 상태 반영 (공유/북마크 가능)

### 2-3. 무한 스크롤 + Cursor-based Pagination

**프론트엔드**: 무한 스크롤 (`useInfiniteQuery` + `IntersectionObserver`)

**백엔드**: Cursor-based pagination

- `LIMIT/OFFSET` 대신 커서 방식 사용 — 목록 중간에 새 기록이 삽입돼도 중복/누락 없음
- 커서는 **불투명 커서(opaque cursor)** 로 설계: `{sort_field, sort_value, id}`를 base64 인코딩해 클라이언트에 전달
  - 클라이언트는 커서 내부 구조를 몰라도 됨
  - 정렬 기준이 바뀌어도 API 인터페이스(`cursor` 파라미터)는 그대로 유지됨
- 기본 정렬: `recorded_at DESC`
- 향후 정렬 기준·방향 확장을 위해 `sort_by`, `order` 쿼리 파라미터를 처음부터 포함
- 커서 내부에 `sort_by`와 `order`를 함께 인코딩 → 다음 페이지 요청 시 커서만 넘기면 정렬 기준이 유지됨

```
GET /api/v1/logs?sort_by=recorded_at&order=desc&cursor=<opaque>&limit=20
```

응답:
```json
{
  "items": [...],
  "next_cursor": "<opaque>",  // 마지막 페이지면 null
  "has_next": true
}
```

**Phase 2 완료 기준**
- 브루 기록을 레시피 포함 완전히 기록할 수 있다
- 타입/날짜 필터로 원하는 기록만 볼 수 있다

---

## Phase 3 — Tasting Tags 자동완성 + Companions 자동완성

> **목표**: 반복 입력을 줄이는 자동완성 UX를 추가한다.

### 3-1. 자동완성 API

```
GET /api/v1/suggestions/tags?q=<검색어>
GET /api/v1/suggestions/companions?q=<검색어>
```

- 해당 `user_id`의 모든 기록에서 `tasting_tags` / `companions` 집계
- 빈도 내림차순 정렬, 검색어로 prefix 필터링
- 응답: `{ "suggestions": string[] }`

### 3-2. 자동완성 컴포넌트 (`src/components/TagInput.tsx`)

- 텍스트 입력 + 드롭다운 추천 목록
- 선택 시 태그 뱃지로 추가, 삭제 가능
- `tasting_tags`, `companions` 두 필드에서 공용

**Phase 3 완료 기준**
- 태그 입력 시 이전에 사용한 태그가 자동완성으로 제안된다
- 동반자 입력 시 이전에 입력한 이름이 자동완성으로 제안된다

---

## Phase 4 — 계정 및 인증

> **목표**: 멀티유저 서비스로 전환. POC의 `X-User-Id` 헤더를 JWT 인증으로 교체한다.

### 4-1. 인증 API

```
POST /api/v1/auth/register   # 회원가입
POST /api/v1/auth/login      # 로그인 → JWT 발급
POST /api/v1/auth/refresh    # 토큰 갱신
POST /api/v1/auth/logout
```

### 4-2. Backend 변경

- `users` 테이블에 `email TEXT UNIQUE`, `password_hash TEXT` 컬럼 추가 (마이그레이션)
- JWT 미들웨어로 `X-User-Id` 헤더 미들웨어 교체
- 비밀번호 해싱: `bcrypt`

### 4-3. Frontend 변경

- 로그인/회원가입 페이지
- JWT를 `httpOnly cookie`에 저장 (`SameSite=Strict`)
- 미인증 시 로그인 페이지로 리다이렉트

**Phase 4 완료 기준**
- 사용자별 계정으로 로그인하고, 본인의 기록만 조회·수정·삭제할 수 있다

---

## 기술 스택 요약

| 영역 | 기술 | 버전 |
|------|------|------|
| Backend 언어 | Go | 1.22+ |
| HTTP 라우터 | chi | v5 |
| DB | SQLite | — |
| SQLite 드라이버 | mattn/go-sqlite3 | — |
| 쿼리 생성 | sqlc | v1 |
| DB 마이그레이션 | golang-migrate | v4 |
| Frontend 번들러 | Vite | 5.x |
| UI 프레임워크 | React | 18.x |
| 언어 | TypeScript | 5.x |
| 스타일 | Tailwind CSS | v3 |
| 서버 상태 | TanStack Query | v5 |
| 라우팅 | React Router | v6 |

---

## 로컬 실행 방법 (목표 상태)

```bash
# Backend
cd backend
go run ./cmd/server

# Frontend
cd frontend
npm run dev
```

- Backend: `http://localhost:8080`
- Frontend: `http://localhost:5173`

---

*Last updated: 2026-03-28 (초안)*
