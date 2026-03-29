/**
 * openapi.yml → schema.ts (자동 생성) → 이 파일 순서로 타입이 흐른다.
 * 이 파일을 직접 편집하지 말고, openapi.yml을 수정한 뒤 `npm run generate`를 실행한다.
 */
import type { components } from './schema'

// --- openapi.yml 스키마 타입 alias ---
// schema.ts의 경로 표현(components['schemas']['...'])을 짧게 재수출한다.

export type LogType = components['schemas']['LogType']
export type RoastLevel = components['schemas']['RoastLevel']
export type BrewMethod = components['schemas']['BrewMethod']
export type CafeDetail = components['schemas']['CafeDetail']
export type BrewDetail = components['schemas']['BrewDetail']

/** API 응답의 원시 커피 기록 타입 */
export type CoffeeLogResponse = components['schemas']['CoffeeLogResponse']

/** POST /api/v1/logs 요청 본문 */
export type CreateLogInput = components['schemas']['CreateLogRequest']

/** PUT /api/v1/logs/:id 요청 본문 */
export type UpdateLogInput = components['schemas']['UpdateLogRequest']

// --- TypeScript 전용 Discriminated Union ---
// OpenAPI 3.0 spec은 cafe/brew 필드를 모두 optional로 표현할 수밖에 없지만,
// 실제 응답은 log_type에 따라 반드시 하나만 존재한다.
// TypeScript 레벨에서 log_type을 좁히면 cafe/brew 접근을 타입 안전하게 처리한다.

export type CafeLogFull = Omit<CoffeeLogResponse, 'log_type' | 'cafe' | 'brew'> & {
  log_type: 'cafe'
  cafe: NonNullable<CoffeeLogResponse['cafe']>
  brew?: never
}

export type BrewLogFull = Omit<CoffeeLogResponse, 'log_type' | 'cafe' | 'brew'> & {
  log_type: 'brew'
  brew: NonNullable<CoffeeLogResponse['brew']>
  cafe?: never
}

export type CoffeeLogFull = CafeLogFull | BrewLogFull
