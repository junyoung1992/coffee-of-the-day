import { RatingDisplay } from './RatingDisplay'

interface RatingInputProps {
  value: number | null
  onChange: (value: number | null) => void
}

const RATING_STEPS = Array.from({ length: 10 }, (_, index) => (index + 1) * 0.5)

export function RatingInput({ value, onChange }: RatingInputProps) {
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-amber-950/10 bg-stone-50 px-4 py-3">
        <div className="space-y-1">
          <p className="text-sm font-medium text-stone-700">Rating</p>
          <RatingDisplay value={value} />
        </div>
        <button
          type="button"
          onClick={() => onChange(null)}
          className="rounded-full border border-amber-950/10 px-3 py-1.5 text-sm font-medium text-stone-600 transition hover:border-amber-900/30 hover:text-amber-900"
        >
          Clear
        </button>
      </div>

      <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
        {RATING_STEPS.map((step) => {
          const selected = value === step
          return (
            <button
              key={step}
              type="button"
              onClick={() => onChange(step)}
              className={[
                'rounded-2xl border px-3 py-2 text-left transition',
                selected
                  ? 'border-amber-900 bg-amber-900 text-amber-50 shadow-[0_10px_30px_rgba(120,69,20,0.2)]'
                  : 'border-amber-950/10 bg-white text-stone-700 hover:border-amber-900/30 hover:bg-amber-50',
              ].join(' ')}
              aria-pressed={selected}
            >
              <span className="block text-sm font-semibold">{step.toFixed(1)}</span>
              <span className="block text-xs opacity-80">out of 5.0</span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
