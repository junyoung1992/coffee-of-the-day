const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

let userId = 'dev-user'

export function setUserId(id: string) {
  userId = id
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      'X-User-Id': userId,
      ...init.headers,
    },
  })

  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    // 백엔드 에러 응답 형식: { "error": "메시지 문자열" }
    const message = typeof body?.error === 'string' ? body.error : res.statusText
    throw new ApiError(res.status, 'UNKNOWN', message)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}
