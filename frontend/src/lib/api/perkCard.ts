import apiClient from '@/lib/apiClient'

export interface PerkCardItem {
  id: string
  from_user_id: string
  to_user_id: string
  title: string
  description: string
  icon_type: string
  status: 'active' | 'used' | 'expired' | string
  used_at: string | null
  expires_at: string | null
  created_at: string
}

export interface CreatePerkCardRequest {
  title: string
  description?: string
  icon_type?: string
  expires_at?: string
}

export function createPerkCard(data: CreatePerkCardRequest): Promise<PerkCardItem> {
  return apiClient.post('/api/v1/perk-cards', data).then((r) => r.data)
}

export function getPerkCards(): Promise<PerkCardItem[]> {
  return apiClient.get('/api/v1/perk-cards').then((r) => r.data)
}

export function usePerkCard(id: string): Promise<PerkCardItem> {
  return apiClient.post(`/api/v1/perk-cards/${id}/use`).then((r) => r.data)
}
