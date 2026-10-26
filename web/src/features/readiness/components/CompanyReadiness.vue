<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useReadinessTable } from '../composables/useReadiness'
import { adviceSummary, policySummary, readinessFacts } from '../domain/readiness'

/** A company's AI readiness on its card: its crawler policy in a line, and the other signals. */
const props = defineProps<{ companyId: string }>()

const { rowOf, isLoading } = useReadinessTable()
const readiness = computed(() => rowOf(props.companyId)?.readiness)
</script>

<template>
  <section aria-labelledby="company-readiness" class="flex flex-col gap-2 stage:gap-4" data-testid="company-readiness">
    <h3 id="company-readiness" class="m-0 text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[26px]">AI readiness</h3>
    <p v-if="isLoading" class="m-0 text-muted stage:text-[28px]">Loading…</p>
    <template v-else>
      <p class="m-0 font-medium stage:text-[30px]">{{ policySummary(readiness) }}</p>
      <template v-if="readiness">
        <p class="m-0 text-sm text-ink-2 stage:text-[26px]">{{ adviceSummary(readiness) }}</p>
        <dl class="m-0 grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1 text-sm stage:gap-x-8 stage:gap-y-2 stage:text-[26px]">
          <template v-for="fact in readinessFacts(readiness)" :key="fact.label">
            <dt class="text-ink-2">{{ fact.label }}</dt>
            <dd class="m-0" :class="fact.ok === false ? 'font-semibold text-orange-ink' : ''">
              {{ fact.value }}<span v-if="fact.ok === false" class="sr-only">, a gap</span>
            </dd>
          </template>
        </dl>
      </template>
      <RouterLink to="/ai-search/readiness" class="self-start text-sm font-medium stage:text-[26px]">Compare with Tribelt and the others</RouterLink>
    </template>
  </section>
</template>
