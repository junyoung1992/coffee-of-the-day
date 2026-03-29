import { useEffect, useMemo, useRef } from 'react'
import { Link } from 'react-router-dom'
import { Layout } from '../components/Layout'
import { LogCard } from '../components/LogCard'
import { useLogList } from '../hooks/useLogs'
import type { ApiErrorLike } from '../types/common'

function getErrorMessage(error: unknown) {
  if (typeof error === 'object' && error !== null && 'message' in error) {
    return String((error as ApiErrorLike).message)
  }
  return '기록을 불러오는 중 오류가 발생했습니다.'
}

export default function HomePage() {
  const {
    data,
    error,
    fetchNextPage,
    hasNextPage,
    isError,
    isFetchingNextPage,
    isLoading,
    isRefetching,
  } = useLogList({ limit: 12 })
  const sentinelRef = useRef<HTMLDivElement | null>(null)

  const logs = useMemo(
    () => data?.pages.flatMap((page) => page.items) ?? [],
    [data?.pages],
  )

  useEffect(() => {
    const node = sentinelRef.current
    if (!node || !hasNextPage || typeof IntersectionObserver === 'undefined') {
      return
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting) && !isFetchingNextPage) {
          void fetchNextPage()
        }
      },
      { rootMargin: '220px 0px' },
    )

    observer.observe(node)
    return () => observer.disconnect()
  }, [fetchNextPage, hasNextPage, isFetchingNextPage, logs.length])

  return (
    <Layout
      title="커피 기록"
      description="카페에서 마신 한 잔과 직접 내린 레시피를 한 화면에서 관리합니다. 목록은 최신순으로 쌓이고, 아래로 스크롤하면 다음 페이지가 이어집니다."
      actions={
        <>
          <Link
            to="/logs/new"
            className="inline-flex items-center justify-center rounded-full border border-amber-900/15 bg-amber-100/70 px-4 py-2 text-sm font-semibold text-amber-950 transition hover:border-amber-900/30 hover:bg-amber-100"
          >
            오늘의 기록 추가
          </Link>
          <Link
            to="/logs/new"
            className="inline-flex items-center justify-center rounded-full border border-stone-950/10 px-4 py-2 text-sm font-semibold text-stone-700 transition hover:border-stone-950/20 hover:bg-stone-100"
          >
            빠른 추가
          </Link>
        </>
      }
    >
      <div className="space-y-6">
        <div className="flex items-center justify-between gap-3 rounded-[1.5rem] border border-amber-950/10 bg-stone-50/80 px-5 py-4">
          <div>
            <p className="text-sm font-semibold text-stone-900">
              {logs.length} {logs.length === 1 ? 'log' : 'logs'}
            </p>
            <p className="text-sm text-stone-500">
              {isRefetching ? '최신 상태를 다시 확인 중입니다.' : '홈 화면에서 최근 기록을 바로 훑어볼 수 있습니다.'}
            </p>
          </div>
          {isLoading ? (
            <span className="rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-900">
              Loading
            </span>
          ) : null}
        </div>

        {isError ? (
          <div className="rounded-[1.5rem] border border-rose-200 bg-rose-50 px-5 py-4 text-sm text-rose-700">
            {getErrorMessage(error)}
          </div>
        ) : null}

        {!isLoading && !isError && logs.length === 0 ? (
          <div className="rounded-[1.75rem] border border-dashed border-amber-900/25 bg-white/70 px-6 py-10 text-center">
            <p className="text-lg font-semibold text-stone-900">첫 커피 기록을 남길 차례입니다.</p>
            <p className="mt-2 text-sm leading-6 text-stone-600">
              아직 저장된 로그가 없습니다. 카페에서 마신 한 잔이든, 집에서 내린 브루든 지금 바로 시작할 수 있습니다.
            </p>
            <Link
              to="/logs/new"
              className="mt-6 inline-flex items-center justify-center rounded-full bg-stone-950 px-4 py-2 text-sm font-semibold !text-white transition hover:bg-amber-900 hover:!text-white"
            >
              Create first log
            </Link>
          </div>
        ) : null}

        {logs.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {logs.map((log) => (
              <LogCard key={log.id} log={log} />
            ))}
          </div>
        ) : null}

        {hasNextPage ? (
          <div className="space-y-3">
            <div ref={sentinelRef} className="h-4" aria-hidden="true" />
            <div className="flex justify-center">
              <button
                type="button"
                onClick={() => void fetchNextPage()}
                disabled={isFetchingNextPage}
                className="rounded-full border border-amber-950/10 bg-white px-4 py-2 text-sm font-semibold text-stone-700 transition hover:border-amber-900/25 hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isFetchingNextPage ? 'Loading more...' : 'Load more'}
              </button>
            </div>
          </div>
        ) : null}

        {!hasNextPage && logs.length > 0 ? (
          <div className="rounded-[1.5rem] border border-amber-950/10 bg-stone-50/70 px-5 py-4 text-center text-sm text-stone-500">
            현재 불러온 기록이 전부입니다.
          </div>
        ) : null}
      </div>
    </Layout>
  )
}
