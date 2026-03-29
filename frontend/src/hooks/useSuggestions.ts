import { useQuery } from '@tanstack/react-query'
import { getSuggestions } from '../api/suggestions'

// 자동완성은 입력 중 반응형으로 동작해야 하므로 staleTime을 짧게 유지한다.
// 단, 동일한 q 값이면 캐시를 재사용해 불필요한 요청을 줄인다.
const SUGGESTIONS_STALE_TIME = 30_000 // 30초

function useSuggestions(type: 'tags' | 'companions', q: string) {
  return useQuery({
    queryKey: ['suggestions', type, q],
    queryFn: () => getSuggestions(type, q),
    staleTime: SUGGESTIONS_STALE_TIME,
    // 입력이 완전히 비어있을 때는 요청하지 않는다
    enabled: q.length > 0,
  })
}

export function useTagSuggestions(q: string) {
  return useSuggestions('tags', q)
}

export function useCompanionSuggestions(q: string) {
  return useSuggestions('companions', q)
}
