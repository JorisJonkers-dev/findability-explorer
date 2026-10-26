<script setup lang="ts">
import { computed, ref } from 'vue'
import type { PromptRun } from '../../../infrastructure/api/types.gen'
import PillButton from '../../../shared/ui/PillButton.vue'
import { type Dimension, shareRows } from '../domain/results'
import ShareCell from './ShareCell.vue'

/** AI share of voice and citation share of one run, per provider, scope or belt family, both modes side by side. */
const props = defineProps<{ run: PromptRun }>()

const DIMENSIONS: { value: Dimension; label: string }[] = [
  { value: 'provider', label: 'Per model' },
  { value: 'scope', label: 'Per scope' },
  { value: 'family', label: 'Per belt family' },
]
const by = ref<Dimension>('provider')
const rows = computed(() => shareRows(props.run, by.value))
const model = (provider: string) => props.run.models.find((m) => m.provider === provider)?.model
</script>

<template>
  <div class="flex flex-col gap-3 stage:gap-5">
    <div role="group" aria-label="Break down" class="flex flex-wrap gap-2 stage:gap-3">
      <PillButton v-for="d in DIMENSIONS" :key="d.value" size="sm" :pressed="by === d.value" @click="by = d.value">{{ d.label }}</PillButton>
    </div>
    <!-- Focusable, so a keyboard can scroll it sideways on a phone. -->
    <div class="overflow-x-auto" tabindex="0" role="region" aria-label="Shares of the run">
      <table class="w-full border-collapse text-sm stage:text-[26px]">
        <caption class="sr-only">AI share of voice and citation share, from the models' own knowledge and with web search</caption>
        <thead>
          <tr class="text-left text-muted">
            <th scope="col" rowspan="2" class="py-1.5 pr-3 align-bottom font-medium stage:py-3">{{ DIMENSIONS.find((d) => d.value === by)?.label.replace('Per ', '') }}</th>
            <th scope="colgroup" colspan="2" class="px-2 py-1.5 font-medium stage:px-4">From its own knowledge</th>
            <th scope="colgroup" colspan="2" class="px-2 py-1.5 font-medium stage:px-4">With web search</th>
          </tr>
          <tr class="border-b-2 border-black text-left text-xs text-muted stage:border-b-[3px] stage:text-[22px]">
            <th scope="col" class="px-2 pb-1.5 font-medium stage:px-4">Names Tribelt</th>
            <th scope="col" class="px-2 pb-1.5 font-medium stage:px-4">Cites Tribelt</th>
            <th scope="col" class="px-2 pb-1.5 font-medium stage:px-4">Names Tribelt</th>
            <th scope="col" class="px-2 pb-1.5 font-medium stage:px-4">Cites Tribelt</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.key" class="border-b border-grey">
            <th scope="row" class="py-2 pr-3 text-left font-medium stage:py-4">
              {{ row.label }}
              <span v-if="by === 'provider'" class="block text-xs font-normal text-muted stage:text-[20px]">{{ model(row.key) }}</span>
            </th>
            <td class="px-2 py-2 stage:px-4"><ShareCell :part="row.knowledge.named" :whole="row.knowledge.answers" /></td>
            <td class="px-2 py-2 stage:px-4"><ShareCell :part="row.knowledge.cited" :whole="row.knowledge.answers" /></td>
            <td class="px-2 py-2 stage:px-4"><ShareCell :part="row.web.named" :whole="row.web.answers" /></td>
            <td class="px-2 py-2 stage:px-4"><ShareCell :part="row.web.cited" :whole="row.web.answers" /></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
