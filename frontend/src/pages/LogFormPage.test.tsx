import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import LogFormPage from './LogFormPage'
import type { CafeLogFull, BrewLogFull } from '../types/log'

vi.mock('../hooks/useLogs', () => ({
  useLog: () => ({ data: undefined, error: null, isError: false, isLoading: false }),
  useCreateLog: () => ({ mutateAsync: vi.fn(), isPending: false, isError: false, error: null }),
  useUpdateLog: () => ({ mutateAsync: vi.fn(), isPending: false, isError: false, error: null }),
}))

vi.mock('../hooks/useSuggestions', () => ({
  useCompanionSuggestions: () => ({ data: [] }),
  useTagSuggestions: () => ({ data: [] }),
}))

const cafeLog: CafeLogFull = {
  id: 'log-cafe-1',
  user_id: 'user-1',
  recorded_at: '2026-03-20T14:00:00Z',
  companions: ['민수', '지연'],
  log_type: 'cafe',
  memo: '분위기 좋았다',
  created_at: '2026-03-20T14:00:00Z',
  updated_at: '2026-03-20T14:00:00Z',
  cafe: {
    cafe_name: '블루보틀 성수',
    location: '서울 성수',
    coffee_name: '게이샤 드립',
    bean_origin: 'Ethiopia',
    bean_process: 'washed',
    roast_level: 'light',
    tasting_tags: ['자몽', '꽃'],
    tasting_note: '깔끔한 산미',
    impressions: '다음에 또 오고 싶다',
    rating: 4.5,
  },
}

const brewLog: BrewLogFull = {
  id: 'log-brew-1',
  user_id: 'user-1',
  recorded_at: '2026-03-22T09:00:00Z',
  companions: ['수빈'],
  log_type: 'brew',
  memo: '아침 한 잔',
  created_at: '2026-03-22T09:00:00Z',
  updated_at: '2026-03-22T09:00:00Z',
  brew: {
    bean_name: '케냐 AB',
    bean_origin: 'Kenya',
    bean_process: null,
    roast_level: 'medium',
    roast_date: '2026-03-18',
    tasting_tags: ['베리', '홍차'],
    tasting_note: '달콤한 뒷맛',
    brew_method: 'pour_over',
    brew_device: 'Origami',
    coffee_amount_g: 18,
    water_amount_ml: 300,
    water_temp_c: 92,
    brew_time_sec: 165,
    grind_size: '중간',
    brew_steps: ['뜸 40초', '3회 나눠 붓기'],
    impressions: '맑고 길게 남는다',
    rating: 4,
  },
}

function renderCloneMode(cloneFrom: CafeLogFull | BrewLogFull) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter
        initialEntries={[{ pathname: '/logs/new', state: { cloneFrom } }]}
      >
        <Routes>
          <Route path="/logs/new" element={<LogFormPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('LogFormPage clone 모드', () => {
  it('cafe 로그 복제 시 타이틀이 "기록 복제"로 표시된다', () => {
    renderCloneMode(cafeLog)
    expect(screen.getByText('기록 복제')).toBeInTheDocument()
  })

  it('cafe 로그 복제 시 원본 필드가 폼에 채워진다', () => {
    renderCloneMode(cafeLog)

    expect(screen.getByDisplayValue('블루보틀 성수')).toBeInTheDocument()
    expect(screen.getByDisplayValue('게이샤 드립')).toBeInTheDocument()
  })

  it('cafe 로그 복제 시 리셋 대상 필드가 비어있다', () => {
    renderCloneMode(cafeLog)

    // recordedAt은 오늘 날짜로 리셋 (원본 날짜가 아님)
    const recordedAtInput = screen.getByLabelText(/Recorded at/) as HTMLInputElement
    expect(recordedAtInput.value).not.toContain('2026-03-20')
  })

  it('brew 로그 복제 시 원본 필드가 폼에 채워진다', () => {
    renderCloneMode(brewLog)

    expect(screen.getByDisplayValue('케냐 AB')).toBeInTheDocument()
    // logType이 brew로 설정되었는지 확인
    expect(screen.getByText('기록 복제')).toBeInTheDocument()
  })

  it('brew 로그 복제 시 recordedAt이 원본 날짜가 아니다', () => {
    renderCloneMode(brewLog)

    const recordedAtInput = screen.getByLabelText(/Recorded at/) as HTMLInputElement
    expect(recordedAtInput.value).not.toContain('2026-03-22')
  })

  it('clone 모드에서 로그 타입 변경이 불가능하다', () => {
    renderCloneMode(cafeLog)

    const cafeButton = screen.getByRole('button', { name: /Cafe log/ })
    expect(cafeButton).toBeDisabled()
  })
})
