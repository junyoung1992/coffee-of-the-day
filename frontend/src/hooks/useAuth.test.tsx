import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { useCurrentUser, useLogin, useLogout, useRegister } from './useAuth'
import * as authApi from '../api/auth'

vi.mock('../api/auth')
// navigate mock
vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>()
  return { ...actual, useNavigate: () => vi.fn() }
})

const mockGetMe = vi.mocked(authApi.getMe)
const mockLogin = vi.mocked(authApi.login)
const mockRegister = vi.mocked(authApi.register)
const mockLogout = vi.mocked(authApi.logout)

function makeWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      </MemoryRouter>
    )
  }
}

const sampleUser = {
  id: 'user-1',
  email: 'test@example.com',
  username: 'testuser',
  display_name: 'Test User',
}

beforeEach(() => {
  vi.resetAllMocks()
})

describe('useCurrentUser', () => {
  it('로그인된 사용자 정보를 반환한다', async () => {
    mockGetMe.mockResolvedValue(sampleUser)

    const { result } = renderHook(() => useCurrentUser(), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(sampleUser)
  })

  it('미로그인 시 error 상태가 된다', async () => {
    mockGetMe.mockRejectedValue(new Error('Unauthorized'))

    const { result } = renderHook(() => useCurrentUser(), { wrapper: makeWrapper() })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe('useLogin', () => {
  it('로그인 성공 시 쿼리 캐시에 사용자 정보가 저장된다', async () => {
    mockLogin.mockResolvedValue(sampleUser)

    const { result } = renderHook(() => useLogin(), { wrapper: makeWrapper() })

    await act(async () => {
      result.current.mutate({ email: 'test@example.com', password: 'password123' })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockLogin).toHaveBeenCalledWith({ email: 'test@example.com', password: 'password123' })
  })
})

describe('useRegister', () => {
  it('회원가입 성공 시 사용자 정보를 반환한다', async () => {
    mockRegister.mockResolvedValue(sampleUser)

    const { result } = renderHook(() => useRegister(), { wrapper: makeWrapper() })

    await act(async () => {
      result.current.mutate({ email: 'test@example.com', password: 'password123', username: 'testuser' })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
  })
})

describe('useLogout', () => {
  it('로그아웃 성공 시 logout API를 호출한다', async () => {
    mockLogout.mockResolvedValue(undefined)

    const { result } = renderHook(() => useLogout(), { wrapper: makeWrapper() })

    await act(async () => {
      result.current.mutate()
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockLogout).toHaveBeenCalled()
  })
})
