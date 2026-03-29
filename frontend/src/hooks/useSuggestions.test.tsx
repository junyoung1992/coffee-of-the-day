import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useTagSuggestions, useCompanionSuggestions } from './useSuggestions'
import * as suggestionsApi from '../api/suggestions'

vi.mock('../api/suggestions')

const mockGetSuggestions = vi.mocked(suggestionsApi.getSuggestions)

// debounce 타이밍 검증은 useDebounce.test.ts에서 완료했다.
// 이 테스트는 enabled 조건과 API 호출 인수에만 집중한다.
// useDebounce는 초기값을 즉시 반환하므로, 초기 q값으로 enabled 여부를 검증할 수 있다.

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  })
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('useTagSuggestions', () => {
  beforeEach(() => {
    mockGetSuggestions.mockResolvedValue(['초콜릿', '체리'])
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('빈 검색어일 때는 API를 호출하지 않는다', () => {
    const { result } = renderHook(() => useTagSuggestions(''), { wrapper: makeWrapper() })

    // enabled: false이면 fetchStatus가 idle로 남는다.
    expect(result.current.fetchStatus).toBe('idle')
    expect(mockGetSuggestions).not.toHaveBeenCalled()
  })

  it('검색어가 있으면 tags 타입으로 API를 호출한다', async () => {
    renderHook(() => useTagSuggestions('초'), { wrapper: makeWrapper() })

    await waitFor(() => expect(mockGetSuggestions).toHaveBeenCalledWith('tags', '초'))
  })

  it('API 응답을 data로 반환한다', async () => {
    const { result } = renderHook(() => useTagSuggestions('초'), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.data).toEqual(['초콜릿', '체리']))
  })
})

describe('useCompanionSuggestions', () => {
  beforeEach(() => {
    mockGetSuggestions.mockResolvedValue(['지수', '민준'])
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('빈 검색어일 때는 API를 호출하지 않는다', () => {
    const { result } = renderHook(() => useCompanionSuggestions(''), { wrapper: makeWrapper() })

    expect(result.current.fetchStatus).toBe('idle')
    expect(mockGetSuggestions).not.toHaveBeenCalled()
  })

  it('검색어가 있으면 companions 타입으로 API를 호출한다', async () => {
    renderHook(() => useCompanionSuggestions('지'), { wrapper: makeWrapper() })

    await waitFor(() => expect(mockGetSuggestions).toHaveBeenCalledWith('companions', '지'))
  })
})
