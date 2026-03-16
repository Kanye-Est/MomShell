import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useAuthStore } from './auth'

export type PanelName = 'auth' | 'role' | 'memory' | 'community' | 'chat' | 'car' | 'bar' | 'whisper' | 'task' | 'shell-gift' | 'ai-memory' | null
export type AuthMode = 'login' | 'register' | 'guest'

export const useUiStore = defineStore('ui', () => {
  const activePanel = ref<PanelName>(null)
  const authMode = ref<AuthMode>('login')
  const showLanding = ref(true)

  // Parallax scrollTo bridge: set by BeachScene, called by tutorial
  let _parallaxScrollTo: ((offset: number) => void) | null = null

  function setParallaxScrollTo(fn: (offset: number) => void) {
    _parallaxScrollTo = fn
  }

  function parallaxScrollTo(offset: number) {
    _parallaxScrollTo?.(offset)
  }

  function openPanel(panel: PanelName) {
    activePanel.value = panel
  }

  function closePanel() {
    activePanel.value = null
  }

  function closeLanding() {
    showLanding.value = false
  }

  function openAuth(mode: AuthMode = 'login') {
    authMode.value = mode
    activePanel.value = 'auth'
  }

  /** Open a feature panel, but intercept if guest → force auth */
  function openFeature(panel: 'car' | 'memory' | 'community' | 'chat' | 'bar' | 'whisper' | 'task' | 'shell-gift') {
    const auth = useAuthStore()
    if (!auth.isAuthenticated && !auth.isGuest) {
      openAuth()
      return
    }
    if (auth.isGuest) {
      openAuth('register')
      return
    }
    activePanel.value = panel
  }

  return {
    activePanel,
    authMode,
    showLanding,
    openPanel,
    closePanel,
    closeLanding,
    openAuth,
    openFeature,
    setParallaxScrollTo,
    parallaxScrollTo,
  }
})
