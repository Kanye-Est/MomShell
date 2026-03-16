import apiClient from '@/lib/apiClient'

export interface ShellGiftItem {
  id: string
  task_id: string
  from_user_id: string
  to_user_id: string
  ai_title: string
  ai_content: string
  cover_url: string
  photo_url: string | null
  is_opened: boolean
  opened_at: string | null
  created_at: string
}

export function getShellGifts(): Promise<ShellGiftItem[]> {
  return apiClient.get('/api/v1/shell-gifts').then((r) => r.data)
}

export function openShellGift(id: string): Promise<ShellGiftItem> {
  return apiClient.post(`/api/v1/shell-gifts/${id}/open`).then((r) => r.data)
}

