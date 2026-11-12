<template>
  <Transition name="car-page">
    <div v-if="visible" class="car-page">
      <button class="close-btn" @click="close" aria-label="关闭">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
          <path d="M5 5L15 15M15 5L5 15" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" />
        </svg>
      </button>
      <button class="mobile-profile-btn" @click="openProfile" aria-label="个人资料">
        <img :src="profileAvatarUrl" alt="avatar" class="mobile-profile-avatar" @error="onAvatarImgError" />
      </button>
      <button class="mobile-suitcase-btn" @click="openSuitcase" aria-label="照片集">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none">
          <rect x="2" y="7" width="20" height="14" rx="2" stroke="currentColor" stroke-width="1.8" />
          <path d="M8 7V5a2 2 0 012-2h4a2 2 0 012 2v2" stroke="currentColor" stroke-width="1.8" />
        </svg>
        <span v-if="allPhotos.length > 0" class="mobile-suitcase-badge">{{ allPhotos.length }}</span>
      </button>

      <div class="car-layout">
        <!-- LEFT: Photo Wall -->
        <div class="photo-wall">
          <!-- Dedicated PearlShell activation zone -->
          <div v-if="!showPearlShell && !pearlShellUnavailable" class="pearl-shell-trigger-zone" @click="activatePearlShell()" />
          <p v-if="pearlShellUnavailable" class="photo-wall-status" role="status">此设备的 3D 相册暂不可用，照片可在照片墙或照片集中查看。</p>

          <!-- Original grid (visible when PearlShell not active or in fullscreen) -->
          <div v-if="!showPearlShell || pearlShellFullscreen" class="photo-grid">
            <div
              v-for="(photo, idx) in wallPhotos"
              :key="photo.id"
              class="photo-frame"
              :style="getStickerStyle(idx)"
              @click.stop="openDetail(photo)"
            >
              <img :src="photo.image_url" :alt="photo.title || 'memory'" />
            </div>
            <div v-for="n in emptySlots" :key="'empty-' + n" class="photo-frame empty" />
          </div>
          <!-- Embedded PearlShell -->
          <div v-if="showPearlShell && !pearlShellFullscreen" class="pearl-shell-embedded">
            <PearlShellWrapper
              :photo-urls="allPhotoUrls"
              :is-fullscreen="false"
              @request-fullscreen="handleRequestFullscreen"
              @unavailable="handlePearlShellUnavailable"
            />
            <button class="pearl-shell-close" @click.stop="showPearlShell = false" aria-label="关闭贝壳">✕</button>
          </div>
        </div>

        <!-- RIGHT: Avatar Frames + Profile Entry -->
        <div class="right-section">
          <div class="avatars" @click="openProfile">
            <div class="avatar-wrapper">
              <img class="avatar-photo" :class="{ 'avatar-custom': hasPartnerCustomAvatar }" :src="partnerAvatarUrl" alt="partner avatar" @error="onAvatarImgError" />
              <img class="avatar-frame" :src="avatarFrame" alt="partner frame" />
            </div>
            <div class="ecg-link">
              <svg viewBox="0 0 100 40" preserveAspectRatio="none">
                <path class="ecg-path-bg" d="M0 20 L35 20 L38 16 L41 24 L45 4 L49 36 L52 20 L100 20" />
                <path class="ecg-path-pulse" d="M0 20 L35 20 L38 16 L41 24 L45 4 L49 36 L52 20 L100 20" />
              </svg>
            </div>
            <div class="avatar-wrapper">
              <img class="avatar-photo" :class="{ 'avatar-custom': hasCustomAvatar }" :src="profileAvatarUrl" alt="my avatar" @error="onAvatarImgError" />
              <img class="avatar-frame" :src="avatarFrame" alt="my frame" />
            </div>
          </div>

          <!-- Pearl Timeline -->
          <div v-if="timelineNodes.length > 0" class="pearl-timeline">
            <div class="timeline-line" />
            <div
              v-for="(node, idx) in timelineNodes"
              :key="node.id"
              class="timeline-node"
              :style="{ animationDelay: `${idx * 0.15}s` }"
              @click.stop="activatePearlShell()"
            >
              <img :src="starImg" class="pearl-dot" :style="{ transform: `rotate(${node.rotate}deg)` }" alt="" />
              <div class="timeline-label">
                <span class="timeline-date">{{ node.date }}</span>
                <span class="timeline-title">{{ node.label }}</span>
              </div>
            </div>
          </div>

          <!-- Box -->
          <div ref="boxRef" class="overflow-box" @click="openSuitcase">
            <img :src="boxImg" class="box-icon" alt="overflow box" />
            <span v-if="allPhotos.length > 0" class="box-badge">{{ allPhotos.length }}</span>
          </div>
        </div>
      </div>

      <!-- Fullscreen PearlShell overlay -->
      <Transition name="pearl-fullscreen">
        <div v-if="pearlShellFullscreen" class="pearl-shell-fullscreen">
          <PearlShellWrapper
            :photo-urls="allPhotoUrls"
            :is-fullscreen="true"
            @exit-fullscreen="handleExitFullscreen"
            @unavailable="handlePearlShellUnavailable"
          />
        </div>
      </Transition>

      <!-- Suitcase Modal (Photo Gallery) -->
      <Transition name="suitcase-panel">
        <div v-if="showSuitcase" class="suitcase-overlay" @click.self="showSuitcase = false">
          <div class="suitcase-modal" :style="modalOrigin">
            <button class="modal-close" @click="showSuitcase = false">✕</button>
            <h3 class="suitcase-title">照片集</h3>

            <!-- Actions -->
            <div class="suitcase-actions">
              <button class="action-btn" @click="triggerPhotoUpload" :disabled="uploading">
                {{ uploading ? '上传中...' : '上传照片' }}
              </button>
              <input
                ref="photoInput"
                type="file"
                accept="image/jpeg,image/png,image/gif,image/webp"
                style="display: none"
                @change="onPhotoUpload"
              />
            </div>

            <div v-if="photoError" class="photo-error">{{ photoError }}</div>

            <!-- Photo Grid -->
            <div v-if="allPhotos.length > 0" class="suitcase-grid">
              <div
                v-for="p in allPhotos"
                :key="p.id"
                class="suitcase-card"
                :class="{ 'on-wall': p.is_on_wall }"
              >
                <img :src="p.image_url" :alt="p.title || 'memory'" class="suitcase-img" @click="openDetail(p)" />
                <div class="suitcase-card-actions">
                  <button
                    class="card-action-btn"
                    :class="{ active: p.is_on_wall }"
                    :title="p.is_on_wall ? '从照片墙移除' : '添加到照片墙'"
                    @click="onToggleWall(p)"
                  >
                    {{ p.is_on_wall ? '✦' : '☆' }}
                  </button>
                  <button
                    class="card-action-btn delete-btn"
                    title="删除"
                    @click="onDeletePhoto(p.id)"
                  >
                    ✕
                  </button>
                </div>
                <span v-if="p.source === 'ai_generated'" class="source-badge">AI</span>
              </div>
            </div>
            <div v-else class="modal-empty">还没有照片，快来上传吧</div>
          </div>
        </div>
      </Transition>

      <!-- Photo Detail Modal -->
      <Transition name="modal-zoom">
        <div v-if="showDetail && detailPhoto" class="modal-overlay" @click.self="closeDetail">
          <div class="detail-modal">
            <button class="modal-close" @click="closeDetail">✕</button>
            <img :src="detailPhoto.image_url" :alt="detailPhoto.title || 'memory'" class="detail-image" />
            <div class="detail-info">
              <div v-if="!editingDetail" class="detail-view">
                <h3 class="detail-title">{{ detailPhoto.title || '未命名' }}</h3>
                <p v-if="detailPhoto.description" class="detail-desc">{{ detailPhoto.description }}</p>
                <div v-if="detailPhoto.tags.length > 0" class="detail-tags">
                  <span v-for="tag in detailPhoto.tags" :key="tag" class="detail-tag">{{ tag }}</span>
                </div>
                <div class="detail-meta">
                  <span>{{ detailPhoto.source === 'ai_generated' ? 'AI 生成' : '上传' }}</span>
                  <span>{{ formatDate(detailPhoto.created_at) }}</span>
                </div>
                <button class="action-btn" @click="startEditDetail">编辑信息</button>
              </div>
              <div v-else class="detail-edit">
                <input v-model="editForm.title" class="form-input" placeholder="标题" maxlength="200" />
                <textarea v-model="editForm.description" class="form-input form-textarea" placeholder="描述" maxlength="2000" />
                <input v-model="editForm.tagsStr" class="form-input" placeholder="标签 (逗号分隔)" />
                <div class="edit-actions">
                  <button class="action-btn" :disabled="editSaving" @click="onSaveDetail">
                    {{ editSaving ? '保存中...' : '保存' }}
                  </button>
                  <button class="action-btn action-cancel" @click="editingDetail = false">取消</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Transition>

      <!-- Profile Modal -->
      <Transition name="profile-panel">
        <div v-if="showProfile" class="profile-overlay" @click.self="closeProfile">
          <div class="modal-content profile-modal">
            <button class="modal-close" @click="closeProfile">✕</button>

            <div class="profile-header">
              <div class="profile-avatar" @click="triggerAvatarUpload">
                <img v-if="profileAvatarUrl" :src="profileAvatarUrl" alt="avatar" @error="onAvatarImgError" />
                <span v-else class="avatar-placeholder">{{ displayInitial }}</span>
                <span class="avatar-edit-hint">{{ avatarUploading ? '...' : '✎' }}</span>
                <input
                  ref="avatarInput"
                  type="file"
                  accept="image/jpeg,image/png,image/gif,image/webp"
                  style="display: none"
                  @change="onAvatarChange"
                />
              </div>
              <div class="profile-info">
                <div v-if="!editingNickname" class="nickname-row" @click="startEditNickname">
                  <h2 class="profile-name">{{ displayNickname }}</h2>
                  <span class="edit-icon">✎</span>
                </div>
                <div v-else class="nickname-edit">
                  <input
                    ref="nicknameInput"
                    v-model="nicknameValue"
                    class="nickname-input"
                    maxlength="20"
                    @keydown.enter="saveNickname"
                    @blur="saveNickname"
                  />
                </div>
                <p class="profile-username">@{{ profile?.username || auth.user?.username }}</p>
                <span v-if="isAdmin" class="role-badge role-admin">管理员</span>
                <span class="role-badge" :class="'role-' + (profile?.role || auth.user?.role || 'mom')">{{ roleName }}</span>
              </div>
            </div>

            <div v-if="nicknameError" class="inline-error">{{ nicknameError }}</div>

            <div class="stats-grid">
              <div class="stat-cell">
                <span class="stat-value">{{ profile?.stats.question_count ?? 0 }}</span>
                <span class="stat-label">发帖</span>
              </div>
              <div class="stat-cell">
                <span class="stat-value">{{ profile?.stats.answer_count ?? 0 }}</span>
                <span class="stat-label">回答</span>
              </div>
              <div class="stat-cell">
                <span class="stat-value">{{ profile?.stats.like_received_count ?? 0 }}</span>
                <span class="stat-label">获赞</span>
              </div>
              <div class="stat-cell">
                <span class="stat-value">{{ profile?.stats.collection_count ?? 0 }}</span>
                <span class="stat-label">收藏</span>
              </div>
            </div>

            <div class="settings-content">
              <!-- Profile Info -->
              <div class="settings-section">
                <h3 class="settings-heading">个人资料</h3>
                <div class="profile-form">
                  <label class="form-label" for="profile-avatar">头像</label>
                  <div class="avatar-upload-row">
                    <div class="avatar-preview-small">
                      <img v-if="profileAvatarUrl" :src="profileAvatarUrl" alt="avatar" @error="onAvatarImgError" />
                      <span v-else class="avatar-placeholder-small">{{ displayInitial }}</span>
                    </div>
                    <button id="profile-avatar" class="upload-btn" :disabled="avatarUploading" @click="triggerAvatarUpload">
                      {{ avatarUploading ? '上传中...' : '更换头像' }}
                    </button>
                    <span class="upload-hint">JPG/PNG/GIF/WebP, 最大 2MB</span>
                  </div>
                  <div v-if="avatarError" class="form-error">{{ avatarError }}</div>
                  <label class="form-label" for="profile-username">用户名</label>
                  <input
                    v-model="profileForm.username"
                    id="profile-username"
                    type="text"
                    class="form-input"
                    placeholder="用户名"
                    maxlength="50"
                    minlength="3"
                  />
                  <label class="form-label" for="profile-nickname">昵称</label>
                  <input
                    v-model="profileForm.nickname"
                    id="profile-nickname"
                    type="text"
                    class="form-input"
                    placeholder="昵称"
                    maxlength="50"
                  />
                  <label class="form-label" for="profile-email">邮箱</label>
                  <input
                    v-model="profileForm.email"
                    id="profile-email"
                    type="email"
                    class="form-input"
                    placeholder="email@example.com"
                  />
                  <div v-if="profileFormError" class="form-error">{{ profileFormError }}</div>
                  <div v-if="profileFormSuccess" class="form-success">{{ profileFormSuccess }}</div>
                  <button
                    class="submit-btn"
                    :disabled="profileFormLoading"
                    @click="onSaveProfile"
                  >
                    {{ profileFormLoading ? '保存中...' : '保存资料' }}
                  </button>
                </div>
              </div>

              <!-- Identity Role -->
              <div class="settings-section">
                <h3 class="settings-heading">身份标签</h3>
                <div v-if="isBound" class="bound-hint">已绑定伴侣，身份不可更改</div>
                <div class="role-selector">
                  <button
                    v-for="opt in roleOptions"
                    :key="opt.key"
                    :class="['role-option', { active: selectedRole === opt.key }]"
                    :disabled="isBound"
                    @click="selectedRole = opt.key"
                  >
                    {{ opt.label }}
                  </button>
                </div>
                <div v-if="roleError" class="form-error">{{ roleError }}</div>
                <div v-if="roleSuccess" class="form-success">{{ roleSuccess }}</div>
                <button
                  v-if="!isBound"
                  class="submit-btn"
                  :disabled="roleLoading || selectedRole === (profile?.role || auth.user?.role)"
                  @click="onSaveRole"
                >
                  {{ roleLoading ? '保存中...' : '保存身份' }}
                </button>
              </div>

              <!-- Shell Code / Partner Binding -->
              <div class="settings-section">
                <h3 class="settings-heading">贝壳码</h3>

                <!-- Already bound: show partner info + unbind -->
                <template v-if="isBound">
                  <div class="partner-card">
                    <span class="partner-label">已绑定：</span>
                    <span class="partner-name">{{ profile?.partner?.nickname }}</span>
                    <span class="partner-role">（{{ profile?.partner?.role === 'mom' ? '溯源者' : '守护者' }}）</span>
                  </div>
                  <button class="submit-btn danger-btn" :disabled="unbindLoading" @click="onUnbindPartner">
                    {{ unbindLoading ? '解绑中...' : '解除绑定' }}
                  </button>
                </template>

                <!-- Mom (溯源者): generate shell code -->
                <template v-else-if="isMom">
                  <p class="shell-hint">生成贝壳码分享给守护者，完成伴侣绑定。</p>
                  <div v-if="profile?.shell_code" class="shell-code-display">
                    {{ profile.shell_code }}
                  </div>
                  <button class="submit-btn" :disabled="shellCodeLoading || !!profile?.shell_code" @click="onGenerateShellCode">
                    {{ shellCodeLoading ? '生成中...' : profile?.shell_code ? '已生成' : '生成贝壳码' }}
                  </button>
                  <div v-if="shellCodeError" class="form-error">{{ shellCodeError }}</div>
                </template>

                <!-- Dad (守护者): enter shell code to bind -->
                <template v-else>
                  <p class="shell-hint">输入溯源者分享的贝壳码，完成伴侣绑定。</p>
                  <input
                    v-model="bindCode"
                    class="form-input"
                    placeholder="请输入贝壳码"
                    maxlength="8"
                  />
                  <div v-if="bindError" class="form-error">{{ bindError }}</div>
                  <div v-if="bindSuccess" class="form-success">{{ bindSuccess }}</div>
                  <button class="submit-btn" :disabled="bindLoading || !bindCode.trim()" @click="onBindPartner">
                    {{ bindLoading ? '绑定中...' : '绑定' }}
                  </button>
                </template>
              </div>

              <!-- Change Password -->
              <div class="settings-section">
                <h3 class="settings-heading">修改密码</h3>
                <form class="password-form" @submit.prevent="onChangePassword">
                  <input
                    v-model="pwForm.old_password"
                    type="password"
                    class="form-input"
                    placeholder="当前密码"
                    required
                    autocomplete="current-password"
                  />
                  <input
                    v-model="pwForm.new_password"
                    type="password"
                    class="form-input"
                    placeholder="新密码"
                    required
                    minlength="6"
                    autocomplete="new-password"
                  />
                  <input
                    v-model="pwForm.confirm_password"
                    type="password"
                    class="form-input"
                    placeholder="确认新密码"
                    required
                    minlength="6"
                    autocomplete="new-password"
                  />
                  <div v-if="pwError" class="form-error">{{ pwError }}</div>
                  <div v-if="pwSuccess" class="form-success">{{ pwSuccess }}</div>
                  <button type="submit" class="submit-btn" :disabled="pwLoading">
                    {{ pwLoading ? '提交中...' : '修改密码' }}
                  </button>
                </form>
              </div>

              <div class="settings-section">
                <h3 class="settings-heading">背景音乐</h3>
                <div class="volume-control">
                  <input
                    class="volume-slider"
                    type="range"
                    min="0"
                    max="100"
                    step="1"
                    :value="backgroundMusicVolumePercent"
                    @input="onBackgroundMusicVolumeInput"
                  />
                  <span class="volume-value">{{ backgroundMusicVolumePercent }}%</span>
                </div>
                <p class="settings-hint">{{ backgroundMusicStatusText }}</p>
              </div>

              <div v-if="isAdmin" class="settings-section">
                <h3 class="settings-heading">管理面板</h3>
                <p class="settings-hint">进入后台管理系统，管理用户、照片和系统配置。</p>
                <a href="/admin" class="submit-btn" style="display: inline-block; text-align: center; text-decoration: none;">
                  打开管理面板
                </a>
              </div>

              <div class="settings-section">
                <button class="logout-btn" @click="onLogout">退出登录</button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, reactive, defineAsyncComponent } from 'vue'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import {
  getPhotos,
  uploadPhoto,
  updatePhoto,
  deletePhoto,
  togglePhotoWall,
  type Photo,
} from '@/lib/api/photo'
import {
  getUserProfile,
  updateUserProfile,
  uploadAvatar,
  generateShellCode,
  bindPartner,
  unbindPartner,
  changePassword,
  type UserProfile,
} from '@/lib/api/user'
import { getErrorMessage } from '@/lib/apiClient'
import { useBackgroundMusicControls } from '@/composables/useBackgroundMusicLoop'

import avatarFrame from '@/assets/images/frame.png'
import avatarDefault from '@/assets/images/avatar.png'
import boxImg from '@/assets/images/box.png'
import starImg from '@/assets/images/star.png'

const PearlShellWrapper = defineAsyncComponent(() => import('@/components/react/PearlShellWrapper.vue'))

const uiStore = useUiStore()
const auth = useAuthStore()

// ── Car page state ──
const allPhotos = ref<Photo[]>([])
const showSuitcase = ref(false)
const boxRef = ref<HTMLElement | null>(null)
const modalOrigin = ref<Record<string, string>>({})

// ── PearlShell state ──
const showPearlShell = ref(false)
const pearlShellFullscreen = ref(false)
const pearlShellUnavailable = ref(false)
const visible = computed(() => uiStore.activePanel === 'car')

const wallPhotos = computed(() =>
  allPhotos.value
    .filter((p) => p.is_on_wall)
    .sort((a, b) => (a.wall_position ?? 99) - (b.wall_position ?? 99))
    .slice(0, 10),
)

const allPhotoUrls = computed(() =>
  allPhotos.value.map((p) => p.image_url),
)

const emptySlots = computed(() => Math.max(0, 10 - wallPhotos.value.length))

const STAR_ROTATIONS = [0, 35, -25]

const timelineNodes = computed(() => {
  const photos = allPhotos.value
  if (photos.length === 0) return []

  const sorted = [...photos].sort(
    (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  )

  let picked: Photo[]
  if (sorted.length <= 3) {
    picked = sorted
  } else {
    picked = [sorted[0]]
    const mid = Math.round((sorted.length - 1) / 2)
    picked.push(sorted[mid], sorted[sorted.length - 1])
  }

  return picked.map((p, i) => {
    let label = p.title
    if (!label) {
      if (i === 0) {
        label = '第一张照片'
      } else if (i === picked.length - 1) {
        label = '最近的回忆'
      } else {
        label = '记忆碎片'
      }
    }
    return {
      id: p.id,
      date: formatDate(p.created_at),
      label,
      rotate: STAR_ROTATIONS[i % STAR_ROTATIONS.length],
    }
  })
})

// ── Photo management state ──
const photoInput = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const photoError = ref('')

// ── Photo detail state ──
const showDetail = ref(false)
const detailPhoto = ref<Photo | null>(null)
const editingDetail = ref(false)
const editForm = reactive({ title: '', description: '', tagsStr: '' })
const editSaving = ref(false)

// ── Profile modal state ──
const showProfile = ref(false)
const profile = ref<UserProfile | null>(null)
const editingNickname = ref(false)
const nicknameValue = ref('')
const nicknameError = ref('')
const nicknameInput = ref<HTMLInputElement | null>(null)

const pwForm = ref({ old_password: '', new_password: '', confirm_password: '' })
const pwError = ref('')
const pwSuccess = ref('')
const pwLoading = ref(false)

// Profile form
const profileForm = reactive({ username: '', nickname: '', email: '' })
const profileFormError = ref('')
const profileFormSuccess = ref('')
const profileFormLoading = ref(false)

// Avatar upload
const avatarInput = ref<HTMLInputElement | null>(null)
const avatarUploading = ref(false)
const avatarError = ref('')

// Identity role
const selectedRole = ref('')
const roleLoading = ref(false)
const roleError = ref('')
const roleSuccess = ref('')
const roleOptions = [
  { key: 'mom', label: '溯源者' },
  { key: 'dad', label: '守护者' },
]

// Shell code / partner binding
const shellCodeLoading = ref(false)
const shellCodeError = ref('')
const bindCode = ref('')
const bindLoading = ref(false)
const bindError = ref('')
const bindSuccess = ref('')
const unbindLoading = ref(false)

const isBound = computed(() => !!profile.value?.partner)
const isMom = computed(() => (profile.value?.role || auth.user?.role) === 'mom')

const roleMap: Record<string, string> = {
  mom: '溯源者',
  dad: '守护者',
  professional: '专业认证',
}

const displayNickname = computed(() => profile.value?.nickname || auth.user?.nickname || '用户')
const displayInitial = computed(() => displayNickname.value.charAt(0))
const profileAvatarUrl = computed(() => profile.value?.avatar_url || auth.user?.avatar_url || avatarDefault)
const hasCustomAvatar = computed(() => !!(profile.value?.avatar_url || auth.user?.avatar_url))
const partnerAvatarUrl = computed(() => profile.value?.partner?.avatar_url || avatarDefault)
const hasPartnerCustomAvatar = computed(() => !!profile.value?.partner?.avatar_url)

function onAvatarImgError(e: Event) {
  const img = e.target as HTMLImageElement
  if (!img.dataset.fallback) {
    img.dataset.fallback = '1'
    img.src = avatarDefault
  }
}
const roleName = computed(() => roleMap[profile.value?.role || auth.user?.role || 'mom'] || '溯源者')
const isAdmin = computed(() => !!(profile.value?.is_admin || auth.user?.is_admin))
const { backgroundMusicVolume, isBackgroundMusicPlaying, setBackgroundMusicVolume } = useBackgroundMusicControls()
const backgroundMusicVolumePercent = computed(() => Math.round(backgroundMusicVolume.value * 100))
const backgroundMusicStatusText = computed(() => isBackgroundMusicPlaying.value ? '拖动滑块调节背景音乐音量' : '如果还没有声音，请先点击页面任意位置以允许播放')

function onBackgroundMusicVolumeInput(event: Event) {
  const target = event.target as HTMLInputElement
  setBackgroundMusicVolume(Number(target.value) / 100)
}

// ── Car page methods ──
function close() {
  showPearlShell.value = false
  pearlShellFullscreen.value = false
  uiStore.closePanel()
}

function activatePearlShell() {
  if (pearlShellUnavailable.value) return
  showPearlShell.value = true
}

function handleRequestFullscreen() {
  pearlShellFullscreen.value = true
}

function handleExitFullscreen() {
  pearlShellFullscreen.value = false
}

function handlePearlShellUnavailable() {
  pearlShellUnavailable.value = true
  showPearlShell.value = false
  pearlShellFullscreen.value = false
}

function openSuitcase() {
  if (boxRef.value) {
    const rect = boxRef.value.getBoundingClientRect()
    const x = rect.left + rect.width / 2
    const y = rect.top + rect.height / 2
    modalOrigin.value = { transformOrigin: `${x}px ${y}px` }
  }
  showSuitcase.value = !showSuitcase.value
}

// ── Photo methods ──
async function fetchPhotos() {
  try {
    const res = await getPhotos(1, 50)
    allPhotos.value = res.photos
  } catch {
    // silent
  }
}

function triggerPhotoUpload() {
  photoInput.value?.click()
}

async function onPhotoUpload(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  input.value = ''

  if (file.size > 5 * 1024 * 1024) {
    photoError.value = '照片大小不能超过 5MB'
    return
  }
  if (!['image/jpeg', 'image/png', 'image/gif', 'image/webp'].includes(file.type)) {
    photoError.value = '仅支持 JPG、PNG、GIF、WebP 格式'
    return
  }

  photoError.value = ''
  uploading.value = true
  try {
    const photo = await uploadPhoto(file)
    allPhotos.value = [photo, ...allPhotos.value]
  } catch (err) {
    photoError.value = getErrorMessage(err)
  } finally {
    uploading.value = false
  }
}

async function onToggleWall(photo: Photo) {
  const newIsOnWall = !photo.is_on_wall
  const wallCount = allPhotos.value.filter((p) => p.is_on_wall).length

  if (newIsOnWall && wallCount >= 10) {
    photoError.value = '照片墙已满 (最多 10 张)'
    return
  }

  let position: number | undefined
  if (newIsOnWall) {
    // Find the lowest free slot (0-8) to avoid position collisions
    const usedPositions = new Set(
      allPhotos.value
        .filter((p) => p.is_on_wall && p.id !== photo.id)
        .map((p) => p.wall_position),
    )
    for (let i = 0; i <= 9; i++) {
      if (!usedPositions.has(i)) {
        position = i
        break
      }
    }
  }

  try {
    const updated = await togglePhotoWall(photo.id, newIsOnWall, position)
    allPhotos.value = allPhotos.value.map((p) => (p.id === updated.id ? updated : p))
    photoError.value = ''
  } catch (err) {
    photoError.value = getErrorMessage(err)
  }
}

async function onDeletePhoto(id: string) {
  try {
    await deletePhoto(id)
    allPhotos.value = allPhotos.value.filter((p) => p.id !== id)
    if (detailPhoto.value?.id === id) {
      closeDetail()
    }
  } catch (err) {
    photoError.value = getErrorMessage(err)
  }
}

// ── Sticker scatter style ──
function getStickerStyle(index: number) {
  const seed = (index + 1) * 137.5
  const offsetX = Math.sin(seed) * 30
  const offsetY = Math.cos(seed * 1.3) * 20
  const rotate = Math.sin(seed * 0.7) * 15
  return {
    '--sticker-x': `${offsetX}px`,
    '--sticker-y': `${offsetY}px`,
    '--sticker-rotate': `${rotate}deg`,
  }
}

// ── Detail modal ──
function openDetail(photo: Photo) {
  detailPhoto.value = photo
  editingDetail.value = false
  showDetail.value = true
}

function closeDetail() {
  showDetail.value = false
  detailPhoto.value = null
  editingDetail.value = false
}

function startEditDetail() {
  if (!detailPhoto.value) return
  editForm.title = detailPhoto.value.title
  editForm.description = detailPhoto.value.description
  editForm.tagsStr = detailPhoto.value.tags.join(', ')
  editingDetail.value = true
}

async function onSaveDetail() {
  if (!detailPhoto.value) return
  editSaving.value = true
  try {
    const tags = editForm.tagsStr
      .split(',')
      .map((t) => t.trim())
      .filter(Boolean)
    const updated = await updatePhoto(detailPhoto.value.id, {
      title: editForm.title,
      description: editForm.description,
      tags,
    })
    allPhotos.value = allPhotos.value.map((p) => (p.id === updated.id ? updated : p))
    detailPhoto.value = updated
    editingDetail.value = false
  } catch (err) {
    photoError.value = getErrorMessage(err)
  } finally {
    editSaving.value = false
  }
}

// ── Profile methods ──
function openProfile() {
  showProfile.value = true
  pwForm.value = { old_password: '', new_password: '', confirm_password: '' }
  pwError.value = ''
  pwSuccess.value = ''
  fetchProfile()
}

function closeProfile() {
  showProfile.value = false
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  const m = d.getMonth() + 1
  const day = d.getDate()
  return `${m}/${day}`
}

function startEditNickname() {
  nicknameValue.value = displayNickname.value
  nicknameError.value = ''
  editingNickname.value = true
  nextTick(() => nicknameInput.value?.focus())
}

async function saveNickname() {
  if (!editingNickname.value) return
  const trimmed = nicknameValue.value.trim()
  if (!trimmed || trimmed === displayNickname.value) {
    editingNickname.value = false
    return
  }
  try {
    const updated = await updateUserProfile({ nickname: trimmed })
    profile.value = updated
    if (auth.user) {
      auth.user.nickname = updated.nickname
    }
    profileForm.nickname = updated.nickname
    nicknameError.value = ''
  } catch (e) {
    nicknameError.value = getErrorMessage(e)
  }
  editingNickname.value = false
}

function syncProfileForm() {
  profileForm.username = profile.value?.username || auth.user?.username || ''
  profileForm.nickname = profile.value?.nickname || auth.user?.nickname || ''
  profileForm.email = profile.value?.email || auth.user?.email || ''
}

async function fetchProfile() {
  try {
    profile.value = await getUserProfile()
    syncProfileForm()
    selectedRole.value = profile.value?.role || auth.user?.role || 'mom'
  } catch {
    syncProfileForm()
  }
}

function triggerAvatarUpload() {
  avatarInput.value?.click()
}

async function onAvatarChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  input.value = ''

  if (file.size > 2 * 1024 * 1024) {
    avatarError.value = '图片大小不能超过 2MB'
    return
  }
  if (!['image/jpeg', 'image/png', 'image/gif', 'image/webp'].includes(file.type)) {
    avatarError.value = '仅支持 JPG、PNG、GIF、WebP 格式'
    return
  }

  avatarError.value = ''
  avatarUploading.value = true
  try {
    const updated = await uploadAvatar(file)
    profile.value = updated
    if (auth.user) {
      auth.user.avatar_url = updated.avatar_url
    }
  } catch (e) {
    avatarError.value = getErrorMessage(e)
  } finally {
    avatarUploading.value = false
  }
}

async function onSaveProfile() {
  profileFormError.value = ''
  profileFormSuccess.value = ''

  const data: Record<string, string> = {}
  const p = profile.value
  const trimmedUsername = profileForm.username.trim()
  const trimmedNickname = profileForm.nickname.trim()
  const trimmedEmail = profileForm.email.trim()

  if (trimmedUsername && trimmedUsername !== (p?.username || auth.user?.username)) {
    if (trimmedUsername.length < 3) {
      profileFormError.value = '用户名至少3个字符'
      return
    }
    data.username = trimmedUsername
  }
  if (trimmedNickname && trimmedNickname !== (p?.nickname || auth.user?.nickname)) {
    data.nickname = trimmedNickname
  }
  if (trimmedEmail && trimmedEmail !== (p?.email || auth.user?.email)) {
    data.email = trimmedEmail
  }

  if (Object.keys(data).length === 0) {
    profileFormSuccess.value = '没有需要更新的内容'
    return
  }

  profileFormLoading.value = true
  try {
    const updated = await updateUserProfile(data)
    profile.value = updated
    if (auth.user) {
      auth.user.username = updated.username
      auth.user.nickname = updated.nickname
      auth.user.email = updated.email
      auth.user.avatar_url = updated.avatar_url
    }
    syncProfileForm()
    profileFormSuccess.value = '资料已更新'
  } catch (e) {
    profileFormError.value = getErrorMessage(e)
  } finally {
    profileFormLoading.value = false
  }
}

async function onSaveRole() {
  if (!selectedRole.value) return
  roleError.value = ''
  roleSuccess.value = ''
  roleLoading.value = true
  try {
    const updated = await updateUserProfile({ role: selectedRole.value })
    profile.value = updated
    if (auth.user) {
      auth.user.role = updated.role
    }
    roleSuccess.value = '身份已更新'
  } catch (e) {
    roleError.value = getErrorMessage(e)
  } finally {
    roleLoading.value = false
  }
}

async function onGenerateShellCode() {
  shellCodeError.value = ''
  shellCodeLoading.value = true
  try {
    const updated = await generateShellCode()
    profile.value = updated
  } catch (e) {
    shellCodeError.value = getErrorMessage(e)
  } finally {
    shellCodeLoading.value = false
  }
}

async function onBindPartner() {
  if (!bindCode.value.trim()) return
  bindError.value = ''
  bindSuccess.value = ''
  bindLoading.value = true
  try {
    const updated = await bindPartner(bindCode.value.trim())
    profile.value = updated
    bindSuccess.value = '绑定成功'
    bindCode.value = ''
  } catch (e) {
    bindError.value = getErrorMessage(e)
  } finally {
    bindLoading.value = false
  }
}

async function onUnbindPartner() {
  unbindLoading.value = true
  try {
    const updated = await unbindPartner()
    profile.value = updated
  } catch (e) {
    bindError.value = getErrorMessage(e)
  } finally {
    unbindLoading.value = false
  }
}

async function onChangePassword() {
  pwError.value = ''
  pwSuccess.value = ''
  if (pwForm.value.new_password !== pwForm.value.confirm_password) {
    pwError.value = '两次输入的新密码不一致'
    return
  }
  if (pwForm.value.new_password.length < 6) {
    pwError.value = '新密码长度不能少于6位'
    return
  }
  pwLoading.value = true
  try {
    const res = await changePassword({
      old_password: pwForm.value.old_password,
      new_password: pwForm.value.new_password,
    })
    pwSuccess.value = res.message || '密码修改成功'
    pwForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (e) {
    pwError.value = getErrorMessage(e)
  } finally {
    pwLoading.value = false
  }
}

function onLogout() {
  auth.logout()
  close()
}

// ── Watchers ──
watch(visible, async (isVisible) => {
  if (isVisible) {
    showSuitcase.value = false
    showProfile.value = false
    showDetail.value = false
    showPearlShell.value = false
    pearlShellFullscreen.value = false
    try {
      await Promise.all([
        fetchPhotos(),
        fetchProfile(),
      ])
    } catch {
      // silent
    }
  }
})
</script>

<style scoped>
.car-page {
  position: fixed;
  inset: 0;
  z-index: 100;
  background: url('@/assets/images/car-bg.png') center / 100% 100% no-repeat;
  background-color: #3a2f28;
  display: flex;
  flex-direction: column;
}

@media (max-width: 768px) {
  .car-page {
    background-image: none;
    background-color: #d4c4a8;
  }
}

.close-btn {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 10;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 50%;
  color: var(--text-primary, #fff);
  cursor: pointer;
  transition: background 0.2s, transform 0.15s;
}

.close-btn:hover {
  background: rgba(255, 255, 255, 0.2);
  transform: scale(1.08);
}

.car-layout {
  display: flex;
  height: 100%;
  padding: 5vh 5vw 5vh 20vw;
  gap: 4vw;
  align-items: center;
}

/* ── Photo Wall (Left) ── */
.photo-wall {
  position: relative;
  flex: 1;
  min-width: 0;
  display: flex;
  align-self: stretch;
  align-items: flex-start;
  justify-content: center;
  padding-top: 21vh;
}

.photo-wall-status {
  position: absolute;
  bottom: 12px;
  padding: 6px 10px;
  border-radius: 8px;
  background: rgba(31, 26, 23, 0.8);
  color: white;
  font-size: 0.875rem;
}

/* ── PearlShell Trigger Zone ── */
.pearl-shell-trigger-zone {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: min(936px, 100%);
  height: min(544px, calc(100% - 34px));
  margin-top: -17px;
  margin-left: 0px;
  z-index: 1; /* Below photos but above background */
  cursor: pointer;
  border-radius: 0;
  animation: pulse-glow 3s ease-in-out infinite;
}

@keyframes pulse-glow {
  0%, 100% { box-shadow: inset 0 0 40px 20px rgba(255, 180, 200, 0.05), inset 0 0 25px 12px rgba(255, 210, 80, 0.05); }
  50% { box-shadow: inset 0 0 80px 40px rgba(255, 180, 200, 0.2), inset 0 0 60px 30px rgba(255, 210, 80, 0.3); }
}

.pearl-shell-trigger-zone:hover {
  animation-name: pulse-glow-hover;
  animation-duration: 2s;
}

@keyframes pulse-glow-hover {
  0%, 100% { box-shadow: inset 0 0 60px 30px rgba(255, 170, 190, 0.2), inset 0 0 40px 20px rgba(255, 210, 80, 0.25); }
  50% { box-shadow: inset 0 0 120px 60px rgba(255, 150, 180, 0.5), inset 0 0 90px 45px rgba(255, 210, 80, 0.55); }
}

.photo-grid {
  position: relative;
  z-index: 2;
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 18px;
  width: 100%;
  max-width: 880px;
  pointer-events: none;
}

/* 玻璃拟态 + 散落 + 黑白默认 */
.photo-frame {
  aspect-ratio: 3 / 4;
  border-radius: 16px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(8px);
  box-shadow: 0 0 12px rgba(255, 255, 255, 0.06);
  cursor: pointer;

  /* 散落定位 */
  transform: translate(var(--sticker-x, 0), var(--sticker-y, 0))
             rotate(var(--sticker-rotate, 0deg));

  /* 默认：原色 + 半透明（沉睡的记忆） */
  filter: brightness(0.8);
  opacity: 0.8;

  transition: transform 0.5s ease, filter 0.5s ease,
              opacity 0.5s ease, box-shadow 0.5s ease, z-index 0s;
}

.photo-frame:not(.empty) {
  pointer-events: auto;
}

/* hover：唤醒回忆 */
.photo-frame:hover:not(.empty) {
  transform: translate(var(--sticker-x, 0), var(--sticker-y, 0))
             rotate(0deg) scale(1.5);
  filter: brightness(1);
  opacity: 1;
  box-shadow: 0 0 24px rgba(255, 255, 255, 0.15);
  z-index: 10;
}

.photo-frame img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.photo-frame.empty {
  background: rgba(255, 255, 255, 0.03);
  border: 1px dashed rgba(255, 255, 255, 0.08);
  cursor: default;
  filter: none;
  opacity: 0.4;
  backdrop-filter: none;
  box-shadow: none;
}

/* ── Right Section ── */
.right-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: min(400px, 30vw);
  align-self: stretch;
  padding-top: 18vh;
  margin-right: 8vw;
}

.avatars {
  display: flex;
  align-items: center;
  gap: 0;
  cursor: pointer;
  margin-bottom: 48px;
}

@keyframes floating-group {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-8px); }
}

.ecg-link {
  width: 120px;
  height: 60px;
  margin: 0 -30px;
  z-index: 5;
  pointer-events: none;
  display: flex;
  align-items: center;
  justify-content: center;
}

.ecg-link svg {
  width: 100%;
  height: 100%;
}

.ecg-path-bg {
  fill: none;
  stroke: rgba(255, 140, 160, 0.2);
  stroke-width: 1.5;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.ecg-path-pulse {
  fill: none;
  stroke: #ff8ca0;
  stroke-width: 2.5;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-dasharray: 200;
  stroke-dashoffset: 200;
  filter: drop-shadow(0 0 8px rgba(255, 140, 160, 0.9));
  animation: ecg-pulse 3s linear infinite;
}

@keyframes ecg-pulse {
  0% { stroke-dashoffset: 200; opacity: 0; }
  10% { opacity: 1; }
  40% { stroke-dashoffset: 0; opacity: 1; }
  50% { stroke-dashoffset: 0; opacity: 0; }
  100% { stroke-dashoffset: 0; opacity: 0; }
}

@keyframes heartbeat {
  0%, 100% { transform: scale(1); filter: drop-shadow(0 0 8px rgba(255, 140, 160, 0.6)); }
  14% { transform: scale(1.1); }
  28% { transform: scale(1); }
  42% { transform: scale(1.1); filter: drop-shadow(0 0 12px rgba(255, 140, 160, 0.9)); }
  70% { transform: scale(1); filter: drop-shadow(0 0 8px rgba(255, 140, 160, 0.6)); }
}

.avatar-wrapper {
  position: relative;
  width: 160px;
  height: 160px;
  transition: transform 0.2s;
}

.avatar-wrapper:hover {
  transform: scale(1.08);
}

.avatar-photo {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 70%;
  height: 70%;
  border-radius: 50%;
  object-fit: cover;
}

.avatar-photo.avatar-custom {
  width: 40%;
  height: 40%;
}

.avatar-frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
}

/* ── Overflow Box ── */
.overflow-box {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: auto;
  margin-bottom: -3vh; /* 落地基准调试：进一步向下移动 */
  cursor: pointer;
  transition: transform 0.2s;
}

.overflow-box:hover {
  transform: scale(1.08);
}

.box-icon {
  width: 360px;
  height: 360px;
  object-fit: contain;
  filter: drop-shadow(0 2px 8px rgba(0, 0, 0, 0.4));
}

.box-badge {
  position: absolute;
  top: 22%;
  right: 18%;
  min-width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 6px;
  background: var(--accent-warm, #ff9f43);
  border-radius: 11px;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
}

/* ── Pearl Timeline ── */
.pearl-timeline {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 80px;
  max-height: 420px;
  padding: 16px 0;
  transform: translateX(-24px);
}

.timeline-line {
  position: absolute;
  top: 16px;
  bottom: 16px;
  left: 50%;
  width: 2px;
  transform: translateX(-50%);
  background: linear-gradient(
    to bottom,
    rgba(255, 180, 200, 0.05),
    rgba(255, 180, 200, 0.3) 15%,
    rgba(255, 210, 80, 0.3) 85%,
    rgba(255, 210, 80, 0.05)
  );
}

.timeline-node {
  position: relative;
  display: flex;
  align-items: center;
  gap: 16px;
  flex: 1;
  cursor: pointer;
  padding: 8px 0;
  opacity: 0;
  animation: timeline-node-enter 0.5s ease forwards;
  transition: transform 0.25s;
}

.timeline-node:hover {
  transform: scale(1.15);
}

@keyframes timeline-node-enter {
  from { opacity: 0; }
  to { opacity: 1; }
}

.pearl-dot {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  object-fit: contain;
  filter: drop-shadow(0 0 6px rgba(255, 210, 80, 0.5));
  animation: pearl-glow-pulse 3s ease-in-out infinite;
  transition: filter 0.3s, transform 0.3s;
}

@keyframes pearl-glow-pulse {
  0%, 100% { filter: drop-shadow(0 0 6px rgba(255, 210, 80, 0.5)); }
  50% { filter: drop-shadow(0 0 12px rgba(255, 210, 80, 0.7)); }
}

.timeline-node:hover .pearl-dot {
  filter: drop-shadow(0 0 14px rgba(255, 210, 80, 0.9));
}

.timeline-label {
  display: flex;
  flex-direction: column;
  gap: 2px;
  white-space: nowrap;
  pointer-events: none;
}

.timeline-date {
  font-size: 15px;
  font-weight: 700;
  color: rgba(255, 210, 80, 1);
}

.timeline-title {
  font-size: 15px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.8);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ── Modal Overlay ── */
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--overlay-backdrop);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
}

.modal-content {
  position: relative;
  width: 70vw;
  max-height: 75vh;
  background: rgba(40, 34, 28, 0.95);
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 32px;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.15) transparent;
}

.modal-close {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 50%;
  color: #fff;
  font-size: 16px;
  cursor: pointer;
  transition: background 0.2s;
  z-index: 2;
}

.modal-close:hover {
  background: rgba(255, 255, 255, 0.2);
}

.modal-empty {
  text-align: center;
  color: rgba(255, 255, 255, 0.7);
  font-size: 16px;
  padding: 48px 0;
}

/* ── Suitcase Overlay (matches OverlayPanel) ── */
.suitcase-overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--overlay-backdrop);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
}

/* ── Suitcase Modal ── */
.suitcase-modal {
  position: relative;
  width: 75vw;
  max-width: 900px;
  max-height: 80vh;
  background: var(--glass-bg-heavy);
  backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
  -webkit-backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
  border-radius: var(--glass-radius);
  border: 1px solid var(--glass-border);
  box-shadow: var(--glass-shadow), var(--glass-inner-glow);
  padding: 32px;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.15) transparent;
}

/* ── Suitcase Panel Transition ── */
.suitcase-panel-enter-active {
  transition: opacity 0.35s cubic-bezier(0.16, 1, 0.3, 1);
}
.suitcase-panel-enter-active .suitcase-modal {
  transition: transform 0.4s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.35s ease;
}
.suitcase-panel-leave-active {
  transition: opacity 0.25s ease;
}
.suitcase-panel-leave-active .suitcase-modal {
  transition: transform 0.25s ease, opacity 0.25s ease;
}
.suitcase-panel-enter-from {
  opacity: 0;
}
.suitcase-panel-enter-from .suitcase-modal {
  transform: scale(0.92) translateY(20px);
  opacity: 0;
}
.suitcase-panel-leave-to {
  opacity: 0;
}
.suitcase-panel-leave-to .suitcase-modal {
  transform: scale(0.95) translateY(10px);
  opacity: 0;
}

.suitcase-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary, #fff);
  margin-bottom: 20px;
}

.suitcase-actions {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}

.action-btn {
  padding: 10px 20px;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 12px;
  color: var(--text-primary, #fff);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s, transform 0.15s;
}

.action-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.18);
  transform: scale(1.02);
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: default;
}

.action-btn.action-cancel {
  background: rgba(255, 255, 255, 0.06);
}

.photo-error {
  padding: 8px 14px;
  margin-bottom: 12px;
  background: rgba(255, 100, 100, 0.12);
  border-radius: 10px;
  font-size: 13px;
  color: #ffaaaa;
}

/* ── Suitcase Grid ── */
.suitcase-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 14px;
}

.suitcase-card {
  position: relative;
  border-radius: 14px;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: transform 0.2s, border-color 0.2s;
}

.suitcase-card:hover {
  transform: scale(1.02);
}

.suitcase-card.on-wall {
  border-color: rgba(255, 182, 100, 0.5);
  box-shadow: 0 0 12px rgba(255, 182, 100, 0.15);
}

.suitcase-img {
  width: 100%;
  aspect-ratio: 3 / 4;
  object-fit: cover;
  cursor: pointer;
  display: block;
}

.suitcase-card-actions {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: space-between;
  padding: 6px 8px;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.6));
  opacity: 0;
  transition: opacity 0.2s;
}

.suitcase-card:hover .suitcase-card-actions {
  opacity: 1;
}

.card-action-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.15);
  border: none;
  border-radius: 50%;
  color: #fff;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}

.card-action-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

.card-action-btn.active {
  color: #ffc88a;
}

.card-action-btn.delete-btn:hover {
  background: rgba(255, 80, 80, 0.4);
}

.source-badge {
  position: absolute;
  top: 6px;
  left: 6px;
  padding: 2px 6px;
  background: rgba(147, 112, 219, 0.8);
  border-radius: 6px;
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.5px;
}

/* ── Detail Modal ── */
.detail-modal {
  position: relative;
  width: 600px;
  max-width: 90vw;
  max-height: 85vh;
  background: var(--glass-bg-heavy);
  backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
  -webkit-backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
  border-radius: var(--glass-radius);
  border: 1px solid var(--glass-border);
  box-shadow: var(--glass-shadow), var(--glass-inner-glow);
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.15) transparent;
}

.detail-image {
  width: 100%;
  max-height: 400px;
  object-fit: contain;
  background: rgba(0, 0, 0, 0.3);
  border-radius: var(--glass-radius) var(--glass-radius) 0 0;
}

.detail-info {
  padding: 24px;
}

.detail-view {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.detail-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary, #fff);
}

.detail-desc {
  font-size: 14px;
  color: var(--text-secondary, rgba(255, 255, 255, 0.7));
  line-height: 1.6;
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.detail-tag {
  padding: 3px 10px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  font-size: 12px;
  color: var(--text-secondary, rgba(255, 255, 255, 0.7));
}

.detail-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: var(--text-secondary, rgba(255, 255, 255, 0.65));
}

.detail-edit {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.form-textarea {
  min-height: 80px;
  resize: vertical;
}

.edit-actions {
  display: flex;
  gap: 8px;
}

/* ── Fade Transition ── */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s, max-height 0.3s;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* ── Profile Overlay (matches OverlayPanel) ── */
.profile-overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--overlay-backdrop);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
}

/* ── Profile Modal ── */
.profile-modal {
  width: 520px;
  max-width: 90vw;
  padding: 32px 28px;
  background: var(--glass-bg-heavy);
  backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
  -webkit-backdrop-filter: blur(var(--glass-blur)) saturate(var(--glass-saturate));
  border: 1px solid var(--glass-border);
  border-radius: var(--glass-radius);
  box-shadow: var(--glass-shadow), var(--glass-inner-glow);
}

/* ── Profile Panel Transition ── */
.profile-panel-enter-active {
  transition: opacity 0.35s cubic-bezier(0.16, 1, 0.3, 1);
}
.profile-panel-enter-active .profile-modal {
  transition: transform 0.4s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.35s ease;
}
.profile-panel-leave-active {
  transition: opacity 0.25s ease;
}
.profile-panel-leave-active .profile-modal {
  transition: transform 0.25s ease, opacity 0.25s ease;
}
.profile-panel-enter-from {
  opacity: 0;
}
.profile-panel-enter-from .profile-modal {
  transform: scale(0.92) translateY(20px);
  opacity: 0;
}
.profile-panel-leave-to {
  opacity: 0;
}
.profile-panel-leave-to .profile-modal {
  transform: scale(0.95) translateY(10px);
  opacity: 0;
}

.profile-header {
  display: flex;
  align-items: center;
  gap: 18px;
  padding-bottom: 24px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  margin-bottom: 20px;
}

.profile-avatar {
  position: relative;
  width: 68px;
  height: 68px;
  border-radius: 50%;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.1);
  border: 2px solid rgba(255, 255, 255, 0.15);
  flex-shrink: 0;
  cursor: pointer;
}

.profile-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: 600;
  color: var(--accent-warm);
}

.avatar-edit-hint {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
  color: #fff;
  font-size: 18px;
  opacity: 0;
  transition: opacity 0.2s;
}

.profile-avatar:hover .avatar-edit-hint {
  opacity: 1;
}

/* Avatar Upload */
.avatar-upload-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.avatar-preview-small {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  flex-shrink: 0;
}

.avatar-preview-small img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder-small {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 600;
  color: var(--accent-warm);
}

.upload-btn {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 10px;
  color: var(--text-primary);
  font-size: 13px;
  cursor: pointer;
  transition: background 0.2s;
}

.upload-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.18);
}

.upload-btn:disabled {
  opacity: 0.5;
  cursor: default;
}

.upload-hint {
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0.6;
}

.profile-info {
  flex: 1;
  min-width: 0;
}

.nickname-row {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.nickname-row:hover .edit-icon {
  opacity: 1;
}

.profile-name {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.edit-icon {
  font-size: 14px;
  color: var(--text-secondary);
  opacity: 0;
  transition: opacity 0.2s;
  flex-shrink: 0;
}

.nickname-edit {
  display: flex;
}

.nickname-input {
  width: 100%;
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 10px;
  color: var(--text-primary);
  font-size: 18px;
  font-weight: 600;
  outline: none;
  font-family: inherit;
}

.nickname-input:focus {
  border-color: var(--accent-warm);
}

.profile-username {
  font-size: 13px;
  color: var(--text-secondary);
  margin-top: 3px;
}

.role-badge {
  display: inline-block;
  margin-top: 6px;
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.role-mom {
  background: rgba(255, 182, 193, 0.2);
  color: #ffd0d6;
}

.role-dad {
  background: rgba(135, 206, 235, 0.2);
  color: #b0e0f0;
}

.role-family {
  background: rgba(255, 218, 185, 0.2);
  color: #ffe8d4;
}

.role-professional {
  background: rgba(144, 238, 144, 0.2);
  color: #b8f0b8;
}

.role-admin {
  background: rgba(255, 215, 0, 0.2);
  color: #ffe566;
}

.inline-error {
  padding: 8px 14px;
  margin-bottom: 16px;
  background: rgba(255, 100, 100, 0.12);
  border-radius: 10px;
  font-size: 13px;
  color: #ffaaaa;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 2px;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 16px;
  overflow: hidden;
  margin-bottom: 22px;
}

.stat-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 16px 8px;
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
}

.stat-label {
  font-size: 11px;
  color: var(--text-secondary);
  letter-spacing: 0.5px;
}

.settings-section {
  margin-bottom: 28px;
}

.settings-heading {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 14px;
}

.settings-hint {
  margin-top: 10px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-secondary);
  opacity: 0.78;
}

.volume-control {
  display: flex;
  align-items: center;
  gap: 12px;
}

.volume-slider {
  flex: 1;
  accent-color: #ffb6c1;
  cursor: pointer;
}

.volume-value {
  min-width: 44px;
  text-align: right;
  font-size: 13px;
  color: var(--text-primary);
}

.password-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.profile-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-label {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: -6px;
}

.form-input {
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 12px;
  color: var(--text-primary);
  font-size: 15px;
  outline: none;
  font-family: inherit;
}

.form-input:focus {
  border-color: rgba(255, 255, 255, 0.3);
}

.form-input::placeholder {
  color: var(--text-secondary);
}

.form-error {
  padding: 8px 14px;
  background: rgba(255, 100, 100, 0.12);
  border-radius: 10px;
  font-size: 13px;
  color: #ffaaaa;
}

.form-success {
  padding: 8px 14px;
  background: rgba(72, 199, 142, 0.12);
  border-radius: 10px;
  font-size: 13px;
  color: #48c78e;
}

.submit-btn {
  padding: 12px;
  background: rgba(255, 255, 255, 0.12);
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 14px;
  color: var(--text-primary);
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.submit-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.18);
}

.submit-btn:disabled {
  opacity: 0.5;
  cursor: default;
}

.logout-btn {
  width: 100%;
  padding: 14px;
  background: rgba(255, 100, 100, 0.1);
  border: 1px solid rgba(255, 100, 100, 0.2);
  border-radius: 14px;
  color: #ffaaaa;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.logout-btn:hover {
  background: rgba(255, 100, 100, 0.18);
}

/* Role Selector */
.role-selector {
  display: flex;
  gap: 8px;
  margin-bottom: 14px;
}

.role-option {
  flex: 1;
  padding: 10px 0;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.role-option:hover {
  background: rgba(255, 255, 255, 0.1);
}

.role-option.active {
  background: rgba(255, 182, 193, 0.2);
  border-color: rgba(255, 182, 193, 0.4);
  color: var(--text-primary);
}

.role-option:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.bound-hint {
  font-size: 12px;
  color: rgba(180, 130, 80, 0.85);
  margin-bottom: 8px;
}

.shell-hint {
  font-size: 13px;
  color: var(--text-secondary);
  margin-bottom: 10px;
  line-height: 1.5;
}

.shell-code-display {
  font-family: inherit;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 4px;
  text-align: center;
  padding: 16px;
  margin-bottom: 10px;
  background: rgba(255, 182, 193, 0.1);
  border: 1px dashed rgba(255, 182, 193, 0.4);
  border-radius: 8px;
  color: var(--text-primary);
  user-select: all;
}

.partner-card {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 12px;
  margin-bottom: 10px;
  background: rgba(255, 182, 193, 0.08);
  border-radius: 8px;
  font-size: 14px;
}

.partner-label {
  color: var(--text-secondary);
}

.partner-name {
  font-weight: 600;
  color: var(--text-primary);
}

.partner-role {
  color: var(--text-secondary);
  font-size: 12px;
}

.danger-btn {
  background: rgba(220, 80, 80, 0.15) !important;
  color: #c44 !important;
}

.danger-btn:hover:not(:disabled) {
  background: rgba(220, 80, 80, 0.25) !important;
}

/* ── PearlShell ── */
.pearl-shell-embedded {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: min(936px, 100%);
  height: min(544px, calc(100% - 34px));
  margin-top: -17px;
  margin-left: 0px;
  border-radius: 2px;
  overflow: hidden;
}

.pearl-shell-close {
  position: absolute;
  top: 8px;
  left: 8px;
  z-index: 30;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 50%;
  color: #fff;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}

.pearl-shell-close:hover {
  background: rgba(0, 0, 0, 0.55);
}

.pearl-shell-fullscreen {
  position: fixed;
  inset: 0;
  z-index: 150;
  background: #d48a56;
}

.pearl-fullscreen-enter-active {
  transition: opacity 0.4s ease, transform 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

.pearl-fullscreen-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.pearl-fullscreen-enter-from,
.pearl-fullscreen-leave-to {
  opacity: 0;
  transform: scale(0.9);
}

/* ── Transitions ── */
.car-page-enter-active {
  transition: opacity 0.35s cubic-bezier(0.16, 1, 0.3, 1), transform 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

.car-page-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.car-page-enter-from {
  opacity: 0;
  transform: scale(0.96);
}

.car-page-leave-to {
  opacity: 0;
  transform: scale(0.97);
}

/* macOS-like zoom animation */
.modal-zoom-enter-active {
  transition: opacity 0.35s ease;
}

.modal-zoom-enter-active .detail-modal {
  transition: transform 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

.modal-zoom-leave-active {
  transition: opacity 0.25s ease;
}

.modal-zoom-leave-active .detail-modal {
  transition: transform 0.25s ease;
}

.modal-zoom-enter-from {
  opacity: 0;
}

.modal-zoom-enter-from .detail-modal {
  transform: scale(0.85) translateY(20px);
}

.modal-zoom-leave-to {
  opacity: 0;
}

.modal-zoom-leave-to .detail-modal {
  transform: scale(0.95) translateY(10px);
}

/* ── Mobile profile button ── */
.mobile-profile-btn {
  display: none;
}

.mobile-suitcase-btn {
  display: none;
}

/* ── Mobile ── */
@media (max-width: 768px) {
  .mobile-profile-btn {
    display: flex;
    position: absolute;
    top: 16px;
    left: 16px;
    z-index: 10;
    width: 44px;
    height: 44px;
    padding: 0;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.1);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 50%;
    cursor: pointer;
    transition: background 0.2s, transform 0.15s;
    overflow: hidden;
  }

  .mobile-profile-btn:hover {
    background: rgba(255, 255, 255, 0.2);
    transform: scale(1.08);
  }

  .mobile-profile-avatar {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: 50%;
  }

  .mobile-suitcase-btn {
    display: flex;
    position: absolute;
    bottom: 24px;
    right: 16px;
    z-index: 10;
    width: 48px;
    height: 48px;
    padding: 0;
    align-items: center;
    justify-content: center;
    background: rgba(138, 106, 74, 0.85);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    border: 1px solid rgba(200, 160, 108, 0.5);
    border-radius: 50%;
    color: #fff;
    cursor: pointer;
    transition: background 0.2s, transform 0.15s;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.2);
  }

  .mobile-suitcase-btn:hover {
    background: rgba(138, 106, 74, 1);
    transform: scale(1.08);
  }

  .mobile-suitcase-badge {
    position: absolute;
    top: -4px;
    right: -4px;
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    background: #e74c5e;
    border-radius: 9px;
    color: #fff;
    font-size: 11px;
    font-weight: 700;
    line-height: 18px;
    text-align: center;
  }

  .close-btn {
    width: 44px;
    height: 44px;
  }

  .car-layout {
    flex-direction: column;
    padding: 0;
    gap: 0;
    height: 100%;
    align-items: center;
    justify-content: center;
    overflow-y: auto;
  }

  .right-section {
    display: none;
  }

  .photo-wall {
    flex: none;
    width: 94vw;
    max-width: none;
    height: calc(100vh - 120px);
    height: calc(100dvh - 120px);
    padding: 0;
    display: block;
    align-self: center;
    background: url('@/assets/images/board_med.png') repeat center / cover;
    border: 8px solid #c8a06c;
    border-radius: 6px;
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.3),
                inset 0 0 12px rgba(0, 0, 0, 0.06);
    overflow-y: auto;
    overflow-x: hidden;
  }

  .pearl-shell-trigger-zone {
    display: none;
  }

  .pearl-shell-embedded {
    display: none;
  }

  .photo-grid {
    position: relative;
    z-index: auto;
    grid-template-columns: repeat(3, 1fr);
    max-width: 100%;
    width: 100%;
    gap: 10px;
    padding: 16px;
    pointer-events: auto;
    box-sizing: border-box;
  }

  .photo-frame {
    transform: none !important;
    filter: brightness(1);
    opacity: 1;
    border-radius: 8px;
    overflow: hidden;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
    background: rgba(255, 255, 255, 0.15);
    border-color: rgba(255, 255, 255, 0.3);
  }

  .photo-frame:hover:not(.empty) {
    transform: none !important;
  }

  .photo-frame.empty {
    display: none;
  }

  .suitcase-modal {
    width: 95vw;
    max-height: 90dvh;
    max-height: 90vh;
    padding: 20px;
  }

  .suitcase-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .detail-modal {
    width: 92vw;
    max-height: 88dvh;
    max-height: 88vh;
  }

  .detail-image {
    max-height: 40vh;
  }

  .profile-modal {
    width: 100vw;
    max-width: 100vw;
    max-height: 100dvh;
    max-height: 100vh;
    border-radius: 0;
    padding: 24px 16px;
  }

  .modal-content {
    width: 100vw;
    max-height: 100dvh;
    max-height: 100vh;
    border-radius: 0;
  }
}

@media (max-width: 480px) {
  .photo-wall {
    width: 96vw;
  }

  .photo-grid {
    gap: 8px;
    padding: 12px;
  }
}
</style>
