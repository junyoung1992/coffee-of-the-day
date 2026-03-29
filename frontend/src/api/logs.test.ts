import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getLogs, getLog, createLog, updateLog, deleteLog } from './logs'
import * as client from './client'

// request 함수를 스파이로 대체한다
vi.mock('./client', () => ({
  request: vi.fn(),
  ApiError: class ApiError extends Error {
    status: number
    code: string
    constructor(status: number, code: string, message: string) {
      super(message)
      this.status = status
      this.code = code
    }
  },
}))

const mockRequest = vi.mocked(client.request)

const sampleCafeLog = {
  id: 'log-1',
  user_id: 'user-1',
  recorded_at: '2026-03-29T10:00:00Z',
  companions: [],
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
    tasting_tags: [],
    tasting_note: null,
    impressions: null,
    rating: 4.5,
  },
}

beforeEach(() => {
  mockRequest.mockReset()
})

describe('getLogs', () => {
  it('파라미터 없이 호출하면 /logs로 요청한다', async () => {
    mockRequest.mockResolvedValue({ items: [], next_cursor: null, has_next: false })

    await getLogs()

    expect(mockRequest).toHaveBeenCalledWith('/logs')
  })

  it('log_type 필터를 쿼리 파라미터로 전달한다', async () => {
    mockRequest.mockResolvedValue({ items: [], next_cursor: null, has_next: false })

    await getLogs({ log_type: 'cafe' })

    expect(mockRequest).toHaveBeenCalledWith('/logs?log_type=cafe')
  })

  it('여러 필터를 동시에 전달할 수 있다', async () => {
    mockRequest.mockResolvedValue({ items: [], next_cursor: null, has_next: false })

    await getLogs({ log_type: 'brew', limit: 10, date_from: '2026-01-01' })

    const call = mockRequest.mock.calls[0][0] as string
    expect(call).toContain('log_type=brew')
    expect(call).toContain('limit=10')
    expect(call).toContain('date_from=2026-01-01')
  })

  it('cursor 파라미터를 전달한다', async () => {
    mockRequest.mockResolvedValue({ items: [], next_cursor: null, has_next: false })

    await getLogs({ cursor: 'abc123' })

    expect(mockRequest).toHaveBeenCalledWith('/logs?cursor=abc123')
  })

  it('응답을 그대로 반환한다', async () => {
    const response = { items: [sampleCafeLog], next_cursor: 'next', has_next: true }
    mockRequest.mockResolvedValue(response)

    const result = await getLogs()

    expect(result).toEqual(response)
  })
})

describe('getLog', () => {
  it('/logs/:id 경로로 요청한다', async () => {
    mockRequest.mockResolvedValue(sampleCafeLog)

    await getLog('log-1')

    expect(mockRequest).toHaveBeenCalledWith('/logs/log-1')
  })

  it('기록을 반환한다', async () => {
    mockRequest.mockResolvedValue(sampleCafeLog)

    const result = await getLog('log-1')

    expect(result).toEqual(sampleCafeLog)
  })
})

describe('createLog', () => {
  it('POST /logs로 요청한다', async () => {
    mockRequest.mockResolvedValue(sampleCafeLog)
    const input = {
      recorded_at: '2026-03-29T10:00:00Z',
      log_type: 'cafe' as const,
      cafe: { cafe_name: '블루보틀', coffee_name: '싱글 오리진' },
    }

    await createLog(input)

    expect(mockRequest).toHaveBeenCalledWith('/logs', {
      method: 'POST',
      body: JSON.stringify(input),
    })
  })
})

describe('updateLog', () => {
  it('PUT /logs/:id로 요청한다', async () => {
    mockRequest.mockResolvedValue(sampleCafeLog)
    const input = {
      recorded_at: '2026-03-29T10:00:00Z',
      log_type: 'cafe' as const,
      cafe: { cafe_name: '블루보틀', coffee_name: '싱글 오리진' },
    }

    await updateLog('log-1', input)

    expect(mockRequest).toHaveBeenCalledWith('/logs/log-1', {
      method: 'PUT',
      body: JSON.stringify(input),
    })
  })
})

describe('deleteLog', () => {
  it('DELETE /logs/:id로 요청한다', async () => {
    mockRequest.mockResolvedValue(undefined)

    await deleteLog('log-1')

    expect(mockRequest).toHaveBeenCalledWith('/logs/log-1', { method: 'DELETE' })
  })
})
