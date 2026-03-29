import { useId, useRef, useState } from 'react'

interface TagInputProps {
  value: string[]
  onChange: (tags: string[]) => void
  suggestions?: string[]
  placeholder?: string
  // suggestions 쿼리용 — 입력값이 변할 때 부모가 검색어를 갱신할 수 있도록 노출
  onQueryChange?: (q: string) => void
}

export function TagInput({ value, onChange, suggestions = [], placeholder, onQueryChange }: TagInputProps) {
  const [inputValue, setInputValue] = useState('')
  const [open, setOpen] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const listboxId = useId()

  function addTag(tag: string) {
    const trimmed = tag.trim()
    // 빈 문자열이거나 이미 추가된 태그는 무시한다
    if (!trimmed || value.includes(trimmed)) return
    onChange([...value, trimmed])
    setInputValue('')
    onQueryChange?.('')
    setOpen(false)
  }

  function removeTag(index: number) {
    onChange(value.filter((_, i) => i !== index))
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    const q = e.target.value
    setInputValue(q)
    onQueryChange?.(q)
    setOpen(q.length > 0)
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter' || e.key === ',') {
      // IME 조합 중(한국어, 중국어 등)에는 Enter/쉼표를 무시한다.
      // isComposing이 true인 동안은 문자가 아직 확정되지 않은 상태다.
      if (e.nativeEvent.isComposing) return
      // 쉼표 또는 엔터로 즉시 태그 추가 — 폼 제출을 막는다
      e.preventDefault()
      addTag(inputValue)
    } else if (e.key === 'Backspace' && inputValue === '' && value.length > 0) {
      // 입력이 비었을 때 Backspace로 마지막 태그 삭제
      removeTag(value.length - 1)
    } else if (e.key === 'Escape') {
      setOpen(false)
    }
  }

  // mousedown에서 preventDefault를 호출해 input의 blur 이벤트를 방지한다.
  // blur 이후에 드롭다운이 닫히면 click이 무시되는 문제를 막기 위함이다.
  function handleOptionMouseDown(e: React.MouseEvent, tag: string) {
    e.preventDefault()
    addTag(tag)
    inputRef.current?.focus()
  }

  // 드롭다운에 표시할 제안 목록 — 이미 추가된 태그는 제외한다
  const filteredSuggestions = suggestions.filter((s) => !value.includes(s))

  return (
    <div className="space-y-2">
      {/* 태그 뱃지 목록 */}
      {value.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {value.map((tag, index) => (
            <span
              key={`${tag}-${index}`}
              className="inline-flex items-center gap-1 rounded-full border border-amber-900/20 bg-amber-50 px-3 py-1 text-xs font-medium text-amber-950"
            >
              {tag}
              <button
                type="button"
                onClick={() => removeTag(index)}
                aria-label={`${tag} 태그 삭제`}
                className="ml-0.5 rounded-full text-amber-700 transition hover:text-amber-950"
              >
                ×
              </button>
            </span>
          ))}
        </div>
      )}

      {/* 입력 + 드롭다운 */}
      <div className="relative">
        <input
          ref={inputRef}
          type="text"
          value={inputValue}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onFocus={() => { if (inputValue.length > 0) setOpen(true) }}
          onBlur={() => setOpen(false)}
          placeholder={placeholder}
          aria-autocomplete="list"
          aria-controls={open ? listboxId : undefined}
          className="w-full rounded-2xl border border-amber-950/10 bg-white px-4 py-3 text-sm text-stone-900 outline-none transition placeholder:text-stone-400 focus:border-amber-900/35 focus:bg-amber-50/40"
        />

        {open && filteredSuggestions.length > 0 && (
          <ul
            id={listboxId}
            role="listbox"
            className="absolute z-10 mt-1 w-full overflow-hidden rounded-2xl border border-amber-950/10 bg-white shadow-lg"
          >
            {filteredSuggestions.map((suggestion) => (
              <li
                key={suggestion}
                role="option"
                aria-selected={false}
                onMouseDown={(e) => handleOptionMouseDown(e, suggestion)}
                className="cursor-pointer px-4 py-2.5 text-sm text-stone-800 transition hover:bg-amber-50"
              >
                {suggestion}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
