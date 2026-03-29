interface RatingDisplayProps {
  value?: number | null
  size?: 'sm' | 'md'
  showValue?: boolean
}

const STAR_PATH =
  'M12 2.75 14.843 8.511 21.2 9.435 16.6 13.919 17.686 20.25 12 17.261 6.314 20.25 7.4 13.919 2.8 9.435 9.157 8.511 12 2.75Z'

function StarRow({ size, tone }: { size: 'sm' | 'md'; tone: string }) {
  const dimension = size === 'sm' ? 'h-4 w-4' : 'h-5 w-5'

  return (
    <div className={`flex ${tone}`}>
      {Array.from({ length: 5 }).map((_, index) => (
        <svg
          key={index}
          viewBox="0 0 24 24"
          className={`${dimension} shrink-0`}
          fill="currentColor"
          aria-hidden="true"
        >
          <path d={STAR_PATH} />
        </svg>
      ))}
    </div>
  )
}

export function RatingDisplay({
  value,
  size = 'md',
  showValue = true,
}: RatingDisplayProps) {
  const normalized =
    typeof value === 'number' && value >= 0.5 && value <= 5 ? value : null
  const width = normalized ? `${(normalized / 5) * 100}%` : '0%'

  return (
    <div className="inline-flex items-center gap-2">
      <span className="relative inline-flex" aria-hidden="true">
        <StarRow size={size} tone="text-stone-200" />
        <span className="pointer-events-none absolute inset-0 overflow-hidden" style={{ width }}>
          <StarRow size={size} tone="text-amber-500" />
        </span>
      </span>
      <span className="sr-only">
        {normalized ? `${normalized.toFixed(1)} out of 5 stars` : 'No rating'}
      </span>
      {showValue ? (
        <span className="text-sm font-medium text-stone-600">
          {normalized ? normalized.toFixed(1) : 'No rating'}
        </span>
      ) : null}
    </div>
  )
}
