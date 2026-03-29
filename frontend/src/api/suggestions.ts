import { request } from './client'
import type { components } from '../types/schema'

type SuggestionsResponse = components['schemas']['SuggestionsResponse']

// 자동완성 타입 — 현재는 tags와 companions만 지원
export type SuggestionType = 'tags' | 'companions'

export async function getSuggestions(type: SuggestionType, q: string): Promise<string[]> {
  const params = new URLSearchParams()
  if (q) params.set('q', q)
  const query = params.toString() ? `?${params.toString()}` : ''
  const res = await request<SuggestionsResponse>(`/suggestions/${type}${query}`)
  return res.suggestions
}
