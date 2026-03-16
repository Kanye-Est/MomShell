<template>
  <OverlayPanel :visible="uiStore.activePanel === 'task'" position="center" @close="uiStore.closePanel()">
    <div class="task-panel">
      <!-- Dad view: daily tasks -->
      <template v-if="isDad">
        <button class="age-menu-btn" @click="showAgeMenu = !showAgeMenu" title="修改宝宝年龄" aria-label="修改宝宝年龄" :aria-expanded="showAgeMenu">⋯</button>
        <h2 class="panel-title">今日任务</h2>
        <p class="panel-subtitle">
          完成任务，一起成长
          <span v-if="currentAge" class="age-badge">{{ ageLabel(currentAge) }}</span>
        </p>

        <!-- Stats bar -->
        <div v-if="stats" class="stats-bar">
          <span class="level-badge">Lv.{{ stats.level }}</span>
          <span class="xp-text">{{ stats.xp }} XP</span>
        </div>

        <div v-if="loading" class="loading-state">{{ currentAge ? '正在生成任务...' : '加载中...' }}</div>

        <div v-else-if="tasks.length === 0" class="empty-state">今天没有任务</div>

        <div v-else class="task-list">
          <div
            v-for="t in sortedTasks"
            :key="t.id"
            :class="[
              'task-card',
              `status-${t.status}`,
              `pri-${t.priority || 'T2'}`,
            ]"
          >
            <div class="task-header">
              <div class="task-badges">
                <span
                  :class="['priority-badge', `pri-${t.priority || 'T2'}`]"
                  :title="priorityTip(t.priority)"
                >
                  {{ priorityLabel(t.priority) }}
                </span>
                <span :class="['category-badge', `cat-${t.category}`]">
                  {{ categoryLabel(t.category) }}
                </span>
              </div>
              <span class="difficulty">{{ difficultyStars(t.difficulty) }}</span>
            </div>

            <div class="task-body">
              <div class="task-kv">
                <div class="task-k">做什么</div>
                <h3 class="task-title">{{ t.title }}</h3>
              </div>
              <div class="task-kv">
                <div class="task-k">怎么做</div>
                <p class="task-desc">{{ t.description }}</p>
              </div>
              <img
                v-if="t.proof_photo_url && t.status !== 'pending'"
                class="proof-thumb"
                :src="t.proof_photo_url"
                alt=""
                loading="lazy"
              />
            </div>

            <div class="task-footer">
              <span v-if="t.status === 'pending'" class="status-text pending">待完成</span>
              <span v-else-if="t.status === 'completed'" class="status-text completed">已完成，等待验收</span>
              <span v-else class="status-text verified">已验收 {{ t.score }}/5</span>
              <button
                v-if="t.status === 'pending'"
                class="action-btn complete-btn"
                :disabled="completing === t.id"
                @click="openCompleteDialog(t)"
              >
                {{ completing === t.id ? '...' : '完成' }}
              </button>
            </div>
            <p v-if="t.comment && t.status === 'verified'" class="score-comment">{{ t.comment }}</p>
          </div>
        </div>
      </template>

      <!-- Mom view: partner tasks review -->
      <template v-else>
        <button class="age-menu-btn" @click="showAgeMenu = !showAgeMenu" title="修改宝宝年龄" aria-label="修改宝宝年龄" :aria-expanded="showAgeMenu">⋯</button>
        <h2 class="panel-title">伴侣任务</h2>
        <p class="panel-subtitle">
          查看他的完成情况
          <span v-if="currentAge" class="age-badge">{{ ageLabel(currentAge) }}</span>
        </p>

        <!-- Stats bar -->
        <div v-if="stats" class="stats-bar">
          <span class="level-badge">Lv.{{ stats.level }}</span>
          <span class="xp-text">{{ stats.xp }} XP</span>
        </div>

        <div v-if="loading" class="loading-state">{{ currentAge ? '正在生成任务...' : '加载中...' }}</div>

        <div v-else-if="tasks.length === 0" class="empty-state">今天还没有任务</div>

        <div v-else class="task-list">
          <div
            v-for="t in sortedTasks"
            :key="t.id"
            :class="[
              'task-card',
              `status-${t.status}`,
              `pri-${t.priority || 'T2'}`,
            ]"
          >
            <div class="task-header">
              <div class="task-badges">
                <span
                  :class="['priority-badge', `pri-${t.priority || 'T2'}`]"
                  :title="priorityTip(t.priority)"
                >
                  {{ priorityLabel(t.priority) }}
                </span>
                <span :class="['category-badge', `cat-${t.category}`]">
                  {{ categoryLabel(t.category) }}
                </span>
              </div>
              <span class="difficulty">{{ difficultyStars(t.difficulty) }}</span>
            </div>

            <div class="task-body">
              <div class="task-kv">
                <div class="task-k">做什么</div>
                <h3 class="task-title">{{ t.title }}</h3>
              </div>
              <div class="task-kv">
                <div class="task-k">怎么做</div>
                <p class="task-desc">{{ t.description }}</p>
              </div>
              <img
                v-if="t.proof_photo_url && t.status !== 'pending'"
                class="proof-thumb"
                :src="t.proof_photo_url"
                alt=""
                loading="lazy"
              />
            </div>

            <div class="task-footer">
              <span v-if="t.status === 'pending'" class="status-text pending">未完成</span>
              <span v-else-if="t.status === 'completed'" class="status-text completed">待验收</span>
              <span v-else class="status-text verified">已验收 {{ t.score }}/5</span>
            </div>

            <!-- Score UI for completed tasks -->
            <div v-if="t.status === 'completed'" class="score-section">
              <div class="score-stars">
                <button
                  v-for="s in 5"
                  :key="s"
                  :class="['star-btn', { active: (scoreMap[t.id]?.score ?? 0) >= s }]"
                  @click="setScore(t.id, s)"
                >
                  {{ (scoreMap[t.id]?.score ?? 0) >= s ? '★' : '☆' }}
                </button>
              </div>
              <input
                v-model="scoreMap[t.id]!.comment"
                class="score-input"
                placeholder="留一句评价..."
                maxlength="500"
              />
              <div class="score-actions">
                <button
                  class="action-btn reject-btn"
                  :disabled="scoring === t.id"
                  @click="onReject(t.id)"
                >
                  {{ scoring === t.id ? '...' : '未完成' }}
                </button>
                <button
                  class="action-btn score-btn"
                  :disabled="scoring === t.id || !(scoreMap[t.id]?.score)"
                  @click="onScore(t.id)"
                >
                  {{ scoring === t.id ? '提交中...' : '验收' }}
                </button>
              </div>
            </div>

            <p v-if="t.comment && t.status === 'verified'" class="score-comment">{{ t.comment }}</p>
          </div>
        </div>
      </template>

      <!-- Complete task: proof photo dialog (Dad) -->
      <Transition name="complete-fade">
        <div
          v-if="showCompleteDialog"
          class="complete-overlay"
          @click.self="closeCompleteDialog"
        >
          <div class="complete-dialog">
            <h3 class="complete-title">完成任务</h3>
            <p v-if="completeTarget" class="complete-subtitle">{{ completeTarget.title }}</p>

            <div class="complete-actions">
              <label class="proof-picker">
                <input
                  type="file"
                  accept="image/*"
                  capture="environment"
                  class="proof-input"
                  :disabled="proofUploading"
                  @change="onProofFileChange"
                />
                <span class="proof-picker-btn">拍照或上传证明</span>
              </label>
            </div>

            <img v-if="proofPreviewUrl" class="proof-preview" :src="proofPreviewUrl" alt="" />

            <p v-if="completeDialogError" class="complete-error">{{ completeDialogError }}</p>

            <div class="complete-footer">
              <button class="action-btn cancel-btn" :disabled="proofUploading" @click="closeCompleteDialog">
                取消
              </button>
              <button
                class="action-btn skip-btn"
                :disabled="proofUploading || !completeTarget"
                @click="submitCompleteWithoutPhoto"
              >
                跳过照片完成
              </button>
              <button
                class="action-btn submit-btn"
                :disabled="proofUploading || !completeTarget || !proofFile"
                @click="submitCompleteWithPhoto"
              >
                {{ proofUploading ? '提交中...' : '上传并完成' }}
              </button>
            </div>
          </div>
        </div>
      </Transition>

      <p v-if="error" class="error-msg">{{ error }}</p>

      <!-- Age picker popup -->
      <Transition name="age-fade">
        <div v-if="showAgeMenu" class="age-picker-overlay" @click.self="showAgeMenu = false">
          <div class="age-picker">
            <h3 class="age-picker-title">选择宝宝年龄</h3>
            <div class="age-options">
              <button
                v-for="opt in AGE_OPTIONS"
                :key="opt.value"
                :class="['age-option', { active: currentAge === opt.value }]"
                :disabled="settingAge"
                @click="onSelectAge(opt.value)"
              >
                {{ opt.label }}
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </OverlayPanel>
</template>

<script setup lang="ts">
import { computed, onUnmounted, reactive, ref, watch } from 'vue'
import OverlayPanel from './OverlayPanel.vue'
import { uploadPhoto } from '@/lib/api/photo'
import {
  completeTask,
  getBabyAge,
  getDailyTasks,
  getPartnerTasks,
  getTaskStats,
  rejectTask,
  scoreTask,
  setBabyAge,
  type TaskStats,
  type UserTaskItem,
} from '@/lib/api/task'
import { getErrorMessage } from '@/lib/apiClient'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'

const uiStore = useUiStore()
const authStore = useAuthStore()

const isDad = computed(() => authStore.user?.role === 'dad')

const tasks = ref<UserTaskItem[]>([])
const stats = ref<TaskStats | null>(null)
const loading = ref(false)
const error = ref('')
const completing = ref('')
const scoring = ref('')
const scoreMap = reactive<Record<string, { score: number; comment: string }>>({})
let pollTimer: ReturnType<typeof setInterval> | null = null

const priorityRank: Record<string, number> = { T0: 3, T1: 2, T2: 1 }
const statusRankDad: Record<string, number> = { pending: 3, completed: 2, verified: 1 }
const statusRankMom: Record<string, number> = { completed: 3, pending: 2, verified: 1 }

const sortedTasks = computed(() => {
  const statusRank = isDad.value ? statusRankDad : statusRankMom
  return [...tasks.value].sort((a, b) => {
    const byStatus = (statusRank[b.status] ?? 0) - (statusRank[a.status] ?? 0)
    if (byStatus !== 0) return byStatus
    return (priorityRank[b.priority || 'T2'] ?? 0) - (priorityRank[a.priority || 'T2'] ?? 0)
  })
})

function priorityLabel(priority?: string): string {
  const p = (priority || 'T2').toUpperCase()
  if (p === 'T0') return '紧急'
  if (p === 'T1') return '里程碑'
  return '日常'
}

function priorityTip(priority?: string): string {
  const p = (priority || 'T2').toUpperCase()
  if (p === 'T0') return 'T0: 突发/情绪干预'
  if (p === 'T1') return 'T1: 关键里程碑'
  return 'T2: 日常守护'
}

// Dad: complete task + optional proof photo
const showCompleteDialog = ref(false)
const completeTarget = ref<UserTaskItem | null>(null)
const proofFile = ref<File | null>(null)
const proofPreviewUrl = ref('')
const proofUploading = ref(false)
const completeDialogError = ref('')

function resetProof() {
  if (proofPreviewUrl.value) {
    URL.revokeObjectURL(proofPreviewUrl.value)
  }
  proofFile.value = null
  proofPreviewUrl.value = ''
}

function openCompleteDialog(task: UserTaskItem) {
  if (!isDad.value) return
  completeTarget.value = task
  completeDialogError.value = ''
  resetProof()
  showCompleteDialog.value = true
}

function closeCompleteDialog() {
  showCompleteDialog.value = false
  completeTarget.value = null
  completeDialogError.value = ''
  resetProof()
}

function onProofFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0] ?? null
  resetProof()
  if (file) {
    proofFile.value = file
    proofPreviewUrl.value = URL.createObjectURL(file)
  }
  // Allow selecting the same file again
  input.value = ''
}

async function submitCompleteWithoutPhoto() {
  if (!completeTarget.value) return
  const id = completeTarget.value.id
  completing.value = id
  proofUploading.value = true
  completeDialogError.value = ''
  try {
    const updated = await completeTask(id)
    tasks.value = tasks.value.map((t) => (t.id === id ? updated : t))
    closeCompleteDialog()
  } catch (e) {
    completeDialogError.value = getErrorMessage(e)
  } finally {
    proofUploading.value = false
    completing.value = ''
  }
}

async function submitCompleteWithPhoto() {
  if (!completeTarget.value || !proofFile.value) return
  const id = completeTarget.value.id
  completing.value = id
  proofUploading.value = true
  completeDialogError.value = ''
  try {
    const uploaded = await uploadPhoto(proofFile.value, `任务证明：${completeTarget.value.title}`)
    const updated = await completeTask(id, { proof_photo_url: uploaded.image_url })
    tasks.value = tasks.value.map((t) => (t.id === id ? updated : t))
    closeCompleteDialog()
  } catch (e) {
    completeDialogError.value = getErrorMessage(e)
  } finally {
    proofUploading.value = false
    completing.value = ''
  }
}

const AGE_OPTIONS = [
  { value: 'pregnancy', label: '孕期' },
  { value: '0-3m', label: '0-3个月' },
  { value: '3-6m', label: '3-6个月' },
  { value: '6-12m', label: '6-12个月' },
  { value: '1-2y', label: '1-2岁' },
  { value: '2-3y', label: '2-3岁' },
  { value: '3-4y', label: '3-4岁' },
  { value: '4-5y', label: '4-5岁' },
] as const

const ageLabelMap: Record<string, string> = Object.fromEntries(
  AGE_OPTIONS.map((o) => [o.value, o.label]),
)

const showAgeMenu = ref(false)
const currentAge = ref('')
const settingAge = ref(false)

function ageLabel(value: string): string {
  return ageLabelMap[value] || value
}

async function fetchBabyAge() {
  try {
    const resp = await getBabyAge()
    currentAge.value = resp.age_stage || ''
  } catch {
    // ignore
  }
}

async function onSelectAge(value: string) {
  settingAge.value = true
  error.value = ''
  try {
    await setBabyAge(value)
    currentAge.value = value
    showAgeMenu.value = false
    // Refresh tasks immediately
    loading.value = true
    await fetchTasks()
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    settingAge.value = false
    loading.value = false
  }
}

async function fetchTasks() {
  const [taskList, taskStats] = await Promise.all([
    isDad.value ? getDailyTasks() : getPartnerTasks(),
    getTaskStats(),
  ])
  tasks.value = taskList
  stats.value = taskStats
  initScoreMap(taskList)
}

function initScoreMap(taskList: UserTaskItem[]) {
  for (const t of taskList) {
    if (t.status === 'completed' && !scoreMap[t.id]) {
      scoreMap[t.id] = { score: 0, comment: '' }
    }
  }
}

function mergeTaskFields(existing: UserTaskItem, incoming: UserTaskItem) {
  if (existing.status !== incoming.status) existing.status = incoming.status
  if (existing.priority !== incoming.priority) existing.priority = incoming.priority
  if (existing.score !== incoming.score) existing.score = incoming.score
  if (existing.comment !== incoming.comment) existing.comment = incoming.comment
  if (existing.proof_photo_url !== incoming.proof_photo_url) existing.proof_photo_url = incoming.proof_photo_url
  if (existing.completed_at !== incoming.completed_at) existing.completed_at = incoming.completed_at
  if (existing.scored_at !== incoming.scored_at) existing.scored_at = incoming.scored_at
}

function mergeTaskList(taskList: UserTaskItem[]) {
  for (const incoming of taskList) {
    const existing = tasks.value.find((t) => t.id === incoming.id)
    if (existing) {
      mergeTaskFields(existing, incoming)
    }
  }
}

async function pollTasks() {
  try {
    const [taskList, taskStats] = await Promise.all([
      isDad.value ? getDailyTasks() : getPartnerTasks(),
      getTaskStats(),
    ])
    mergeTaskList(taskList)
    if (stats.value?.xp !== taskStats.xp || stats.value?.level !== taskStats.level) {
      stats.value = taskStats
    }
    initScoreMap(taskList)
  } catch {
    // Silently ignore poll errors
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(pollTasks, 10000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

onUnmounted(stopPolling)

watch(
  () => uiStore.activePanel,
  async (panel) => {
    if (panel === 'task') {
      error.value = ''
      loading.value = true
      try {
        await Promise.all([fetchTasks(), fetchBabyAge()])
      } catch (e) {
        error.value = getErrorMessage(e)
      } finally {
        loading.value = false
      }
      startPolling()
    } else {
      stopPolling()
    }
  },
)

function setScore(taskId: string, score: number) {
  if (!scoreMap[taskId]) scoreMap[taskId] = { score: 0, comment: '' }
  scoreMap[taskId].score = score
}

async function onScore(id: string) {
  const entry = scoreMap[id]
  if (!entry || !entry.score) return

  scoring.value = id
  error.value = ''
  try {
    const updated = await scoreTask(id, {
      score: entry.score,
      comment: entry.comment || undefined,
    })
    tasks.value = tasks.value.map((t) => (t.id === id ? updated : t))
    stats.value = await getTaskStats()
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    scoring.value = ''
  }
}

async function onReject(id: string) {
  const entry = scoreMap[id]
  scoring.value = id
  error.value = ''
  try {
    const updated = await rejectTask(id, {
      comment: entry?.comment || undefined,
    })
    tasks.value = tasks.value.map((t) => (t.id === id ? updated : t))
    delete scoreMap[id]
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    scoring.value = ''
  }
}

const categoryLabels: Record<string, string> = {
  housework: '家务',
  parenting: '育儿',
  health: '健康',
  emotional: '情感',
}

function categoryLabel(cat: string) {
  return categoryLabels[cat] || cat
}

function difficultyStars(d: number) {
  return '★'.repeat(d)
}
</script>

<style scoped>
.task-panel {
  position: relative;
  padding: 32px 28px 28px;
}

.panel-title {
  font-size: 22px;
  font-weight: 600;
  color: var(--text-primary);
  text-align: center;
  margin-bottom: 6px;
}

.panel-subtitle {
  text-align: center;
  color: var(--text-secondary);
  font-size: 14px;
  margin-bottom: 16px;
}

.stats-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 20px;
}

.level-badge {
  padding: 4px 14px;
  background: var(--accent-warm);
  color: #fff;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 700;
}

.xp-text {
  color: var(--text-secondary);
  font-size: 14px;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.task-card {
  position: relative;
  padding: 16px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  transition: border-color 0.2s;
  overflow: hidden;
}

.task-card::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 6px;
  background: rgba(255, 255, 255, 0.06);
}

.task-card.pri-T0::before { background: rgba(255, 90, 90, 0.8); }
.task-card.pri-T1::before { background: rgba(90, 160, 255, 0.75); }
.task-card.pri-T2::before { background: rgba(255, 255, 255, 0.08); }

.task-card.status-completed {
  border-color: rgba(255, 200, 80, 0.3);
}

.task-card.status-verified {
  border-color: rgba(80, 200, 120, 0.3);
  background: rgba(80, 200, 120, 0.05);
}

.task-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.task-badges {
  display: flex;
  align-items: center;
  gap: 8px;
}

.priority-badge {
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 700;
}

.priority-badge.pri-T0 { background: rgba(255, 90, 90, 0.18); color: #ffb8b8; }
.priority-badge.pri-T1 { background: rgba(90, 160, 255, 0.18); color: #b6d6ff; }
.priority-badge.pri-T2 { background: rgba(255, 255, 255, 0.06); color: rgba(255, 255, 255, 0.6); }

.category-badge {
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 600;
}

.cat-housework { background: rgba(100, 180, 255, 0.15); color: #b0d8ff; }
.cat-parenting { background: rgba(255, 180, 100, 0.15); color: #ffd49a; }
.cat-health { background: rgba(100, 220, 150, 0.15); color: #a8f0c0; }
.cat-emotional { background: rgba(220, 140, 255, 0.15); color: #e4c0ff; }

.difficulty {
  color: var(--accent-warm);
  font-size: 14px;
  letter-spacing: 2px;
}

.task-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 10px;
}

.task-kv {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}

.task-k {
  width: 56px;
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.55);
  letter-spacing: 1px;
}

.task-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.task-desc {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.4;
}

.proof-thumb {
  width: 100%;
  max-height: 180px;
  object-fit: cover;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.task-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.status-text {
  font-size: 13px;
  font-weight: 500;
}

.status-text.pending { color: var(--text-secondary); }
.status-text.completed { color: #ffd080; }
.status-text.verified { color: #a8f0c0; }

.action-btn {
  padding: 8px 18px;
  border: none;
  border-radius: 10px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s, transform 0.15s;
}

.action-btn:active { transform: scale(0.96); }
.action-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.complete-btn {
  background: var(--accent-warm);
  color: #fff;
}

.complete-btn:hover { background: var(--accent-warm-hover); }

.score-section {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.score-stars {
  display: flex;
  gap: 6px;
}

.star-btn {
  width: 36px;
  height: 36px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 22px;
  cursor: pointer;
  transition: all 0.15s;
  padding: 0;
  line-height: 1;
}

.star-btn.active {
  color: var(--accent-warm);
}

.star-btn:hover {
  transform: scale(1.2);
}

.score-input {
  padding: 10px 14px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 10px;
  color: var(--text-primary);
  font-size: 14px;
  outline: none;
}

.score-input::placeholder { color: var(--text-secondary); }

.score-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.score-btn {
  background: rgba(80, 200, 120, 0.8);
  color: #fff;
}

.score-btn:hover { background: rgba(80, 200, 120, 1); }

.reject-btn {
  background: rgba(255, 160, 80, 0.3);
  color: #ffc480;
}

.reject-btn:hover { background: rgba(255, 160, 80, 0.5); }

.score-comment {
  margin-top: 8px;
  padding: 10px 14px;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 10px;
  font-size: 13px;
  color: var(--text-secondary);
  font-style: italic;
}

.complete-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 11;
  border-radius: inherit;
}

.complete-dialog {
  width: 320px;
  max-width: 92%;
  background: var(--surface-elevated, #2a2a3a);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 16px;
  padding: 18px 16px 14px;
}

.complete-title {
  text-align: center;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.complete-subtitle {
  text-align: center;
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 12px;
}

.complete-actions {
  display: flex;
  justify-content: center;
  margin-bottom: 12px;
}

.proof-picker {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.proof-input {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}

.proof-picker-btn {
  padding: 10px 14px;
  border-radius: 12px;
  border: 1px dashed rgba(255, 255, 255, 0.2);
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.proof-picker-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.28);
}

.proof-preview {
  width: 100%;
  max-height: 200px;
  object-fit: cover;
  border-radius: 12px;
  margin-bottom: 10px;
}

.complete-error {
  margin-top: 8px;
  margin-bottom: 10px;
  padding: 8px 10px;
  background: rgba(220, 60, 60, 0.15);
  border: 1px solid rgba(220, 60, 60, 0.25);
  border-radius: 10px;
  color: #ffbbbb;
  font-size: 13px;
}

.complete-footer {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  flex-wrap: wrap;
}

.cancel-btn {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.8);
}

.cancel-btn:hover {
  background: rgba(255, 255, 255, 0.12);
}

.skip-btn {
  background: rgba(255, 160, 80, 0.22);
  color: #ffc480;
}

.skip-btn:hover {
  background: rgba(255, 160, 80, 0.3);
}

.submit-btn {
  background: rgba(80, 200, 120, 0.8);
  color: #fff;
}

.submit-btn:hover {
  background: rgba(80, 200, 120, 1);
}

.complete-fade-enter-active,
.complete-fade-leave-active {
  transition: opacity 0.2s ease;
}

.complete-fade-enter-from,
.complete-fade-leave-to {
  opacity: 0;
}

.error-msg {
  margin-top: 16px;
  padding: 10px 14px;
  background: rgba(220, 60, 60, 0.15);
  border: 1px solid rgba(220, 60, 60, 0.25);
  border-radius: 10px;
  color: #ffbbbb;
  font-size: 13px;
}

.loading-state, .empty-state {
  text-align: center;
  padding: 40px 20px;
  color: var(--text-secondary);
  font-size: 14px;
}

.panel-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.age-menu-btn {
  position: absolute;
  top: 16px;
  left: 16px;
  width: 36px;
  height: 36px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 50%;
  color: var(--text-secondary);
  font-size: 18px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
  line-height: 1;
  letter-spacing: 1px;
  z-index: 2;
}

.age-menu-btn:hover {
  background: rgba(255, 255, 255, 0.18);
  color: var(--text-primary);
}

.age-badge {
  display: inline-block;
  margin-left: 8px;
  padding: 2px 10px;
  background: rgba(255, 200, 80, 0.15);
  color: #ffd080;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 600;
}

.age-picker-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10;
  border-radius: inherit;
}

.age-picker {
  background: var(--surface-elevated, #2a2a3a);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 16px;
  padding: 24px;
  width: 280px;
  max-width: 90%;
}

.age-picker-title {
  text-align: center;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 16px;
}

.age-options {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.age-option {
  padding: 10px 8px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  color: var(--text-primary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.age-option:hover {
  background: rgba(255, 200, 80, 0.12);
  border-color: rgba(255, 200, 80, 0.3);
}

.age-option.active {
  background: rgba(255, 200, 80, 0.2);
  border-color: rgba(255, 200, 80, 0.5);
  color: #ffd080;
  font-weight: 600;
}

.age-option:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.age-fade-enter-active,
.age-fade-leave-active {
  transition: opacity 0.2s ease;
}

.age-fade-enter-from,
.age-fade-leave-to {
  opacity: 0;
}

/* ── Mobile ── */
@media (max-width: 768px) {
  .task-panel {
    padding: 24px 16px 20px;
  }

  .panel-title {
    font-size: 20px;
  }

  .task-card {
    padding: 14px;
  }

  .task-title {
    font-size: 15px;
  }

  .task-desc {
    font-size: 12px;
  }

  .star-btn {
    width: 44px;
    height: 44px;
    font-size: 24px;
  }

  .score-input {
    font-size: 16px;
  }

  .age-menu-btn {
    min-width: 44px;
    min-height: 44px;
  }
}
</style>
