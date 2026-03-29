import { useEffect, useState } from 'react'

// 값이 변경된 뒤 delay ms가 지나야 반영되는 debounced 값을 반환한다.
// 자동완성처럼 타이핑마다 요청이 발생하는 경우 불필요한 네트워크 호출을 줄이기 위해 사용한다.
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedValue(value), delay)
    return () => clearTimeout(timer)
  }, [value, delay])

  return debouncedValue
}
