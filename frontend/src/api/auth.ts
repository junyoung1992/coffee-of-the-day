import { request } from './client'
import type { User, LoginRequest, RegisterRequest } from '../types/auth'

export function register(body: RegisterRequest): Promise<User> {
  return request('/auth/register', { method: 'POST', body: JSON.stringify(body) })
}

export function login(body: LoginRequest): Promise<User> {
  return request('/auth/login', { method: 'POST', body: JSON.stringify(body) })
}

export function logout(): Promise<void> {
  return request('/auth/logout', { method: 'POST' })
}

export function refresh(): Promise<void> {
  return request('/auth/refresh', { method: 'POST' })
}

export function getMe(): Promise<User> {
  return request('/auth/me')
}
