import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { createLog, deleteLog, getLog, getLogs, updateLog } from '../api/logs'
import type { ListLogsParams } from '../api/logs'
import type { UpdateLogInput, CreateLogInput } from '../types/log'

// 쿼리 키 상수: 무효화(invalidate) 범위를 명확히 하기 위해 한 곳에서 관리한다
const LOG_KEYS = {
  all: ['logs'] as const,
  list: (params: ListLogsParams) => ['logs', 'list', params] as const,
  detail: (id: string) => ['logs', 'detail', id] as const,
}

/**
 * 커피 기록 무한 스크롤 목록 조회.
 * 각 페이지는 CursorPage<CoffeeLogFull> 형태이며,
 * data.pages 배열로 모든 페이지에 접근할 수 있다.
 */
export function useLogList(params: Omit<ListLogsParams, 'cursor'> = {}) {
  return useInfiniteQuery({
    queryKey: LOG_KEYS.list(params),
    queryFn: ({ pageParam }) =>
      getLogs({ ...params, cursor: pageParam }),
    // next_cursor가 null이면 undefined를 반환해야 TanStack Query가 마지막 페이지로 인식한다
    getNextPageParam: (lastPage) => lastPage.next_cursor ?? undefined,
    initialPageParam: undefined as string | undefined,
  })
}

/** 커피 기록 단건 조회 */
export function useLog(id: string) {
  return useQuery({
    queryKey: LOG_KEYS.detail(id),
    queryFn: () => getLog(id),
    enabled: !!id,
  })
}

/** 커피 기록 생성. 성공 시 목록 쿼리를 무효화하여 자동으로 재조회한다 */
export function useCreateLog() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createLog,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: LOG_KEYS.all })
    },
  })
}

/** 커피 기록 수정. 성공 시 해당 기록과 목록을 모두 무효화한다 */
export function useUpdateLog(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: UpdateLogInput) => updateLog(id, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: LOG_KEYS.detail(id) })
      queryClient.invalidateQueries({ queryKey: LOG_KEYS.all })
    },
  })
}

/** 커피 기록 삭제. 성공 시 목록 쿼리를 무효화한다 */
export function useDeleteLog() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteLog,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: LOG_KEYS.all })
    },
  })
}

// 내부 타입 재수출: 폼 컴포넌트 등에서 직접 임포트할 수 있도록
export type { CreateLogInput, UpdateLogInput }
