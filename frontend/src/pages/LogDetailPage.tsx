import { Link, useNavigate, useParams } from 'react-router-dom'
import { Layout } from '../components/Layout'
import { RatingDisplay } from '../components/RatingDisplay'
import { useDeleteLog, useLog } from '../hooks/useLogs'
import type { ApiErrorLike } from '../types/common'

const roastLabels = {
  dark: 'Dark',
  light: 'Light',
  medium: 'Medium',
} as const

const brewMethodLabels = {
  aeropress: 'AeroPress',
  cold_brew: 'Cold Brew',
  espresso: 'Espresso',
  immersion: 'Immersion',
  moka_pot: 'Moka Pot',
  other: 'Other',
  pour_over: 'Pour Over',
  siphon: 'Siphon',
} as const

function formatDateTime(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat('ko-KR', {
    dateStyle: 'full',
    timeStyle: 'short',
  }).format(parsed)
}

function getErrorMessage(error: unknown) {
  if (typeof error === 'object' && error !== null && 'message' in error) {
    return String((error as ApiErrorLike).message)
  }
  return '오류가 발생했습니다.'
}

function DetailField({ label, value }: { label: string; value?: string | null }) {
  return (
    <div className="rounded-2xl border border-amber-950/10 bg-stone-50/80 p-4">
      <dt className="text-xs font-semibold uppercase tracking-[0.2em] text-stone-500">{label}</dt>
      <dd className="mt-2 text-sm leading-6 text-stone-800">{value || '-'}</dd>
    </div>
  )
}

export default function LogDetailPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const deleteMutation = useDeleteLog()
  const { data: log, error, isError, isLoading } = useLog(id ?? '')
  const brewSteps = log?.log_type === 'brew' ? log.brew.brew_steps ?? [] : []

  async function handleDelete() {
    if (!id) {
      return
    }
    if (!window.confirm('이 기록을 삭제하시겠습니까?')) {
      return
    }

    await deleteMutation.mutateAsync(id)
    navigate('/')
  }

  return (
    <Layout
      title="기록 상세"
      description="기록한 맛과 레시피를 빠르게 다시 읽을 수 있도록 공통 정보와 타입별 세부 정보를 분리해 보여줍니다."
      actions={
        <>
          <Link
            to="/"
            className="inline-flex items-center justify-center whitespace-nowrap rounded-full border border-stone-950/10 px-4 py-2 text-sm font-semibold text-stone-700 transition hover:border-stone-950/20 hover:bg-stone-100"
          >
            목록으로
          </Link>
          {id && log ? (
            <button
              type="button"
              onClick={() => navigate('/logs/new', { state: { cloneFrom: log } })}
              className="inline-flex items-center justify-center whitespace-nowrap rounded-full border border-stone-950/10 px-4 py-2 text-sm font-semibold text-stone-700 transition hover:border-stone-950/20 hover:bg-stone-100"
            >
              복제
            </button>
          ) : null}
          {id ? (
            <Link
              to={`/logs/${id}/edit`}
              className="inline-flex items-center justify-center whitespace-nowrap rounded-full bg-stone-950 px-4 py-2 text-sm font-semibold !text-white transition hover:bg-amber-900 hover:!text-white"
            >
              수정
            </Link>
          ) : null}
        </>
      }
    >
      {isLoading ? (
        <div className="rounded-[1.5rem] border border-amber-950/10 bg-stone-50/80 px-5 py-10 text-center text-sm text-stone-500">
          기록을 불러오는 중입니다.
        </div>
      ) : null}

      {isError ? (
        <div className="rounded-[1.5rem] border border-rose-200 bg-rose-50 px-5 py-4 text-sm text-rose-700">
          {getErrorMessage(error)}
        </div>
      ) : null}

      {log ? (
        <div className="space-y-6">
          <section className="rounded-[1.75rem] border border-amber-950/10 bg-[linear-gradient(180deg,rgba(255,250,243,0.96),rgba(248,240,229,0.9))] p-6">
            <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
              <div className="space-y-3">
                <div className="inline-flex rounded-full border border-amber-900/15 bg-amber-100/80 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-amber-950/70">
                  {log.log_type}
                </div>
                <div>
                  <h2 className="text-3xl font-semibold tracking-tight text-stone-950">
                    {log.log_type === 'cafe' ? log.cafe.coffee_name : log.brew.bean_name}
                  </h2>
                  <p className="mt-2 text-sm text-stone-600">
                    {log.log_type === 'cafe'
                      ? log.cafe.cafe_name
                      : brewMethodLabels[log.brew.brew_method]}
                  </p>
                </div>
              </div>
              <div className="space-y-3 rounded-[1.5rem] border border-amber-950/10 bg-white/70 px-4 py-3">
                <p className="text-xs font-semibold uppercase tracking-[0.2em] text-stone-500">Rating</p>
                <RatingDisplay
                  value={log.log_type === 'cafe' ? log.cafe.rating : log.brew.rating}
                />
              </div>
            </div>
          </section>

          <section className="grid gap-4 md:grid-cols-2">
            <DetailField label="Recorded at" value={formatDateTime(log.recorded_at)} />
            <DetailField
              label="Companions"
              value={log.companions.length > 0 ? log.companions.join(', ') : '혼자'}
            />
            <DetailField label="Memo" value={log.memo} />
            <DetailField label="Updated at" value={formatDateTime(log.updated_at)} />
          </section>

          {log.log_type === 'cafe' ? (
            <section className="space-y-4">
              <h3 className="text-lg font-semibold text-stone-950">카페 상세</h3>
              <div className="grid gap-4 md:grid-cols-2">
                <DetailField label="Cafe" value={log.cafe.cafe_name} />
                <DetailField label="Location" value={log.cafe.location} />
                <DetailField label="Coffee" value={log.cafe.coffee_name} />
                <DetailField label="Bean origin" value={log.cafe.bean_origin} />
                <DetailField label="Bean process" value={log.cafe.bean_process} />
                <DetailField
                  label="Roast level"
                  value={log.cafe.roast_level ? roastLabels[log.cafe.roast_level] : null}
                />
                <DetailField
                  label="Tasting tags"
                  value={log.cafe.tasting_tags?.join(', ') || null}
                />
                <DetailField label="Tasting note" value={log.cafe.tasting_note} />
                <div className="md:col-span-2">
                  <DetailField label="Impressions" value={log.cafe.impressions} />
                </div>
              </div>
            </section>
          ) : (
            <section className="space-y-4">
              <h3 className="text-lg font-semibold text-stone-950">브루 상세</h3>
              <div className="grid gap-4 md:grid-cols-2">
                <DetailField label="Bean" value={log.brew.bean_name} />
                <DetailField label="Brew method" value={brewMethodLabels[log.brew.brew_method]} />
                <DetailField label="Brew device" value={log.brew.brew_device} />
                <DetailField label="Roast date" value={log.brew.roast_date} />
                <DetailField label="Bean origin" value={log.brew.bean_origin} />
                <DetailField label="Bean process" value={log.brew.bean_process} />
                <DetailField
                  label="Roast level"
                  value={log.brew.roast_level ? roastLabels[log.brew.roast_level] : null}
                />
                <DetailField
                  label="Tasting tags"
                  value={log.brew.tasting_tags?.join(', ') || null}
                />
                <DetailField label="Water" value={log.brew.water_amount_ml ? `${log.brew.water_amount_ml} ml` : null} />
                <DetailField label="Coffee dose" value={log.brew.coffee_amount_g ? `${log.brew.coffee_amount_g} g` : null} />
                <DetailField label="Water temp" value={log.brew.water_temp_c ? `${log.brew.water_temp_c} C` : null} />
                <DetailField label="Brew time" value={log.brew.brew_time_sec ? `${log.brew.brew_time_sec} sec` : null} />
                <DetailField label="Grind size" value={log.brew.grind_size} />
                <DetailField label="Tasting note" value={log.brew.tasting_note} />
                <div className="md:col-span-2">
                  <DetailField label="Impressions" value={log.brew.impressions} />
                </div>
                <div className="md:col-span-2 rounded-2xl border border-amber-950/10 bg-stone-50/80 p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.2em] text-stone-500">
                    Brew steps
                  </p>
                  {brewSteps.length > 0 ? (
                    <ol className="mt-3 space-y-2 text-sm leading-6 text-stone-800">
                      {brewSteps.map((step, index) => (
                        <li key={`${index}-${step}`} className="rounded-xl bg-white px-4 py-3">
                          {index + 1}. {step}
                        </li>
                      ))}
                    </ol>
                  ) : (
                    <p className="mt-2 text-sm text-stone-500">기록된 추출 단계가 없습니다.</p>
                  )}
                </div>
              </div>
            </section>
          )}

          <div className="flex flex-wrap justify-end gap-3 border-t border-amber-950/10 pt-4">
            <button
              type="button"
              onClick={() => void handleDelete()}
              disabled={deleteMutation.isPending}
              className="rounded-full border border-rose-300 bg-rose-50 px-4 py-2 text-sm font-semibold text-rose-700 transition hover:border-rose-400 hover:bg-rose-100 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {deleteMutation.isPending ? '삭제 중...' : '삭제'}
            </button>
          </div>

          {deleteMutation.isError ? (
            <div className="rounded-[1.5rem] border border-rose-200 bg-rose-50 px-5 py-4 text-sm text-rose-700">
              {getErrorMessage(deleteMutation.error)}
            </div>
          ) : null}
        </div>
      ) : null}
    </Layout>
  )
}
