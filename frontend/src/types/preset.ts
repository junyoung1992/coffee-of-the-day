/**
 * openapi.yml → schema.ts (자동 생성) → 이 파일 순서로 타입이 흐른다.
 * 이 파일을 직접 편집하지 말고, openapi.yml을 수정한 뒤 `npm run generate`를 실행한다.
 */
import type { components } from './schema'

// --- openapi.yml 스키마 타입 alias ---

export type CafePresetDetail = components['schemas']['CafePresetDetail']
export type BrewPresetDetail = components['schemas']['BrewPresetDetail']

/** API 응답의 원시 프리셋 타입 */
export type PresetResponse = components['schemas']['PresetResponse']

/** POST /api/v1/presets 요청 본문 */
export type CreatePresetInput = components['schemas']['CreatePresetRequest']

/** PUT /api/v1/presets/:id 요청 본문 */
export type UpdatePresetInput = components['schemas']['UpdatePresetRequest']

// --- TypeScript 전용 Discriminated Union ---
// log_type에 따라 cafe/brew 중 하나만 존재한다.

export type CafePresetFull = Omit<PresetResponse, 'log_type' | 'cafe' | 'brew'> & {
  log_type: 'cafe'
  cafe: NonNullable<PresetResponse['cafe']>
  brew?: never
}

export type BrewPresetFull = Omit<PresetResponse, 'log_type' | 'cafe' | 'brew'> & {
  log_type: 'brew'
  brew: NonNullable<PresetResponse['brew']>
  cafe?: never
}

export type PresetFull = CafePresetFull | BrewPresetFull
