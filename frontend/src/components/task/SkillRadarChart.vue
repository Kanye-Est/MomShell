<template>
  <div class="radar-wrap" role="img" aria-label="六维战力雷达">
    <svg class="radar" viewBox="0 0 200 200">
      <!-- Rings -->
      <polygon
        v-for="(p, idx) in ringPolygons"
        :key="idx"
        :points="p"
        class="ring"
        :class="`ring-${idx + 1}`"
      />

      <!-- Axes -->
      <line
        v-for="(a, idx) in axes"
        :key="a.key"
        class="axis"
        x1="100"
        y1="100"
        :x2="axisPoints[idx].x"
        :y2="axisPoints[idx].y"
      />

      <!-- Value polygon -->
      <polygon :points="valuePolygon" class="value-fill" />
      <polyline :points="valuePolygon" class="value-stroke" />

      <!-- Labels -->
      <g class="labels">
        <text
          v-for="(a, idx) in axes"
          :key="a.key"
          class="label"
          :x="labelPoints[idx].x"
          :y="labelPoints[idx].y"
          text-anchor="middle"
          dominant-baseline="middle"
        >
          {{ a.label }}
        </text>
      </g>
    </svg>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface SkillRadarValues {
  nutrition: number
  cleaning: number
  emotional: number
  logistics: number
  health: number
  playtime: number
}

const props = defineProps<{
  values: SkillRadarValues
}>()

const axes = [
  { key: 'nutrition', label: '营养' },
  { key: 'cleaning', label: '清洁' },
  { key: 'logistics', label: '后勤' },
  { key: 'health', label: '健康' },
  { key: 'playtime', label: '陪伴' },
  { key: 'emotional', label: '安抚' },
] as const

const valueList = computed(() => axes.map((a) => Math.max(0, props.values[a.key])))
const maxValue = computed(() => Math.max(10, ...valueList.value))

function polarPoint(angle: number, r: number) {
  return { x: 100 + Math.cos(angle) * r, y: 100 + Math.sin(angle) * r }
}

const baseAngles = computed(() => {
  // Start at top, clockwise
  const start = -Math.PI / 2
  const step = (Math.PI * 2) / axes.length
  return axes.map((_, i) => start + step * i)
})

const axisPoints = computed(() => baseAngles.value.map((a) => polarPoint(a, 78)))
const labelPoints = computed(() => baseAngles.value.map((a) => polarPoint(a, 94)))

function toPolygon(points: Array<{ x: number; y: number }>): string {
  return points.map((p) => `${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
}

const ringPolygons = computed(() => {
  const radii = [26, 52, 78]
  return radii.map((r) => toPolygon(baseAngles.value.map((a) => polarPoint(a, r))))
})

const valuePolygon = computed(() => {
  const points = baseAngles.value.map((angle, i) => {
    const v = valueList.value[i]
    const r = (v / maxValue.value) * 78
    return polarPoint(angle, r)
  })
  return toPolygon(points)
})
</script>

<style scoped>
.radar-wrap {
  width: 100%;
  max-width: 520px;
  margin: 0 auto;
  padding: 14px;
  border-radius: 18px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.05);
}

.radar {
  width: 100%;
  height: auto;
  display: block;
}

.ring {
  fill: transparent;
  stroke: rgba(255, 255, 255, 0.1);
  stroke-width: 1;
}

.ring-3 {
  stroke: rgba(255, 255, 255, 0.14);
}

.axis {
  stroke: rgba(255, 255, 255, 0.12);
  stroke-width: 1;
}

.value-fill {
  fill: rgba(255, 200, 80, 0.18);
  stroke: none;
}

.value-stroke {
  fill: transparent;
  stroke: rgba(255, 200, 80, 0.85);
  stroke-width: 2;
}

.label {
  font-size: 11px;
  font-weight: 700;
  fill: rgba(255, 255, 255, 0.75);
  letter-spacing: 1px;
}
</style>
