import { createBrowserRouter } from 'react-router-dom'
import ProtectedRoute from './components/ProtectedRoute'
import HomePage from './pages/HomePage'
import LogDetailPage from './pages/LogDetailPage'
import LogFormPage from './pages/LogFormPage'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'

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
