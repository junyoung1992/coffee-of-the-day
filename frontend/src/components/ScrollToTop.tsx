import { useCallback, useEffect, useState } from 'react'

// easeInOutCubic: 처음에 천천히 → 중간에 빠르게 → 끝에서 감속
// 스크롤 거리와 무관하게 고정 시간(500ms) 안에 완료되므로
// 긴 페이지에서도 쭉 타고 올라가는 느낌 없이 자연스럽다.
function easeInOutCubic(t: number): number {
  return t < 0.5 ? 4 * t * t * t : 1 - (-2 * t + 2) ** 3 / 2
}

export function animateScrollToTop(duration: number) {
  const start = window.scrollY
  if (start === 0) return

  const startTime = performance.now()

  function step(now: number) {
    const elapsed = now - startTime
    const progress = Math.min(elapsed / duration, 1)
    const eased = easeInOutCubic(progress)

    window.scrollTo(0, start * (1 - eased))

    if (progress < 1) {
      requestAnimationFrame(step)
    }
  }

  requestAnimationFrame(step)
}

const SCROLL_THRESHOLD = 300

export function ScrollToTop() {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    let ticking = false
    function handleScroll() {
      if (!ticking) {
        requestAnimationFrame(() => {
          setVisible(window.scrollY > SCROLL_THRESHOLD)
          ticking = false
        })
        ticking = true
      }
    }

    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [])

  const scrollToTop = useCallback(() => {
    animateScrollToTop(500)
  }, [])

  return (
    <button
      type="button"
      onClick={scrollToTop}
      aria-label="맨 위로 스크롤"
      className={`fixed bottom-6 right-6 z-50 flex h-11 w-11 items-center justify-center rounded-full border border-amber-950/10 bg-white/70 text-stone-600 shadow-lg backdrop-blur-sm transition-opacity duration-300 hover:bg-white hover:text-stone-900 ${
        visible ? 'opacity-100' : 'pointer-events-none opacity-0'
      }`}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 20 20"
        fill="currentColor"
        className="h-5 w-5"
      >
        <path
          fillRule="evenodd"
          d="M10 17a.75.75 0 0 1-.75-.75V5.612L5.29 9.77a.75.75 0 0 1-1.08-1.04l5.25-5.5a.75.75 0 0 1 1.08 0l5.25 5.5a.75.75 0 1 1-1.08 1.04l-3.96-4.158V16.25A.75.75 0 0 1 10 17Z"
          clipRule="evenodd"
        />
      </svg>
    </button>
  )
}
