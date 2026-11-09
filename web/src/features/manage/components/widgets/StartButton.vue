<script setup lang="ts">
import { computed, ref } from 'vue'
import { type BudgetName, confirmLine, estimateLine, overBudgetLine } from '../../domain/estimate'

/**
 * A start button with its estimate. When the work costs more than the budget has left, a warning
 * says so before the click, and the first click asks for a second, Start anyway.
 */
const props = defineProps<{
  label: string
  usd: number
  left: number
  /** The key whose monthly budget pays for it. */
  budget: BudgetName
  /** What the estimate covers, such as "200 responses". */
  note?: string
  /** Why the button is off; it is on without one. */
  disabledNote?: string
  busy?: boolean
}>()
const emit = defineEmits<{ start: [] }>()
const asking = ref(false)
const over = computed(() => overBudgetLine(props.usd, props.left, props.budget))

function start(): void {
  if (over.value !== null && !asking.value) {
    asking.value = true
    return
  }
  asking.value = false
  emit('start')
}

const PRIMARY = 'min-h-11 rounded-full border-0 bg-black px-6 font-semibold text-white hover:bg-ink-2 disabled:opacity-60'
</script>

<template>
  <div class="flex flex-col gap-3">
    <p v-if="over && !disabledNote" class="m-0 font-medium text-orange-ink" data-part="over-budget">{{ over }}</p>
    <div v-if="asking" role="alert" class="flex flex-wrap items-center gap-3">
      <span>{{ confirmLine(usd, left, budget) }}</span>
      <button type="button" :class="PRIMARY" @click="start">Start anyway</button>
      <button type="button" class="min-h-11 rounded-full border-2 border-silver bg-white px-4 font-semibold hover:border-black" @click="asking = false">Cancel</button>
    </div>
    <div v-else class="flex flex-wrap items-center gap-x-4 gap-y-2">
      <button type="button" :class="PRIMARY" :disabled="disabledNote !== undefined || busy" @click="start">{{ label }}</button>
      <span v-if="disabledNote" class="text-ink-2" data-part="disabled-note">{{ disabledNote }}</span>
      <span v-else class="text-ink-2" data-part="estimate">{{ estimateLine(usd, left, budget, note) }}</span>
    </div>
  </div>
</template>
