import { request } from './client'
import type { CoffeeLogFull, CreateLogInput, UpdateLogInput } from '../types/log'
import type { CursorPage } from '../types/common'

export interface ListLogsParams {
  log_type?: 'cafe' | 'brew'
  date_from?: string
  date_to?: string
  cursor?: string
  limit?: number
  /**
   * 목록 status 필터. 미지정 시 백엔드가 published만 반환한다(기존 호환).
   * 'draft'는 작성 중 로그만, 'all'은 published+draft 전체.
   */
  status?: 'draft' | 'published' | 'all'
}

export function getLogs(params: ListLogsParams = {}): Promise<CursorPage<CoffeeLogFull>> {
  const q = new URLSearchParams()
  if (params.log_type) q.set('log_type', params.log_type)
  if (params.date_from) q.set('date_from', params.date_from)
  if (params.date_to) q.set('date_to', params.date_to)
  if (params.cursor) q.set('cursor', params.cursor)
  if (params.limit !== undefined) q.set('limit', String(params.limit))
  if (params.status) q.set('status', params.status)
  const qs = q.toString()
  return request(`/logs${qs ? `?${qs}` : ''}`)
}

export function getLog(id: string): Promise<CoffeeLogFull> {
  return request(`/logs/${id}`)
}

export function createLog(body: CreateLogInput): Promise<CoffeeLogFull> {
  return request('/logs', { method: 'POST', body: JSON.stringify(body) })
}

export function updateLog(id: string, body: UpdateLogInput): Promise<CoffeeLogFull> {
  return request(`/logs/${id}`, { method: 'PUT', body: JSON.stringify(body) })
}

export function deleteLog(id: string): Promise<void> {
  return request(`/logs/${id}`, { method: 'DELETE' })
}
