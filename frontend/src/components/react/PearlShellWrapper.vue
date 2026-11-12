<template>
  <div ref="containerRef" class="pearl-shell-container" />
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { createRoot, type Root } from 'react-dom/client'
import { createElement } from 'react'
import PearlShell, { type PearlShellProps } from './PearlShell'

const props = defineProps<{
  photoUrls: string[]
  isFullscreen: boolean
  showDebug?: boolean
}>()

const emit = defineEmits<{
  requestFullscreen: []
  exitFullscreen: []
  unavailable: []
}>()

const containerRef = ref<HTMLDivElement | null>(null)
let root: Root | null = null
const reportUnavailable = () => emit('unavailable')

function renderReact() {
  if (!root) return
  root.render(
    createElement(PearlShell, {
      photoUrls: props.photoUrls,
      isFullscreen: props.isFullscreen,
      showDebug: props.showDebug ?? false,
      onRequestFullscreen: () => emit('requestFullscreen'),
      onExitFullscreen: () => emit('exitFullscreen'),
      onUnavailable: reportUnavailable,
    } satisfies PearlShellProps),
  )
}

onMounted(() => {
  if (containerRef.value) {
    root = createRoot(containerRef.value)
    renderReact()
  }
})

watch(
  () => [props.photoUrls, props.isFullscreen, props.showDebug],
  renderReact,
  { deep: true },
)

onBeforeUnmount(() => {
  root?.unmount()
  root = null
})
</script>

<style scoped>
.pearl-shell-container {
  width: 100%;
  height: 100%;
}
</style>
