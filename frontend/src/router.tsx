import { createBrowserRouter } from 'react-router-dom'
import HomePage from './pages/HomePage'
import LogDetailPage from './pages/LogDetailPage'
import LogFormPage from './pages/LogFormPage'

export const router = createBrowserRouter([
  { path: '/', element: <HomePage /> },
  { path: '/logs/new', element: <LogFormPage /> },
  { path: '/logs/:id', element: <LogDetailPage /> },
  { path: '/logs/:id/edit', element: <LogFormPage /> },
])
