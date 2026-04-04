import { describe, it, expect, vi, afterEach } from 'vitest'
import { formatLocalDate, getDefaultDateFrom, getDefaultDateTo } from './date'

afterEach(() => {
  vi.useRealTimers()
})

describe('formatLocalDate', () => {
  it('로컬 시간 기준으로 YYYY-MM-DD 형식을 반환한다', () => {
    const date = new Date(2026, 2, 15) // 3월 15일 (month는 0-indexed)
    expect(formatLocalDate(date)).toBe('2026-03-15')
  })

  it('월과 일을 2자리로 패딩한다', () => {
    const date = new Date(2026, 0, 5) // 1월 5일
    expect(formatLocalDate(date)).toBe('2026-01-05')
  })
})

describe('getDefaultDateFrom', () => {
  it('오늘 기준 30일 전을 반환한다', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 3, 4)) // 4월 4일
    expect(getDefaultDateFrom()).toBe('2026-03-05')
  })

  it('연도를 넘어가는 경우에도 올바르게 동작한다', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 0, 15)) // 1월 15일
    expect(getDefaultDateFrom()).toBe('2025-12-16')
  })
})

describe('getDefaultDateTo', () => {
  it('오늘 날짜를 로컬 시간 기준으로 반환한다', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 2, 30, 3, 0)) // 3월 30일 03:00 KST
    expect(getDefaultDateTo()).toBe('2026-03-30')
  })

  it('KST 자정 직후에도 오늘 날짜를 반환한다 (UTC 기준 어제가 아님)', () => {
    vi.useFakeTimers()
    // KST 2026-03-30 00:30 = UTC 2026-03-29 15:30
    // toISOString()이었다면 2026-03-29를 반환했을 것
    vi.setSystemTime(new Date(2026, 2, 30, 0, 30))
    expect(getDefaultDateTo()).toBe('2026-03-30')
  })
})
