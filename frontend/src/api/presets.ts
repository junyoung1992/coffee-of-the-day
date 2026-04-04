import { request } from './client'
import type { PresetFull, CreatePresetInput, UpdatePresetInput } from '../types/preset'

export interface ListPresetsResult {
  items: PresetFull[]
}

export function getPresets(): Promise<ListPresetsResult> {
  return request('/presets')
}

export function getPreset(id: string): Promise<PresetFull> {
  return request(`/presets/${id}`)
}

export function createPreset(body: CreatePresetInput): Promise<PresetFull> {
  return request('/presets', { method: 'POST', body: JSON.stringify(body) })
}

export function updatePreset(id: string, body: UpdatePresetInput): Promise<PresetFull> {
  return request(`/presets/${id}`, { method: 'PUT', body: JSON.stringify(body) })
}

export function deletePreset(id: string): Promise<void> {
  return request(`/presets/${id}`, { method: 'DELETE' })
}

export function usePresetApi(id: string): Promise<void> {
  return request(`/presets/${id}/use`, { method: 'POST' })
}
