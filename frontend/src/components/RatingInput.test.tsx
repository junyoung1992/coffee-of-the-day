import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { RatingInput } from './RatingInput'

describe('RatingInput', () => {
  it('선택한 반 단위 별점을 전달한다', () => {
    const handleChange = vi.fn()

    render(<RatingInput value={null} onChange={handleChange} />)

    fireEvent.click(screen.getByRole('button', { name: /4.5/i }))

    expect(handleChange).toHaveBeenCalledWith(4.5)
  })

  it('clear 버튼으로 값을 지울 수 있다', () => {
    const handleChange = vi.fn()

    render(<RatingInput value={3} onChange={handleChange} />)

    fireEvent.click(screen.getByRole('button', { name: /clear/i }))

    expect(handleChange).toHaveBeenCalledWith(null)
  })
})
