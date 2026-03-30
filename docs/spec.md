# Coffee of the Day — 서비스 명세서

> 이 문서는 도메인 규칙, 비즈니스 요구사항, API 동작 규칙의 기준입니다.
> 아키텍처는 `docs/arch/`를, API 상세는 `docs/openapi.yml`을 참조하세요.

---

## 1. 서비스 개요

**Coffee of the Day**는 내가 오늘 마신 커피를 기록하는 개인 로그 서비스입니다.

- 여러 사용자가 각자의 커피 기록을 관리합니다.
- **카페에서 주문한 커피**와 **직접 추출한 커피**를 구분하여 기록합니다.
- 누구와, 언제, 어디서, 어떤 커피를 마셨는지 추적합니다.
- 멀티유저 DB 구조를 기반으로 JWT 쿠키 인증을 사용합니다.

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

---

## 5. Companions 설계

**현재 결정**: 텍스트 배열로 저장. 프론트엔드에서 이전에 입력한 이름을 자동완성으로 제안.

> POC 이후: 사용자별 Companion 목록 테이블(`user_companions`)로 마이그레이션 가능.
> (서비스 가입자가 아니어도 등록 가능한 인물 목록)

---

## 6. 기능 요구사항

### 6.1 커피 기록 관리

| ID | 기능 | 설명 |
|----|------|------|
| F-01 | 기록 생성 | 카페 또는 브루 기록을 새로 작성한다 |
| F-02 | 기록 조회 (목록) | 날짜 역순으로 기록 목록을 조회한다 |
| F-03 | 기록 조회 (상세) | 특정 기록의 상세 내용을 조회한다 |
| F-04 | 기록 수정 | 작성된 기록을 수정한다 |
| F-05 | 기록 삭제 | 작성된 기록을 삭제한다 |

### 6.2 탐색 및 필터

| ID | 기능 | 설명 |
|----|------|------|
| F-06 | 날짜 필터 | 특정 날짜 또는 기간으로 기록을 필터링한다 |
| F-07 | 타입 필터 | `cafe` / `brew`로 필터링한다 |

### 6.3 자동완성

| ID | 기능 | 설명 |
|----|------|------|
| F-08 | 태그 자동완성 | 사용자의 이전 tasting_tags 기반으로 추천 |
| F-09 | 동반자 자동완성 | 사용자의 이전 companions 기반으로 추천 |

---

## 7. API 동작 규칙

API 엔드포인트와 스키마 상세는 `docs/openapi.yml`을 참조하세요.

- 날짜 형식: RFC3339 datetime (`2024-03-28T14:30:00+09:00`). date-only(`YYYY-MM-DD`)는 허용하지 않는다.
- 에러 응답: `{ "error": string, "field"?: string }` — `field`는 ValidationError일 때만 포함되며 오류가 발생한 필드 경로를 나타낸다 (예: `"cafe.cafe_name"`)
- 커서 기반 페이지네이션: `next_cursor`가 `null`이면 마지막 페이지. 커서 내부에 `sort_by`와 `order`가 인코딩되어 있어 다음 페이지 요청 시 커서만 전달하면 정렬 기준이 유지된다.
- 요청 바디는 중첩 Discriminated Union 구조: 공통 필드는 최상위에, 타입별 필드는 `cafe` 또는 `brew` 서브 객체로 분리. `log_type`이 판별자 역할.

---

## 8. UI 화면 구성

| 화면 | 경로 | 인증 | 설명 |
|------|------|------|------|
| 로그인 | `/login` | - | 로그인 |
| 회원가입 | `/register` | - | 회원가입 |
| 홈 (피드) | `/` | 필요 | 최근 커피 기록 목록, 타입/날짜 필터 |
| 기록 상세 | `/logs/:id` | 필요 | 기록 상세 보기 |
| 기록 작성 | `/logs/new` | 필요 | 신규 기록 작성 (cafe/brew 선택 후 폼 분기) |
| 기록 수정 | `/logs/:id/edit` | 필요 | 기록 수정 |

---

*Last updated: 2026-03-30 (v0.5 — 아키텍처/API 상세 분리, stale 항목 갱신)*
