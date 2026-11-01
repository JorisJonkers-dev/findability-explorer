<script setup lang="ts">
import { computed } from 'vue'
import type { LlmBudget } from '../../../infrastructure/api/types.gen'
import ProgressBar from '../../../shared/ui/ProgressBar.vue'
import { budgetMeter } from '../domain/budget'

/** This month's tokens against the budget: the line, and the bar. */
const props = defineProps<{ budget: LlmBudget }>()
const meter = computed(() => budgetMeter(props.budget))
</script>

<template>
  <div class="flex flex-col gap-2">
    <p class="m-0">{{ meter.text }}</p>
    <ProgressBar thick :fraction="meter.fraction" label="Tokens used this month" :text="meter.text" />
  </div>
</template>
