<script setup lang="ts">
import { computed } from 'vue'

/** One bubble: the recommendations with the same benefit and effort, each from 1 to 5, and how many. */
export type BenefitEffortBubble = { key: string; benefit: number; effort: number; count: number; label: string }

const props = defineProps<{ bubbles: readonly BenefitEffortBubble[]; picked?: string }>()
const emit = defineEmits<{ pick: [key: string] }>()

const SCORES = [1, 2, 3, 4, 5] as const

/** Five rows, the most benefit at the top, each of five cells from the least work; a cell holds its bubble, if any. */
const rows = computed(() => {
  const most = Math.max(1, ...props.bubbles.map((b) => b.count))
  return [...SCORES].reverse().map((benefit) => ({
    benefit,
    cells: SCORES.map((effort) => {
      const bubble = props.bubbles.find((b) => b.benefit === benefit && b.effort === effort)
      // A bubble grows with its count, from 1.75 to 3.25 rem across.
      const size = bubble ? `${String(1.75 + 1.5 * Math.sqrt(bubble.count / most))}rem` : ''
      // Quick wins, a large benefit for little work, sit at the top left.
      return { effort, bubble, size, quick: benefit >= 4 && effort <= 2 }
    }),
  }))
})
</script>

<template>
  <div class="grid grid-cols-[22px_minmax(0,1fr)] gap-2">
    <div aria-hidden="true" class="rotate-180 text-center text-[13px] font-semibold text-muted [writing-mode:vertical-rl]">More benefit →</div>
    <div class="flex min-w-0 flex-col gap-1.5">
      <div role="group" aria-label="Benefit against effort" class="grid grid-cols-[1.25rem_repeat(5,minmax(0,1fr))]">
        <template v-for="row in rows" :key="row.benefit">
          <span aria-hidden="true" class="flex items-center text-[13px] text-muted">{{ row.benefit }}</span>
          <div
            v-for="cell in row.cells"
            :key="cell.effort"
            class="flex h-16 items-center justify-center border-grey"
            :class="[cell.quick ? 'bg-orange-pale' : '', cell.effort > 1 ? 'border-l' : '', row.benefit < 5 ? 'border-t' : '']"
          >
            <button
              v-if="cell.bubble"
              type="button"
              :aria-label="cell.bubble.label"
              :title="cell.bubble.label"
              :aria-pressed="picked === cell.bubble.key"
              class="inline-flex cursor-pointer items-center justify-center rounded-full border-2 border-solid p-0 text-[15px] font-bold transition-transform hover:scale-110"
              :class="picked === cell.bubble.key ? 'border-black bg-black text-white' : 'border-orange bg-orange text-black'"
              :style="{ width: cell.size, height: cell.size }"
              @click="emit('pick', cell.bubble.key)"
            >
              {{ cell.bubble.count }}
            </button>
          </div>
        </template>
        <span aria-hidden="true"></span>
        <span v-for="e in SCORES" :key="e" aria-hidden="true" class="pt-1.5 text-center text-[13px] text-muted">{{ e }}</span>
      </div>
      <div aria-hidden="true" class="text-center text-[13px] font-semibold text-muted">More work →</div>
    </div>
  </div>
</template>
