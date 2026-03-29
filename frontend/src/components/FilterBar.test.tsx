import { describe, expect, it, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { FilterBar } from './FilterBar'

describe('FilterBar', () => {
  const noop = () => {}

  it('전체/카페/브루 탭을 렌더링한다', () => {
    render(
      <FilterBar
        logType={undefined}
        dateFrom=""
        dateTo=""
        onLogTypeChange={noop}
        onDateFromChange={noop}
        onDateToChange={noop}
      />,
    )

    expect(screen.getByRole('button', { name: '전체' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '카페' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '브루' })).toBeInTheDocument()
  })

  it('현재 logType에 해당하는 탭이 활성 스타일을 가진다', () => {
    render(
      <FilterBar
        logType="cafe"
        dateFrom=""
        dateTo=""
        onLogTypeChange={noop}
        onDateFromChange={noop}
        onDateToChange={noop}
      />,
    )

    // 활성 탭은 bg-white 클래스를 가진다
    const cafeButton = screen.getByRole('button', { name: '카페' })
    expect(cafeButton.className).toContain('bg-white')

    // 비활성 탭은 bg-white 클래스를 갖지 않는다
    const allButton = screen.getByRole('button', { name: '전체' })
    expect(allButton.className).not.toContain('bg-white')
  })

  it('탭 클릭 시 onLogTypeChange가 올바른 값으로 호출된다', () => {
    const onLogTypeChange = vi.fn()

    render(
      <FilterBar
        logType={undefined}
        dateFrom=""
        dateTo=""
        onLogTypeChange={onLogTypeChange}
        onDateFromChange={noop}
        onDateToChange={noop}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: '브루' }))
    expect(onLogTypeChange).toHaveBeenCalledWith('brew')

    fireEvent.click(screen.getByRole('button', { name: '전체' }))
    expect(onLogTypeChange).toHaveBeenCalledWith(undefined)
  })

  it('날짜 입력 변경 시 각 핸들러가 호출된다', () => {
    const onDateFromChange = vi.fn()
    const onDateToChange = vi.fn()

    render(
      <FilterBar
        logType={undefined}
        dateFrom=""
        dateTo=""
        onLogTypeChange={noop}
        onDateFromChange={onDateFromChange}
        onDateToChange={onDateToChange}
      />,
    )

    const fromInput = screen.getByLabelText('시작 날짜')
    fireEvent.change(fromInput, { target: { value: '2026-03-01' } })
    expect(onDateFromChange).toHaveBeenCalledWith('2026-03-01')

    const toInput = screen.getByLabelText('종료 날짜')
    fireEvent.change(toInput, { target: { value: '2026-03-29' } })
    expect(onDateToChange).toHaveBeenCalledWith('2026-03-29')
  })
})
