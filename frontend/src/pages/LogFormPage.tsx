import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from 'react'
import { createPortal } from 'react-dom'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { Layout } from '../components/Layout'
import { RatingInput } from '../components/RatingInput'
import { TagInput } from '../components/TagInput'
import { ApiError } from '../api/client'
import { useCreateLog, useLog, useLogList, useUpdateLog } from '../hooks/useLogs'
import { usePresetList, useUsePreset } from '../hooks/usePresets'
import { useCompanionSuggestions, useTagSuggestions } from '../hooks/useSuggestions'
import {
  brewMethodOptions,
  buildLogPayload,
  cloneToFormState,
  createEmptyFormState,
  hasOptionalValues,
  logToFormState,
  presetToFormState,
  recipeToFormState,
  roastLevelOptions,
  type FormLogType,
  type LogFormState,
} from './logFormState'
import type { BrewLogFull, CoffeeLogFull } from '../types/log'
import type { PresetFull } from '../types/preset'

// --- 공통 UI 유틸 ---

function inputClassName() {
  return 'w-full rounded-2xl border border-amber-950/10 bg-white px-4 py-3 text-sm text-stone-900 outline-none transition placeholder:text-stone-400 focus:border-amber-900/35 focus:bg-amber-50/40'
}

function textareaClassName() {
  return `${inputClassName()} min-h-[116px] resize-y`
}

function Field({
  label,
  required,
  error,
  children,
}: {
  label: string
  required?: boolean
  error?: string
  children: ReactNode
}) {
  return (
    <label className="space-y-2">
      <span className="text-sm font-medium text-stone-800">
        {label}
        {required ? <span className="ml-1 text-amber-900">*</span> : null}
      </span>
      {children}
      {error ? <span className="block text-xs text-rose-600">{error}</span> : null}
    </label>
  )
}

function Section({
  title,
  description,
  error,
  children,
}: {
  title: string
  description: string
  error?: string
  children: ReactNode
}) {
  return (
    <section className="space-y-5 rounded-[1.75rem] border border-amber-950/10 bg-stone-50/65 p-5 sm:p-6">
      <div className="space-y-2">
        <h2 className="text-lg font-semibold text-stone-950">{title}</h2>
        <p className="text-sm leading-6 text-stone-600">{description}</p>
        {error ? <p className="text-xs text-rose-600">{error}</p> : null}
      </div>
      {children}
    </section>
  )
}

// --- 폼 상태 헬퍼 ---

function updateCafeField<K extends keyof LogFormState['cafe']>(
  state: LogFormState,
  key: K,
  value: LogFormState['cafe'][K],
) {
  return { ...state, cafe: { ...state.cafe, [key]: value } }
}

function updateBrewField<K extends keyof LogFormState['brew']>(
  state: LogFormState,
  key: K,
  value: LogFormState['brew'][K],
) {
  return { ...state, brew: { ...state.brew, [key]: value } }
}

// --- 섹션 컴포넌트 ---

type FieldErrors = Record<string, string>

function LogTypeSection({
  form,
  setForm,
  disabled,
  error,
}: {
  form: LogFormState
  setForm: React.Dispatch<React.SetStateAction<LogFormState>>
  disabled: boolean
  error?: string
}) {
  function handleLogTypeChange(logType: FormLogType) {
    if (disabled) return
    setForm((prev) => ({ ...prev, logType }))
  }

  return (
    <Section
      title="로그 유형"
      description={
        disabled
          ? '기존 로그 타입은 백엔드 제약에 따라 변경할 수 없습니다.'
          : '바리스타가 만들어준 커피는 cafe, 내가 직접 추출한 커피는 brew로 기록합니다.'
      }
      error={error}
    >
      <div className="grid gap-3 sm:grid-cols-2">
        {(['cafe', 'brew'] as const).map((type) => {
          const selected = form.logType === type
          return (
            <button
              key={type}
              type="button"
              onClick={() => handleLogTypeChange(type)}
              disabled={disabled}
              className={[
                'rounded-[1.5rem] border p-5 text-left transition',
                selected
                  ? 'border-amber-900 bg-amber-900 !text-white shadow-[0_16px_40px_rgba(123,79,34,0.22)]'
                  : 'border-amber-950/10 bg-white text-stone-800 hover:border-amber-900/25 hover:bg-amber-50/60',
                disabled ? 'cursor-not-allowed opacity-70' : '',
              ].join(' ')}
            >
              <p className="text-base font-semibold">{type === 'cafe' ? 'Cafe log' : 'Brew log'}</p>
              <p className={`mt-2 text-sm ${selected ? '!text-white/85' : 'text-stone-500'}`}>
                {type === 'cafe'
                  ? '카페 이름, 메뉴, 인상 중심으로 기록'
                  : '원두, 추출 방식, 레시피 중심으로 기록'}
              </p>
            </button>
          )
        })}
      </div>
    </Section>
  )
}

function PresetSection({
  logType,
  onSelect,
}: {
  logType: FormLogType
  onSelect: (preset: PresetFull) => void
}) {
  const { data: presets = [] } = usePresetList()
  const usePreset = useUsePreset()

  // 현재 logType에 맞는 프리셋만 필터링
  const filtered = presets.filter((p) => p.log_type === logType)

  if (filtered.length === 0) return null

  function handleSelect(preset: PresetFull) {
    onSelect(preset)
    usePreset.mutate(preset.id)
  }

  return (
    <section className="space-y-3 rounded-[1.75rem] border border-amber-950/10 bg-stone-50/65 p-5 sm:p-6">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold text-stone-950">프리셋</h2>
        <p className="text-sm text-stone-600">저장된 조합을 선택하면 필드가 자동으로 채워집니다.</p>
      </div>
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {filtered.map((preset) => (
          <button
            key={preset.id}
            type="button"
            onClick={() => handleSelect(preset)}
            className="rounded-2xl border border-amber-950/10 bg-white p-4 text-left transition hover:border-amber-900/25 hover:bg-amber-50/60"
          >
            <p className="text-sm font-semibold text-stone-900">{preset.name}</p>
            <p className="mt-1 text-xs text-stone-500">
              {preset.log_type === 'cafe'
                ? `${preset.cafe.cafe_name} · ${preset.cafe.coffee_name}`
                : `${preset.brew.bean_name} · ${preset.brew.brew_method.replace('_', ' ')}`}
            </p>
          </button>
        ))}
      </div>
    </section>
  )
}

// --- 레시피 불러오기 ---

const brewMethodLabelMap: Record<string, string> = {
  pour_over: 'Pour Over',
  immersion: 'Immersion',
  aeropress: 'AeroPress',
  espresso: 'Espresso',
  moka_pot: 'Moka Pot',
  siphon: 'Siphon',
  cold_brew: 'Cold Brew',
  other: 'Other',
}

const dateFormatter = new Intl.DateTimeFormat('ko-KR', { dateStyle: 'medium' })

function RecipePickerModal({
  open,
  onClose,
  onSelect,
}: {
  open: boolean
  onClose: () => void
  onSelect: (log: BrewLogFull) => void
}) {
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    useLogList({ log_type: 'brew' })

  if (!open) return null

  const logs = data?.pages.flatMap((p) => p.items) ?? []

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="mx-4 w-full max-w-md rounded-[1.75rem] border border-amber-950/10 bg-white shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between border-b border-amber-950/10 px-6 pt-5 pb-4">
          <h3 className="text-lg font-semibold text-stone-950">이전 레시피 불러오기</h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-full p-1.5 text-stone-500 transition hover:bg-stone-100 hover:text-stone-800"
          >
            ✕
          </button>
        </div>

        <div className="max-h-[60vh] overflow-y-auto p-4">
          {isLoading ? (
            <p className="py-8 text-center text-sm text-stone-500">불러오는 중...</p>
          ) : logs.length === 0 ? (
            <p className="py-8 text-center text-sm text-stone-500">
              아직 brew 기록이 없습니다.
            </p>
          ) : (
            <div className="space-y-2">
              {logs.map((log) => (
                <button
                  key={log.id}
                  type="button"
                  onClick={() => onSelect(log as BrewLogFull)}
                  className="w-full rounded-2xl border border-amber-950/10 bg-stone-50/65 p-4 text-left transition hover:border-amber-900/25 hover:bg-amber-50/60"
                >
                  <p className="text-sm font-semibold text-stone-900">
                    {(log as BrewLogFull).brew.bean_name}
                  </p>
                  <p className="mt-1 text-xs text-stone-500">
                    {brewMethodLabelMap[(log as BrewLogFull).brew.brew_method] ?? (log as BrewLogFull).brew.brew_method}
                    {' · '}
                    {dateFormatter.format(new Date(log.recorded_at))}
                  </p>
                </button>
              ))}
              {hasNextPage ? (
                <button
                  type="button"
                  onClick={() => void fetchNextPage()}
                  disabled={isFetchingNextPage}
                  className="w-full rounded-full border border-amber-950/10 bg-white px-4 py-2.5 text-sm font-semibold text-stone-700 transition hover:border-amber-900/25 hover:bg-amber-50/60 disabled:opacity-50"
                >
                  {isFetchingNextPage ? '불러오는 중...' : '더 보기'}
                </button>
              ) : null}
            </div>
          )}
        </div>
      </div>
    </div>,
    document.body,
  )
}

function RecipePickerSection({
  onSelect,
}: {
  onSelect: (log: BrewLogFull) => void
}) {
  const [open, setOpen] = useState(false)

  return (
    <>
      <section className="space-y-3 rounded-[1.75rem] border border-amber-950/10 bg-stone-50/65 p-5 sm:p-6">
        <div className="space-y-1">
          <h2 className="text-lg font-semibold text-stone-950">레시피 불러오기</h2>
          <p className="text-sm text-stone-600">이전 브루 기록의 레시피를 불러와 수정할 수 있습니다.</p>
        </div>
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="rounded-full border border-amber-950/10 bg-white px-4 py-2.5 text-sm font-semibold text-stone-700 transition hover:border-amber-900/25 hover:bg-amber-50/60"
        >
          이전 레시피 불러오기
        </button>
      </section>
      <RecipePickerModal
        open={open}
        onClose={() => setOpen(false)}
        onSelect={(log) => {
          onSelect(log)
          setOpen(false)
        }}
      />
    </>
  )
}

// --- 공통 선택 필드 (companions, memo) ---
// CafeFieldsSection / BrewFieldsSection 선택 영역에서 공통으로 렌더링한다.

function OptionalCommonFields({
  form,
  setForm,
}: {
  form: LogFormState
  setForm: React.Dispatch<React.SetStateAction<LogFormState>>
}) {
  const [companionQuery, setCompanionQuery] = useState('')
  const { data: companionSuggestions = [] } = useCompanionSuggestions(companionQuery)

  return (
    <>
      <div className="md:col-span-2">
        <Field label="Companions">
          <TagInput
            value={form.companions}
            onChange={(tags) => setForm((prev) => ({ ...prev, companions: tags }))}
            suggestions={companionSuggestions}
            onQueryChange={setCompanionQuery}
            placeholder="민수, 지연"
          />
        </Field>
      </div>
      <div className="md:col-span-2">
        <Field label="Memo">
          <textarea
            className={textareaClassName()}
            name="memo"
            value={form.memo}
            onChange={(e) => setForm((prev) => ({ ...prev, memo: e.target.value }))}
            placeholder="오늘의 한 잔이 남긴 기억을 자유롭게 적어보세요."
          />
        </Field>
      </div>
    </>
  )
}

function ToggleButton({
  expanded,
  onToggle,
}: {
  expanded: boolean
  onToggle: () => void
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      aria-expanded={expanded}
      className="w-full rounded-full border border-amber-950/10 bg-white px-4 py-2.5 text-sm font-semibold text-stone-700 transition hover:border-amber-900/25 hover:bg-amber-50/60"
    >
      {expanded ? '접기' : '더 기록하기'}
    </button>
  )
}

function CafeFieldsSection({
  form,
  setForm,
  fieldErrors,
  expanded,
  onToggle,
}: {
  form: LogFormState
  setForm: React.Dispatch<React.SetStateAction<LogFormState>>
  fieldErrors: FieldErrors
  expanded: boolean
  onToggle: () => void
}) {
  const [tagsQuery, setTagsQuery] = useState('')
  const { data: tagSuggestions = [] } = useTagSuggestions(tagsQuery)

  return (
    <div className="space-y-4">
      {/* 필수 영역 */}
      <Section
        title="카페 정보"
        description="카페 이름, 메뉴, 평가를 입력합니다."
        error={fieldErrors['cafe']}
      >
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Recorded at" required error={fieldErrors['recorded_at']}>
            <input
              className={inputClassName()}
              type="datetime-local"
              value={form.recordedAt}
              onChange={(e) => setForm((prev) => ({ ...prev, recordedAt: e.target.value }))}
              required
            />
          </Field>
          <Field label="Cafe name" required error={fieldErrors['cafe.cafe_name']}>
            <input
              className={inputClassName()}
              value={form.cafe.cafeName}
              onChange={(e) => setForm((prev) => updateCafeField(prev, 'cafeName', e.target.value))}
              required
            />
          </Field>
          <Field label="Coffee name" required error={fieldErrors['cafe.coffee_name']}>
            <input
              className={inputClassName()}
              value={form.cafe.coffeeName}
              onChange={(e) => setForm((prev) => updateCafeField(prev, 'coffeeName', e.target.value))}
              required
            />
          </Field>
          <div className="md:col-span-2">
            <Field label="Rating" error={fieldErrors['cafe.rating']}>
              <RatingInput
                value={form.cafe.rating ? Number(form.cafe.rating) : null}
                onChange={(value) =>
                  setForm((prev) =>
                    updateCafeField(prev, 'rating', value ? value.toFixed(1) : ''),
                  )
                }
              />
            </Field>
          </div>
        </div>
      </Section>

      {/* 토글 버튼 */}
      <ToggleButton expanded={expanded} onToggle={onToggle} />

      {/* 선택 영역 */}
      {expanded && (
        <Section
          title="추가 정보"
          description="장소, 원두, 테이스팅 노트 등 상세 정보를 기록합니다."
        >
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Location">
              <input
                className={inputClassName()}
                value={form.cafe.location}
                onChange={(e) => setForm((prev) => updateCafeField(prev, 'location', e.target.value))}
                placeholder="서울 성수"
              />
            </Field>
            <Field label="Roast level">
              <select
                className={inputClassName()}
                value={form.cafe.roastLevel}
                onChange={(e) =>
                  setForm((prev) =>
                    updateCafeField(prev, 'roastLevel', e.target.value as LogFormState['cafe']['roastLevel']),
                  )
                }
              >
                {roastLevelOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Bean origin">
              <input
                className={inputClassName()}
                value={form.cafe.beanOrigin}
                onChange={(e) => setForm((prev) => updateCafeField(prev, 'beanOrigin', e.target.value))}
              />
            </Field>
            <Field label="Bean process">
              <input
                className={inputClassName()}
                value={form.cafe.beanProcess}
                onChange={(e) => setForm((prev) => updateCafeField(prev, 'beanProcess', e.target.value))}
              />
            </Field>
            <div className="md:col-span-2">
              <Field label="Tasting tags">
                <TagInput
                  value={form.cafe.tastingTags}
                  onChange={(tags) => setForm((prev) => updateCafeField(prev, 'tastingTags', tags))}
                  suggestions={tagSuggestions}
                  onQueryChange={setTagsQuery}
                  placeholder="초콜릿, 체리, 헤이즐넛"
                />
              </Field>
            </div>
            <div className="md:col-span-2">
              <Field label="Tasting note">
                <textarea
                  className={textareaClassName()}
                  value={form.cafe.tastingNote}
                  onChange={(e) =>
                    setForm((prev) => updateCafeField(prev, 'tastingNote', e.target.value))
                  }
                />
              </Field>
            </div>
            <div className="md:col-span-2">
              <Field label="Impressions">
                <textarea
                  className={textareaClassName()}
                  value={form.cafe.impressions}
                  onChange={(e) =>
                    setForm((prev) => updateCafeField(prev, 'impressions', e.target.value))
                  }
                />
              </Field>
            </div>
            <OptionalCommonFields form={form} setForm={setForm} />
          </div>
        </Section>
      )}
    </div>
  )
}

function BrewFieldsSection({
  form,
  setForm,
  fieldErrors,
  expanded,
  onToggle,
}: {
  form: LogFormState
  setForm: React.Dispatch<React.SetStateAction<LogFormState>>
  fieldErrors: FieldErrors
  expanded: boolean
  onToggle: () => void
}) {
  const [tagsQuery, setTagsQuery] = useState('')
  const { data: tagSuggestions = [] } = useTagSuggestions(tagsQuery)

  // coffee/water 비율은 brew 섹션 내에서만 사용하므로 여기서 계산한다
  const ratio = useMemo(() => {
    const coffee = Number(form.brew.coffeeAmountG)
    const water = Number(form.brew.waterAmountMl)
    if (!Number.isFinite(coffee) || !Number.isFinite(water) || coffee <= 0 || water <= 0) {
      return null
    }
    return (water / coffee).toFixed(1)
  }, [form.brew.coffeeAmountG, form.brew.waterAmountMl])

  function updateStep(index: number, value: string) {
    setForm((prev) =>
      updateBrewField(prev, 'brewSteps', prev.brew.brewSteps.map((step, i) => (i === index ? value : step))),
    )
  }

  function addStep() {
    setForm((prev) => updateBrewField(prev, 'brewSteps', [...prev.brew.brewSteps, '']))
  }

  function moveStep(index: number, direction: -1 | 1) {
    setForm((prev) => {
      const nextIndex = index + direction
      if (nextIndex < 0 || nextIndex >= prev.brew.brewSteps.length) return prev
      const nextSteps = [...prev.brew.brewSteps]
      ;[nextSteps[index], nextSteps[nextIndex]] = [nextSteps[nextIndex], nextSteps[index]]
      return updateBrewField(prev, 'brewSteps', nextSteps)
    })
  }

  function removeStep(index: number) {
    setForm((prev) => {
      const nextSteps = prev.brew.brewSteps.filter((_, i) => i !== index)
      return updateBrewField(prev, 'brewSteps', nextSteps.length > 0 ? nextSteps : [''])
    })
  }

  return (
    <div className="space-y-4">
      {/* 필수 영역 */}
      <Section
        title="브루 정보"
        description="원두와 추출 방식, 평가를 입력합니다."
        error={fieldErrors['brew']}
      >
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Recorded at" required error={fieldErrors['recorded_at']}>
            <input
              className={inputClassName()}
              type="datetime-local"
              value={form.recordedAt}
              onChange={(e) => setForm((prev) => ({ ...prev, recordedAt: e.target.value }))}
              required
            />
          </Field>
          <div className="md:col-span-2">
            <Field label="Bean name" required error={fieldErrors['brew.bean_name']}>
              <input
                className={inputClassName()}
                value={form.brew.beanName}
                onChange={(e) => setForm((prev) => updateBrewField(prev, 'beanName', e.target.value))}
                required
              />
            </Field>
          </div>

          {/* 추출 방식 — 버튼 그룹 */}
          <div className="md:col-span-2">
            <Field label="Brew method" required error={fieldErrors['brew.brew_method']}>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                {brewMethodOptions.map((option) => {
                  const selected = form.brew.brewMethod === option.value
                  return (
                    <button
                      key={option.value}
                      type="button"
                      onClick={() =>
                        setForm((prev) =>
                          updateBrewField(prev, 'brewMethod', option.value),
                        )
                      }
                      className={[
                        'rounded-2xl border px-3 py-2.5 text-left transition',
                        selected
                          ? 'border-amber-900 bg-amber-900 shadow-[0_8px_24px_rgba(123,79,34,0.18)]'
                          : 'border-amber-950/10 bg-white hover:border-amber-900/25 hover:bg-amber-50/60',
                      ].join(' ')}
                    >
                      <p className={`text-sm font-semibold ${selected ? 'text-white' : 'text-stone-800'}`}>
                        {option.label}
                      </p>
                      <p className={`mt-0.5 text-xs ${selected ? 'text-white/75' : 'text-stone-500'}`}>
                        {option.description}
                      </p>
                    </button>
                  )
                })}
              </div>
            </Field>
          </div>

          <div className="md:col-span-2">
            <Field label="Rating" error={fieldErrors['brew.rating']}>
              <RatingInput
                value={form.brew.rating ? Number(form.brew.rating) : null}
                onChange={(value) =>
                  setForm((prev) =>
                    updateBrewField(prev, 'rating', value ? value.toFixed(1) : ''),
                  )
                }
              />
            </Field>
          </div>
        </div>
      </Section>

      {/* 토글 버튼 */}
      <ToggleButton expanded={expanded} onToggle={onToggle} />

      {/* 선택 영역 */}
      {expanded && (
        <Section
          title="추가 정보"
          description="레시피, 원두 상세, 테이스팅 노트 등을 기록합니다."
        >
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Brew device">
              <input
                className={inputClassName()}
                value={form.brew.brewDevice}
                onChange={(e) =>
                  setForm((prev) => updateBrewField(prev, 'brewDevice', e.target.value))
                }
                placeholder="Origami, AeroPress Go, Breville..."
              />
            </Field>
            <Field label="Grind size">
              <input
                className={inputClassName()}
                value={form.brew.grindSize}
                onChange={(e) => setForm((prev) => updateBrewField(prev, 'grindSize', e.target.value))}
                placeholder="중간, 20 clicks"
              />
            </Field>

            <Field label="Bean origin">
              <input
                className={inputClassName()}
                value={form.brew.beanOrigin}
                onChange={(e) => setForm((prev) => updateBrewField(prev, 'beanOrigin', e.target.value))}
              />
            </Field>
            <Field label="Bean process">
              <input
                className={inputClassName()}
                value={form.brew.beanProcess}
                onChange={(e) =>
                  setForm((prev) => updateBrewField(prev, 'beanProcess', e.target.value))
                }
              />
            </Field>
            <Field label="Roast level">
              <select
                className={inputClassName()}
                value={form.brew.roastLevel}
                onChange={(e) =>
                  setForm((prev) =>
                    updateBrewField(prev, 'roastLevel', e.target.value as LogFormState['brew']['roastLevel']),
                  )
                }
              >
                {roastLevelOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Roast date">
              <input
                className={inputClassName()}
                type="date"
                value={form.brew.roastDate}
                onChange={(e) => setForm((prev) => updateBrewField(prev, 'roastDate', e.target.value))}
              />
            </Field>

            {/* 레시피 — 원두량:물량 비율 인라인 표시 */}
            <div className="md:col-span-2 rounded-[1.75rem] border border-amber-950/10 bg-white p-4 space-y-4">
              <p className="text-sm font-semibold text-stone-900">Recipe</p>

              <div className="grid grid-cols-[1fr_auto_1fr] items-end gap-3">
                <Field label="Coffee (g)" error={fieldErrors['brew.coffee_amount_g']}>
                  <input
                    className={inputClassName()}
                    type="number"
                    min="0"
                    step="0.1"
                    value={form.brew.coffeeAmountG}
                    onChange={(e) =>
                      setForm((prev) => updateBrewField(prev, 'coffeeAmountG', e.target.value))
                    }
                  />
                </Field>

                <div className="flex flex-col items-center gap-1 pb-1">
                  <span className="text-xs text-stone-400">ratio</span>
                  <span
                    className={`text-base font-bold tabular-nums transition-colors ${
                      ratio ? 'text-amber-900' : 'text-stone-300'
                    }`}
                  >
                    {ratio ? `1 : ${ratio}` : '1 : —'}
                  </span>
                </div>

                <Field label="Water (ml)" error={fieldErrors['brew.water_amount_ml']}>
                  <input
                    className={inputClassName()}
                    type="number"
                    min="0"
                    step="0.1"
                    value={form.brew.waterAmountMl}
                    onChange={(e) =>
                      setForm((prev) => updateBrewField(prev, 'waterAmountMl', e.target.value))
                    }
                  />
                </Field>
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <Field label="Water temperature (°C)" error={fieldErrors['brew.water_temp_c']}>
                  <input
                    className={inputClassName()}
                    type="number"
                    min="0"
                    step="0.1"
                    value={form.brew.waterTempC}
                    onChange={(e) =>
                      setForm((prev) => updateBrewField(prev, 'waterTempC', e.target.value))
                    }
                  />
                </Field>
                <Field label="Brew time (sec)" error={fieldErrors['brew.brew_time_sec']}>
                  <input
                    className={inputClassName()}
                    type="number"
                    min="0"
                    step="1"
                    value={form.brew.brewTimeSec}
                    onChange={(e) =>
                      setForm((prev) => updateBrewField(prev, 'brewTimeSec', e.target.value))
                    }
                  />
                </Field>
              </div>
            </div>

            <div className="md:col-span-2">
              <Field label="Tasting tags">
                <TagInput
                  value={form.brew.tastingTags}
                  onChange={(tags) => setForm((prev) => updateBrewField(prev, 'tastingTags', tags))}
                  suggestions={tagSuggestions}
                  onQueryChange={setTagsQuery}
                  placeholder="꽃, 베리, 카카오"
                />
              </Field>
            </div>
            <div className="md:col-span-2">
              <Field label="Tasting note">
                <textarea
                  className={textareaClassName()}
                  value={form.brew.tastingNote}
                  onChange={(e) =>
                    setForm((prev) => updateBrewField(prev, 'tastingNote', e.target.value))
                  }
                />
              </Field>
            </div>
            <div className="md:col-span-2">
              <Field label="Brew steps">
                <div className="space-y-3">
                  {form.brew.brewSteps.map((step, index) => (
                    <div
                      key={`brew-step-${index}`}
                      className="rounded-[1.5rem] border border-amber-950/10 bg-white p-4"
                    >
                      <div className="flex items-center justify-between gap-3">
                        <p className="text-sm font-semibold text-stone-900">Step {index + 1}</p>
                        <div className="flex flex-wrap gap-2">
                          <button
                            type="button"
                            onClick={() => moveStep(index, -1)}
                            disabled={index === 0}
                            className="rounded-full border border-amber-950/10 px-3 py-1.5 text-xs font-semibold text-stone-600 transition hover:border-amber-900/25 hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            Up
                          </button>
                          <button
                            type="button"
                            onClick={() => moveStep(index, 1)}
                            disabled={index === form.brew.brewSteps.length - 1}
                            className="rounded-full border border-amber-950/10 px-3 py-1.5 text-xs font-semibold text-stone-600 transition hover:border-amber-900/25 hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            Down
                          </button>
                          <button
                            type="button"
                            onClick={() => removeStep(index)}
                            className="rounded-full border border-rose-300 bg-rose-50 px-3 py-1.5 text-xs font-semibold text-rose-700 transition hover:border-rose-400 hover:bg-rose-100"
                          >
                            삭제
                          </button>
                        </div>
                      </div>
                      <textarea
                        className={`${textareaClassName()} mt-3 min-h-[90px]`}
                        aria-label={`Brew step ${index + 1}`}
                        value={step}
                        onChange={(e) => updateStep(index, e.target.value)}
                        placeholder="예: 30초 뜸, 3회 나눠 붓기"
                      />
                    </div>
                  ))}
                  <button
                    type="button"
                    onClick={addStep}
                    className="rounded-full border border-amber-900/15 bg-amber-100/70 px-4 py-2 text-sm font-semibold text-amber-950 transition hover:border-amber-900/30 hover:bg-amber-100"
                  >
                    단계 추가
                  </button>
                </div>
              </Field>
            </div>
            <div className="md:col-span-2">
              <Field label="Impressions">
                <textarea
                  className={textareaClassName()}
                  value={form.brew.impressions}
                  onChange={(e) =>
                    setForm((prev) => updateBrewField(prev, 'impressions', e.target.value))
                  }
                />
              </Field>
            </div>
            <OptionalCommonFields form={form} setForm={setForm} />
          </div>
        </Section>
      )}
    </div>
  )
}

// --- 메인 페이지 ---

function getErrorMessage(error: unknown) {
  if (error instanceof ApiError) return error.message
  if (typeof error === 'object' && error !== null && 'message' in error) {
    return String((error as { message: unknown }).message)
  }
  return '저장 중 오류가 발생했습니다.'
}

export default function LogFormPage() {
  const { id } = useParams()
  const location = useLocation()
  const navigate = useNavigate()
  const cloneFrom = (location.state as { cloneFrom?: CoffeeLogFull } | null)?.cloneFrom
  const isEditMode = Boolean(id)
  const isCloneMode = !isEditMode && cloneFrom != null
  const [form, setForm] = useState(() => createEmptyFormState())
  const [expanded, setExpanded] = useState(false)
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const hydratedLogIDRef = useRef<string | null>(null)

  const createMutation = useCreateLog()
  const updateMutation = useUpdateLog(id ?? '')
  const { data: log, error: loadError, isError: isLoadError, isLoading } = useLog(id ?? '')

  useEffect(() => {
    if (!isEditMode) {
      hydratedLogIDRef.current = null
      return
    }
    if (!log || log.id === hydratedLogIDRef.current) {
      return
    }

    // 비동기 조회 결과를 수정 가능한 폼 draft로 옮기는 초기 hydrate 단계다.
    // 사용자 입력 이후에는 hydratedLogIDRef로 재주입을 막는다.
    const formState = logToFormState(log)
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setForm(formState)
    // 선택 필드에 값이 있으면 자동으로 펼친다
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setExpanded(hasOptionalValues(formState))
    hydratedLogIDRef.current = log.id
  }, [isEditMode, log])

  // clone 모드: 전달받은 로그 데이터로 폼 초기화
  useEffect(() => {
    if (!isCloneMode || hydratedLogIDRef.current === 'clone') {
      return
    }
    const formState = cloneToFormState(cloneFrom)
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setForm(formState)
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setExpanded(hasOptionalValues(formState))
    hydratedLogIDRef.current = 'clone'
  }, [isCloneMode, cloneFrom])

  const activeMutation = isEditMode ? updateMutation : createMutation
  const toggleExpanded = () => setExpanded((prev) => !prev)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFieldErrors({})
    const payload = buildLogPayload(form)

    try {
      const saved = isEditMode && id
        ? await updateMutation.mutateAsync(payload)
        : await createMutation.mutateAsync(payload)
      navigate(`/logs/${saved.id}`)
    } catch (err) {
      // 필드 단위 validation 에러는 해당 필드 아래 인라인으로 표시한다
      if (err instanceof ApiError && err.field) {
        setFieldErrors({ [err.field]: err.message })
      }
    }
  }

  return (
    <Layout
      title={isEditMode ? '기록 수정' : isCloneMode ? '기록 복제' : '커피 기록 추가'}
      description={isCloneMode ? '이전 기록을 바탕으로 새 기록을 작성합니다. 날짜와 평가는 초기화됩니다.' : '필수 정보만 빠르게 기록하고, 더 기록하기를 눌러 상세 정보를 추가할 수 있습니다.'}
      actions={
        <>
          <Link
            to={isEditMode && id ? `/logs/${id}` : '/'}
            className="inline-flex items-center justify-center whitespace-nowrap rounded-full border border-stone-950/10 px-4 py-2 text-sm font-semibold text-stone-700 transition hover:border-stone-950/20 hover:bg-stone-100"
          >
            {isEditMode ? '상세로' : '목록으로'}
          </Link>
          <button
            type="submit"
            form="log-form"
            disabled={activeMutation.isPending || (isEditMode && isLoading)}
            className="inline-flex items-center justify-center whitespace-nowrap rounded-full bg-stone-950 px-4 py-2 text-sm font-semibold !text-white transition hover:bg-amber-900 hover:!text-white disabled:cursor-not-allowed disabled:opacity-60"
          >
            {activeMutation.isPending ? '저장 중...' : isEditMode ? '변경 저장' : '기록 추가'}
          </button>
        </>
      }
    >
      {isEditMode && isLoading ? (
        <div className="rounded-[1.5rem] border border-amber-950/10 bg-stone-50/80 px-5 py-10 text-center text-sm text-stone-500">
          수정할 기록을 불러오는 중입니다.
        </div>
      ) : null}

      {isLoadError ? (
        <div className="rounded-[1.5rem] border border-rose-200 bg-rose-50 px-5 py-4 text-sm text-rose-700">
          {getErrorMessage(loadError)}
        </div>
      ) : null}

      {!isLoadError && (!isEditMode || log) ? (
        <form id="log-form" className="space-y-6" onSubmit={(event) => void handleSubmit(event)}>
          <LogTypeSection
            form={form}
            setForm={setForm}
            disabled={isEditMode || isCloneMode}
            error={fieldErrors['log_type']}
          />
          {!isEditMode && !isCloneMode ? (
            <PresetSection
              logType={form.logType}
              onSelect={(preset) => {
                const formState = presetToFormState(preset)
                setForm(formState)
                setExpanded(hasOptionalValues(formState))
              }}
            />
          ) : null}
          {!isEditMode && !isCloneMode && form.logType === 'brew' ? (
            <RecipePickerSection
              onSelect={(log) => {
                const formState = recipeToFormState(log)
                setForm(formState)
                setExpanded(hasOptionalValues(formState))
              }}
            />
          ) : null}
          {form.logType === 'cafe' ? (
            <CafeFieldsSection form={form} setForm={setForm} fieldErrors={fieldErrors} expanded={expanded} onToggle={toggleExpanded} />
          ) : (
            <BrewFieldsSection form={form} setForm={setForm} fieldErrors={fieldErrors} expanded={expanded} onToggle={toggleExpanded} />
          )}

          {activeMutation.isError && !fieldErrors[activeMutation.error instanceof ApiError && activeMutation.error.field ? activeMutation.error.field : ''] ? (
            <div className="rounded-[1.5rem] border border-rose-200 bg-rose-50 px-5 py-4 text-sm text-rose-700">
              {getErrorMessage(activeMutation.error)}
            </div>
          ) : null}
        </form>
      ) : null}
    </Layout>
  )
}
