import { describe, expect, it } from 'vitest'
import { buildLogPayload, cloneToFormState, createEmptyFormState, hasOptionalValues, logToFormState } from './logFormState'
import type { CafeLogFull, BrewLogFull, CoffeeLogFull } from '../types/log'

describe('createEmptyFormState', () => {
  it('신규 폼의 기본값을 만든다', () => {
    const state = createEmptyFormState(new Date('2026-03-29T10:15:00Z'))

    expect(state.logType).toBe('cafe')
    expect(state.recordedAt).toMatch(/^2026-03-29T/)
    expect(state.brew.brewMethod).toBe('pour_over')
    expect(state.brew.brewSteps).toEqual([''])
  })
})

describe('buildLogPayload', () => {
  it('cafe 폼 상태를 API 요청 본문으로 변환한다', () => {
    const state = createEmptyFormState(new Date('2026-03-29T10:15:00Z'))
    state.recordedAt = '2026-03-29T19:30'
    state.companions = ['민수', '지연']
    state.memo = ' 주말 기록 '
    state.cafe.cafeName = '블루보틀 성수'
    state.cafe.coffeeName = '게이샤 드립'
    state.cafe.location = '서울 성수'
    state.cafe.tastingTags = ['초콜릿', '자몽']
    state.cafe.rating = '4.5'

    const payload = buildLogPayload(state)

    expect(payload.log_type).toBe('cafe')
    expect(payload.companions).toEqual(['민수', '지연'])
    expect(payload.memo).toBe('주말 기록')
    expect(payload.cafe).toMatchObject({
      cafe_name: '블루보틀 성수',
      coffee_name: '게이샤 드립',
      location: '서울 성수',
      tasting_tags: ['초콜릿', '자몽'],
      rating: 4.5,
    })
    expect(new Date(payload.recorded_at).toISOString()).toBe(new Date('2026-03-29T19:30').toISOString())
  })

  it('brew 폼 상태를 API 요청 본문으로 변환하면서 빈 값을 제거한다', () => {
    const state = createEmptyFormState(new Date('2026-03-29T10:15:00Z'))
    state.logType = 'brew'
    state.brew.beanName = '에티오피아 예가체프'
    state.brew.brewMethod = 'aeropress'
    state.brew.tastingTags = ['복숭아', '꽃']
    state.brew.brewSteps = ['뜸들이기 30초', '본 추출', '  ']
    state.brew.coffeeAmountG = '18'
    state.brew.waterAmountMl = '250'
    state.brew.brewTimeSec = '140'
    state.brew.rating = '5'

    const payload = buildLogPayload(state)

    expect(payload.brew).toMatchObject({
      bean_name: '에티오피아 예가체프',
      brew_method: 'aeropress',
      tasting_tags: ['복숭아', '꽃'],
      brew_steps: ['뜸들이기 30초', '본 추출'],
      coffee_amount_g: 18,
      water_amount_ml: 250,
      brew_time_sec: 140,
      rating: 5,
    })
    expect(payload).not.toHaveProperty('cafe')
  })
})

describe('logToFormState', () => {
  it('응답 데이터를 수정용 폼 상태로 변환한다', () => {
    const log: CoffeeLogFull = {
      id: 'log-1',
      user_id: 'user-1',
      recorded_at: '2026-03-29T10:00:00Z',
      companions: ['민수', '지연'],
      log_type: 'brew',
      memo: '같이 비교 시음',
      created_at: '2026-03-29T10:00:00Z',
      updated_at: '2026-03-29T10:00:00Z',
      brew: {
        bean_name: '케냐 AB',
        bean_origin: 'Kenya',
        bean_process: null,
        roast_level: 'light',
        roast_date: '2026-03-25',
        tasting_tags: ['베리', '홍차'],
        tasting_note: null,
        brew_method: 'pour_over',
        brew_device: 'Origami',
        coffee_amount_g: 18,
        water_amount_ml: 300,
        water_temp_c: 92,
        brew_time_sec: 165,
        grind_size: '중간',
        brew_steps: ['뜸 40초', '3회 나눠 붓기'],
        impressions: '맑고 길게 남는다',
        rating: 4.5,
      },
    }

    const state = logToFormState(log)

    expect(state.logType).toBe('brew')
    expect(state.companions).toEqual(['민수', '지연'])
    expect(state.memo).toBe('같이 비교 시음')
    expect(state.brew.beanName).toBe('케냐 AB')
    expect(state.brew.tastingTags).toEqual(['베리', '홍차'])
    expect(state.brew.brewSteps).toEqual(['뜸 40초', '3회 나눠 붓기'])
    expect(state.brew.rating).toBe('4.5')
  })
})

describe('cloneToFormState', () => {
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

  const cloneNow = new Date('2026-04-04T08:00:00')

  it('cafe 로그 복제 시 리셋 대상 필드가 초기화된다', () => {
    const state = cloneToFormState(cafeLog, cloneNow)

    // 리셋 대상
    expect(state.recordedAt).toBe('2026-04-04T08:00')
    expect(state.companions).toEqual([])
    expect(state.memo).toBe('')
    expect(state.cafe.rating).toBe('')
    expect(state.cafe.impressions).toBe('')
  })

  it('cafe 로그 복제 시 나머지 필드는 원본 값을 유지한다', () => {
    const state = cloneToFormState(cafeLog, cloneNow)

    expect(state.logType).toBe('cafe')
    expect(state.cafe.cafeName).toBe('블루보틀 성수')
    expect(state.cafe.location).toBe('서울 성수')
    expect(state.cafe.coffeeName).toBe('게이샤 드립')
    expect(state.cafe.beanOrigin).toBe('Ethiopia')
    expect(state.cafe.beanProcess).toBe('washed')
    expect(state.cafe.roastLevel).toBe('light')
    expect(state.cafe.tastingTags).toEqual(['자몽', '꽃'])
    expect(state.cafe.tastingNote).toBe('깔끔한 산미')
  })

  it('brew 로그 복제 시 tastingNote가 유지된다', () => {
    const state = cloneToFormState(brewLog, cloneNow)

    expect(state.brew.tastingNote).toBe('달콤한 뒷맛')
  })

  it('brew 로그 복제 시 리셋 대상 필드가 초기화된다', () => {
    const state = cloneToFormState(brewLog, cloneNow)

    // 리셋 대상
    expect(state.recordedAt).toBe('2026-04-04T08:00')
    expect(state.companions).toEqual([])
    expect(state.memo).toBe('')
    expect(state.brew.rating).toBe('')
    expect(state.brew.impressions).toBe('')
  })

  it('brew 로그 복제 시 나머지 필드는 원본 값을 유지한다', () => {
    const state = cloneToFormState(brewLog, cloneNow)

    expect(state.logType).toBe('brew')
    expect(state.brew.beanName).toBe('케냐 AB')
    expect(state.brew.beanOrigin).toBe('Kenya')
    expect(state.brew.roastLevel).toBe('medium')
    expect(state.brew.roastDate).toBe('2026-03-18')
    expect(state.brew.tastingTags).toEqual(['베리', '홍차'])
    expect(state.brew.tastingNote).toBe('달콤한 뒷맛')
    expect(state.brew.brewMethod).toBe('pour_over')
    expect(state.brew.brewDevice).toBe('Origami')
    expect(state.brew.coffeeAmountG).toBe('18')
    expect(state.brew.waterAmountMl).toBe('300')
    expect(state.brew.waterTempC).toBe('92')
    expect(state.brew.brewTimeSec).toBe('165')
    expect(state.brew.grindSize).toBe('중간')
    expect(state.brew.brewSteps).toEqual(['뜸 40초', '3회 나눠 붓기'])
  })
})

describe('hasOptionalValues', () => {
  it('빈 폼 상태에서는 false를 반환한다', () => {
    const state = createEmptyFormState()
    expect(hasOptionalValues(state)).toBe(false)
  })

  it('cafe 선택 필드에 값이 있으면 true를 반환한다', () => {
    const state = createEmptyFormState()
    state.cafe.tastingTags = ['초콜릿']
    expect(hasOptionalValues(state)).toBe(true)
  })

  it('brew 선택 필드에 값이 있으면 true를 반환한다', () => {
    const state = createEmptyFormState()
    state.logType = 'brew'
    state.brew.coffeeAmountG = '18'
    expect(hasOptionalValues(state)).toBe(true)
  })

  it('공통 선택 필드(companions)에 값이 있으면 true를 반환한다', () => {
    const state = createEmptyFormState()
    state.companions = ['민수']
    expect(hasOptionalValues(state)).toBe(true)
  })

  it('공통 선택 필드(memo)에 값이 있으면 true를 반환한다', () => {
    const state = createEmptyFormState()
    state.memo = '오늘 커피가 특히 맛있었다'
    expect(hasOptionalValues(state)).toBe(true)
  })

  it('cafe의 impressions에 값이 있으면 true를 반환한다', () => {
    const state = createEmptyFormState()
    state.cafe.impressions = '다음에 또 가고 싶다'
    expect(hasOptionalValues(state)).toBe(true)
  })

  it('brew의 brewSteps가 빈 스텝만 있으면 false를 반환한다', () => {
    const state = createEmptyFormState()
    state.logType = 'brew'
    state.brew.brewSteps = ['', '  ']
    expect(hasOptionalValues(state)).toBe(false)
  })

  it('brew의 brewSteps에 실제 내용이 있으면 true를 반환한다', () => {
    const state = createEmptyFormState()
    state.logType = 'brew'
    state.brew.brewSteps = ['뜸 30초']
    expect(hasOptionalValues(state)).toBe(true)
  })
})
