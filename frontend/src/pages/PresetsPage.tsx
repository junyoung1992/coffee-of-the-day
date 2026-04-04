import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Layout } from '../components/Layout'
import {
  usePresetList,
  useDeletePreset,
  useUpdatePreset,
} from '../hooks/usePresets'
import type { PresetFull, UpdatePresetInput } from '../types/preset'
import type { ApiErrorLike } from '../types/common'

const brewMethodLabels: Record<string, string> = {
  aeropress: 'AeroPress',
  cold_brew: 'Cold Brew',
  espresso: 'Espresso',
  immersion: 'Immersion',
  moka_pot: 'Moka Pot',
  other: 'Other',
  pour_over: 'Pour Over',
  siphon: 'Siphon',
}

function formatRelativeDate(value: string | null | undefined) {
  if (!value) return '사용 기록 없음'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value
  return new Intl.DateTimeFormat('ko-KR', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(parsed)
}

function getErrorMessage(error: unknown) {
  if (typeof error === 'object' && error !== null && 'message' in error) {
    return String((error as ApiErrorLike).message)
  }
  return '오류가 발생했습니다.'
}

// --- 수정 모달 ---

function EditModal({
  preset,
  onClose,
}: {
  preset: PresetFull
  onClose: () => void
}) {
  const [name, setName] = useState(preset.name)
  const updateMutation = useUpdatePreset(preset.id)

  function buildUpdateBody(): UpdatePresetInput {
    const body: UpdatePresetInput = { name: name.trim() }
    if (preset.log_type === 'cafe') {
      body.cafe = {
        cafe_name: preset.cafe.cafe_name,
        coffee_name: preset.cafe.coffee_name,
        tasting_tags: preset.cafe.tasting_tags ?? [],
      }
    } else {
      body.brew = {
        bean_name: preset.brew.bean_name,
        brew_method: preset.brew.brew_method,
        recipe_detail: preset.brew.recipe_detail ?? undefined,
        brew_steps: preset.brew.brew_steps ?? [],
      }
    }
    return body
  }

  async function handleSave() {
    if (!name.trim()) return
    await updateMutation.mutateAsync(buildUpdateBody())
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 backdrop-blur-sm">
      <div className="mx-4 w-full max-w-md space-y-4 rounded-[1.75rem] border border-amber-950/10 bg-white p-6 shadow-xl">
        <h3 className="text-lg font-semibold text-stone-950">프리셋 수정</h3>
        <label className="block space-y-2">
          <span className="text-sm font-medium text-stone-800">이름</span>
          <input
            className="w-full rounded-2xl border border-amber-950/10 bg-white px-4 py-3 text-sm text-stone-900 outline-none transition placeholder:text-stone-400 focus:border-amber-900/35 focus:bg-amber-50/40"
            value={name}
            onChange={(e) => setName(e.target.value)}
            autoFocus
            onKeyDown={(e) => {
              if (e.key === 'Enter') void handleSave()
              if (e.key === 'Escape') onClose()
            }}
          />
        </label>
        {updateMutation.isError ? (
          <p className="text-xs text-rose-600">{getErrorMessage(updateMutation.error)}</p>
        ) : null}
        <div className="flex justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded-full border border-stone-950/10 px-4 py-2 text-sm font-semibold text-stone-700 transition hover:bg-stone-100"
          >
            취소
          </button>
          <button
            type="button"
            onClick={() => void handleSave()}
            disabled={updateMutation.isPending || !name.trim()}
            className="rounded-full bg-stone-950 px-4 py-2 text-sm font-semibold text-white transition hover:bg-amber-900 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {updateMutation.isPending ? '저장 중...' : '저장'}
          </button>
        </div>
      </div>
    </div>
  )
}

// --- 프리셋 카드 ---

function PresetCard({
  preset,
  onEdit,
}: {
  preset: PresetFull
  onEdit: () => void
}) {
  const deleteMutation = useDeletePreset()

  async function handleDelete() {
    if (!window.confirm(`"${preset.name}" 프리셋을 삭제하시겠습니까?`)) return
    await deleteMutation.mutateAsync(preset.id)
  }

  return (
    <div className="rounded-[1.75rem] border border-amber-950/10 bg-white p-5 transition hover:border-amber-900/15 hover:shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="space-y-2">
          <div className="inline-flex rounded-full border border-amber-900/15 bg-amber-100/80 px-2.5 py-0.5 text-xs font-semibold uppercase tracking-[0.2em] text-amber-950/70">
            {preset.log_type}
          </div>
          <h3 className="text-base font-semibold text-stone-950">{preset.name}</h3>
          <p className="text-sm text-stone-600">
            {preset.log_type === 'cafe'
              ? `${preset.cafe.cafe_name} · ${preset.cafe.coffee_name}`
              : `${preset.brew.bean_name} · ${brewMethodLabels[preset.brew.brew_method] ?? preset.brew.brew_method}`}
          </p>
          {preset.log_type === 'cafe' && preset.cafe.tasting_tags && preset.cafe.tasting_tags.length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {preset.cafe.tasting_tags.map((tag) => (
                <span
                  key={tag}
                  className="rounded-full bg-stone-100 px-2.5 py-0.5 text-xs text-stone-600"
                >
                  {tag}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      </div>

      <div className="mt-4 flex items-center justify-between border-t border-amber-950/5 pt-3">
        <p className="text-xs text-stone-400">
          {formatRelativeDate(preset.last_used_at)}
        </p>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={onEdit}
            className="rounded-full border border-amber-950/10 px-3 py-1.5 text-xs font-semibold text-stone-600 transition hover:border-amber-900/25 hover:bg-amber-50"
          >
            수정
          </button>
          <button
            type="button"
            onClick={() => void handleDelete()}
            disabled={deleteMutation.isPending}
            className="rounded-full border border-rose-300 bg-rose-50 px-3 py-1.5 text-xs font-semibold text-rose-700 transition hover:border-rose-400 hover:bg-rose-100 disabled:opacity-60"
          >
            {deleteMutation.isPending ? '삭제 중...' : '삭제'}
          </button>
        </div>
      </div>

      {deleteMutation.isError ? (
        <p className="mt-2 text-xs text-rose-600">{getErrorMessage(deleteMutation.error)}</p>
      ) : null}
    </div>
  )
}

// --- 메인 페이지 ---

export default function PresetsPage() {
  const { data: presets = [], isLoading, isError, error } = usePresetList()
  const [editingPreset, setEditingPreset] = useState<PresetFull | null>(null)

  return (
    <Layout
      title="프리셋 관리"
      description="자주 사용하는 카페+메뉴 또는 원두+추출방식 조합을 관리합니다."
      actions={
        <Link
          to="/"
          className="inline-flex items-center justify-center whitespace-nowrap rounded-full border border-stone-950/10 px-4 py-2 text-sm font-semibold text-stone-700 transition hover:border-stone-950/20 hover:bg-stone-100"
        >
          목록으로
        </Link>
      }
    >
      {isLoading ? (
        <div className="rounded-[1.5rem] border border-amber-950/10 bg-stone-50/80 px-5 py-10 text-center text-sm text-stone-500">
          프리셋을 불러오는 중입니다.
        </div>
      ) : null}

      {isError ? (
        <div className="rounded-[1.5rem] border border-rose-200 bg-rose-50 px-5 py-4 text-sm text-rose-700">
          {getErrorMessage(error)}
        </div>
      ) : null}

      {!isLoading && !isError && presets.length === 0 ? (
        <div className="rounded-[1.5rem] border border-amber-950/10 bg-stone-50/80 px-5 py-10 text-center">
          <p className="text-sm text-stone-500">아직 저장된 프리셋이 없습니다.</p>
          <p className="mt-2 text-xs text-stone-400">
            로그 상세 화면에서 "프리셋 저장" 버튼으로 프리셋을 추가할 수 있습니다.
          </p>
        </div>
      ) : null}

      {presets.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2">
          {presets.map((preset) => (
            <PresetCard
              key={preset.id}
              preset={preset}
              onEdit={() => setEditingPreset(preset)}
            />
          ))}
        </div>
      ) : null}

      {editingPreset ? (
        <EditModal
          preset={editingPreset}
          onClose={() => setEditingPreset(null)}
        />
      ) : null}
    </Layout>
  )
}
