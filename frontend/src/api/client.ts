const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'

// refresh single-flight: 토큰 만료 시 동시에 여러 요청이 refresh를 중복 호출하지 않도록 한다.
// 진행 중인 refresh Promise를 공유하여 refresh rotation 도입 시 race condition을 방지한다.
let refreshPromise: Promise<void> | null = null

export class ApiError extends Error {
  status: number
  code: string
  field?: string

  constructor(status: number, code: string, message: string, field?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.field = field
  }
}

// credentials: 'include'는 쿠키를 cross-origin 요청에도 전송하기 위해 필요하다.
// 서버의 Access-Control-Allow-Credentials: true 설정과 짝을 이룬다.
function doFetch(path: string, init: RequestInit = {}) {
  return fetch(`${BASE_URL}${path}`, {
    ...init,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...init.headers },
  })
}

async function requestInternal<T>(path: string, init: RequestInit, isRetry: boolean): Promise<T> {
  const res = await doFetch(path, init)

  if (res.status === 401 && !isRetry && !path.startsWith('/auth/')) {
    // 액세스 토큰 만료 시 리프레시 토큰으로 갱신을 시도한다.
    // 진행 중인 refresh가 있으면 같은 Promise를 await해 중복 호출을 방지한다.
    if (!refreshPromise) {
      refreshPromise = doFetch('/auth/refresh', { method: 'POST' })
        .then((r) => {
          if (!r.ok) throw new ApiError(401, 'UNAUTHORIZED', '세션이 만료되었습니다')
        })
        .finally(() => {
          refreshPromise = null
        })
    }
    try {
      await refreshPromise
      return requestInternal(path, init, true)
    } catch {
      throw new ApiError(401, 'UNAUTHORIZED', '세션이 만료되었습니다')
    }
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    // 백엔드 에러 응답 형식: { "error": "메시지", "field": "필드명(선택)" }
    const message = typeof body?.error === 'string' ? body.error : res.statusText
    const field = typeof body?.field === 'string' ? body.field : undefined
    throw new ApiError(res.status, 'UNKNOWN', message, field)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}

export function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  return requestInternal(path, init, false)
}
