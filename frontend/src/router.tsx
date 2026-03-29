import { createBrowserRouter, Navigate, Outlet } from 'react-router-dom'
import { useCurrentUser } from './hooks/useAuth'
import HomePage from './pages/HomePage'
import LogDetailPage from './pages/LogDetailPage'
import LogFormPage from './pages/LogFormPage'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'

/**
 * ProtectedRoute: 인증된 사용자만 하위 라우트에 접근할 수 있도록 보호한다.
 * - 로딩 중: 빈 화면 (깜빡임 방지)
 * - 미인증: /login으로 리다이렉트
 * - 인증됨: <Outlet />으로 자식 라우트 렌더링
 *
 * Spring Security의 SecurityFilterChain과 유사한 역할이다.
 */
function ProtectedRoute() {
  const { data: user, isLoading, isError } = useCurrentUser()

  if (isLoading) return null
  if (isError || !user) return <Navigate to="/login" replace />
  return <Outlet />
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/register', element: <RegisterPage /> },
  {
    element: <ProtectedRoute />,
    children: [
      { path: '/', element: <HomePage /> },
      { path: '/logs/new', element: <LogFormPage /> },
      { path: '/logs/:id', element: <LogDetailPage /> },
      { path: '/logs/:id/edit', element: <LogFormPage /> },
    ],
  },
])
