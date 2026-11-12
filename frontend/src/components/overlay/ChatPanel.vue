<template>
  <OverlayPanel :visible="uiStore.activePanel === 'chat'" position="right" @close="uiStore.closePanel()">
    <div class="chat-panel" :style="ambientStyle">
      <!-- Ambient background layer -->
      <div class="ambient-bg" :style="ambientGradientStyle" />

      <!-- Ripple on send -->
      <Transition name="ripple-fx">
        <div v-if="showRipple" class="send-ripple" />
      </Transition>

      <div class="chat-header">
        <button v-if="authStore.isAuthenticated" class="memory-btn" @click="uiStore.openPanel('ai-memory')">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
            <path d="M12 3C6.5 3 2 6.58 2 11c0 2.42 1.34 4.58 3.43 6.04L4 21l4.53-2.21C9.62 19.26 10.78 19.5 12 19.5c5.5 0 10-3.58 10-8S17.5 3 12 3z" fill="currentColor" />
          </svg>
        </button>
      </div>

      <Transition name="toast">
        <div v-if="showMemoryToast" class="memory-toast">✨ 记住了</div>
      </Transition>

      <!-- Messages -->
      <div class="messages" ref="messagesEl">
        <Transition name="msg-replace" mode="out-in">
          <div v-if="messages.length === 0 && !sending" key="empty" class="chat-empty">
            <p class="empty-greeting" v-if="preferredName">{{ preferredName }}，你好</p>
            <p class="empty-greeting" v-else>你好</p>
            <p class="empty-subtitle">我是小石光，有什么想聊的吗？</p>
          </div>
          <div v-else-if="sending" key="loading" class="msg-center">
            <p v-if="latestUserMsg" class="msg-text user-text">{{ latestUserMsg.text }}</p>
            <div class="typing-dots">
              <span /><span /><span />
            </div>
          </div>
          <div v-else-if="latestAssistantMsg" :key="latestAssistantMsg.id" class="msg-center">
            <p v-if="latestUserMsg" class="msg-text user-text">{{ latestUserMsg.text }}</p>
            <p
              :class="[
                'msg-text assistant-text',
                latestAssistantMsg.showEffect && latestAssistantMsg.visualMeta
                  ? `effect-${latestAssistantMsg.visualMeta.effect_type}` : '',
              ]"
              :style="getBubbleStyle(latestAssistantMsg)"
            >
              {{ displayedAssistantText }}<span v-if="!typingComplete" class="typing-cursor" />
            </p>
          </div>
        </Transition>
      </div>

      <p v-if="sendError" class="send-error" role="alert">{{ sendError }}</p>
      <form class="chat-input-area" @submit.prevent="onSend">
        <input
          v-model="input"
          type="text"
          class="chat-input"
          placeholder="说说你的心情..."
          :disabled="sending"
        />
        <button type="submit" class="send-btn" :disabled="!input.trim() || sending">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M12 19V5M5 12l7-7 7 7" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" fill="none" />
          </svg>
        </button>
      </form>
    </div>
  </OverlayPanel>
</template>

<script lang="ts">
interface ChatMessage {
  id: number
  role: 'user' | 'assistant'
  text: string
  visualMeta?: import('@/lib/api/chat').VisualMetadata
  showEffect: boolean
}

const _messages: ChatMessage[] = []
let _sessionId: string | null = null
let _sessionCounter = 0
let _profileLoaded = false
let _preferredName: string | null = null
</script>

<script setup lang="ts">
import { ref, watch, nextTick, computed } from 'vue'
import OverlayPanel from './OverlayPanel.vue'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { getErrorMessage } from '@/lib/apiClient'
import {
  sendChatMessage,
  getChatProfile,
  type ChatResponse,
  type ChatProfile,
} from '@/lib/api/chat'

const uiStore = useUiStore()
const authStore = useAuthStore()

const messages = ref<ChatMessage[]>(_messages.map(m => ({ ...m, showEffect: false })))
const input = ref('')
const sending = ref(false)
const sendError = ref('')
const messagesEl = ref<HTMLElement | null>(null)
const preferredName = ref<string | null>(_preferredName)
const showMemoryToast = ref(false)
const showRipple = ref(false)
const typedLength = ref(0)
const typingComplete = ref(true)
let typingTimer: ReturnType<typeof setInterval> | null = null

function startTypewriter(text: string) {
  typedLength.value = 0
  typingComplete.value = false
  if (typingTimer) clearInterval(typingTimer)
  typingTimer = setInterval(() => {
    typedLength.value++
    if (typedLength.value >= text.length) {
      typingComplete.value = true
      clearInterval(typingTimer!)
      typingTimer = null
    }
  }, 30)
}

const isOpen = computed(() => uiStore.activePanel === 'chat')

const colorToneMap: Record<string, string> = {
  soft_pink: 'rgba(255,182,193,0.15)',
  warm_gold: 'rgba(255,215,0,0.15)',
  gentle_blue: 'rgba(135,206,235,0.12)',
  lavender: 'rgba(200,162,255,0.15)',
  neutral_white: 'rgba(255,255,255,0.1)',
  coral: 'rgba(255,127,80,0.15)',
  sage: 'rgba(143,188,143,0.15)',
}

const ambientColorMap: Record<string, { primary: string; secondary: string }> = {
  soft_pink: { primary: 'rgba(255,182,193,0.25)', secondary: 'rgba(255,228,225,0.1)' },
  warm_gold: { primary: 'rgba(255,215,0,0.2)', secondary: 'rgba(255,248,220,0.08)' },
  gentle_blue: { primary: 'rgba(135,206,235,0.2)', secondary: 'rgba(176,224,230,0.08)' },
  lavender: { primary: 'rgba(200,162,255,0.2)', secondary: 'rgba(230,230,250,0.08)' },
  neutral_white: { primary: 'rgba(255,255,255,0.12)', secondary: 'rgba(255,255,255,0.04)' },
  coral: { primary: 'rgba(255,127,80,0.2)', secondary: 'rgba(255,160,122,0.08)' },
  sage: { primary: 'rgba(143,188,143,0.2)', secondary: 'rgba(193,225,193,0.08)' },
}

const focusColorMap: Record<string, { border: string; glow: string }> = {
  soft_pink:     { border: 'rgba(255,182,193,0.35)', glow: 'rgba(255,182,193,0.10)' },
  warm_gold:     { border: 'rgba(255,215,0,0.35)',   glow: 'rgba(255,215,0,0.10)' },
  gentle_blue:   { border: 'rgba(135,206,235,0.35)', glow: 'rgba(135,206,235,0.10)' },
  lavender:      { border: 'rgba(200,162,255,0.35)', glow: 'rgba(200,162,255,0.10)' },
  neutral_white: { border: 'rgba(255,255,255,0.25)', glow: 'rgba(255,255,255,0.06)' },
  coral:         { border: 'rgba(255,127,80,0.35)',   glow: 'rgba(255,127,80,0.10)' },
  sage:          { border: 'rgba(143,188,143,0.35)', glow: 'rgba(143,188,143,0.10)' },
}

const latestColorTone = computed(() => {
  for (let i = messages.value.length - 1; i >= 0; i--) {
    const m = messages.value[i]
    if (m.role === 'assistant' && m.visualMeta?.color_tone) return m.visualMeta.color_tone
  }
  return 'gentle_blue'
})

const ambientStyle = computed(() => ({
  '--ambient-primary': ambientColorMap[latestColorTone.value]?.primary ?? ambientColorMap.gentle_blue.primary,
  '--ambient-secondary': ambientColorMap[latestColorTone.value]?.secondary ?? ambientColorMap.gentle_blue.secondary,
  '--focus-border': focusColorMap[latestColorTone.value]?.border ?? focusColorMap.gentle_blue.border,
  '--focus-glow': focusColorMap[latestColorTone.value]?.glow ?? focusColorMap.gentle_blue.glow,
}))

const ambientGradientStyle = computed(() => ({
  background: `radial-gradient(ellipse at 30% 40%, var(--ambient-primary), var(--ambient-secondary), transparent)`,
}))

const latestUserMsg = computed(() => {
  for (let i = messages.value.length - 1; i >= 0; i--) {
    if (messages.value[i].role === 'user') return messages.value[i]
  }
  return null
})

const latestAssistantMsg = computed(() => {
  for (let i = messages.value.length - 1; i >= 0; i--) {
    if (messages.value[i].role === 'assistant') return messages.value[i]
  }
  return null
})

const displayedAssistantText = computed(() => {
  if (!latestAssistantMsg.value) return ''
  if (typingComplete.value) return latestAssistantMsg.value.text
  return latestAssistantMsg.value.text.slice(0, typedLength.value)
})

function generateSessionId(): string {
  // crypto.randomUUID() requires a secure context (HTTPS or localhost).
  // Fall back to crypto.getRandomValues for non-secure HTTP environments.
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  // crypto.getRandomValues is available in all modern browsers;
  // guard retained for SSR / test environments where crypto may be absent.
  if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
    const bytes = new Uint8Array(16)
    crypto.getRandomValues(bytes)
    return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
  }
  // Fallback: timestamp + counter (non-crypto, but unique per session)
  return Date.now().toString(36) + (++_sessionCounter).toString(36)
}

function syncPersistent() {
  _messages.length = 0
  _messages.push(...messages.value.map(m => ({ ...m, showEffect: false })))
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesEl.value) {
      messagesEl.value.scrollTop = messagesEl.value.scrollHeight
    }
  })
}

function getBubbleStyle(msg: ChatMessage): Record<string, string> {
  if (msg.role !== 'assistant' || !msg.visualMeta) return {}
  const color = colorToneMap[msg.visualMeta.color_tone] ?? colorToneMap.neutral_white
  if (msg.showEffect) {
    return {
      '--effect-color': color,
      '--effect-intensity': String(0.5 + msg.visualMeta.intensity * 0.5),
    }
  }
  return {}
}

async function loadProfile() {
  if (_profileLoaded) {
    preferredName.value = _preferredName
    return
  }
  try {
    const profile: ChatProfile = await getChatProfile()
    _preferredName = profile.preferred_name
    _profileLoaded = true
    preferredName.value = _preferredName
  } catch {
    _profileLoaded = true
  }
}

watch(isOpen, (open) => {
  if (open) {
    scrollToBottom()
    if (authStore.isAuthenticated && !_profileLoaded) {
      loadProfile()
    }
  }
})

async function onSend() {
  const text = input.value.trim()
  if (!text || sending.value) return

  if (!_sessionId) {
    _sessionId = generateSessionId()
  }

  messages.value = [
    ...messages.value,
    { id: Date.now(), role: 'user', text, showEffect: false },
  ]
  input.value = ''
  sendError.value = ''
  sending.value = true
  showRipple.value = true
  setTimeout(() => { showRipple.value = false }, 800)
  syncPersistent()
  scrollToBottom()

  try {
    const res: ChatResponse = await sendChatMessage({
      content: text,
      session_id: _sessionId,
    })
    const hasVisual = !!res.visual_metadata
    const msgId = Date.now()
    const assistantMsg: ChatMessage = {
      id: msgId,
      role: 'assistant',
      text: res.text,
      visualMeta: res.visual_metadata,
      showEffect: false,
    }
    messages.value = [...messages.value, assistantMsg]
    startTypewriter(res.text)

    if (hasVisual) {
      const effectDelay = res.text.length * 30 + 200
      setTimeout(() => {
        const idx = messages.value.findIndex(m => m.id === msgId)
        if (idx !== -1) {
          messages.value[idx] = { ...messages.value[idx], showEffect: true }
        }
      }, effectDelay)
      setTimeout(() => {
        const idx = messages.value.findIndex(m => m.id === msgId)
        if (idx !== -1) {
          messages.value[idx] = { ...messages.value[idx], showEffect: false }
        }
        syncPersistent()
      }, effectDelay + 3000)
    }

    if (res.memory_updated) {
      showMemoryToast.value = true
      setTimeout(() => {
        showMemoryToast.value = false
      }, 2000)
    }
  } catch (err) {
    messages.value = messages.value.slice(0, -1)
    input.value = text
    sendError.value = getErrorMessage(err)
  } finally {
    sending.value = false
    syncPersistent()
    scrollToBottom()
  }
}
</script>

<style scoped>
.send-error {
  margin: 0 16px 8px;
  color: #ffcebf;
  font-size: 0.875rem;
  text-align: center;
}

.chat-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-height: 90vh;
  overflow: hidden;
  position: relative;
}

/* ── Ambient background ── */
.ambient-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  transition: background 2s ease-in-out;
  animation: ambientBreath 4s ease-in-out infinite;
}

@keyframes ambientBreath {
  0%, 100% { opacity: 0.6; }
  50% { opacity: 1; }
}

/* ── Send ripple ── */
.send-ripple {
  position: absolute;
  left: 50%;
  bottom: 80px;
  width: 0;
  height: 0;
  border-radius: 50%;
  border: 2px solid var(--ambient-primary, rgba(135, 206, 235, 0.3));
  transform: translate(-50%, 50%);
  pointer-events: none;
  z-index: 1;
}

.ripple-fx-enter-active {
  animation: rippleExpand 0.8s ease-out forwards;
}
.ripple-fx-leave-active {
  opacity: 0;
  transition: opacity 0.1s;
}

@keyframes rippleExpand {
  0% { width: 0; height: 0; opacity: 0.8; }
  100% { width: 500px; height: 500px; opacity: 0; }
}

/* ── Header ── */
.chat-header {
  display: flex;
  align-items: center;
  padding: 24px 56px 16px 24px;
  flex-shrink: 0;
  gap: 12px;
  position: relative;
  z-index: 2;
}

.memory-btn {
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
  color: var(--text-primary);
  cursor: pointer;
  transition: background 0.2s, transform 0.15s;
}

.memory-btn:hover {
  background: rgba(255, 255, 255, 0.2);
  transform: scale(1.08);
}

/* ── Memory toast ── */
.memory-toast {
  position: absolute;
  top: 24px;
  left: 50%;
  transform: translateX(-50%);
  padding: 6px 16px;
  background: rgba(255, 215, 0, 0.12);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 215, 0, 0.18);
  border-radius: 16px;
  color: var(--accent-warm);
  font-size: 13px;
  letter-spacing: 0.3px;
  z-index: 5;
  pointer-events: none;
  white-space: nowrap;
}

.toast-enter-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}
.toast-leave-active {
  transition: opacity 0.6s ease, transform 0.6s ease;
}
.toast-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(-6px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-4px);
}

/* ── Messages ── */
.messages {
  flex: 1;
  overflow-y: auto;
  padding: 0 32px 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  overscroll-behavior: contain;
  position: relative;
  z-index: 2;
}

.msg-center {
  text-align: center;
  width: 100%;
  max-width: 560px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

/* ── Empty state ── */
.chat-empty {
  text-align: center;
  color: var(--text-secondary);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.empty-greeting {
  font-size: 18px;
  font-weight: 300;
  color: var(--text-primary);
  opacity: 0.8;
}

.empty-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
  opacity: 0.6;
}

/* ── Replace transition ── */
.msg-replace-enter-active {
  transition: opacity 0.6s ease, transform 0.6s ease;
}
.msg-replace-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}
.msg-replace-enter-from {
  opacity: 0;
  transform: translateY(12px);
}
.msg-replace-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* ── Text styles ── */
.msg-text {
  word-break: break-word;
}

.user-text {
  font-size: 14px;
  color: var(--text-secondary);
  opacity: 0.5;
  font-weight: 400;
  line-height: 1.6;
}

.assistant-text {
  font-size: 20px;
  color: var(--text-primary);
  font-weight: 300;
  line-height: 1.8;
  max-width: 480px;
  transition: text-shadow 0.6s ease;
}

/* ── Visual effects ── */
.effect-ripple { animation: effectRipple 3s ease-out forwards; }
.effect-sunlight { animation: effectSunlight 3s ease-out forwards; }
.effect-calm { animation: effectCalm 3s ease-in-out forwards; }
.effect-warm_glow { animation: effectWarmGlow 3s ease-out forwards; }
.effect-gentle_wave { animation: effectGentleWave 3s ease-in-out forwards; }

@keyframes effectRipple {
  0% { text-shadow: 0 0 0 var(--effect-color, rgba(255, 255, 255, 0.1)); }
  30% { text-shadow: 0 0 20px var(--effect-color, rgba(255, 255, 255, 0.3)); }
  100% { text-shadow: 0 0 0 transparent; }
}
@keyframes effectSunlight {
  0% { text-shadow: 4px 0 24px var(--effect-color, rgba(255, 215, 0, 0.3)); }
  100% { text-shadow: 0 0 0 transparent; }
}
@keyframes effectCalm {
  0% { transform: scale(1); text-shadow: 0 0 10px var(--effect-color, rgba(255, 255, 255, 0.2)); }
  50% { transform: scale(1.01); text-shadow: 0 0 18px var(--effect-color, rgba(255, 255, 255, 0.3)); }
  100% { transform: scale(1); text-shadow: 0 0 0 transparent; }
}
@keyframes effectWarmGlow {
  0% { text-shadow: 0 0 24px var(--effect-color, rgba(255, 215, 0, 0.3)); }
  100% { text-shadow: 0 0 0 transparent; }
}
@keyframes effectGentleWave {
  0%, 100% { transform: translateY(0); }
  25% { transform: translateY(-2px); }
  75% { transform: translateY(2px); }
}

/* ── Typing indicator ── */
.typing-dots {
  display: flex;
  gap: 6px;
  justify-content: center;
  padding: 8px 0;
}

.typing-dots span {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-secondary);
  animation: typingDot 1.2s infinite ease-in-out;
}

.typing-dots span:nth-child(2) { animation-delay: 0.15s; }
.typing-dots span:nth-child(3) { animation-delay: 0.3s; }

@keyframes typingDot {
  0%, 60%, 100% { opacity: 0.3; transform: translateY(0); }
  30% { opacity: 1; transform: translateY(-6px); }
}

/* ── Input area ── */
.chat-input-area {
  display: flex;
  gap: 10px;
  padding: 0 24px 24px;
  flex-shrink: 0;
  position: relative;
  z-index: 2;
}

.chat-input {
  flex: 1;
  padding: 14px 20px;
  background: rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 24px;
  color: var(--text-primary);
  font-size: 15px;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.3s;
}

.chat-input:focus {
  border-color: var(--focus-border, rgba(255, 255, 255, 0.25));
  box-shadow: 0 0 16px var(--focus-glow, rgba(255, 255, 255, 0.06));
}

.chat-input::placeholder {
  color: var(--text-secondary);
}

.send-btn {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--accent-warm);
  border: none;
  border-radius: 50%;
  color: #fff;
  cursor: pointer;
  transition: background 0.2s, transform 0.15s, box-shadow 0.2s;
  flex-shrink: 0;
  box-shadow: 0 2px 12px rgba(255, 170, 85, 0.2);
}

.send-btn:hover {
  background: var(--accent-warm-hover);
  box-shadow: 0 4px 20px rgba(255, 170, 85, 0.3);
}
.send-btn:active { transform: scale(0.93); }
.send-btn:disabled { opacity: 0.4; cursor: not-allowed; box-shadow: none; }

/* ── Typing cursor ── */
.typing-cursor {
  display: inline-block;
  width: 2px;
  height: 1em;
  background: var(--text-primary);
  margin-left: 2px;
  vertical-align: text-bottom;
  animation: cursorBlink 0.8s steps(2) infinite;
}

@keyframes cursorBlink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* ── Mobile ── */
@media (max-width: 768px) {
  .chat-header {
    padding: 16px 48px 12px 16px;
  }

  .messages {
    padding: 0 16px 12px;
  }

  .msg-center {
    max-width: 100%;
  }

  .assistant-text {
    max-width: 100%;
  }

  .chat-input-area {
    padding: 0 16px 16px;
  }

  .chat-input {
    padding: 12px 16px;
    font-size: 16px;
  }

  .memory-btn {
    width: 44px;
    height: 44px;
  }
}
</style>
