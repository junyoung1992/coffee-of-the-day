import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { useLogList, useLog, useCreateLog, useUpdateLog, useDeleteLog } from './useLogs'
import * as logsApi from '../api/logs'

vi.mock('../api/logs')

const mockGetLogs = vi.mocked(logsApi.getLogs)
const mockGetLog = vi.mocked(logsApi.getLog)
const mockCreateLog = vi.mocked(logsApi.createLog)
const mockUpdateLog = vi.mocked(logsApi.updateLog)
const mockDeleteLog = vi.mocked(logsApi.deleteLog)

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

const sampleCafeLog = {
  id: 'log-1',
  user_id: 'user-1',
  recorded_at: '2026-03-29T10:00:00Z',
  companions: [] as string[],
  log_type: 'cafe' as const,
  memo: null,
  created_at: '2026-03-29T10:00:00Z',
  updated_at: '2026-03-29T10:00:00Z',
  cafe: {
    cafe_name: '블루보틀',
    coffee_name: '싱글 오리진',
    location: null,
    bean_origin: null,
    bean_process: null,
    roast_level: null,
    tasting_tags: [] as string[],
    tasting_note: null,
    impressions: null,
    rating: 4.5,
  },
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('useLogList', () => {
  it('첫 페이지를 로드한다', async () => {
    mockGetLogs.mockResolvedValue({
      items: [sampleCafeLog],
      next_cursor: null,
      has_next: false,
    })

    const { result } = renderHook(() => useLogList(), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data?.pages[0].items).toHaveLength(1)
    expect(result.current.data?.pages[0].items[0].id).toBe('log-1')
  })

  it('has_next가 false이면 hasNextPage가 false이다', async () => {
    mockGetLogs.mockResolvedValue({ items: [], next_cursor: null, has_next: false })

    const { result } = renderHook(() => useLogList(), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.hasNextPage).toBe(false)
  })

  it('has_next가 true이면 hasNextPage가 true이다', async () => {
    mockGetLogs.mockResolvedValue({
      items: [sampleCafeLog],
      next_cursor: 'cursor-abc',
      has_next: true,
    })

    const { result } = renderHook(() => useLogList(), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.hasNextPage).toBe(true)
  })

  it('필터 파라미터를 getLogs에 전달한다', async () => {
    mockGetLogs.mockResolvedValue({ items: [], next_cursor: null, has_next: false })

    const { result } = renderHook(() => useLogList({ log_type: 'cafe', limit: 10 }), {
      wrapper: makeWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockGetLogs).toHaveBeenCalledWith(
      expect.objectContaining({ log_type: 'cafe', limit: 10 }),
    )
  })
})

describe('useLog', () => {
  it('id로 단건을 조회한다', async () => {
    mockGetLog.mockResolvedValue(sampleCafeLog)

    const { result } = renderHook(() => useLog('log-1'), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toEqual(sampleCafeLog)
    expect(mockGetLog).toHaveBeenCalledWith('log-1')
  })

  it('id가 빈 문자열이면 쿼리를 실행하지 않는다', async () => {
    const { result } = renderHook(() => useLog(''), { wrapper: makeWrapper() })

    // fetchStatus가 idle이어야 한다 (enabled: false)
    expect(result.current.fetchStatus).toBe('idle')
    expect(mockGetLog).not.toHaveBeenCalled()
  })
})

describe('useCreateLog', () => {
  it('createLog를 호출하고 성공 시 isSuccess가 된다', async () => {
    mockCreateLog.mockResolvedValue(sampleCafeLog)

    const { result } = renderHook(() => useCreateLog(), { wrapper: makeWrapper() })

    await act(async () => {
      result.current.mutate({
        recorded_at: '2026-03-29T10:00:00Z',
        log_type: 'cafe',
        cafe: { cafe_name: '블루보틀', coffee_name: '싱글 오리진' },
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockCreateLog).toHaveBeenCalledTimes(1)
  })
})

describe('useUpdateLog', () => {
  it('updateLog를 호출하고 성공 시 isSuccess가 된다', async () => {
    mockUpdateLog.mockResolvedValue(sampleCafeLog)

    const { result } = renderHook(() => useUpdateLog('log-1'), { wrapper: makeWrapper() })

    await act(async () => {
      result.current.mutate({
        recorded_at: '2026-03-29T10:00:00Z',
        log_type: 'cafe',
        cafe: { cafe_name: '블루보틀', coffee_name: '싱글 오리진' },
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockUpdateLog).toHaveBeenCalledWith('log-1', expect.any(Object))
  })
})

describe('useDeleteLog', () => {
  it('deleteLog를 호출하고 성공 시 isSuccess가 된다', async () => {
    mockDeleteLog.mockResolvedValue(undefined)

    const { result } = renderHook(() => useDeleteLog(), { wrapper: makeWrapper() })

    await act(async () => {
      result.current.mutate('log-1')
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    // TanStack Query v5는 mutationFn에 (variables, context)를 전달한다
    expect(mockDeleteLog).toHaveBeenCalledWith('log-1', expect.any(Object))
  })
})
