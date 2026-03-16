<template>
  <BeachScene />
  <NavBar />
  <LandingOverlay />
  <AuthPanel />
  <RoleSelectPanel />
  <CarPage />
  <CommunityPanel />
  <BarPage />
  <ChatPanel />
  <MemoryPanel />
  <AiMemoryPanel />
  <WhisperPanel />
  <TaskPanel />
  <ShellGiftPanel />
</template>

<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useBackgroundMusicLoop } from '@/composables/useBackgroundMusicLoop'
import BeachScene from '@/components/scene/BeachScene.vue'
import NavBar from '@/components/scene/NavBar.vue'
import LandingOverlay from '@/components/overlay/LandingOverlay.vue'
import AuthPanel from '@/components/overlay/AuthPanel.vue'
import RoleSelectPanel from '@/components/overlay/RoleSelectPanel.vue'
import CarPage from '@/components/overlay/CarPage.vue'
import CommunityPanel from '@/components/overlay/CommunityPanel.vue'
import BarPage from '@/components/overlay/BarPage.vue'
import ChatPanel from '@/components/overlay/ChatPanel.vue'
import MemoryPanel from '@/components/overlay/MemoryPanel.vue'
import AiMemoryPanel from '@/components/overlay/AiMemoryPanel.vue'
import WhisperPanel from '@/components/overlay/WhisperPanel.vue'
import TaskPanel from '@/components/overlay/TaskPanel.vue'
import ShellGiftPanel from '@/components/overlay/ShellGiftPanel.vue'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { useTutorial } from '@/composables/useTutorial'

const authStore = useAuthStore()
const uiStore = useUiStore()
const { startTutorial, stopTutorial, cancelPending } = useTutorial()

useBackgroundMusicLoop()

onMounted(async () => {
  await authStore.init()
  if (authStore.isAuthenticated) {
    uiStore.closeLanding()
  }
})

// Trigger tutorial when landing closes + user is authenticated + no panel open
watch(
  () => ({
    landing: uiStore.showLanding,
    auth: authStore.isAuthenticated || authStore.isGuest,
    panel: uiStore.activePanel,
  }),
  (state) => {
    if (!state.landing && state.auth && !state.panel) {
      setTimeout(() => {
        startTutorial()
      }, 1000)
    }
  },
  { deep: true, immediate: true }
)

// If a panel opens while the tutorial is pending or active, stop/cancel it
watch(
  () => uiStore.activePanel,
  (panel) => {
    if (panel) {
      cancelPending()
      stopTutorial()
    }
  }
)


</script>
