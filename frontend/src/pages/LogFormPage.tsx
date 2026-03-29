import {
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type FormEvent,
  type ReactNode,
} from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Layout } from '../components/Layout'
import { RatingInput } from '../components/RatingInput'
import { useCreateLog, useLog, useUpdateLog } from '../hooks/useLogs'
import type { ApiErrorLike } from '../types/common'
import {
  brewMethodOptions,
  buildLogPayload,
  createEmptyFormState,
  logToFormState,
  roastLevelOptions,
  type FormLogType,
  type LogFormState,
} from './logFormState'

function getErrorMessage(error: unknown) {
  if (typeof error === 'object' && error !== null && 'message' in error) {
    return String((error as ApiErrorLike).message)
  }
  return '저장 중 오류가 발생했습니다.'
}

function Field({
  label,
  required,
  children,
}: {
  label: string
  required?: boolean
  children: ReactNode
}) {
  return (
    <label className="space-y-2">
      <span className="text-sm font-medium text-stone-800">
        {label}
        {required ? <span className="ml-1 text-amber-900">*</span> : null}
      </span>
      {children}
    </label>
  )
}

function inputClassName() {
  return 'w-full rounded-2xl border border-amber-950/10 bg-white px-4 py-3 text-sm text-stone-900 outline-none transition placeholder:text-stone-400 focus:border-amber-900/35 focus:bg-amber-50/40'
}

function textareaClassName() {
  return `${inputClassName()} min-h-[116px] resize-y`
}

function Section({
  title,
  description,
  children,
}: {
  title: string
  description: string
  children: ReactNode
}) {
  return (
    <section className="space-y-5 rounded-[1.75rem] border border-amber-950/10 bg-stone-50/65 p-5 sm:p-6">
      <div className="space-y-2">
        <h2 className="text-lg font-semibold text-stone-950">{title}</h2>
        <p className="text-sm leading-6 text-stone-600">{description}</p>
      </div>
      {children}
    </section>
  )
}

function updateCafeField<K extends keyof LogFormState['cafe']>(
  state: LogFormState,
  key: K,
  value: LogFormState['cafe'][K],
) {
  return {
    ...state,
    cafe: {
      ...state.cafe,
      [key]: value,
    },
  }
}

function updateBrewField<K extends keyof LogFormState['brew']>(
  state: LogFormState,
  key: K,
  value: LogFormState['brew'][K],
) {
  return {
    ...state,
    brew: {
      ...state.brew,
      [key]: value,
    },
  }
}

export default function LogFormPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const isEditMode = Boolean(id)
  const [form, setForm] = useState(() => createEmptyFormState())
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
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setForm(logToFormState(log))
    hydratedLogIDRef.current = log.id
  }, [isEditMode, log])

  const activeMutation = isEditMode ? updateMutation : createMutation
  const ratio = useMemo(() => {
    const coffee = Number(form.brew.coffeeAmountG)
    const water = Number(form.brew.waterAmountMl)
    if (!Number.isFinite(coffee) || !Number.isFinite(water) || coffee <= 0 || water <= 0) {
      return null
    }
    return (water / coffee).toFixed(1)
  }, [form.brew.coffeeAmountG, form.brew.waterAmountMl])

  function handleBaseChange(
    event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>,
  ) {
    const { name, value } = event.target
    setForm((prev) => ({ ...prev, [name]: value }))
  }

  function handleLogTypeChange(logType: FormLogType) {
    if (isEditMode) {
      return
    }
    setForm((prev) => ({ ...prev, logType }))
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const payload = buildLogPayload(form)

    const saved = isEditMode && id
      ? await updateMutation.mutateAsync(payload)
      : await createMutation.mutateAsync(payload)

    navigate(`/logs/${saved.id}`)
  }

  function updateBrewStep(index: number, value: string) {
    setForm((prev) =>
      updateBrewField(prev, 'brewSteps', prev.brew.brewSteps.map((step, stepIndex) => (
        stepIndex === index ? value : step
      ))),
    )
  }

  function addBrewStep() {
    setForm((prev) => updateBrewField(prev, 'brewSteps', [...prev.brew.brewSteps, '']))
  }

  function moveBrewStep(index: number, direction: -1 | 1) {
    setForm((prev) => {
      const nextIndex = index + direction
      if (nextIndex < 0 || nextIndex >= prev.brew.brewSteps.length) {
        return prev
      }

      const nextSteps = [...prev.brew.brewSteps]
      ;[nextSteps[index], nextSteps[nextIndex]] = [nextSteps[nextIndex], nextSteps[index]]
      return updateBrewField(prev, 'brewSteps', nextSteps)
    })
  }

  function removeBrewStep(index: number) {
    setForm((prev) => {
      const nextSteps = prev.brew.brewSteps.filter((_, stepIndex) => stepIndex !== index)
      return updateBrewField(prev, 'brewSteps', nextSteps.length > 0 ? nextSteps : [''])
    })
  }

  return (
    <Layout
      title={isEditMode ? 'Refine the cup' : 'Capture a coffee moment'}
      description="공통 필드와 타입별 세부 필드를 한 폼에 묶었습니다. 카페 로그는 장소와 메뉴 중심으로, 브루 로그는 레시피 중심으로 입력합니다."
      actions={
        <>
          <Link
            to={isEditMode && id ? `/logs/${id}` : '/'}
            className="inline-flex items-center justify-center rounded-full border border-stone-950/10 px-4 py-2 text-sm font-semibold text-stone-700 transition hover:border-stone-950/20 hover:bg-stone-100"
          >
            {isEditMode ? 'Back to detail' : 'Back to list'}
          </Link>
          <button
            type="submit"
            form="log-form"
            disabled={activeMutation.isPending || (isEditMode && isLoading)}
            className="inline-flex items-center justify-center rounded-full bg-stone-950 px-4 py-2 text-sm font-semibold !text-white transition hover:bg-amber-900 hover:!text-white disabled:cursor-not-allowed disabled:opacity-60"
          >
            {activeMutation.isPending ? 'Saving...' : isEditMode ? 'Save changes' : 'Create log'}
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
          <Section
            title="Log type"
            description={
              isEditMode
                ? '기존 로그 타입은 백엔드 제약에 따라 변경할 수 없습니다.'
                : '바리스타가 만들어준 커피는 cafe, 내가 직접 추출한 커피는 brew로 기록합니다.'
            }
          >
            <div className="grid gap-3 sm:grid-cols-2">
              {(['cafe', 'brew'] as const).map((type) => {
                const selected = form.logType === type
                return (
                  <button
                    key={type}
                    type="button"
                    onClick={() => handleLogTypeChange(type)}
                    disabled={isEditMode}
                    className={[
                      'rounded-[1.5rem] border p-5 text-left transition',
                      selected
                        ? 'border-amber-900 bg-amber-900 !text-white shadow-[0_16px_40px_rgba(123,79,34,0.22)]'
                        : 'border-amber-950/10 bg-white text-stone-800 hover:border-amber-900/25 hover:bg-amber-50/60',
                      isEditMode ? 'cursor-not-allowed opacity-70' : '',
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

          <Section
            title="Common fields"
            description="모든 커피 로그가 공유하는 시간, 함께한 사람, 메모를 먼저 입력합니다."
          >
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="Recorded at" required>
                <input
                  className={inputClassName()}
                  type="datetime-local"
                  name="recordedAt"
                  value={form.recordedAt}
                  onChange={handleBaseChange}
                  required
                />
              </Field>
              <Field label="Companions">
                <input
                  className={inputClassName()}
                  type="text"
                  name="companionsText"
                  value={form.companionsText}
                  onChange={handleBaseChange}
                  placeholder="민수, 지연"
                />
              </Field>
              <div className="md:col-span-2">
                <Field label="Memo">
                  <textarea
                    className={textareaClassName()}
                    name="memo"
                    value={form.memo}
                    onChange={handleBaseChange}
                    placeholder="오늘의 한 잔이 남긴 기억을 자유롭게 적어보세요."
                  />
                </Field>
              </div>
            </div>
          </Section>

          {form.logType === 'cafe' ? (
            <Section
              title="Cafe section"
              description="카페에서 마신 커피의 장소 정보와 메뉴 정보를 입력합니다."
            >
              <div className="grid gap-4 md:grid-cols-2">
                <Field label="Cafe name" required>
                  <input
                    className={inputClassName()}
                    value={form.cafe.cafeName}
                    onChange={(event) =>
                      setForm((prev) => updateCafeField(prev, 'cafeName', event.target.value))
                    }
                    required
                  />
                </Field>
                <Field label="Location">
                  <input
                    className={inputClassName()}
                    value={form.cafe.location}
                    onChange={(event) =>
                      setForm((prev) => updateCafeField(prev, 'location', event.target.value))
                    }
                    placeholder="서울 성수"
                  />
                </Field>
                <Field label="Coffee name" required>
                  <input
                    className={inputClassName()}
                    value={form.cafe.coffeeName}
                    onChange={(event) =>
                      setForm((prev) => updateCafeField(prev, 'coffeeName', event.target.value))
                    }
                    required
                  />
                </Field>
                <Field label="Roast level">
                  <select
                    className={inputClassName()}
                    value={form.cafe.roastLevel}
                    onChange={(event) =>
                      setForm((prev) =>
                        updateCafeField(
                          prev,
                          'roastLevel',
                          event.target.value as LogFormState['cafe']['roastLevel'],
                        ),
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
                    onChange={(event) =>
                      setForm((prev) => updateCafeField(prev, 'beanOrigin', event.target.value))
                    }
                  />
                </Field>
                <Field label="Bean process">
                  <input
                    className={inputClassName()}
                    value={form.cafe.beanProcess}
                    onChange={(event) =>
                      setForm((prev) => updateCafeField(prev, 'beanProcess', event.target.value))
                    }
                  />
                </Field>
                <div className="md:col-span-2">
                  <Field label="Tasting tags">
                    <input
                      className={inputClassName()}
                      value={form.cafe.tastingTagsText}
                      onChange={(event) =>
                        setForm((prev) =>
                          updateCafeField(prev, 'tastingTagsText', event.target.value),
                        )
                      }
                      placeholder="초콜릿, 체리, 헤이즐넛"
                    />
                  </Field>
                </div>
                <div className="md:col-span-2">
                  <Field label="Tasting note">
                    <textarea
                      className={textareaClassName()}
                      value={form.cafe.tastingNote}
                      onChange={(event) =>
                        setForm((prev) =>
                          updateCafeField(prev, 'tastingNote', event.target.value),
                        )
                      }
                    />
                  </Field>
                </div>
                <div className="md:col-span-2">
                  <Field label="Impressions">
                    <textarea
                      className={textareaClassName()}
                      value={form.cafe.impressions}
                      onChange={(event) =>
                        setForm((prev) =>
                          updateCafeField(prev, 'impressions', event.target.value),
                        )
                      }
                    />
                  </Field>
                </div>
                <div className="md:col-span-2">
                  <Field label="Rating">
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
          ) : (
            <Section
              title="Brew section"
              description="브루 로그는 원두 정보와 추출 레시피를 함께 기록합니다."
            >
              <div className="grid gap-4 md:grid-cols-2">
                <Field label="Bean name" required>
                  <input
                    className={inputClassName()}
                    value={form.brew.beanName}
                    onChange={(event) =>
                      setForm((prev) => updateBrewField(prev, 'beanName', event.target.value))
                    }
                    required
                  />
                </Field>
                <Field label="Brew method" required>
                  <select
                    className={inputClassName()}
                    value={form.brew.brewMethod}
                    onChange={(event) =>
                      setForm((prev) =>
                        updateBrewField(
                          prev,
                          'brewMethod',
                          event.target.value as LogFormState['brew']['brewMethod'],
                        ),
                      )
                    }
                    required
                  >
                    {brewMethodOptions.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="Bean origin">
                  <input
                    className={inputClassName()}
                    value={form.brew.beanOrigin}
                    onChange={(event) =>
                      setForm((prev) => updateBrewField(prev, 'beanOrigin', event.target.value))
                    }
                  />
                </Field>
                <Field label="Bean process">
                  <input
                    className={inputClassName()}
                    value={form.brew.beanProcess}
                    onChange={(event) =>
                      setForm((prev) => updateBrewField(prev, 'beanProcess', event.target.value))
                    }
                  />
                </Field>
                <Field label="Roast level">
                  <select
                    className={inputClassName()}
                    value={form.brew.roastLevel}
                    onChange={(event) =>
                      setForm((prev) =>
                        updateBrewField(
                          prev,
                          'roastLevel',
                          event.target.value as LogFormState['brew']['roastLevel'],
                        ),
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
                    onChange={(event) =>
                      setForm((prev) => updateBrewField(prev, 'roastDate', event.target.value))
                    }
                  />
                </Field>
                <Field label="Brew device">
                  <input
                    className={inputClassName()}
                    value={form.brew.brewDevice}
                    onChange={(event) =>
                      setForm((prev) => updateBrewField(prev, 'brewDevice', event.target.value))
                    }
                    placeholder="Origami, AeroPress Go"
                  />
                </Field>
                <Field label="Grind size">
                  <input
                    className={inputClassName()}
                    value={form.brew.grindSize}
                    onChange={(event) =>
                      setForm((prev) => updateBrewField(prev, 'grindSize', event.target.value))
                    }
                    placeholder="중간, 20 clicks"
                  />
                </Field>
                <Field label="Coffee amount (g)">
                  <input
                    className={inputClassName()}
                    type="number"
                    min="0"
                    step="0.1"
                    value={form.brew.coffeeAmountG}
                    onChange={(event) =>
                      setForm((prev) =>
                        updateBrewField(prev, 'coffeeAmountG', event.target.value),
                      )
                    }
                  />
                </Field>
                <Field label="Water amount (ml)">
                  <input
                    className={inputClassName()}
                    type="number"
                    min="0"
                    step="0.1"
                    value={form.brew.waterAmountMl}
                    onChange={(event) =>
                      setForm((prev) =>
                        updateBrewField(prev, 'waterAmountMl', event.target.value),
                      )
                    }
                  />
                </Field>
                <Field label="Water temperature (C)">
                  <input
                    className={inputClassName()}
                    type="number"
                    min="0"
                    step="0.1"
                    value={form.brew.waterTempC}
                    onChange={(event) =>
                      setForm((prev) => updateBrewField(prev, 'waterTempC', event.target.value))
                    }
                  />
                </Field>
                <Field label="Brew time (sec)">
                  <input
                    className={inputClassName()}
                    type="number"
                    min="0"
                    step="1"
                    value={form.brew.brewTimeSec}
                    onChange={(event) =>
                      setForm((prev) => updateBrewField(prev, 'brewTimeSec', event.target.value))
                    }
                  />
                </Field>
                <div className="md:col-span-2 rounded-[1.5rem] border border-dashed border-amber-900/20 bg-white/70 px-4 py-4">
                  <p className="text-sm font-semibold text-stone-900">Recipe ratio</p>
                  <p className="mt-2 text-sm text-stone-600">
                    {ratio ? `1:${ratio}` : '원두량과 물량을 입력하면 비율이 계산됩니다.'}
                  </p>
                </div>
                <div className="md:col-span-2">
                  <Field label="Tasting tags">
                    <input
                      className={inputClassName()}
                      value={form.brew.tastingTagsText}
                      onChange={(event) =>
                        setForm((prev) =>
                          updateBrewField(prev, 'tastingTagsText', event.target.value),
                        )
                      }
                      placeholder="꽃, 베리, 카카오"
                    />
                  </Field>
                </div>
                <div className="md:col-span-2">
                  <Field label="Tasting note">
                    <textarea
                      className={textareaClassName()}
                      value={form.brew.tastingNote}
                      onChange={(event) =>
                        setForm((prev) =>
                          updateBrewField(prev, 'tastingNote', event.target.value),
                        )
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
                                onClick={() => moveBrewStep(index, -1)}
                                disabled={index === 0}
                                className="rounded-full border border-amber-950/10 px-3 py-1.5 text-xs font-semibold text-stone-600 transition hover:border-amber-900/25 hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-50"
                              >
                                Up
                              </button>
                              <button
                                type="button"
                                onClick={() => moveBrewStep(index, 1)}
                                disabled={index === form.brew.brewSteps.length - 1}
                                className="rounded-full border border-amber-950/10 px-3 py-1.5 text-xs font-semibold text-stone-600 transition hover:border-amber-900/25 hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-50"
                              >
                                Down
                              </button>
                              <button
                                type="button"
                                onClick={() => removeBrewStep(index)}
                                className="rounded-full border border-rose-300 bg-rose-50 px-3 py-1.5 text-xs font-semibold text-rose-700 transition hover:border-rose-400 hover:bg-rose-100"
                              >
                                Delete
                              </button>
                            </div>
                          </div>
                          <textarea
                            className={`${textareaClassName()} mt-3 min-h-[90px]`}
                            value={step}
                            onChange={(event) => updateBrewStep(index, event.target.value)}
                            placeholder="예: 30초 뜸, 3회 나눠 붓기"
                          />
                        </div>
                      ))}
                      <button
                        type="button"
                        onClick={addBrewStep}
                        className="rounded-full border border-amber-900/15 bg-amber-100/70 px-4 py-2 text-sm font-semibold text-amber-950 transition hover:border-amber-900/30 hover:bg-amber-100"
                      >
                        Add brew step
                      </button>
                    </div>
                  </Field>
                </div>
                <div className="md:col-span-2">
                  <Field label="Impressions">
                    <textarea
                      className={textareaClassName()}
                      value={form.brew.impressions}
                      onChange={(event) =>
                        setForm((prev) =>
                          updateBrewField(prev, 'impressions', event.target.value),
                        )
                      }
                    />
                  </Field>
                </div>
                <div className="md:col-span-2">
                  <Field label="Rating">
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
          )}

          {activeMutation.isError ? (
            <div className="rounded-[1.5rem] border border-rose-200 bg-rose-50 px-5 py-4 text-sm text-rose-700">
              {getErrorMessage(activeMutation.error)}
            </div>
          ) : null}
        </form>
      ) : null}
    </Layout>
  )
}
