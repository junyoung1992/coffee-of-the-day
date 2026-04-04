import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import {
  usePresetList,
  usePreset,
  useCreatePreset,
  useUpdatePreset,
  useDeletePreset,
  useUsePreset,
} from './usePresets'
import * as presetsApi from '../api/presets'

vi.mock('../api/presets')

const mockGetPresets = vi.mocked(presetsApi.getPresets)
const mockGetPreset = vi.mocked(presetsApi.getPreset)
const mockCreatePreset = vi.mocked(presetsApi.createPreset)
const mockUpdatePreset = vi.mocked(presetsApi.updatePreset)
const mockDeletePreset = vi.mocked(presetsApi.deletePreset)
const mockUsePresetApi = vi.mocked(presetsApi.usePresetApi)

// 각 테스트마다 새 QueryClient를 생성하여 캐시가 오염되지 않도록 한다
function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

const sampleCafePreset = {
  id: 'preset-1',
  user_id: 'user-1',
  name: '출근길 아메리카노',
  log_type: 'cafe' as const,
  last_used_at: null,
  created_at: '2026-04-01T00:00:00Z',
  updated_at: '2026-04-01T00:00:00Z',
  cafe: {
    cafe_name: '블루보틀',
    coffee_name: '싱글 오리진',
    tasting_tags: ['fruity'],
  },
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('usePresetList', () => {
  it('프리셋 목록을 조회한다', async () => {
    mockGetPresets.mockResolvedValue({ items: [sampleCafePreset] })

    const { result } = renderHook(() => usePresetList(), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toHaveLength(1)
    expect(result.current.data?.[0].id).toBe('preset-1')
  })

  it('빈 목록을 처리한다', async () => {
    mockGetPresets.mockResolvedValue({ items: [] })

    const { result } = renderHook(() => usePresetList(), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toHaveLength(0)
  })
})

describe('usePreset', () => {
  it('id로 단건을 조회한다', async () => {
    mockGetPreset.mockResolvedValue(sampleCafePreset)

    const { result } = renderHook(() => usePreset('preset-1'), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toEqual(sampleCafePreset)
    expect(mockGetPreset).toHaveBeenCalledWith('preset-1')
  })

  it('id가 빈 문자열이면 쿼리를 실행하지 않는다', async () => {
    const { result } = renderHook(() => usePreset(''), { wrapper: makeWrapper() })

    expect(result.current.fetchStatus).toBe('idle')
    expect(mockGetPreset).not.toHaveBeenCalled()
  })
})

describe('useCreatePreset', () => {
  it('createPreset을 호출하고 성공한다', async () => {
    mockCreatePreset.mockResolvedValue(sampleCafePreset)

    const { result } = renderHook(() => useCreatePreset(), { wrapper: makeWrapper() })

    await act(async () => {
      result.current.mutate({
        name: '출근길 아메리카노',
        log_type: 'cafe',
        cafe: { cafe_name: '블루보틀', coffee_name: '싱글 오리진' },
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockCreatePreset).toHaveBeenCalledTimes(1)
  })
})

describe('useUpdatePreset', () => {
  it('updatePreset을 호출하고 성공한다', async () => {
    mockUpdatePreset.mockResolvedValue(sampleCafePreset)

    const { result } = renderHook(() => useUpdatePreset('preset-1'), { wrapper: makeWrapper() })

    await act(async () => {
      result.current.mutate({
        name: '바뀐 이름',
        cafe: { cafe_name: '스타벅스', coffee_name: '아메리카노' },
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockUpdatePreset).toHaveBeenCalledWith('preset-1', expect.any(Object))
  })
})

describe('useDeletePreset', () => {
  it('deletePreset을 호출하고 성공한다', async () => {
    mockDeletePreset.mockResolvedValue(undefined)

    const { result } = renderHook(() => useDeletePreset(), { wrapper: makeWrapper() })

    await act(async () => {
      result.current.mutate('preset-1')
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockDeletePreset).toHaveBeenCalledWith('preset-1', expect.any(Object))
  })
})

describe('useUsePreset', () => {
  it('usePresetApi를 호출하고 성공한다', async () => {
    mockUsePresetApi.mockResolvedValue(undefined)

    const { result } = renderHook(() => useUsePreset(), { wrapper: makeWrapper() })

    await act(async () => {
      result.current.mutate('preset-1')
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockUsePresetApi).toHaveBeenCalledWith('preset-1', expect.any(Object))
  })
})
