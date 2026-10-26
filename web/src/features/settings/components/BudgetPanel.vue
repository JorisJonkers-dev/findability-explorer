<script setup lang="ts">
import { computed } from 'vue'
import type { LlmBudget } from '../../../infrastructure/api/types.gen'
import { percent } from '../../../shared/percent'
import { budgetMeter } from '../domain/budget'

const props = defineProps<{ budget: LlmBudget }>()
const meter = computed(() => budgetMeter(props.budget))
</script>

<template>
  <section aria-labelledby="budget-title" class="flex flex-col gap-4">
    <h2 id="budget-title" class="m-0 font-display text-3xl font-semibold tracking-[-0.01em]">Budget</h2>
    <p class="m-0 text-lg">{{ meter.text }}</p>
    <div aria-hidden="true" class="h-5 bg-grey">
      <div class="h-full bg-orange" :style="{ width: percent(meter.fraction) }"></div>
    </div>
    <p class="m-0 text-ink-2">The month runs in UTC. Once the tokens are used, scans run without product matching until the next month.</p>
    <div class="flex flex-col gap-1" data-testid="per-scan">
      <p class="m-0 text-lg font-semibold">{{ meter.perScan }}</p>
      <p v-if="meter.left" class="m-0 text-ink-2">{{ meter.left }}</p>
    </div>
  </section>
</template>
