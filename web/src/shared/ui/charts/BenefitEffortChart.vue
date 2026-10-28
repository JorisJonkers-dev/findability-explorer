<script setup lang="ts">
import { computed } from 'vue'

/** One recommendation as a dot: benefit and effort run 1 to 5. */
export type BenefitEffortDot = { id: string; rank: number; benefit: number; effort: number; tip: string }

const props = defineProps<{ dots: readonly BenefitEffortDot[]; picked?: string }>()
const emit = defineEmits<{ pick: [id: string] }>()

const placed = computed(() =>
  props.dots.map((d) => {
    const same = props.dots.filter((o) => o.benefit === d.benefit && o.effort === d.effort)
    const nudge = (same.indexOf(d) - (same.length - 1) / 2) * 7
    return { ...d, x: ((d.effort - 1) / 4) * 100 + nudge, y: ((5 - d.benefit) / 4) * 100 }
  }),
)
</script>

<template>
  <div class="grid grid-cols-[22px_minmax(0,1fr)] gap-2">
    <div aria-hidden="true" class="rotate-180 text-center text-[13px] font-semibold text-muted [writing-mode:vertical-rl]">More benefit →</div>
    <div class="flex flex-col gap-1.5">
      <div class="relative grid h-[340px] grid-cols-2 grid-rows-2 overflow-hidden rounded-2xl" role="group" aria-label="Benefit against effort">
        <div class="bg-orange-pale p-3 text-[13px] font-bold text-orange-ink">Quick wins</div>
        <div class="bg-mist p-3 text-right text-[13px] font-bold text-muted">Big projects</div>
        <div class="flex items-end bg-mist p-3 text-[13px] font-bold text-muted">Small extras</div>
        <div class="flex items-end justify-end bg-white p-3 text-[13px] font-bold text-muted">Probably later</div>
        <!-- Dots live in an inner box that stays clear of the corner labels. -->
        <div class="pointer-events-none absolute inset-x-8 inset-y-16">
        <button
          v-for="d in placed"
          :key="d.id"
          type="button"
          :title="d.tip"
          :aria-label="d.tip"
          :aria-pressed="picked === d.id"
          class="pointer-events-auto absolute size-10 -translate-x-1/2 -translate-y-1/2 cursor-pointer rounded-full border-solid p-0 text-[15px] font-extrabold text-black transition-transform hover:scale-110"
          :class="picked === d.id ? 'border-[3px] border-black bg-orange shadow-[0_0_0_4px_#fff]' : 'border-2 border-black bg-white'"
          :style="{ left: `${String(d.x)}%`, top: `${String(d.y)}%` }"
          @click="emit('pick', d.id)"
        >
          {{ d.rank }}
        </button>
        </div>
      </div>
      <div aria-hidden="true" class="text-center text-[13px] font-semibold text-muted">More work →</div>
    </div>
  </div>
</template>
