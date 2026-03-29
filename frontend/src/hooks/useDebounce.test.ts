import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useDebounce } from './useDebounce'

describe('useDebounce', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('초기값을 즉시 반환한다', () => {
    const { result } = renderHook(() => useDebounce('초콜릿', 200))
    expect(result.current).toBe('초콜릿')
  })

  it('delay가 지나기 전에는 이전 값을 유지한다', () => {
    const { result, rerender } = renderHook(({ value }) => useDebounce(value, 200), {
      initialProps: { value: '초' },
    })

    rerender({ value: '초콜' })
    // 아직 200ms가 지나지 않았으므로 이전 값이 유지된다.
    expect(result.current).toBe('초')
  })

  it('delay가 지난 후 새 값으로 업데이트된다', () => {
    const { result, rerender } = renderHook(({ value }) => useDebounce(value, 200), {
      initialProps: { value: '초' },
    })

    rerender({ value: '초콜릿' })

    act(() => {
      vi.advanceTimersByTime(200)
    })

    expect(result.current).toBe('초콜릿')
  })

  it('연속으로 값이 바뀌면 마지막 값만 반영된다', () => {
    const { result, rerender } = renderHook(({ value }) => useDebounce(value, 200), {
      initialProps: { value: '초' },
    })

    rerender({ value: '초콜' })
    act(() => { vi.advanceTimersByTime(100) })

    rerender({ value: '초콜릿' })
    act(() => { vi.advanceTimersByTime(100) })

    // 아직 두 번째 타이머가 완료되지 않았다.
    expect(result.current).toBe('초')

    act(() => { vi.advanceTimersByTime(100) })

    // 마지막으로 입력된 "초콜릿"만 반영된다.
    expect(result.current).toBe('초콜릿')
  })
})
