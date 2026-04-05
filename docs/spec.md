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

### 공통

- **최소 입력 길이**: `q` 파라미터가 빈 문자열이거나 1자 미만이면 빈 배열 `[]`을 반환한다. 쿼리를 실행하지 않는다.

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

### 6.3 로그 복제

기존 로그를 기반으로 새 로그를 생성한다. 기존 create API를 그대로 사용.

**진입점:**
- 로그 상세 화면: "이 기록으로 다시 쓰기" 버튼
- 로그 목록 카드: 카드 액션 메뉴(⋯)에 "복제" 항목

**복제 필드 규칙:**

| 구분 | 필드 | 동작 |
|------|------|------|
| 복제 | log_type, cafe/brew 전용 필드, tasting_tags | 원본 값 유지 |
| 리셋 | recorded_at | 오늘 날짜로 초기화 |
| 리셋 | rating, memo, companions | 빈 값으로 초기화 |

**동작 규칙:**
- 복제된 데이터가 초기값으로 채워진 작성 폼이 열림
- 선택 영역에 복제된 값이 있으면 자동으로 펼쳐진 상태
- 저장 시 새 로그 생성 (원본 변경 없음)

### 6.4 브루 레시피 불러오기

브루 로그 작성 시 이전 브루 로그의 레시피 필드만 선택적으로 불러온다.
동일 원두로 파라미터를 미세 조정하는 홈브루잉 워크플로우를 지원한다.

**진입점:**
- 브루 로그 작성 폼의 "레시피 불러오기" 섹션에서 "이전 레시피 불러오기" 버튼
- 새 로그 작성 모드에서만 표시 (수정 모드, 복제 모드에서는 숨김)

**필드 채움 규칙:**

| 구분 | 필드 | 동작 |
|------|------|------|
| 채움 | bean_name, brew_method, brew_device, coffee_amount_g, water_amount_ml, water_temp_c, brew_time_sec, grind_size, brew_steps | 원본 값으로 채움 |
| 채움 | bean_origin, bean_process, roast_level, roast_date | 원본 값으로 채움 |
| 리셋 | recorded_at | 현재 시각으로 초기화 |
| 리셋 | rating, tasting_tags, tasting_note, impressions, memo, companions | 빈 값으로 초기화 |

**동작 규칙:**
- 버튼 클릭 시 모달에 최근 brew 로그 목록을 최신순으로 표시 (원두 이름 + 추출 방식 + 날짜)
- 로그 선택 시 레시피 필드가 채워지고, 선택 영역이 자동으로 펼쳐짐
- 불러온 레시피를 자유롭게 수정할 수 있음
- 백엔드 변경 없음 (기존 로그 조회 API 사용)

---

## 7. 즐겨찾기 프리셋

자주 반복하는 카페+메뉴 또는 원두+추출방식 조합을 프리셋으로 저장하고, 새 로그 작성 시 프리셋을 선택하면 관련 필드가 자동으로 채워진다.

### 7.1 프리셋 데이터 모델

프리셋은 `log_type`에 따라 cafe/brew로 구분된다. 로그와 독립적인 엔티티.

**공통 필드:**

| 필드 | 타입 | 필수 | 설명 |
|------|------|:----:|------|
| id | uuid | ✓ | 고유 식별자 |
| user_id | uuid | ✓ | 소유자 (FK → users) |
| name | string | ✓ | 프리셋 이름 (예: "출근길 아메리카노") |
| log_type | enum | ✓ | `cafe` \| `brew` |
| last_used_at | datetime | | 마지막 사용 시각 (정렬용) |
| created_at | datetime | ✓ | 레코드 생성 시각 |
| updated_at | datetime | ✓ | 레코드 수정 시각 |

**Cafe 프리셋 (CafePresetDetail):**

| 필드 | 타입 | 필수 | 설명 |
|------|------|:----:|------|
| cafe_name | string | ✓ | 카페 이름 |
| coffee_name | string | ✓ | 커피 이름 |
| tasting_tags | string[] | | 테이스팅 태그 |

**Brew 프리셋 (BrewPresetDetail):**

| 필드 | 타입 | 필수 | 설명 |
|------|------|:----:|------|
| bean_name | string | ✓ | 원두 이름 |
| brew_method | BrewMethod | ✓ | 추출 방식 |
| recipe_detail | string | | 레시피 상세 (자유 서술) |
| brew_steps | string[] | | 추출 절차 단계별 메모 |

### 7.2 프리셋 진입점 및 흐름

**프리셋으로 기록 작성:**
1. 새 로그 작성 화면 진입
2. 상단에 저장된 프리셋 목록이 표시됨 (최근 사용순 정렬)
3. 프리셋 선택 → 필드 자동 채움 (복제와 동일한 방식으로 폼 초기값 설정)
4. 별점, 메모 등 나머지 입력 → 저장
5. 프리셋 사용 시 `last_used_at` 갱신

**프리셋 등록 (로그에서):**
1. 기존 로그 상세 화면에서 "프리셋으로 저장" 버튼
2. 프리셋 이름 입력 (모달 또는 인라인)
3. 저장 → 프리셋 목록에 추가

**프리셋 관리:**
- 별도 관리 화면 (`/presets`)에서 목록 조회, 수정, 삭제
- 목록은 최근 사용순 정렬 (last_used_at DESC, 미사용 프리셋은 created_at 기준)

### 7.3 프리셋 필드 채움 규칙

| 구분 | 필드 | 동작 |
|------|------|------|
| 채움 | log_type, cafe/brew 전용 필드 | 프리셋 값으로 설정 |
| 리셋 | recorded_at | 오늘 날짜로 초기화 |
| 리셋 | rating, memo, companions, impressions | 빈 값으로 초기화 |

### 7.4 프리셋 API

엔드포인트 및 스키마 상세는 `docs/openapi.yml` 참조.

| 엔드포인트 | 메서드 | 설명 |
|-----------|--------|------|
| `/api/v1/presets` | POST | 프리셋 생성 |
| `/api/v1/presets` | GET | 프리셋 목록 조회 (최근 사용순) |
| `/api/v1/presets/{id}` | GET | 프리셋 단건 조회 |
| `/api/v1/presets/{id}` | PUT | 프리셋 수정 |
| `/api/v1/presets/{id}` | DELETE | 프리셋 삭제 |
| `/api/v1/presets/{id}/use` | POST | 프리셋 사용 기록 (last_used_at 갱신) |

### 7.5 UI 화면

| 화면 | 경로 | 인증 | 설명 |
|------|------|:----:|------|
| 프리셋 관리 | `/presets` | ✓ | 프리셋 목록, 수정, 삭제 |

---

*Last updated: 2026-04-05 (v0.10 — 브루 레시피 불러오기 기능 추가)*
