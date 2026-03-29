import type { LogType } from '../types/log'

// 탭 정의: value가 undefined이면 전체 조회를 의미한다
const LOG_TYPE_TABS: { label: string; value: LogType | undefined }[] = [
  { label: '전체', value: undefined },
  { label: '카페', value: 'cafe' },
  { label: '브루', value: 'brew' },
]

interface FilterBarProps {
  logType: LogType | undefined
  dateFrom: string
  dateTo: string
  onLogTypeChange: (value: LogType | undefined) => void
  onDateFromChange: (value: string) => void
  onDateToChange: (value: string) => void
}

export function FilterBar({
  logType,
  dateFrom,
  dateTo,
  onLogTypeChange,
  onDateFromChange,
  onDateToChange,
}: FilterBarProps) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      {/* 기록 타입 탭 필터 */}
      <div className="flex gap-1 rounded-full border border-amber-950/10 bg-stone-100/80 p-1">
        {LOG_TYPE_TABS.map((tab) => {
          const isActive = logType === tab.value
          return (
            <button
              key={tab.label}
              type="button"
              onClick={() => onLogTypeChange(tab.value)}
              className={[
                'rounded-full px-4 py-1.5 text-sm font-semibold transition',
                isActive
                  ? 'bg-white text-stone-900 shadow-sm'
                  : 'text-stone-500 hover:text-stone-700',
              ].join(' ')}
            >
              {tab.label}
            </button>
          )
        })}
      </div>

      {/* 날짜 범위 필터 */}
      <div className="flex items-center gap-2 text-sm text-stone-500">
        <input
          type="date"
          value={dateFrom}
          onChange={(e) => onDateFromChange(e.target.value)}
          className="rounded-full border border-amber-950/10 bg-stone-50 px-3 py-1.5 text-sm text-stone-700 transition focus:border-amber-900/30 focus:outline-none focus:ring-2 focus:ring-amber-900/10"
          aria-label="시작 날짜"
        />
        <span className="shrink-0">–</span>
        <input
          type="date"
          value={dateTo}
          onChange={(e) => onDateToChange(e.target.value)}
          className="rounded-full border border-amber-950/10 bg-stone-50 px-3 py-1.5 text-sm text-stone-700 transition focus:border-amber-900/30 focus:outline-none focus:ring-2 focus:ring-amber-900/10"
          aria-label="종료 날짜"
        />
      </div>
    </div>
  )
}
