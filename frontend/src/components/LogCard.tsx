import { Link } from 'react-router-dom'
import type { CoffeeLogFull } from '../types/log'
import { RatingDisplay } from './RatingDisplay'

const brewMethodLabelMap = {
  aeropress: 'AeroPress',
  cold_brew: 'Cold Brew',
  espresso: 'Espresso',
  immersion: 'Immersion',
  moka_pot: 'Moka Pot',
  other: 'Other',
  pour_over: 'Pour Over',
  siphon: 'Siphon',
} as const

function formatRecordedAt(value: string) {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return new Intl.DateTimeFormat('ko-KR', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(parsed)
}

function summarizeCompanions(companions: string[]) {
  if (companions.length === 0) {
    return '혼자'
  }
  if (companions.length <= 2) {
    return companions.join(', ')
  }
  return `${companions.slice(0, 2).join(', ')} +${companions.length - 2}`
}

// 로딩 중 레이아웃 이동(CLS) 없이 카드 자리를 채우는 스켈레톤 컴포넌트
export function LogCardSkeleton() {
  return (
    <div className="flex h-full flex-col justify-between rounded-[1.75rem] border border-amber-950/10 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(247,239,229,0.9))] p-5 shadow-[0_16px_50px_rgba(72,44,17,0.08)]">
      <div className="animate-pulse space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div className="space-y-2">
            <div className="h-5 w-14 rounded-full bg-amber-100" />
            <div className="space-y-1.5">
              <div className="h-6 w-36 rounded-lg bg-stone-200" />
              <div className="h-4 w-24 rounded-lg bg-stone-100" />
            </div>
          </div>
          <div className="h-5 w-16 rounded-full bg-stone-100" />
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <div className="h-3 w-16 rounded bg-stone-100" />
            <div className="h-4 w-28 rounded bg-stone-200" />
          </div>
          <div className="space-y-1.5">
            <div className="h-3 w-8 rounded bg-stone-100" />
            <div className="h-4 w-20 rounded bg-stone-200" />
          </div>
        </div>

        <div className="flex gap-2">
          <div className="h-6 w-14 rounded-full bg-stone-100" />
          <div className="h-6 w-20 rounded-full bg-stone-100" />
        </div>

        <div className="space-y-1.5">
          <div className="h-4 w-full rounded bg-stone-100" />
          <div className="h-4 w-4/5 rounded bg-stone-100" />
        </div>
      </div>

      <div className="mt-5 flex items-center justify-between border-t border-amber-950/10 pt-4">
        <div className="h-4 w-10 animate-pulse rounded bg-stone-100" />
        <div className="h-4 w-14 animate-pulse rounded bg-stone-100" />
      </div>
    </div>
  )
}

export function LogCard({ log }: { log: CoffeeLogFull }) {
  const title = log.log_type === 'cafe' ? log.cafe.coffee_name : log.brew.bean_name
  const subtitle =
    log.log_type === 'cafe'
      ? `${log.cafe.cafe_name}${log.cafe.location ? ` · ${log.cafe.location}` : ''}`
      : `${brewMethodLabelMap[log.brew.brew_method]}${log.brew.brew_device ? ` · ${log.brew.brew_device}` : ''}`

  const tags =
    log.log_type === 'cafe' ? log.cafe.tasting_tags ?? [] : log.brew.tasting_tags ?? []
  const rating = log.log_type === 'cafe' ? log.cafe.rating : log.brew.rating
  const note = log.log_type === 'cafe' ? log.cafe.impressions : log.brew.impressions

  return (
    <Link
      to={`/logs/${log.id}`}
      className="group flex h-full flex-col justify-between rounded-[1.75rem] border border-amber-950/10 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(247,239,229,0.9))] p-5 shadow-[0_16px_50px_rgba(72,44,17,0.08)] transition hover:-translate-y-0.5 hover:border-amber-900/20 hover:shadow-[0_24px_70px_rgba(72,44,17,0.16)]"
    >
      <div className="space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div className="space-y-2">
            <div className="inline-flex rounded-full border border-amber-900/15 bg-amber-100/70 px-3 py-1 text-xs font-semibold uppercase tracking-[0.24em] text-amber-950/70">
              {log.log_type}
            </div>
            <div>
              <h2 className="text-xl font-semibold tracking-tight text-stone-950">{title}</h2>
              <p className="mt-1 text-sm text-stone-600">{subtitle}</p>
            </div>
          </div>
          <RatingDisplay value={rating} size="sm" />
        </div>

        <dl className="grid gap-3 text-sm text-stone-600 sm:grid-cols-2">
          <div>
            <dt className="text-xs font-semibold uppercase tracking-[0.2em] text-stone-500">
              Recorded
            </dt>
            <dd className="mt-1 text-stone-700">{formatRecordedAt(log.recorded_at)}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold uppercase tracking-[0.2em] text-stone-500">
              With
            </dt>
            <dd className="mt-1 text-stone-700">{summarizeCompanions(log.companions)}</dd>
          </div>
        </dl>

        {tags.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {tags.slice(0, 4).map((tag) => (
              <span
                key={tag}
                className="rounded-full bg-stone-950/6 px-3 py-1 text-xs font-medium text-stone-700"
              >
                {tag}
              </span>
            ))}
          </div>
        ) : null}

        {note ? (
          <p className="line-clamp-3 text-sm leading-6 text-stone-600">{note}</p>
        ) : (
          <p className="text-sm text-stone-400">기록된 인상 메모가 없습니다.</p>
        )}
      </div>

      <div className="mt-5 flex items-center justify-between border-t border-amber-950/10 pt-4 text-sm font-medium text-stone-700">
        <span>Detail</span>
        <span className="transition group-hover:translate-x-1">View log</span>
      </div>
    </Link>
  )
}
