import type { CoffeeLogFull, CreateLogInput } from '../types/log'

export type FormLogType = 'cafe' | 'brew'
export type RoastLevelValue = 'light' | 'medium' | 'dark' | ''
export type BrewMethodValue =
  | 'pour_over'
  | 'immersion'
  | 'aeropress'
  | 'espresso'
  | 'moka_pot'
  | 'siphon'
  | 'cold_brew'
  | 'other'

export interface LogFormState {
  logType: FormLogType
  recordedAt: string
  companions: string[]
  memo: string
  cafe: {
    cafeName: string
    location: string
    coffeeName: string
    beanOrigin: string
    beanProcess: string
    roastLevel: RoastLevelValue
    tastingTags: string[]
    tastingNote: string
    impressions: string
    rating: string
  }
  brew: {
    beanName: string
    beanOrigin: string
    beanProcess: string
    roastLevel: RoastLevelValue
    roastDate: string
    tastingTags: string[]
    tastingNote: string
    brewMethod: BrewMethodValue
    brewDevice: string
    coffeeAmountG: string
    waterAmountMl: string
    waterTempC: string
    brewTimeSec: string
    grindSize: string
    brewSteps: string[]
    impressions: string
    rating: string
  }
}

export const roastLevelOptions = [
  { label: 'Select roast', value: '' },
  { label: 'Light', value: 'light' },
  { label: 'Medium', value: 'medium' },
  { label: 'Dark', value: 'dark' },
] as const

export const brewMethodOptions = [
  { label: 'Pour Over', description: '핸드드립 계열', value: 'pour_over' },
  { label: 'Immersion', description: '침지 계열', value: 'immersion' },
  { label: 'AeroPress', description: '압력 + 침지', value: 'aeropress' },
  { label: 'Espresso', description: '고압 추출', value: 'espresso' },
  { label: 'Moka Pot', description: '스토브탑 증기압', value: 'moka_pot' },
  { label: 'Siphon', description: '진공 사이폰', value: 'siphon' },
  { label: 'Cold Brew', description: '저온 장시간 침지', value: 'cold_brew' },
  { label: 'Other', description: '기타', value: 'other' },
] as const

function toDateTimeLocal(value: Date) {
  const year = value.getFullYear()
  const month = String(value.getMonth() + 1).padStart(2, '0')
  const day = String(value.getDate()).padStart(2, '0')
  const hours = String(value.getHours()).padStart(2, '0')
  const minutes = String(value.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hours}:${minutes}`
}

function fromRecordedAt(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return ''
  }
  return toDateTimeLocal(parsed)
}

function toApiRecordedAt(value: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    return ''
  }

  const parsed = new Date(trimmed)
  if (Number.isNaN(parsed.getTime())) {
    return trimmed
  }

  return parsed.toISOString()
}

function normalizeText(value: string) {
  const trimmed = value.trim()
  return trimmed ? trimmed : undefined
}

function normalizeNumber(value: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    return undefined
  }

  const parsed = Number(trimmed)
  return Number.isFinite(parsed) ? parsed : undefined
}


export function createEmptyFormState(now = new Date()): LogFormState {
  return {
    logType: 'cafe',
    recordedAt: toDateTimeLocal(now),
    companions: [],
    memo: '',
    cafe: {
      cafeName: '',
      location: '',
      coffeeName: '',
      beanOrigin: '',
      beanProcess: '',
      roastLevel: '',
      tastingTags: [],
      tastingNote: '',
      impressions: '',
      rating: '',
    },
    brew: {
      beanName: '',
      beanOrigin: '',
      beanProcess: '',
      roastLevel: '',
      roastDate: '',
      tastingTags: [],
      tastingNote: '',
      brewMethod: 'pour_over',
      brewDevice: '',
      coffeeAmountG: '',
      waterAmountMl: '',
      waterTempC: '',
      brewTimeSec: '',
      grindSize: '',
      brewSteps: [''],
      impressions: '',
      rating: '',
    },
  }
}

export function logToFormState(log: CoffeeLogFull): LogFormState {
  const base = createEmptyFormState()
  const state: LogFormState = {
    ...base,
    logType: log.log_type,
    recordedAt: fromRecordedAt(log.recorded_at),
    companions: log.companions ?? [],
    memo: log.memo ?? '',
  }

  if (log.log_type === 'cafe') {
    state.cafe = {
      cafeName: log.cafe.cafe_name,
      location: log.cafe.location ?? '',
      coffeeName: log.cafe.coffee_name,
      beanOrigin: log.cafe.bean_origin ?? '',
      beanProcess: log.cafe.bean_process ?? '',
      roastLevel: (log.cafe.roast_level ?? '') as RoastLevelValue,
      tastingTags: log.cafe.tasting_tags ?? [],
      tastingNote: log.cafe.tasting_note ?? '',
      impressions: log.cafe.impressions ?? '',
      rating: log.cafe.rating ? String(log.cafe.rating) : '',
    }
  }

  if (log.log_type === 'brew') {
    state.brew = {
      beanName: log.brew.bean_name,
      beanOrigin: log.brew.bean_origin ?? '',
      beanProcess: log.brew.bean_process ?? '',
      roastLevel: (log.brew.roast_level ?? '') as RoastLevelValue,
      roastDate: log.brew.roast_date ?? '',
      tastingTags: log.brew.tasting_tags ?? [],
      tastingNote: log.brew.tasting_note ?? '',
      brewMethod: log.brew.brew_method as BrewMethodValue,
      brewDevice: log.brew.brew_device ?? '',
      coffeeAmountG: log.brew.coffee_amount_g ? String(log.brew.coffee_amount_g) : '',
      waterAmountMl: log.brew.water_amount_ml ? String(log.brew.water_amount_ml) : '',
      waterTempC: log.brew.water_temp_c ? String(log.brew.water_temp_c) : '',
      brewTimeSec: log.brew.brew_time_sec ? String(log.brew.brew_time_sec) : '',
      grindSize: log.brew.grind_size ?? '',
      brewSteps:
        log.brew.brew_steps && log.brew.brew_steps.length > 0 ? log.brew.brew_steps : [''],
      impressions: log.brew.impressions ?? '',
      rating: log.brew.rating ? String(log.brew.rating) : '',
    }
  }

  return state
}

/**
 * 기존 로그를 복제하여 새 로그 작성 폼의 초기값으로 변환한다.
 * logToFormState()로 원본을 변환한 뒤, 복제 시 리셋할 필드를 초기화한다.
 */
export function cloneToFormState(log: CoffeeLogFull, now = new Date()): LogFormState {
  const state = logToFormState(log)

  // 리셋 대상 필드
  state.recordedAt = toDateTimeLocal(now)
  state.companions = []
  state.memo = ''

  if (state.logType === 'cafe') {
    state.cafe.rating = ''
    state.cafe.impressions = ''
  } else {
    state.brew.rating = ''
    state.brew.impressions = ''
  }

  return state
}

export function buildLogPayload(state: LogFormState): CreateLogInput {
  const payload: CreateLogInput = {
    recorded_at: toApiRecordedAt(state.recordedAt),
    companions: state.companions,
    log_type: state.logType,
  }

  const memo = normalizeText(state.memo)
  if (memo) {
    payload.memo = memo
  }

  if (state.logType === 'cafe') {
    payload.cafe = {
      cafe_name: state.cafe.cafeName.trim(),
      coffee_name: state.cafe.coffeeName.trim(),
      tasting_tags: state.cafe.tastingTags,
    }

    const location = normalizeText(state.cafe.location)
    const beanOrigin = normalizeText(state.cafe.beanOrigin)
    const beanProcess = normalizeText(state.cafe.beanProcess)
    const roastLevel = state.cafe.roastLevel || undefined
    const tastingNote = normalizeText(state.cafe.tastingNote)
    const impressions = normalizeText(state.cafe.impressions)
    const rating = normalizeNumber(state.cafe.rating)

    if (location) payload.cafe.location = location
    if (beanOrigin) payload.cafe.bean_origin = beanOrigin
    if (beanProcess) payload.cafe.bean_process = beanProcess
    if (roastLevel) payload.cafe.roast_level = roastLevel
    if (tastingNote) payload.cafe.tasting_note = tastingNote
    if (impressions) payload.cafe.impressions = impressions
    if (rating !== undefined) payload.cafe.rating = rating
  }

  if (state.logType === 'brew') {
    payload.brew = {
      bean_name: state.brew.beanName.trim(),
      brew_method: state.brew.brewMethod,
      tasting_tags: state.brew.tastingTags,
      brew_steps: state.brew.brewSteps.map((step) => step.trim()).filter(Boolean),
    }

    const beanOrigin = normalizeText(state.brew.beanOrigin)
    const beanProcess = normalizeText(state.brew.beanProcess)
    const roastLevel = state.brew.roastLevel || undefined
    const roastDate = normalizeText(state.brew.roastDate)
    const tastingNote = normalizeText(state.brew.tastingNote)
    const brewDevice = normalizeText(state.brew.brewDevice)
    const coffeeAmountG = normalizeNumber(state.brew.coffeeAmountG)
    const waterAmountMl = normalizeNumber(state.brew.waterAmountMl)
    const waterTempC = normalizeNumber(state.brew.waterTempC)
    const brewTimeSec = normalizeNumber(state.brew.brewTimeSec)
    const grindSize = normalizeText(state.brew.grindSize)
    const impressions = normalizeText(state.brew.impressions)
    const rating = normalizeNumber(state.brew.rating)

    if (beanOrigin) payload.brew.bean_origin = beanOrigin
    if (beanProcess) payload.brew.bean_process = beanProcess
    if (roastLevel) payload.brew.roast_level = roastLevel
    if (roastDate) payload.brew.roast_date = roastDate
    if (tastingNote) payload.brew.tasting_note = tastingNote
    if (brewDevice) payload.brew.brew_device = brewDevice
    if (coffeeAmountG !== undefined) payload.brew.coffee_amount_g = coffeeAmountG
    if (waterAmountMl !== undefined) payload.brew.water_amount_ml = waterAmountMl
    if (waterTempC !== undefined) payload.brew.water_temp_c = waterTempC
    if (brewTimeSec !== undefined) payload.brew.brew_time_sec = Math.round(brewTimeSec)
    if (grindSize) payload.brew.grind_size = grindSize
    if (impressions) payload.brew.impressions = impressions
    if (rating !== undefined) payload.brew.rating = rating
  }

  return payload
}

/**
 * 선택 영역 필드에 값이 하나라도 있는지 검사한다.
 * 수정 모드 진입 시 토글 자동 펼침 여부를 결정하는 데 사용한다.
 */
export function hasOptionalValues(state: LogFormState): boolean {
  // 공통 선택 필드
  if (state.companions.length > 0 || state.memo.trim() !== '') {
    return true
  }

  if (state.logType === 'cafe') {
    const c = state.cafe
    return (
      c.location.trim() !== '' ||
      c.beanOrigin.trim() !== '' ||
      c.beanProcess.trim() !== '' ||
      c.roastLevel !== '' ||
      c.tastingTags.length > 0 ||
      c.tastingNote.trim() !== '' ||
      c.impressions.trim() !== ''
    )
  }

  const b = state.brew
  return (
    b.beanOrigin.trim() !== '' ||
    b.beanProcess.trim() !== '' ||
    b.roastLevel !== '' ||
    b.roastDate.trim() !== '' ||
    b.brewDevice.trim() !== '' ||
    b.coffeeAmountG.trim() !== '' ||
    b.waterAmountMl.trim() !== '' ||
    b.waterTempC.trim() !== '' ||
    b.brewTimeSec.trim() !== '' ||
    b.grindSize.trim() !== '' ||
    b.tastingTags.length > 0 ||
    b.tastingNote.trim() !== '' ||
    b.brewSteps.some((step) => step.trim() !== '') ||
    b.impressions.trim() !== ''
  )
}
