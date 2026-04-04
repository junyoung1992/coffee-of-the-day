# Coffee of the Day — 서비스 명세서

> 도메인 규칙과 비즈니스 요구사항의 단일 소스.
> 아키텍처 → `docs/arch/`, API 스키마 → `docs/openapi.yml` 참조.

---

## 1. 서비스 개요

개인 커피 기록 서비스. 카페에서 주문한 커피와 직접 추출한 커피를 구분하여 기록한다.

- 멀티유저 DB 구조, JWT 쿠키 인증
- 현재 POC 단계 (단일 사용자 운용)

---

## 2. 핵심 개념: log_type

구분 기준은 **장소가 아니라 누가 추출했는가**.

| log_type | 의미 | 예시 |
|----------|------|------|
| `cafe` | 바리스타가 추출 | 카페에서 주문한 라떼, 핸드드립 |
| `brew` | 내가 직접 추출 | 집 V60, 사무실 에어로프레스 |

- `cafe`: 장소·메뉴 중심 기록
- `brew`: 도구·레시피·과정 중심 기록

---

## 3. 도메인 모델

### 3.1 사용자 (User)

| 필드 | 타입 | 필수 | 설명 |
|------|------|:----:|------|
| id | uuid | ✓ | 고유 식별자 |
| username | string | ✓ | 사용자명 (유니크) |
| display_name | string | ✓ | 표시 이름 |
| created_at | datetime | ✓ | 계정 생성 시각 |

### 3.2 커피 기록 공통 (CoffeeLog)

| 필드 | 타입 | 필수 | 설명 |
|------|------|:----:|------|
| id | uuid | ✓ | 고유 식별자 |
| user_id | uuid | ✓ | 작성자 (FK → users) |
| recorded_at | datetime | ✓ | 커피를 마신 일시 |
| log_type | enum | ✓ | `cafe` \| `brew` |
| companions | string[] | | 함께한 사람들 (없으면 혼자) |
| memo | string | | 자유 메모 |
| created_at | datetime | ✓ | 레코드 생성 시각 |
| updated_at | datetime | ✓ | 레코드 수정 시각 |

### 3.3 카페 기록 (CafeLog)

`coffee_logs`와 1:1. `log_id`가 FK이자 PK.

| 필드 | 타입 | 필수 | 설명 |
|------|------|:----:|------|
| log_id | uuid | ✓ | FK → coffee_logs.id |
| cafe_name | string | ✓ | 카페 이름 |
| coffee_name | string | ✓ | 주문한 커피 이름 |
| rating | real(0.5–5.0) | | 0.5 단위 평가 |
| location | string | | 카페 위치/주소 |
| bean_origin | string | | 원두 원산지 |
| bean_process | string | | 가공 방식 (워시드, 내추럴 등) |
| roast_level | RoastLevel | | 로스팅 단계 |
| tasting_tags | string[] | | 구조화된 테이스팅 태그 |
| tasting_note | string | | 자유 서술형 테이스팅 노트 |
| impressions | string | | 전반적인 인상·감상 |

### 3.4 브루 기록 (BrewLog)

`coffee_logs`와 1:1. `log_id`가 FK이자 PK.

| 필드 | 타입 | 필수 | 설명 |
|------|------|:----:|------|
| log_id | uuid | ✓ | FK → coffee_logs.id |
| bean_name | string | ✓ | 원두 이름/상품명 |
| brew_method | BrewMethod | ✓ | 추출 방식 분류 |
| rating | real(0.5–5.0) | | 0.5 단위 평가 |
| bean_origin | string | | 원두 원산지 |
| bean_process | string | | 가공 방식 |
| roast_level | RoastLevel | | 로스팅 단계 |
| roast_date | date | | 로스팅 날짜 (신선도 파악용) |
| brew_device | string | | 구체적 도구명 (예: "Origami") |
| coffee_amount_g | real | | 원두 사용량 (g) |
| water_amount_ml | real | | 물 사용량 (ml) |
| water_temp_c | real | | 물 온도 (°C) |
| brew_time_sec | int | | 총 추출 시간 (초) |
| grind_size | string | | 분쇄도 (예: "Comandante 20클릭") |
| brew_steps | string[] | | 추출 절차 단계별 메모 |
| tasting_tags | string[] | | 구조화된 테이스팅 태그 |
| tasting_note | string | | 자유 서술형 테이스팅 노트 |
| impressions | string | | 전반적인 인상·감상 |

### 3.5 Enum 정의

**RoastLevel**

| 값 | 설명 |
|----|------|
| `light` | 라이트 로스트 |
| `medium` | 미디엄 로스트 |
| `dark` | 다크 로스트 |

**BrewMethod** — 추출 방식(물리적 메커니즘) 기준 분류. 구체적 도구명은 `brew_device`에 기록.

| 값 | 설명 | 예시 도구 |
|----|------|-----------|
| `pour_over` | 중력 + 필터 투과 | V60, Chemex, Origami, Kalita Wave |
| `immersion` | 침지(steeping) | French Press, Clever Dripper |
| `aeropress` | 압력 + 침지 + 필터 | AeroPress |
| `espresso` | 고압 추출 | 에스프레소 머신 |
| `moka_pot` | 스토브탑 증기압 | 모카포트 |
| `siphon` | 진공 사이폰 | 사이폰 |
| `cold_brew` | 저온 장시간 침지 | 콜드브루어 |
| `other` | 기타 | — |

---

## 4. 자동완성 규칙

### Tasting Tags

- 별도 정규화 테이블 없이, 해당 사용자의 과거 `tasting_tags`를 빈도순 집계하여 추천
- 프론트엔드: 입력 시 자동완성 드롭다운

### Companions

- 텍스트 배열로 저장. 해당 사용자의 과거 입력값을 자동완성으로 제안.
- POC 이후 `user_companions` 테이블로 마이그레이션 가능

---

## 5. API 동작 규칙

엔드포인트 및 스키마 상세는 `docs/openapi.yml` 참조.

| 규칙 | 상세 |
|------|------|
| 날짜 형식 | RFC3339 datetime만 허용. date-only(`YYYY-MM-DD`) 불가 |
| 에러 응답 | `{ "error": string, "field"?: string }` — `field`는 ValidationError 시 오류 필드 경로 (예: `"cafe.cafe_name"`) |
| 페이지네이션 | 커서 기반. `next_cursor`가 `null`이면 마지막 페이지. 커서에 `sort_by`·`order` 인코딩됨 |
| 요청 구조 | 중첩 Discriminated Union — 공통 필드는 최상위, 타입별 필드는 `cafe`/`brew` 서브 객체. `log_type`이 판별자 |

---

## 6. UI 화면 및 동작 규칙

### 6.1 화면 목록

| 화면 | 경로 | 인증 | 설명 |
|------|------|:----:|------|
| 로그인 | `/login` | | 로그인 |
| 회원가입 | `/register` | | 회원가입 |
| 홈 (피드) | `/` | ✓ | 최근 커피 기록 목록, 타입/날짜 필터 |
| 기록 상세 | `/logs/:id` | ✓ | 기록 상세 보기 |
| 기록 작성 | `/logs/new` | ✓ | 신규 기록 작성 |
| 기록 수정 | `/logs/:id/edit` | ✓ | 기록 수정 |

### 6.2 로그 작성/수정 폼

단일 폼 컴포넌트가 create/edit을 모두 처리한다. `log_type` 선택에 따라 cafe/brew 필드가 분기된다.

**폼 영역 구분**

폼은 "필수 영역"(항상 노출)과 "선택 영역"(토글로 접기)으로 나뉜다.

Cafe 필수 영역:
- `cafe_name`, `coffee_name`, `rating`

Cafe 선택 영역:
- `location`, `bean_origin`, `bean_process`, `roast_level`
- `tasting_tags`, `tasting_note`, `impressions`
- `companions`, `memo` (공통 필드)

Brew 필수 영역:
- `bean_name`, `brew_method`, `rating`

Brew 선택 영역:
- `bean_origin`, `bean_process`, `roast_level`, `roast_date`
- `brew_device`, `coffee_amount_g`, `water_amount_ml`, `water_temp_c`, `brew_time_sec`, `grind_size`
- `brew_steps`
- `tasting_tags`, `tasting_note`, `impressions`
- `companions`, `memo` (공통 필드)

**동작 규칙:**
- 선택 영역은 기본적으로 접힌 상태
- "더 기록하기" 탭으로 펼침
- 수정 모드 진입 시, 선택 영역에 값이 하나라도 있으면 자동으로 펼쳐진 상태로 표시
- 접힌 상태에서도 선택 필드의 기존 값은 유지 (접는다고 삭제되지 않음)
- `recorded_at`은 필수 영역에 항상 노출 (API required)
- `log_type`은 작성 시에만 선택 가능, 수정 시 변경 불가

---

*Last updated: 2026-04-04 (v0.6 — AI agent 친화적 구조 개편, 폼 영역 구분 추가)*
