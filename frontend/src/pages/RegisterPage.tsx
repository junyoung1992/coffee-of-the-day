import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useRegister } from '../hooks/useAuth'
import { ApiError } from '../api/client'

export default function RegisterPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [username, setUsername] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})

  const register = useRegister()

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setFieldErrors({})
    register.mutate(
      { email, password, username },
      {
        onError: (err) => {
          if (err instanceof ApiError) {
            if (err.field) {
              setFieldErrors({ [err.field]: err.message })
            } else {
              setFieldErrors({ _form: err.message })
            }
          }
        },
      },
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-amber-50/40 px-4">
      <div className="w-full max-w-sm space-y-6 rounded-2xl border border-white/60 bg-white/80 p-8 shadow-[0_24px_80px_rgba(72,44,17,0.10)] backdrop-blur-sm">
        <div className="space-y-1 text-center">
          <p className="text-xs font-semibold uppercase tracking-[0.28em] text-amber-950/45">
            Coffee of the Day
          </p>
          <h1 className="text-2xl font-semibold tracking-tight text-stone-900">회원가입</h1>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1">
            <label htmlFor="email" className="block text-sm font-medium text-stone-700">
              Email
            </label>
            <input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-lg border border-stone-200 px-3 py-2 text-sm text-stone-900 placeholder-stone-400 outline-none transition focus:border-amber-800 focus:ring-2 focus:ring-amber-800/20"
              placeholder="you@example.com"
            />
            {fieldErrors.email ? (
              <p role="alert" className="text-xs text-red-600">
                {fieldErrors.email}
              </p>
            ) : null}
          </div>

          <div className="space-y-1">
            <label htmlFor="password" className="block text-sm font-medium text-stone-700">
              Password
            </label>
            <input
              id="password"
              type="password"
              autoComplete="new-password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-lg border border-stone-200 px-3 py-2 text-sm text-stone-900 placeholder-stone-400 outline-none transition focus:border-amber-800 focus:ring-2 focus:ring-amber-800/20"
              placeholder="8자 이상"
            />
            {fieldErrors.password ? (
              <p role="alert" className="text-xs text-red-600">
                {fieldErrors.password}
              </p>
            ) : null}
          </div>

          <div className="space-y-1">
            <label htmlFor="username" className="block text-sm font-medium text-stone-700">
              Username
            </label>
            <input
              id="username"
              type="text"
              autoComplete="username"
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="w-full rounded-lg border border-stone-200 px-3 py-2 text-sm text-stone-900 placeholder-stone-400 outline-none transition focus:border-amber-800 focus:ring-2 focus:ring-amber-800/20"
              placeholder="coffeelover"
            />
            {fieldErrors.username ? (
              <p role="alert" className="text-xs text-red-600">
                {fieldErrors.username}
              </p>
            ) : null}
          </div>

          {fieldErrors._form ? (
            <p role="alert" className="text-sm text-red-600">
              {fieldErrors._form}
            </p>
          ) : null}

          <button
            type="submit"
            disabled={register.isPending}
            className="w-full rounded-full bg-stone-950 py-2 text-sm font-semibold text-white transition hover:bg-amber-900 disabled:opacity-50"
          >
            {register.isPending ? '가입 중…' : '회원가입'}
          </button>
        </form>

        <p className="text-center text-sm text-stone-500">
          이미 계정이 있으신가요?{' '}
          <Link to="/login" className="font-medium text-amber-900 hover:underline">
            로그인
          </Link>
        </p>
      </div>
    </div>
  )
}
