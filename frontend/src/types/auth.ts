/**
 * openapi.yml → schema.ts (자동 생성) → 이 파일 순서로 타입이 흐른다.
 * 이 파일을 직접 편집하지 말고, openapi.yml을 수정한 뒤 `npm run generate`를 실행한다.
 */
import type { components } from './schema'

export type User = components['schemas']['UserResponse']
export type LoginRequest = components['schemas']['LoginRequest']
export type RegisterRequest = components['schemas']['RegisterRequest']
