import { describe, expect, it } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { render, screen } from '@testing-library/react'
import { LogCard, LogCardSkeleton } from './LogCard'
import type { CoffeeLogFull } from '../types/log'

describe('LogCard', () => {
  it('cafe 로그 핵심 정보를 보여준다', () => {
    const log: CoffeeLogFull = {
      id: 'log-1',
      user_id: 'user-1',
      recorded_at: '2026-03-29T10:00:00Z',
      companions: ['민수'],
      log_type: 'cafe',
      memo: null,
      created_at: '2026-03-29T10:00:00Z',
      updated_at: '2026-03-29T10:00:00Z',
      cafe: {
        cafe_name: '블루보틀',
        location: '성수',
        coffee_name: '게이샤 드립',
        bean_origin: null,
        bean_process: null,
        roast_level: null,
        tasting_tags: ['초콜릿'],
        tasting_note: null,
        impressions: '깔끔하고 단맛이 좋다',
        rating: 4.5,
      },
    }

    render(
      <MemoryRouter>
        <LogCard log={log} />
      </MemoryRouter>,
    )

    expect(screen.getByText('게이샤 드립')).toBeInTheDocument()
    expect(screen.getByText('블루보틀 · 성수')).toBeInTheDocument()
    expect(screen.getByText('초콜릿')).toBeInTheDocument()
  })

  it('brew 로그에서는 추출 방식과 도구를 보여준다', () => {
    const log: CoffeeLogFull = {
      id: 'log-2',
      user_id: 'user-1',
      recorded_at: '2026-03-29T10:00:00Z',
      companions: [],
      log_type: 'brew',
      memo: null,
      created_at: '2026-03-29T10:00:00Z',
      updated_at: '2026-03-29T10:00:00Z',
      brew: {
        bean_name: '에티오피아 G1',
        bean_origin: null,
        bean_process: null,
        roast_level: null,
        roast_date: null,
        tasting_tags: [],
        tasting_note: null,
        brew_method: 'aeropress',
        brew_device: 'AeroPress Go',
        coffee_amount_g: null,
        water_amount_ml: null,
        water_temp_c: null,
        brew_time_sec: null,
        grind_size: null,
        brew_steps: [],
        impressions: null,
        rating: null,
      },
    }

    render(
      <MemoryRouter>
        <LogCard log={log} />
      </MemoryRouter>,
    )

    expect(screen.getByText('에티오피아 G1')).toBeInTheDocument()
    expect(screen.getByText('AeroPress · AeroPress Go')).toBeInTheDocument()
    expect(screen.getByText('혼자')).toBeInTheDocument()
  })
})

describe('LogCardSkeleton', () => {
  it('실제 카드와 동일한 컨테이너 형태로 렌더링된다', () => {
    const { container } = render(<LogCardSkeleton />)
    // 스켈레톤은 링크가 아닌 일반 div로 렌더링되어야 한다
    expect(container.querySelector('a')).toBeNull()
    expect(container.querySelector('.animate-pulse')).toBeInTheDocument()
  })
})
