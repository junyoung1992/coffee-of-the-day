import { describe, it, expect, vi, beforeEach } from 'vitest'

// 모듈 스코프의 refreshPromise 상태를 초기화하기 위해 각 테스트 전에 모듈을 재임포트한다.
// vi.resetModules()로 모듈 캐시를 지워 테스트 간 상태 격리를 보장한다.
async function importFreshClient() {
  vi.resetModules()
  return import('./client')
}

describe('request - refresh single-flight', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('동시 401 상황에서 /auth/refresh는 1회만 호출된다', async () => {
    let refreshCallCount = 0

    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string) => {
        if (url.includes('/auth/refresh')) {
          refreshCallCount++
          // refresh는 항상 성공
          return Promise.resolve(new Response(null, { status: 204 }))
        }
        if (refreshCallCount === 0) {
          // refresh 전: 401 반환
          return Promise.resolve(new Response(JSON.stringify({ error: 'unauthorized' }), { status: 401 }))
        }
        // refresh 후 재시도: 200 반환
        return Promise.resolve(new Response('{}', { status: 200 }))
      }),
    )

    const { request } = await importFreshClient()

    // 3개 요청을 동시에 발생시킨다
    await Promise.all([
      request('/logs'),
      request('/logs'),
      request('/logs'),
    ])

    expect(refreshCallCount).toBe(1)
  })

  it('refresh 실패 시 모든 대기 요청이 ApiError를 던진다', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockImplementation((url: string) => {
        if (url.includes('/auth/refresh')) {
          return Promise.resolve(new Response(JSON.stringify({ error: 'session expired' }), { status: 401 }))
        }
        return Promise.resolve(new Response(JSON.stringify({ error: 'unauthorized' }), { status: 401 }))
      }),
    )

    const { request, ApiError } = await importFreshClient()

    const results = await Promise.allSettled([request('/logs'), request('/logs')])

    for (const result of results) {
      expect(result.status).toBe('rejected')
      if (result.status === 'rejected') {
        expect(result.reason).toBeInstanceOf(ApiError)
      }
    }
  })
})
