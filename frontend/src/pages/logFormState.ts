import type { BrewLogFull, CoffeeLogFull, CreateLogInput, LogStatus } from '../types/log'
import type { PresetFull } from '../types/preset'

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
  status: LogStatus
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
    // 신규 폼은 발행 의도로 시작한다. 사용자가 "임시 저장"을 누를 때만
    // buildLogPayload(state, { status: 'draft' })로 override된다.
    status: 'published',
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
    status: log.status,
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

  // 리셋 대상 필드 — 복제는 새 published 로그 작성을 의도한다.
  state.status = 'published'
  state.recordedAt = toDateTimeLocal(now)
  state.companions = []
  state.memo = ''

  if (state.logType === 'cafe') {
    state.cafe = { ...state.cafe, rating: '', impressions: '' }
  } else {
    state.brew = { ...state.brew, rating: '', impressions: '' }
  }

  return state
}

/**
 * 이전 brew 로그에서 레시피 필드만 추출하여 새 로그 작성 폼의 초기��으로 변환한다.
 * cloneToFormState()가 전체 복제 후 일부 리셋(subtractive)하는 것과 달리,
 * 빈 폼에서 시작하여 레시피 필드만 채우는(additive) 방식이다.
 */
export function recipeToFormState(log: BrewLogFull, now = new Date()): LogFormState {
  const state = createEmptyFormState(now)
  state.logType = 'brew'

  state.brew = {
    ...state.brew,
    beanName: log.brew.bean_name,
    beanOrigin: log.brew.bean_origin ?? '',
    beanProcess: log.brew.bean_process ?? '',
    roastLevel: (log.brew.roast_level ?? '') as RoastLevelValue,
    roastDate: log.brew.roast_date ?? '',
    brewMethod: log.brew.brew_method as BrewMethodValue,
    brewDevice: log.brew.brew_device ?? '',
    coffeeAmountG: log.brew.coffee_amount_g ? String(log.brew.coffee_amount_g) : '',
    waterAmountMl: log.brew.water_amount_ml ? String(log.brew.water_amount_ml) : '',
    waterTempC: log.brew.water_temp_c ? String(log.brew.water_temp_c) : '',
    brewTimeSec: log.brew.brew_time_sec ? String(log.brew.brew_time_sec) : '',
    grindSize: log.brew.grind_size ?? '',
    brewSteps:
      log.brew.brew_steps && log.brew.brew_steps.length > 0 ? log.brew.brew_steps : [''],
  }

  return state
}

/**
 * 프리셋을 새 로그 작성 폼의 초기값으로 변���한다.
 * 프리셋의 log_type과 전용 필드를 채우고, 나머지(날짜, 평가 등)는 빈 값으로 초기화한다.
 */
export function presetToFormState(preset: PresetFull, now = new Date()): LogFormState {
  const state = createEmptyFormState(now)
  state.logType = preset.log_type

  if (preset.log_type === 'cafe') {
    state.cafe = {
      ...state.cafe,
      cafeName: preset.cafe.cafe_name,
      coffeeName: preset.cafe.coffee_name,
      tastingTags: preset.cafe.tasting_tags ?? [],
    }
  }

  if (preset.log_type === 'brew') {
    state.brew = {
      ...state.brew,
      beanName: preset.brew.bean_name,
      brewMethod: preset.brew.brew_method as BrewMethodValue,
      brewSteps:
        preset.brew.brew_steps && preset.brew.brew_steps.length > 0
          ? preset.brew.brew_steps
          : [''],
    }
    if (preset.brew.recipe_detail) {
      state.memo = preset.brew.recipe_detail
    }
  }

  return state
}

export interface BuildLogPayloadOptions {
  /**
   * 저장 의도. 호출자가 명시한다(저장 버튼별로 결정).
   * draft는 cafe_name/coffee_name이나 bean_name/brew_method가 한쪽만 채워져도 통과,
   * published는 기존 필수 필드 검증을 그대로 적용받는다.
   */
  status: LogStatus
}

export function buildLogPayload(
  state: LogFormState,
  options: BuildLogPayloadOptions,
): CreateLogInput {
  const payload: CreateLogInput = {
    recorded_at: toApiRecordedAt(state.recordedAt),
    companions: state.companions,
    log_type: state.logType,
    status: options.status,
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
 * 드래프트(임시 저장)의 최소 조건을 만족하는지 검사한다.
 * - cafe: cafe_name 또는 coffee_name 중 하나가 비어있지 않아야 함
 * - brew: bean_name 또는 brew_method 중 하나가 채워져 있어야 함
 *
 * brewMethod는 createEmptyFormState에서 'pour_over' 기본값이므로 brew 폼은
 * 사실상 항상 true가 되지만, 사용자가 명시적으로 비웠을 가능성에 대비해 검사한다.
 */
export function canSaveAsDraft(state: LogFormState): boolean {
  if (state.logType === 'cafe') {
    return state.cafe.cafeName.trim() !== '' || state.cafe.coffeeName.trim() !== ''
  }
  // BrewMethodValue 타입상 brewMethod는 항상 enum 값 중 하나로 채워져 있으므로
  // 두 번째 조건은 사실상 항상 true다. 사용자가 의도적으로 brewMethod만 선택한
  // 케이스를 허용하기 위해 비교는 그대로 유지한다(plan.md의 알려진 한계 참고).
  return state.brew.beanName.trim() !== '' || Boolean(state.brew.brewMethod)
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
