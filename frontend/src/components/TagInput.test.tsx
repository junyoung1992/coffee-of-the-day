import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TagInput } from './TagInput'

describe('TagInput', () => {
  // ---------------------------------------------------------------------------
  // 기본 태그 추가/삭제
  // ---------------------------------------------------------------------------

  it('Enter로 태그를 추가한다', () => {
    const onChange = vi.fn()
    render(<TagInput value={[]} onChange={onChange} />)

    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: '초콜릿' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    expect(onChange).toHaveBeenCalledWith(['초콜릿'])
  })

  it('쉼표로 태그를 추가한다', () => {
    const onChange = vi.fn()
    render(<TagInput value={[]} onChange={onChange} />)

    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: '체리' } })
    fireEvent.keyDown(input, { key: ',' })

    expect(onChange).toHaveBeenCalledWith(['체리'])
  })

  it('공백만 있는 입력은 태그로 추가하지 않는다', () => {
    const onChange = vi.fn()
    render(<TagInput value={[]} onChange={onChange} />)

    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: '   ' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    expect(onChange).not.toHaveBeenCalled()
  })

  it('이미 추가된 태그는 중복 추가하지 않는다', () => {
    const onChange = vi.fn()
    render(<TagInput value={['초콜릿']} onChange={onChange} />)

    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: '초콜릿' } })
    fireEvent.keyDown(input, { key: 'Enter' })

    expect(onChange).not.toHaveBeenCalled()
  })

  it('Backspace로 마지막 태그를 삭제한다', () => {
    const onChange = vi.fn()
    render(<TagInput value={['초콜릿', '체리']} onChange={onChange} />)

    const input = screen.getByRole('textbox')
    // 입력값이 비어있을 때만 Backspace가 태그 삭제로 동작한다.
    fireEvent.keyDown(input, { key: 'Backspace' })

    expect(onChange).toHaveBeenCalledWith(['초콜릿'])
  })

  it('삭제 버튼으로 특정 태그를 제거한다', () => {
    const onChange = vi.fn()
    render(<TagInput value={['초콜릿', '체리']} onChange={onChange} />)

    fireEvent.click(screen.getByRole('button', { name: '초콜릿 태그 삭제' }))

    expect(onChange).toHaveBeenCalledWith(['체리'])
  })

  // ---------------------------------------------------------------------------
  // IME 조합 처리
  // ---------------------------------------------------------------------------

  it('IME 조합 중 Enter는 태그를 추가하지 않는다', () => {
    const onChange = vi.fn()
    render(<TagInput value={[]} onChange={onChange} />)

    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: '초' } })
    // isComposing을 KeyboardEventInit에 직접 전달해야 nativeEvent.isComposing이 true로 설정된다.
    fireEvent.keyDown(input, { key: 'Enter', isComposing: true })

    expect(onChange).not.toHaveBeenCalled()
  })

  it('IME 조합 중 쉼표는 태그를 추가하지 않는다', () => {
    const onChange = vi.fn()
    render(<TagInput value={[]} onChange={onChange} />)

    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: '초' } })
    fireEvent.keyDown(input, { key: ',', isComposing: true })

    expect(onChange).not.toHaveBeenCalled()
  })

  // ---------------------------------------------------------------------------
  // 드롭다운 제안 선택
  // ---------------------------------------------------------------------------

  it('제안 항목 클릭으로 태그를 추가한다', () => {
    const onChange = vi.fn()
    render(
      <TagInput value={[]} onChange={onChange} suggestions={['초콜릿', '체리']} />,
    )

    const input = screen.getByRole('textbox')
    // 드롭다운이 열리도록 입력값을 설정한다.
    fireEvent.change(input, { target: { value: '초' } })

    fireEvent.mouseDown(screen.getByRole('option', { name: '초콜릿' }))

    expect(onChange).toHaveBeenCalledWith(['초콜릿'])
  })

  it('이미 추가된 태그는 제안 목록에서 제외한다', () => {
    render(
      <TagInput value={['초콜릿']} onChange={vi.fn()} suggestions={['초콜릿', '체리']} />,
    )

    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: '초' } })

    // "초콜릿"은 이미 추가된 태그이므로 드롭다운에 나타나지 않아야 한다.
    expect(screen.queryByRole('option', { name: '초콜릿' })).not.toBeInTheDocument()
    expect(screen.getByRole('option', { name: '체리' })).toBeInTheDocument()
  })

  it('Escape로 드롭다운을 닫는다', () => {
    render(
      <TagInput value={[]} onChange={vi.fn()} suggestions={['초콜릿']} />,
    )

    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: '초' } })
    expect(screen.getByRole('listbox')).toBeInTheDocument()

    fireEvent.keyDown(input, { key: 'Escape' })
    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
  })
})
