import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { getMe, login, logout, register } from '../api/auth'
import type { LoginRequest, RegisterRequest } from '../types/auth'

// 인증 상태를 식별하는 쿼리 키. 앱 전반에서 동일한 키를 사용해 캐시를 공유한다.
export const AUTH_KEY = ['auth', 'me'] as const

/**
 * 현재 로그인된 사용자 정보를 조회한다.
 * - 로그인 상태: data에 사용자 정보 존재
 * - 미로그인 / 토큰 만료: isError = true
 * - retry: false — 401에서 재시도하면 쿠키 갱신 여부 없이 그냥 실패하므로 비활성화
 */
export function useCurrentUser() {
  return useQuery({
    queryKey: AUTH_KEY,
    queryFn: getMe,
    retry: false,
    staleTime: 5 * 60 * 1_000, // 5분: 페이지 이동 시 불필요한 재요청 방지
  })
}

/** 로그인. 성공 시 사용자 정보를 쿼리 캐시에 즉시 반영하고 홈으로 이동한다 */
export function useLogin() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  return useMutation({
    mutationFn: (body: LoginRequest) => login(body),
    onSuccess: (user) => {
      // 네트워크 요청 없이 캐시를 직접 채운다 — /me 요청을 생략할 수 있다
      queryClient.setQueryData(AUTH_KEY, user)
      navigate('/')
    },
  })
}

/** 회원가입. 성공 시 자동 로그인 처리 (서버가 토큰을 함께 발급) */
export function useRegister() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  return useMutation({
    mutationFn: (body: RegisterRequest) => register(body),
    onSuccess: (user) => {
      queryClient.setQueryData(AUTH_KEY, user)
      navigate('/')
    },
  })
}

/** 로그아웃. 쿠키 만료 후 모든 쿼리 캐시를 초기화하고 로그인 페이지로 이동한다 */
export function useLogout() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  return useMutation({
    mutationFn: logout,
    onSuccess: () => {
      queryClient.clear()
      navigate('/login')
    },
  })
}
