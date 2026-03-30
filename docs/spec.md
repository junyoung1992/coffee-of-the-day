# Coffee of the Day — 서비스 명세서

> 이 문서는 SDD(Spec-Driven Development) 기반으로 작성된 요구사항 명세입니다.
> 구현 전 합의된 스펙으로서, 도메인 모델·API·UI 흐름의 기준이 됩니다.

---

## 1. 서비스 개요

**Coffee of the Day**는 내가 오늘 마신 커피를 기록하는 개인 로그 서비스입니다.

- 여러 사용자가 각자의 커피 기록을 관리합니다.
- **카페에서 주문한 커피**와 **직접 추출한 커피**를 구분하여 기록합니다.
- 누구와, 언제, 어디서, 어떤 커피를 마셨는지 추적합니다.
- POC 단계에서는 인증 없이 로컬 환경 실행을 목표로 하되, 멀티유저 DB 구조를 처음부터 설계합니다.
- POC 이후 계정(인증) 기능을 추가합니다.

---

## 2. 핵심 개념

### log_type: `cafe` vs `brew`

구분의 기준은 **장소(카페 vs 집)가 아니라, 누가 추출했는가**입니다.

| 상황 | log_type |
|------|----------|
| 카페에서 바리스타가 만들어준 라떼 | `cafe` |
| 카페에서 주문한 핸드드립 | `cafe` |
| 집에서 V60으로 내린 드립 | `brew` |
| 집에서 에스프레소 머신으로 추출 | `brew` |
| 사무실에서 에어로프레스로 추출 | `brew` |

- `cafe`: 카페를 방문해 바리스타가 만든 커피를 마신 기록. 장소·메뉴 중심.
- `brew`: 내가 직접 추출한 커피 기록. 도구·레시피·과정 중심.

---

## 3. 도메인 모델

### 3.1 사용자 (User)

| 필드 | 타입 | 설명 |
|------|------|------|
| id | uuid | 고유 식별자 |
| username | string | 사용자명 (유니크) |
| display_name | string | 표시 이름 |
| created_at | datetime | 계정 생성 시각 |

> POC에서는 인증 없이 `X-User-Id` 헤더로 user_id를 전달합니다.
> 계정 기능 추가 시 `email`, `password_hash` 등을 이 테이블에 확장합니다.

### 3.2 커피 기록 공통 (CoffeeLog)

| 필드 | 타입 | 설명 |
|------|------|------|
| id | uuid | 고유 식별자 |
| user_id | uuid | 작성자 (FK → users) |
| recorded_at | datetime | 커피를 마신 일시 |
| companions | string[] | 함께한 사람들 (없으면 혼자) |
| log_type | enum | `cafe` \| `brew` |
| memo | string? | 자유 메모 |
| created_at | datetime | 레코드 생성 시각 |
| updated_at | datetime | 레코드 수정 시각 |

### 3.3 카페 기록 (CafeLog) — `cafe_logs` 테이블

`coffee_logs`와 1:1 관계. `log_id`가 FK이자 PK.

| 필드 | 타입 | 설명 |
|------|------|------|
| log_id | uuid | FK → coffee_logs.id (PK 겸용) |
| cafe_name | string | 카페 이름 |
| location | string? | 카페 위치/주소 |
| coffee_name | string | 주문한 커피 이름 (예: 아메리카노, 플랫화이트) |
| bean_origin | string? | 원두 원산지 (예: 에티오피아 예가체프) |
| bean_process | string? | 가공 방식 (예: 워시드, 내추럴, 허니) |
| roast_level | enum? | `light` \| `medium` \| `dark` |
| tasting_tags | string[] | 구조화된 테이스팅 태그 (예: ["초콜릿", "체리"]) |
| tasting_note | string? | 자유 서술형 테이스팅 노트 |
| impressions | string? | 전반적인 인상·감상 (맛 외의 분위기, 경험 등) |
| rating | real? | 0.5 단위 평가 (0.5 – 5.0) |

### 3.4 브루 기록 (BrewLog) — `brew_logs` 테이블

`coffee_logs`와 1:1 관계. `log_id`가 FK이자 PK.

| 필드 | 타입 | 설명 |
|------|------|------|
| log_id | uuid | FK → coffee_logs.id (PK 겸용) |
| bean_name | string | 원두 이름/상품명 |
| bean_origin | string? | 원두 원산지 |
| bean_process | string? | 가공 방식 |
| roast_level | enum? | `light` \| `medium` \| `dark` |
| roast_date | date? | 로스팅 날짜 (신선도 파악용) |
| tasting_tags | string[] | 구조화된 테이스팅 태그 |
| tasting_note | string? | 자유 서술형 테이스팅 노트 |
| brew_method | enum | 추출 방식 분류 (아래 참조) |
| brew_device | string? | 구체적인 도구명 (예: "Origami", "Kalita Wave 155") |
| coffee_amount_g | real? | 원두 사용량 (g) |
| water_amount_ml | real? | 물 사용량 (ml) |
| water_temp_c | real? | 물 온도 (°C) |
| brew_time_sec | int? | 총 추출 시간 (초) |
| grind_size | string? | 분쇄도 (예: "중간", "Comandante 20클릭") |
| brew_steps | string[] | 추출 절차 단계별 메모 |
| impressions | string? | 전반적인 인상·감상 |
| rating | real? | 0.5 단위 평가 (0.5 – 5.0) |

**brew_method 분류** — 추출 방식(물리적 메커니즘) 기준

| 값 | 설명 | 예시 도구 |
|----|------|-----------|
| `pour_over` | 중력 + 필터 투과 (핸드드립 계열) | V60, Chemex, Origami, Kalita Wave, Melitta |
| `immersion` | 침지(steeping) | French Press, Clever Dripper |
| `aeropress` | 압력 + 침지 + 필터 (독자적 방식) | AeroPress |
| `espresso` | 고압 추출 | 에스프레소 머신 |
| `moka_pot` | 스토브탑 증기압 | 모카포트 |
| `siphon` | 진공 사이폰 | 사이폰 |
| `cold_brew` | 저온 장시간 침지 | 콜드브루어 |
| `other` | 기타 | — |

> 구체적인 도구명은 `brew_device` 자유 입력 필드에 기록합니다.

---

## 4. Tasting Tags 자동완성

- 태그는 별도 정규화 테이블 없이 사용자가 과거에 입력한 태그에서 추천합니다.
- 백엔드: 해당 `user_id`의 모든 `tasting_tags` 배열을 집계하여 빈도 순으로 반환하는 엔드포인트 제공.
- 프론트엔드: 입력 시 자동완성 드롭다운으로 제안.

```
GET /api/v1/suggestions/tags?q=<검색어>
```

---

## 5. Companions 설계

**현재 결정**: 텍스트 배열로 저장. 프론트엔드에서 이전에 입력한 이름을 자동완성으로 제안.

```
GET /api/v1/suggestions/companions?q=<검색어>
```

> POC 이후: 사용자별 Companion 목록 테이블(`user_companions`)로 마이그레이션 가능.
> (서비스 가입자가 아니어도 등록 가능한 인물 목록)

---

## 6. DB 테이블 구조 (ERD 요약)

```
users
  id, username, display_name, created_at

coffee_logs
  id, user_id → users, recorded_at, companions[], log_type,
  memo, created_at, updated_at

cafe_logs                               brew_logs
  log_id → coffee_logs (1:1)              log_id → coffee_logs (1:1)
  cafe_name, location                     bean_name, bean_origin
  coffee_name                             bean_process, roast_level, roast_date
  bean_origin, bean_process               tasting_tags[], tasting_note
  roast_level                             brew_method, brew_device
  tasting_tags[], tasting_note            coffee_amount_g, water_amount_ml
  impressions                             water_temp_c, brew_time_sec
  rating (real)                           grind_size, brew_steps[]
                                          impressions
                                          rating (real)
```

---

## 7. 기능 요구사항

### 7.1 커피 기록 관리

| ID | 기능 | 설명 |
|----|------|------|
| F-01 | 기록 생성 | 카페 또는 브루 기록을 새로 작성한다 |
| F-02 | 기록 조회 (목록) | 날짜 역순으로 기록 목록을 조회한다 |
| F-03 | 기록 조회 (상세) | 특정 기록의 상세 내용을 조회한다 |
| F-04 | 기록 수정 | 작성된 기록을 수정한다 |
| F-05 | 기록 삭제 | 작성된 기록을 삭제한다 |

### 7.2 탐색 및 필터

| ID | 기능 | 설명 |
|----|------|------|
| F-06 | 날짜 필터 | 특정 날짜 또는 기간으로 기록을 필터링한다 |
| F-07 | 타입 필터 | `cafe` / `brew`로 필터링한다 |

### 7.3 자동완성

| ID | 기능 | 설명 |
|----|------|------|
| F-08 | 태그 자동완성 | 사용자의 이전 tasting_tags 기반으로 추천 |
| F-09 | 동반자 자동완성 | 사용자의 이전 companions 기반으로 추천 |

### 7.4 POC 제외 범위

- 인증/로그인 (계정 기능은 POC 이후 Phase에서 구현)
- 관리자(Admin) UI
- 이미지 업로드
- Companion 목록 관리 (자동완성으로 대체)

---

## 8. 아키텍처

### 8.1 전체 구조

```
coffee-of-the-day/
├── backend/          # Go API 서버
├── frontend/         # TypeScript Web 앱
└── spec.md
```

### 8.2 Backend (Go)

- **언어**: Go 1.22+
- **프레임워크**: `net/http` + `chi` 라우터
- **DB**: SQLite (로컬 POC, 파일 기반)
- **쿼리**: `sqlc` (SQL → Go 타입 자동 생성)
- **마이그레이션**: `golang-migrate`
- **아키텍처 패턴**: Layered Architecture (handler → service → repository)

```
backend/
├── cmd/server/
├── internal/
│   ├── handler/
│   ├── service/
│   ├── repository/
│   └── domain/
├── db/
│   ├── migrations/
│   └── queries/
└── config/
```

### 8.3 Frontend (TypeScript)

- **언어**: TypeScript
- **프레임워크**: React + Vite
- **스타일**: Tailwind CSS
- **서버 상태**: TanStack Query
- **라우팅**: React Router v6

```
frontend/
├── src/
│   ├── pages/
│   ├── components/
│   ├── api/
│   ├── types/
│   └── hooks/
```

### 8.4 통신

- **프로토콜**: REST API (JSON)
- **Base URL (local)**: `http://localhost:8080/api/v1`
- **CORS**: `localhost:5173` 허용
- **사용자 식별 (POC)**: `X-User-Id` 헤더 (인증 추가 시 JWT로 교체)

---

## 9. API 명세

### 공통

- 날짜 형식: RFC3339 datetime (`2024-03-28T14:30:00+09:00`). date-only(`YYYY-MM-DD`)는 허용하지 않는다.
- 에러 응답: `{ "error": string, "field"?: string }` — `field`는 ValidationError일 때만 포함되며 오류가 발생한 필드 경로를 나타낸다 (예: `"cafe.cafe_name"`)
- 인증 헤더 (POC): `X-User-Id: <uuid>`

### 엔드포인트

```
# 커피 기록
GET    /api/v1/logs              # 목록 조회
POST   /api/v1/logs              # 기록 생성
GET    /api/v1/logs/:id          # 단건 조회
PUT    /api/v1/logs/:id          # 수정
DELETE /api/v1/logs/:id          # 삭제

# 자동완성
GET    /api/v1/suggestions/tags?q=<검색어>
GET    /api/v1/suggestions/companions?q=<검색어>
```

### 목록 조회 쿼리 파라미터

| 파라미터 | 타입 | 설명 |
|----------|------|------|
| log_type | string? | `cafe` \| `brew` |
| date_from | date? | `YYYY-MM-DD` |
| date_to | date? | `YYYY-MM-DD` |
| sort_by | string? | 정렬 기준 필드. 기본값: `recorded_at` |
| order | string? | 정렬 방향 `asc` \| `desc`. 기본값: `desc` |
| cursor | string? | 불투명 커서 (이전 응답의 `next_cursor` 값) |
| limit | int? | 페이지당 개수. 기본값: 20 |

### 목록 조회 응답

```json
{
  "items": [...],
  "next_cursor": "<opaque base64>",
  "has_next": true
}
```

> `next_cursor`가 `null`이면 마지막 페이지입니다.
> 커서 내부에 `sort_by`와 `order`가 인코딩되어 있어, 다음 페이지 요청 시 커서만 전달하면 정렬 기준이 유지됩니다.

### 요청 바디 구조 — 중첩 Discriminated Union

공통 필드는 최상위에, 타입별 필드는 서브 객체로 분리합니다.
`log_type`이 판별자(discriminant) 역할을 하며, 해당 타입의 서브 객체만 유효합니다.

```jsonc
// 카페 기록 생성
{
  "log_type": "cafe",
  "recorded_at": "2024-03-28T14:30:00+09:00",
  "companions": ["지수", "민준"],
  "memo": "오랜만에 방문",
  "cafe": {
    "cafe_name": "블루보틀 삼청",
    "coffee_name": "싱글오리진 드립",
    "bean_origin": "에티오피아 예가체프",
    "bean_process": "내추럴",
    "roast_level": "light",
    "tasting_tags": ["블루베리", "플로럴", "밝은 산미"],
    "tasting_note": "첫 모금에 블루베리잼 같은 달콤함이 느껴졌고, 식을수록 꽃향기가 더 살아났다.",
    "impressions": "조용한 오후에 잘 어울리는 커피.",
    "rating": 4.5
  }
}

// 브루 기록 생성 (집 에스프레소)
{
  "log_type": "brew",
  "recorded_at": "2024-03-28T08:00:00+09:00",
  "companions": [],
  "brew": {
    "bean_name": "커피리브레 에티오피아 구지",
    "roast_level": "light",
    "roast_date": "2024-03-20",
    "tasting_tags": ["복숭아", "재스민"],
    "tasting_note": "패키지 노트보다 훨씬 달콤했다.",
    "brew_method": "espresso",
    "brew_device": "Breville Barista Express",
    "coffee_amount_g": 18,
    "water_amount_ml": 36,
    "water_temp_c": 93,
    "brew_time_sec": 28,
    "grind_size": "fine (EK43 기준 3.5)",
    "brew_steps": [
      "포터필터 예열",
      "탬핑 후 추출 시작",
      "28초에 36ml 추출 완료"
    ],
    "impressions": "비율이 딱 맞았다. 다음엔 온도를 92도로 낮춰보고 싶다.",
    "rating": 4.0
  }
}
```

---

## 10. UI 화면 구성

| 화면 | 경로 | 설명 |
|------|------|------|
| 홈 (피드) | `/` | 최근 커피 기록 목록, 타입/날짜 필터 |
| 기록 상세 | `/logs/:id` | 기록 상세 보기 |
| 기록 작성 | `/logs/new` | 신규 기록 작성 (cafe/brew 선택 후 폼 분기) |
| 기록 수정 | `/logs/:id/edit` | 기록 수정 |

---

*Last updated: 2026-03-29 (v0.4 — 에러 응답 형식 실제 구현 반영, recorded_at RFC3339 datetime-only 확정)*
