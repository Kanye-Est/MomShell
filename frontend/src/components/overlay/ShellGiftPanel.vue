<template>
  <OverlayPanel :visible="uiStore.activePanel === 'shell-gift'" position="center" @close="uiStore.closePanel()">
    <div class="shell-gift-panel">
      <div class="panel-header">
        <h2 class="panel-title">盲盒贝壳</h2>
        <button class="refresh-btn" :disabled="loading" @click="fetchGifts">刷新</button>
      </div>
      <p class="panel-subtitle">
        来自他的完成记录
        <span v-if="unopenedCount > 0" class="count-badge">{{ unopenedCount }} 新</span>
      </p>

      <div v-if="!isMom" class="empty-state">仅妈妈角色可查看</div>

      <div v-else-if="loading" class="loading-state">加载中...</div>

      <div v-else-if="selectedGift" class="gift-detail">
        <button class="back-btn" @click="selectedGift = null">返回</button>
        <img v-if="selectedGift.cover_url" class="detail-cover" :src="selectedGift.cover_url" alt="" />
        <h3 class="detail-title">{{ selectedGift.ai_title }}</h3>
        <p class="detail-content">{{ selectedGift.ai_content }}</p>
        <img v-if="selectedGift.photo_url" class="detail-photo" :src="selectedGift.photo_url" alt="" />
      </div>

      <div v-else-if="error" class="error-msg">{{ error }}</div>

      <div v-else-if="gifts.length === 0" class="empty-state">还没有收到贝壳</div>

      <div v-else class="gift-list">
        <button
          v-for="g in gifts"
          :key="g.id"
          class="gift-card"
          :disabled="opening === g.id"
          @click="onSelectGift(g)"
        >
          <img class="gift-cover" :src="g.cover_url" alt="" loading="lazy" />
          <div class="gift-meta">
            <div class="gift-topline">
              <span class="gift-title">{{ g.ai_title || '一枚贝壳' }}</span>
              <span :class="['gift-status', g.is_opened ? 'opened' : 'sealed']">
                {{ g.is_opened ? '已打开' : '未开封' }}
              </span>
            </div>
            <p class="gift-snippet">
              {{
                g.is_opened
                  ? (g.ai_content || '').slice(0, 64)
                  : '点击打开，看看他做了什么'
              }}
            </p>
          </div>
        </button>
      </div>
    </div>
  </OverlayPanel>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import OverlayPanel from './OverlayPanel.vue'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { getErrorMessage } from '@/lib/apiClient'
import { getShellGifts, openShellGift, type ShellGiftItem } from '@/lib/api/shellGift'

const uiStore = useUiStore()
const authStore = useAuthStore()

const isMom = computed(() => authStore.user?.role === 'mom')

const gifts = ref<ShellGiftItem[]>([])
const selectedGift = ref<ShellGiftItem | null>(null)
const loading = ref(false)
const opening = ref('')
const error = ref('')

const unopenedCount = computed(() => gifts.value.filter((g) => !g.is_opened).length)

async function fetchGifts() {
  if (!isMom.value) return
  loading.value = true
  error.value = ''
  try {
    gifts.value = await getShellGifts()
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    loading.value = false
  }
}

async function onSelectGift(g: ShellGiftItem) {
  if (!isMom.value) return
  if (g.is_opened) {
    selectedGift.value = g
    return
  }
  opening.value = g.id
  error.value = ''
  try {
    const opened = await openShellGift(g.id)
    gifts.value = gifts.value.map((it) => (it.id === g.id ? opened : it))
    selectedGift.value = opened
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    opening.value = ''
  }
}

watch(
  () => uiStore.activePanel,
  (panel) => {
    if (panel === 'shell-gift') {
      selectedGift.value = null
      fetchGifts()
    } else {
      selectedGift.value = null
      opening.value = ''
      error.value = ''
    }
  },
)
</script>

<style scoped>
.shell-gift-panel {
  padding: 32px 28px 28px;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 6px;
}

.panel-title {
  font-size: 22px;
  font-weight: 600;
  color: var(--text-primary);
}

.refresh-btn {
  padding: 8px 12px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.refresh-btn:hover {
  background: rgba(255, 255, 255, 0.12);
}

.refresh-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.panel-subtitle {
  text-align: left;
  color: var(--text-secondary);
  font-size: 14px;
  margin-bottom: 18px;
}

.count-badge {
  display: inline-block;
  margin-left: 8px;
  padding: 2px 10px;
  background: rgba(255, 200, 80, 0.15);
  color: #ffd080;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 700;
}

.gift-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.gift-card {
  display: grid;
  grid-template-columns: 90px 1fr;
  gap: 12px;
  padding: 12px;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.06);
  cursor: pointer;
  text-align: left;
}

.gift-card:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.gift-card:hover {
  border-color: rgba(255, 200, 80, 0.25);
  background: rgba(255, 255, 255, 0.08);
}

.gift-cover {
  width: 90px;
  height: 68px;
  object-fit: cover;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.gift-meta {
  min-width: 0;
}

.gift-topline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 6px;
}

.gift-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gift-status {
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.gift-status.sealed {
  background: rgba(255, 200, 80, 0.14);
  color: #ffd080;
}

.gift-status.opened {
  background: rgba(80, 200, 120, 0.14);
  color: #a8f0c0;
}

.gift-snippet {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.35;
}

.gift-detail {
  padding-top: 8px;
}

.back-btn {
  padding: 8px 12px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  margin-bottom: 12px;
}

.detail-cover {
  width: 100%;
  max-height: 220px;
  object-fit: cover;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  margin-bottom: 12px;
}

.detail-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 10px;
}

.detail-content {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.85);
  line-height: 1.55;
  white-space: pre-wrap;
  margin: 0 0 14px;
}

.detail-photo {
  width: 100%;
  max-height: 320px;
  object-fit: cover;
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.08);
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

.loading-state,
.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: var(--text-secondary);
  font-size: 14px;
}

@media (max-width: 768px) {
  .shell-gift-panel {
    padding: 24px 16px 20px;
  }
}
</style>
