import apiClient from "@/lib/apiClient";

export interface UserTaskItem {
  id: string;
  title: string;
  description: string;
  category: string;
  difficulty: number;
  priority: string;
  status: string;
  score: number | null;
  comment: string | null;
  proof_photo_url: string | null;
  completed_at: string | null;
  scored_at: string | null;
  date: string;
}

export interface TaskStats {
  xp: number;
  level: number;
}

export interface BabyAgeResponse {
  age_stage: string;
  source: string;
}

export function getDailyTasks(): Promise<UserTaskItem[]> {
  return apiClient.get("/api/v1/tasks/daily").then((r) => r.data);
}

export function completeTask(
  id: string,
  data?: { proof_photo_url?: string },
): Promise<UserTaskItem> {
  return apiClient
    .post(`/api/v1/tasks/${id}/complete`, data ?? {})
    .then((r) => r.data);
}

export function getPartnerTasks(): Promise<UserTaskItem[]> {
  return apiClient.get("/api/v1/tasks/partner").then((r) => r.data);
}

export function scoreTask(
  id: string,
  data: { score: number; comment?: string },
): Promise<UserTaskItem> {
  return apiClient.post(`/api/v1/tasks/${id}/score`, data).then((r) => r.data);
}

export function rejectTask(
  id: string,
  data?: { comment?: string },
): Promise<UserTaskItem> {
  return apiClient
    .post(`/api/v1/tasks/${id}/reject`, data ?? {})
    .then((r) => r.data);
}

export function getTaskStats(): Promise<TaskStats> {
  return apiClient.get("/api/v1/tasks/stats").then((r) => r.data);
}

export function getBabyAge(): Promise<BabyAgeResponse> {
  return apiClient.get("/api/v1/tasks/baby-age").then((r) => r.data);
}

export function setBabyAge(ageStage: string): Promise<void> {
  return apiClient
    .put("/api/v1/tasks/baby-age", { age_stage: ageStage })
    .then(() => undefined);
}
