<script setup lang="ts">
import { vScrollHint } from '../../../shared/scrollHint'
import { computed, ref, toRef, watch } from 'vue'
import type { AnalysisSummary } from '../../../infrastructure/api/types.gen'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import { useScanComparison } from '../composables/useCompanyHistory'
import {
  type ChangeKind,
  changeCounts,
  changeDetail,
  changedGroups,
  defaultPair,
  finishedScans,
  groupLabel,
  kindLabel,
  scanDate,
  scanName,
} from '../domain/history'

const props = defineProps<{ companyId: string; scans: readonly AnalysisSummary[] }>()

const finished = computed(() => finishedScans(props.scans))
const from = ref('')
const to = ref('')
watch(
  () => defaultPair(props.scans),
  (pair) => {
    if (pair && (from.value === '' || to.value === '')) {
      from.value = pair.from
      to.value = pair.to
    }
  },
  { immediate: true },
)
const { comparison, isLoading, error } = useScanComparison(toRef(() => props.companyId), from, to)
const counts = computed(() => (comparison.value ? changeCounts(comparison.value.groups) : null))
const groups = computed(() => (comparison.value ? changedGroups(comparison.value.groups) : []))

const KINDS: ChangeKind[] = ['added', 'removed', 'moved']
/** Each kind's colour: added orange, removed grey and struck through, moved blue-grey. The word says it too. */
const MARK: Record<ChangeKind, string> = { added: 'bg-orange', removed: 'bg-silver', moved: 'bg-blue-grey' }
const PHRASE: Record<ChangeKind, string> = { added: 'font-semibold', removed: 'text-muted line-through', moved: '' }
</script>

<template>
  <section aria-labelledby="compare-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-7">
    <SectionHeading id="compare-title">Compare two scans</SectionHeading>
    <p v-if="finished.length < 2" class="m-0 text-ink-2 stage:text-[30px]">Two finished scans are needed to compare. Re-scan the company to get a second.</p>
    <template v-else>
      <div class="grid grid-cols-2 gap-3 stage:gap-6">
        <label class="flex flex-col gap-1.5 text-sm text-ink-2 stage:gap-2 stage:text-[26px]">
          Older scan
          <select v-model="from" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[76px] stage:border-[3px] stage:px-5 stage:text-[28px]">
            <option v-for="s in finished" :key="s.id" :value="s.id">{{ scanName(s) }}, {{ scanDate(s) }}</option>
          </select>
        </label>
        <label class="flex flex-col gap-1.5 text-sm text-ink-2 stage:gap-2 stage:text-[26px]">
          Newer scan
          <select v-model="to" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[76px] stage:border-[3px] stage:px-5 stage:text-[28px]">
            <option v-for="s in finished" :key="s.id" :value="s.id">{{ scanName(s) }}, {{ scanDate(s) }}</option>
          </select>
        </label>
      </div>
      <p v-if="from === to" class="m-0 text-ink-2 stage:text-[28px]">Pick two different scans.</p>
      <p v-else-if="error" role="alert" class="m-0 text-orange-ink stage:text-[28px]">{{ error }}</p>
      <p v-else-if="isLoading" class="m-0 text-muted stage:text-[28px]">Comparing…</p>
      <template v-else-if="comparison && counts">
        <p class="m-0 flex flex-wrap gap-x-5 gap-y-1 text-sm stage:gap-x-8 stage:text-[28px]" aria-live="polite">
          <span v-for="kind in KINDS" :key="kind" class="inline-flex items-center gap-2 stage:gap-3">
            <span aria-hidden="true" class="size-3 rounded-full stage:size-5" :class="MARK[kind]"></span>{{ counts[kind] }} {{ kindLabel(kind).toLowerCase() }}
          </span>
        </p>
        <p v-if="groups.length === 0" class="m-0 text-ink-2 stage:text-[28px]">The two scans found the same terms.</p>
        <div v-scroll-hint class="scroll-hint flex flex-col gap-5 stage:min-h-0 stage:flex-1 stage:gap-8 stage:overflow-y-auto">
          <section v-for="g in groups" :key="g.group" :aria-label="groupLabel(g.group)" class="flex flex-col gap-2 stage:gap-3">
            <h3 class="m-0 text-base font-semibold stage:text-[32px]">{{ groupLabel(g.group) }}</h3>
            <ul class="m-0 flex list-none flex-col gap-1 p-0 text-sm stage:gap-2 stage:text-[26px]">
              <template v-for="kind in KINDS" :key="kind">
                <li v-for="c in g[kind]" :key="`${kind}-${c.phrase}-${c.language}`" class="flex flex-wrap items-baseline gap-x-2.5 stage:gap-x-4">
                  <span class="inline-flex items-center gap-1.5 self-center rounded-full bg-mist px-2 py-0.5 text-xs font-semibold stage:gap-2.5 stage:px-4 stage:py-1 stage:text-[22px]">
                    <span aria-hidden="true" class="size-2 rounded-full stage:size-3.5" :class="MARK[kind]"></span>{{ kindLabel(kind) }}
                  </span>
                  <span :class="PHRASE[kind]">{{ c.phrase }}</span>
                  <span class="text-muted">({{ c.language }}) {{ changeDetail(c, kind) }}</span>
                </li>
              </template>
            </ul>
          </section>
        </div>
      </template>
    </template>
  </section>
</template>
