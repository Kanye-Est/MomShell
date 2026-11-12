<template>
  <OverlayPanel :visible="uiStore.activePanel === 'memory'" position="center" @close="uiStore.closePanel()">
    <div class="memory-panel">
      <h2 class="memory-title">哪段记忆在闪光？</h2>
      <p class="memory-subtitle">请输入或选择记忆砂砾</p>

      <!-- Tags -->
      <div class="tag-section">
        <div v-if="loadingTags" class="loading-state">加载中...</div>
        <div v-else class="tag-cloud">
          <button
            v-for="tag in allTags"
            :key="tag.id"
            :class="['tag-chip', { selected: selectedTag === tag.id }]"
            @click="onSelectTag(tag)"
          >
            {{ tag.content }}
          </button>
          <span v-if="allTags.length === 0" class="empty-hint">暂无标签，可以手动输入</span>
        </div>
      </div>

      <!-- Input -->
      <div class="memory-input-area">
        <input
          v-model="inputText"
          type="text"
          class="memory-input"
          placeholder="输入一段记忆碎片..."
          @keydown.enter="onGenerate"
        />
        <button class="generate-btn" :disabled="generating || (!inputText.trim() && !selectedTag)" @click="onGenerate">
          {{ generating ? '生成中...' : '生成贴纸' }}
        </button>
      </div>

      <!-- Error -->
      <p v-if="error" class="memory-error">{{ error }}</p>

      <!-- Result -->
      <div v-if="result" class="result-card">
        <div v-if="generatingImage" class="result-img-loading">图片生成中...</div>
        <img v-else-if="resultImageUrl" :src="resultImageUrl" alt="" class="result-img" />
        <div class="result-body">
          <h3 v-if="!editingResult" class="result-title">{{ result.title }}</h3>
          <input v-else v-model="editTitle" class="result-title-input" placeholder="标题" />
          <p v-if="!editingResult" class="result-content">{{ result.content }}</p>
          <textarea v-else v-model="editContent" class="result-content-input" placeholder="内容" />
          <div class="result-actions">
            <template v-if="!editingResult">
              <button class="result-action-btn" @click="startEdit">编辑文字</button>
              <button v-if="result.image_generation_enabled !== false" class="result-action-btn accent" :disabled="generatingImage" @click="onRegenerateImage">
                {{ generatingImage ? '生成中...' : '重新生成图片' }}
              </button>
            </template>
            <template v-else>
              <button class="result-action-btn accent" @click="confirmEdit">确认</button>
              <button class="result-action-btn" @click="editingResult = false">取消</button>
            </template>
          </div>
        </div>
      </div>
    </div>
  </OverlayPanel>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import OverlayPanel from './OverlayPanel.vue'
import { useUiStore } from '@/stores/ui'
import {
  getIdentityTags,
  generateMemoir,
  type IdentityTag,
  type IdentityTagList,
  type GeneratedMemoir,
} from '@/lib/api/echo'
import { generatePhoto, deletePhoto } from '@/lib/api/photo'
import { getErrorMessage } from '@/lib/apiClient'

const uiStore = useUiStore()

const tags = ref<IdentityTagList | null>(null)
const loadingTags = ref(false)
const selectedTag = ref<string | null>(null)
const inputText = ref('')
const generating = ref(false)
const generatingImage = ref(false)
const generatedImageUrl = ref('')
const generatedPhotoId = ref<string | null>(null)
const result = ref<GeneratedMemoir | null>(null)
const resultImageUrl = computed(() => generatedImageUrl.value || result.value?.cover_image_url)
const error = ref('')
const editingResult = ref(false)
const editTitle = ref('')
const editContent = ref('')

const allTags = computed<IdentityTag[]>(() => {
  if (!tags.value) return []
  return [
    ...tags.value.music,
    ...tags.value.sound,
    ...tags.value.literature,
    ...tags.value.memory,
  ]
})

watch(
  () => uiStore.activePanel,
  async (panel) => {
    if (panel === 'memory') {
      loadingTags.value = true
      try {
        tags.value = await getIdentityTags()
      } catch {
        // silent
      } finally {
        loadingTags.value = false
      }
    }
  },
)

function onSelectTag(tag: IdentityTag) {
  if (selectedTag.value === tag.id) {
    selectedTag.value = null
    inputText.value = ''
  } else {
    selectedTag.value = tag.id
    inputText.value = tag.content
  }
}

async function onGenerate() {
  const theme = inputText.value.trim()
  if (!theme && !selectedTag.value) return

  error.value = ''
  generating.value = true
  generatingImage.value = false
  generatedImageUrl.value = ''
  generatedPhotoId.value = null
  result.value = null
  try {
    const memoir = await generateMemoir(theme || undefined)
    result.value = memoir
    editingResult.value = false

    if (memoir.image_generation_enabled !== false) {
      triggerImageGeneration(memoir.content || memoir.title || theme || '记忆贴纸')
    }
  } catch (e) {
    error.value = getErrorMessage(e)
  } finally {
    generating.value = false
  }
}

function triggerImageGeneration(prompt: string) {
  generatingImage.value = true
  generatedImageUrl.value = ''

  // Delete old generated photo if exists
  const oldId = generatedPhotoId.value
  if (oldId) {
    deletePhoto(oldId).catch(() => {})
    generatedPhotoId.value = null
  }

  generatePhoto(prompt)
    .then((photo) => {
      generatedImageUrl.value = photo.image_url
      generatedPhotoId.value = photo.id
    })
    .catch(() => {
      // The result falls back to the built-in cover.
    })
    .finally(() => {
      generatingImage.value = false
    })
}

function startEdit() {
  if (!result.value) return
  editTitle.value = result.value.title
  editContent.value = result.value.content
  editingResult.value = true
}

function confirmEdit() {
  if (!result.value) return
  result.value = { ...result.value, title: editTitle.value, content: editContent.value }
  editingResult.value = false
}

function onRegenerateImage() {
  if (!result.value) return
  triggerImageGeneration(result.value.content || result.value.title || '记忆贴纸')
}
</script>

<style scoped>
.memory-panel {
  padding: 32px 28px 28px;
}

.memory-title {
  font-size: 22px;
  font-weight: 600;
  color: var(--text-primary);
  text-align: center;
  margin-bottom: 6px;
}

.memory-subtitle {
  text-align: center;
  color: var(--text-secondary);
  font-size: 14px;
  margin-bottom: 24px;
}

.tag-section {
  margin-bottom: 20px;
}

.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
}

.tag-chip {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 20px;
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.tag-chip:hover {
  background: rgba(255, 255, 255, 0.14);
}

.tag-chip.selected {
  background: var(--accent-warm);
  border-color: var(--accent-warm);
  color: #fff;
}

.empty-hint {
  color: var(--text-secondary);
  font-size: 13px;
}

.memory-input-area {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}

.memory-input {
  flex: 1;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 14px;
  color: var(--text-primary);
  font-size: 15px;
  outline: none;
  transition: border-color 0.2s;
}

.memory-input:focus {
  border-color: rgba(255, 255, 255, 0.3);
}

.memory-input::placeholder {
  color: var(--text-secondary);
}

.generate-btn {
  padding: 12px 20px;
  background: var(--accent-warm);
  color: #fff;
  border: none;
  border-radius: 14px;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.2s, transform 0.15s;
}

.generate-btn:hover { background: var(--accent-warm-hover); }
.generate-btn:active { transform: scale(0.97); }
.generate-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.memory-error {
  padding: 10px 14px;
  background: rgba(220, 60, 60, 0.15);
  border: 1px solid rgba(220, 60, 60, 0.25);
  border-radius: 10px;
  color: #ffbbbb;
  font-size: 13px;
  margin-bottom: 16px;
}

.result-card {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 18px;
  overflow: hidden;
}

.result-img {
  width: 100%;
  aspect-ratio: 4/3;
  object-fit: cover;
}

.result-img-loading {
  width: 100%;
  aspect-ratio: 4/3;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-secondary);
  font-size: 14px;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 0.6; }
  50% { opacity: 1; }
}

.result-body {
  padding: 16px 18px 18px;
}

.result-title {
  font-size: 17px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.result-content {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.6;
}

.result-title-input {
  width: 100%;
  padding: 8px 12px;
  margin-bottom: 8px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 10px;
  color: var(--text-primary);
  font-size: 17px;
  font-weight: 600;
  outline: none;
  font-family: inherit;
}

.result-title-input:focus {
  border-color: rgba(255, 255, 255, 0.35);
}

.result-content-input {
  width: 100%;
  min-height: 120px;
  padding: 10px 12px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 10px;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.6;
  outline: none;
  resize: vertical;
  font-family: inherit;
}

.result-content-input:focus {
  border-color: rgba(255, 255, 255, 0.35);
}

.result-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.result-action-btn {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 10px;
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.result-action-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.15);
}

.result-action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.result-action-btn.accent {
  background: var(--accent-warm);
  border-color: var(--accent-warm);
  color: #fff;
}

.result-action-btn.accent:hover:not(:disabled) {
  background: var(--accent-warm-hover);
}

.loading-state {
  text-align: center;
  padding: 20px;
  color: var(--text-secondary);
  font-size: 14px;
}

/* ── Mobile ── */
@media (max-width: 768px) {
  .memory-panel {
    padding: 24px 16px 20px;
  }

  .memory-title {
    font-size: 20px;
  }

  .memory-input-area {
    flex-direction: column;
  }

  .memory-input {
    font-size: 16px;
  }

  .generate-btn {
    width: 100%;
    min-height: 44px;
  }

  .result-actions {
    flex-wrap: wrap;
  }

  .result-action-btn {
    min-height: 44px;
    flex: 1;
    min-width: 120px;
  }
}
</style>
