import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import { ScrollToTop } from './ScrollToTop'

// window.scrollY를 모킹하기 위한 헬퍼
function setScrollY(value: number) {
  Object.defineProperty(window, 'scrollY', { value, writable: true })
}

// rAF throttle을 flush하기 위해 두 프레임을 기다린다
async function flushRAF() {
  await act(async () => {
    await new Promise((resolve) => requestAnimationFrame(resolve))
    await new Promise((resolve) => requestAnimationFrame(resolve))
  })
}

describe('ScrollToTop', () => {
  beforeEach(() => {
    setScrollY(0)
    window.scrollTo = vi.fn()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('스크롤이 300px 이하이면 버튼이 보이지 않는다', () => {
    render(<ScrollToTop />)

    const button = screen.getByRole('button', { name: '맨 위로 스크롤' })
    expect(button.className).toContain('opacity-0')
    expect(button.className).toContain('pointer-events-none')
  })

  it('스크롤이 300px을 넘으면 버튼이 보인다', async () => {
    render(<ScrollToTop />)

    setScrollY(400)
    fireEvent.scroll(window)
    await flushRAF()

    const button = screen.getByRole('button', { name: '맨 위로 스크롤' })
    expect(button.className).toContain('opacity-100')
    expect(button.className).not.toContain('pointer-events-none')
  })

  it('버튼 클릭 시 스크롤 애니메이션이 시작된다', async () => {
    render(<ScrollToTop />)

    setScrollY(500)
    fireEvent.scroll(window)
    await flushRAF()

    const button = screen.getByRole('button', { name: '맨 위로 스크롤' })
    fireEvent.click(button)
    await flushRAF()

    expect(window.scrollTo).toHaveBeenCalled()
  })
})
