const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'

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
    // 백엔드 에러 응답 형식: { "error": "메시지", "field": "필드명(선택)" }
    const message = typeof body?.error === 'string' ? body.error : res.statusText
    const field = typeof body?.field === 'string' ? body.field : undefined
    throw new ApiError(res.status, 'UNKNOWN', message, field)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}
