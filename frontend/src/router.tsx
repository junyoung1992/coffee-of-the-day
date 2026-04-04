import { createBrowserRouter, Outlet, ScrollRestoration } from 'react-router-dom'
import ProtectedRoute from './components/ProtectedRoute'
import { ScrollToTop } from './components/ScrollToTop'
import HomePage from './pages/HomePage'
import LogDetailPage from './pages/LogDetailPage'
import LogFormPage from './pages/LogFormPage'
import PresetsPage from './pages/PresetsPage'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'

function RootLayout() {
  return (
    <>
      <ScrollRestoration />
      <Outlet />
      <ScrollToTop />
    </>
  )
}

export const router = createBrowserRouter([
  {
    element: <RootLayout />,
    children: [
      { path: '/login', element: <LoginPage /> },
      { path: '/register', element: <RegisterPage /> },
      {
        element: <ProtectedRoute />,
        children: [
          { path: '/', element: <HomePage /> },
          { path: '/logs/new', element: <LogFormPage /> },
          { path: '/logs/:id', element: <LogDetailPage /> },
          { path: '/logs/:id/edit', element: <LogFormPage /> },
          { path: '/presets', element: <PresetsPage /> },
        ],
      },
    ],
  },
])
