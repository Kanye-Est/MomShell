import apiClient from '@/lib/apiClient'

export interface IdentityTag {
  id: string
  tag_type: string
  content: string
  created_at: string
}

export interface IdentityTagList {
  music: IdentityTag[]
  sound: IdentityTag[]
  literature: IdentityTag[]
  memory: IdentityTag[]
}

export interface Memoir {
  id: string
  title: string
  content: string
  cover_image_url: string | null
  user_rating: number | null
  created_at: string
}

export interface GeneratedMemoir extends Memoir {
  image_generation_enabled?: boolean
}

export interface MemoirList {
  memoirs: Memoir[]
  total: number
}

export function getIdentityTags(): Promise<IdentityTagList> {
  return apiClient.get('/api/v1/echo/identity-tags').then((r) => r.data)
}

export function createIdentityTag(tag_type: string, content: string): Promise<IdentityTag> {
  return apiClient.post('/api/v1/echo/identity-tags', { tag_type, content }).then((r) => r.data)
}

export function deleteIdentityTag(tagId: string): Promise<void> {
  return apiClient.delete(`/api/v1/echo/identity-tags/${tagId}`)
}

export function getMemoirs(limit = 10, offset = 0): Promise<MemoirList> {
  return apiClient.get('/api/v1/echo/memoirs', { params: { limit, offset } }).then((r) => r.data)
}

export function generateMemoir(theme?: string): Promise<GeneratedMemoir> {
  return apiClient.post('/api/v1/echo/memoirs/generate', theme ? { theme } : {}).then((r) => r.data)
}

export function rateMemoir(memoirId: string, rating: number): Promise<Memoir> {
  return apiClient.post(`/api/v1/echo/memoirs/${memoirId}/rate`, { rating }).then((r) => r.data)
}
