import {
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import {
  createPreset,
  deletePreset,
  getPreset,
  getPresets,
  updatePreset,
  usePresetApi,
} from '../api/presets'
import type { UpdatePresetInput, CreatePresetInput } from '../types/preset'

// 쿼리 키 상수: 무효화(invalidate) 범위를 명확히 하기 위해 한 곳에서 관리한다
export const PRESET_KEYS = {
  all: ['presets'] as const,
  list: () => ['presets', 'list'] as const,
  detail: (id: string) => ['presets', 'detail', id] as const,
}

/** 프리셋 전체 목록 조회 (최근 사용순 정렬) */
export function usePresetList() {
  return useQuery({
    queryKey: PRESET_KEYS.list(),
    queryFn: () => getPresets(),
    select: (data) => data.items,
  })
}

/** 프리셋 단건 조회 */
export function usePreset(id: string) {
  return useQuery({
    queryKey: PRESET_KEYS.detail(id),
    queryFn: () => getPreset(id),
    enabled: !!id,
  })
}

/** 프리셋 생성. 성공 시 목록 쿼리를 무효화한다 */
export function useCreatePreset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createPreset,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PRESET_KEYS.all })
    },
  })
}

/** 프리셋 수정. 성공 시 해당 프리셋과 목록을 무효화한다 */
export function useUpdatePreset(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: UpdatePresetInput) => updatePreset(id, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PRESET_KEYS.all })
    },
  })
}

/** 프리셋 삭제. 성공 시 목록 쿼리를 무효화한다 */
export function useDeletePreset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deletePreset,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PRESET_KEYS.all })
    },
  })
}

/** 프리셋 사용 기록. last_used_at을 optimistic update로 목록 캐시에 반영한다 */
export function useUsePreset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: usePresetApi,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PRESET_KEYS.all })
    },
  })
}

// 내부 타입 재수출
export type { CreatePresetInput, UpdatePresetInput }
