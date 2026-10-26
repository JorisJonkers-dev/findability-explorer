<script setup lang="ts">
import { computed, ref } from 'vue'
import type { PromptRun, PromptScope } from '../../../infrastructure/api/types.gen'
import { providerName, SCOPES, trendRows } from '../domain/results'
import ShareCell from './ShareCell.vue'

/** How the shares move across runs, the October pilot first, for all answers or one model and scope. */
const props = defineProps<{ runs: readonly PromptRun[] }>()

const provider = ref('')
const scope = ref<PromptScope | ''>('')
const providers = computed(() => [...new Set(props.runs.flatMap((r) => r.models.map((m) => m.provider)))])
const rows = computed(() =>
  trendRows(props.runs, { provider: provider.value === '' ? undefined : provider.value, scope: scope.value === '' ? undefined : scope.value }),
)
</script>

<template>
  <div class="flex flex-col gap-3 stage:gap-5">
    <div class="flex flex-wrap gap-3 stage:gap-6">
      <label class="flex flex-col gap-1 text-sm text-ink-2 stage:text-[24px]">
        Model
        <select v-model="provider" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[72px] stage:text-[26px]">
          <option value="">Every model</option>
          <option v-for="p in providers" :key="p" :value="p">{{ providerName(p) }}</option>
        </select>
      </label>
      <label class="flex flex-col gap-1 text-sm text-ink-2 stage:text-[24px]">
        Scope
        <select v-model="scope" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[72px] stage:text-[26px]">
          <option value="">Every scope</option>
          <option v-for="s in SCOPES" :key="s.value" :value="s.value">{{ s.label }}</option>
        </select>
      </label>
    </div>
    <div class="overflow-x-auto">
      <table class="w-full border-collapse text-sm stage:text-[26px]">
        <caption class="sr-only">AI share of voice and citation share across Prompt runs, oldest first</caption>
        <thead>
          <tr class="text-left text-muted">
            <th scope="col" rowspan="2" class="py-1.5 pr-3 align-bottom font-medium">Run</th>
            <th scope="colgroup" colspan="3" class="px-2 py-1.5 font-medium">Names Tribelt</th>
            <th scope="colgroup" colspan="3" class="px-2 py-1.5 font-medium">Cites Tribelt</th>
          </tr>
          <tr class="border-b-2 border-black text-left text-xs text-muted stage:border-b-[3px] stage:text-[22px]">
            <template v-for="measure in ['names', 'cites']" :key="measure">
              <th scope="col" class="px-2 pb-1.5 font-medium">In all</th>
              <th scope="col" class="px-2 pb-1.5 font-medium">Own knowledge</th>
              <th scope="col" class="px-2 pb-1.5 font-medium">Web search</th>
            </template>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.label" class="border-b border-grey">
            <th scope="row" class="py-2 pr-3 text-left font-medium">
              {{ row.label }}<span class="block text-xs font-normal text-muted stage:text-[20px]">{{ row.note }}</span>
            </th>
            <template v-for="measure in (['named', 'cited'] as const)" :key="measure">
              <td class="px-2 py-2"><ShareCell :part="row.overall[measure]" :whole="row.overall.answers" /></td>
              <td class="px-2 py-2"><ShareCell v-if="row.knowledge" :part="row.knowledge[measure]" :whole="row.knowledge.answers" /><span v-else class="text-muted">–</span></td>
              <td class="px-2 py-2"><ShareCell v-if="row.web" :part="row.web[measure]" :whole="row.web.answers" /><span v-else class="text-muted">–</span></td>
            </template>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
