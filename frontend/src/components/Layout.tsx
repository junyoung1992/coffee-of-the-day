import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useCurrentUser, useLogout } from '../hooks/useAuth'

interface LayoutProps {
  title: string
  description?: string
  actions?: ReactNode
  children: ReactNode
}

export function Layout({ title, description, actions, children }: LayoutProps) {
  const { data: user } = useCurrentUser()
  const logout = useLogout()

  return (
    <div className="min-h-screen">
      <header className="border-b border-amber-950/10 bg-white/50 backdrop-blur-sm">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-4 sm:px-6 lg:px-8">
          <Link to="/" className="group space-y-1">
            <div className="text-xs font-semibold uppercase tracking-[0.28em] text-amber-950/45">
              Coffee of the Day
            </div>
            <div className="text-lg font-semibold text-stone-900 transition group-hover:text-amber-900">
              One cup, one memory
            </div>
          </Link>

          <div className="flex items-center gap-3">
            {user ? (
              <span className="hidden text-sm text-stone-500 sm:block">{user.display_name}</span>
            ) : null}
            <Link
              to="/logs/new"
              className="inline-flex items-center justify-center whitespace-nowrap rounded-full bg-stone-950 px-4 py-2 text-sm font-semibold !text-white transition hover:bg-amber-900 hover:!text-white"
            >
              New Log
            </Link>
            <button
              onClick={() => logout.mutate()}
              disabled={logout.isPending}
              className="whitespace-nowrap rounded-full border border-stone-200 px-4 py-2 text-sm font-medium text-stone-600 transition hover:border-stone-400 hover:text-stone-900 disabled:opacity-50"
            >
              로그아웃
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
        <section className="rounded-[2rem] border border-white/60 bg-white/72 p-6 shadow-[0_24px_80px_rgba(72,44,17,0.12)] backdrop-blur-sm sm:p-8">
          <div className="flex flex-col gap-5 border-b border-amber-950/10 pb-6 sm:flex-row sm:items-end sm:justify-between">
            <div className="max-w-2xl space-y-3">
              <p className="text-xs font-semibold uppercase tracking-[0.28em] text-amber-900/55">
                Personal coffee journal
              </p>
              <h1 className="text-2xl font-semibold tracking-tight text-stone-950 sm:text-3xl">
                {title}
              </h1>
              {description ? (
                <p className="text-sm leading-6 text-stone-600 sm:text-base">{description}</p>
              ) : null}
            </div>
            {actions ? <div className="flex w-full shrink-0 flex-wrap gap-3 sm:w-auto">{actions}</div> : null}
          </div>

          <div className="pt-6">{children}</div>
        </section>
      </main>
    </div>
  )
}
