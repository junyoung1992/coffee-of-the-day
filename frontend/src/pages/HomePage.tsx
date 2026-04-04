import { useCallback, useEffect, useMemo, useRef } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { FilterBar } from '../components/FilterBar'
import { Layout } from '../components/Layout'
import { LogCard, LogCardSkeleton } from '../components/LogCard'
import { useLogList } from '../hooks/useLogs'
import type { LogType } from '../types/log'
import type { ApiErrorLike } from '../types/common'

function getErrorMessage(error: unknown) {
  if (typeof error === 'object' && error !== null && 'message' in error) {
    return String((error as ApiErrorLike).message)
  }
  return '기록을 불러오는 중 오류가 발생했습니다.'
}

// URL 쿼리 파라미터에서 log_type 값을 검증하여 파싱한다
function parseLogType(value: string | null): LogType | undefined {
  if (value === 'cafe' || value === 'brew') return value
  return undefined
}

import { getDefaultDateFrom, getDefaultDateTo } from '../utils/date'

export default function HomePage() {
  const [searchParams, setSearchParams] = useSearchParams()

  // URL 파라미터에서 현재 필터 상태를 파싱한다.
  // 날짜 파라미터가 없으면 당월 1일~오늘을 기본값으로 사용한다.
  // 기본값은 URL에 쓰지 않아 깨끗한 URL을 유지한다.
  const logType = parseLogType(searchParams.get('log_type'))
  const dateFrom = searchParams.get('date_from') ?? getDefaultDateFrom()
  const dateTo = searchParams.get('date_to') ?? getDefaultDateTo()

  const {
    data,
    error,
    fetchNextPage,
    hasNextPage,
    isError,
    isFetchingNextPage,
    isLoading,
    isRefetching,
  } = useLogList({
    limit: 12,
    log_type: logType,
    date_from: dateFrom || undefined,
    date_to: dateTo || undefined,
  })
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

  // 필터 변경 핸들러: URL 파라미터를 업데이트한다
  const handleLogTypeChange = useCallback(
    (value: LogType | undefined) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        if (value) {
          next.set('log_type', value)
        } else {
          next.delete('log_type')
        }
        return next
      })
    },
    [setSearchParams],
  )

  const handleDateFromChange = useCallback(
    (value: string) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        if (value) {
          next.set('date_from', value)
        } else {
          next.delete('date_from')
        }
        return next
      })
    },
    [setSearchParams],
  )

  const handleDateToChange = useCallback(
    (value: string) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        if (value) {
          next.set('date_to', value)
        } else {
          next.delete('date_to')
        }
        return next
      })
    },
    [setSearchParams],
  )

  const handleClearFilters = useCallback(() => {
    setSearchParams({})
  }, [setSearchParams])

  // 필터가 하나라도 활성화된 상태인지 확인한다.
  // 빈 상태 메시지를 "전체 데이터 없음"과 "필터 결과 없음"으로 분기하는 데 사용한다.
  const hasActiveFilter = !!logType || !!dateFrom || !!dateTo

  return (
    <Layout
      title="커피 기록"
      description="카페에서 마신 한 잔과 직접 내린 레시피를 한 화면에서 관리합니다. 목록은 최신순으로 쌓이고, 아래로 스크롤하면 다음 페이지가 이어집니다."
      actions={
        <>
          <Link
            to="/presets"
            className="inline-flex items-center justify-center whitespace-nowrap rounded-full border border-stone-950/10 px-4 py-2 text-sm font-semibold text-stone-700 transition hover:border-stone-950/20 hover:bg-stone-100"
          >
            프리셋
          </Link>
          <Link
            to="/logs/new"
            className="inline-flex items-center justify-center whitespace-nowrap rounded-full border border-amber-900/15 bg-amber-100/70 px-4 py-2 text-sm font-semibold text-amber-950 transition hover:border-amber-900/30 hover:bg-amber-100"
          >
            기록 추가
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

        <FilterBar
          logType={logType}
          dateFrom={dateFrom}
          dateTo={dateTo}
          onLogTypeChange={handleLogTypeChange}
          onDateFromChange={handleDateFromChange}
          onDateToChange={handleDateToChange}
        />

        {isError ? (
          <div className="rounded-[1.5rem] border border-rose-200 bg-rose-50 px-5 py-4 text-sm text-rose-700">
            {getErrorMessage(error)}
          </div>
        ) : null}

        {/* 초기 로딩 중: 빈 상태 대신 스켈레톤 그리드를 표시하여 레이아웃 이동을 방지한다 */}
        {isLoading ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <LogCardSkeleton key={i} />
            ))}
          </div>
        ) : null}

        {!isLoading && !isError && logs.length === 0 ? (
          hasActiveFilter ? (
            // 필터 결과가 0건인 경우: 데이터가 없는 게 아니라 조건에 맞는 항목이 없는 것이다
            <div className="rounded-[1.75rem] border border-dashed border-amber-900/25 bg-white/70 px-6 py-10 text-center">
              <p className="text-lg font-semibold text-stone-900">조건에 맞는 기록이 없습니다.</p>
              <p className="mt-2 text-sm leading-6 text-stone-600">
                선택한 필터에 해당하는 로그를 찾지 못했습니다. 필터를 바꾸거나 초기화해 보세요.
              </p>
              <button
                type="button"
                onClick={handleClearFilters}
                className="mt-6 inline-flex items-center justify-center rounded-full bg-stone-950 px-4 py-2 text-sm font-semibold text-white transition hover:bg-amber-900"
              >
                필터 초기화
              </button>
            </div>
          ) : (
            // 기록이 한 건도 없는 첫 방문 상태
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
          )
        ) : null}

        {!isLoading && logs.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {logs.map((log) => (
              <LogCard key={log.id} log={log} />
            ))}
            {/* 다음 페이지 로딩 중: 그리드 하단에 스켈레톤 카드를 이어 붙인다 */}
            {isFetchingNextPage
              ? Array.from({ length: 3 }).map((_, i) => (
                  <LogCardSkeleton key={`skeleton-next-${i}`} />
                ))
              : null}
          </div>
        ) : null}

        {/* sentinel: hasNextPage일 때만 DOM에 존재하면 되므로 조건부 렌더링이 적절하다 */}
        {hasNextPage ? <div ref={sentinelRef} className="h-1" aria-hidden="true" /> : null}

        {!hasNextPage && logs.length > 0 && !isLoading ? (
          <div className="rounded-[1.5rem] border border-amber-950/10 bg-stone-50/70 px-5 py-4 text-center text-sm text-stone-500">
            더 이상 기록이 없습니다.
          </div>
        ) : null}
      </div>
    </Layout>
  )
}
