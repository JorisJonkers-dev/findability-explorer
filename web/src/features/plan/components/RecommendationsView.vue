<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { vScrollHint } from '../../../shared/scrollHint'
import BenefitEffortChart from '../../../shared/ui/charts/BenefitEffortChart.vue'
import PageFrame from '../../shell/components/PageFrame.vue'
import { usePlan } from '../composables/usePlan'
import {
  cells,
  dataLine,
  doneLine,
  inCell,
  KIND_LABEL,
  KIND_PLURAL,
  kindOf,
  kindsIn,
  lcFirst,
  listed,
  measureName,
  open,
  scoreLine,
  scrollCue,
  SORTS,
  whereText,
  type Direction,
  type Kind,
  type Sort,
} from '../domain/recommendations'
import RecSection from './RecSection.vue'

const { plan, isLoading, error } = usePlan()

const kind = ref<Kind | 'all'>('all')
const cell = ref<string | undefined>()
const sort = ref<Sort>('value')
const direction = ref<Direction>('desc')

const items = computed(() => plan.value?.improvements ?? [])
const kinds = computed(() => kindsIn(items.value))
const stillOpen = computed(() => open(items.value))
const shown = computed(() => (kind.value === 'all' ? stillOpen.value : stillOpen.value.filter((i) => kindOf(i) === kind.value)))
const rows = computed(() => listed(inCell(shown.value, cell.value), sort.value, direction.value))
const bubbles = computed(() => cells(shown.value))
const finished = computed(() => items.value.filter((i) => i.done))
const count = (n: number, word: string) => `${String(n)} ${word}${n === 1 ? '' : 's'}`

function pickKind(next: Kind | 'all'): void {
  kind.value = next
  cell.value = undefined
}
function pickCell(key: string): void {
  cell.value = cell.value === key ? undefined : key
}
/** Picking the sort already in use turns it round. */
function pickSort(next: Sort): void {
  if (sort.value === next) direction.value = direction.value === 'desc' ? 'asc' : 'desc'
  else {
    sort.value = next
    direction.value = 'desc'
  }
}

const PURPOSE =
  'Changes to make on tribelt.nl, ranked by benefit against effort. Some apply in general. The rest come from the latest Recommendation run and name a page, search term, or site.'
const pill = 'min-h-10 cursor-pointer rounded-full border-2 border-solid px-4 text-[15px] font-semibold'
const on = 'border-black bg-black text-white'
const off = 'border-silver bg-white text-black hover:border-black'
</script>

<template>
  <PageFrame
    section="recommendations"
    title="Recommendations"
    :purpose="PURPOSE"
    :data-line="plan ? dataLine(plan.catalogueVersion) : undefined"
    :manage="{ to: '/manage/recommendations', label: 'Manage the recommendations' }"
    help-topic="recommendations"
  >
    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the recommendations…</p>
    <template v-else-if="plan">
      <div class="flex flex-wrap items-center gap-x-4 gap-y-3 rounded-3xl bg-mist px-5 py-4">
        <template v-if="kinds.length > 1">
          <span class="font-semibold">Show</span>
          <div role="group" aria-label="Show" class="flex flex-wrap gap-2">
            <button type="button" :aria-pressed="kind === 'all'" :class="[pill, kind === 'all' ? on : off]" @click="pickKind('all')">All</button>
            <button v-for="k in kinds" :key="k" type="button" :aria-pressed="kind === k" :class="[pill, kind === k ? on : off]" @click="pickKind(k)">
              {{ KIND_PLURAL[k] }}
            </button>
          </div>
        </template>
        <span class="text-[15px] text-ink-2">{{ count(shown.length, 'open recommendation') }}{{ kind === 'all' ? '' : ' of this kind' }}</span>
      </div>

      <div class="grid items-start gap-10 min-[1100px]:grid-cols-2">
        <RecSection title="Benefit against effort" heading-id="chart-title">
          <template #intro>Each bubble groups recommendations with the same benefit and effort. Quick wins sit at the top left. Pick a bubble to list only those.</template>
          <div class="rounded-3xl bg-mist p-4 sm:p-6">
            <BenefitEffortChart :bubbles="bubbles" :picked="cell" @pick="pickCell" />
          </div>
        </RecSection>

        <RecSection title="Ranked list" heading-id="list-title">
          <template #intro>{{ count(rows.length, 'recommendation') }}. Best value is benefit divided by effort. Ties go to the larger benefit.</template>
          <template v-if="cell" #aside>
            <button type="button" class="min-h-10 cursor-pointer rounded-full border-2 border-solid border-silver bg-white px-4 text-[15px] font-semibold hover:border-black" @click="cell = undefined">
              Show every bubble
            </button>
          </template>
          <p v-if="stillOpen.length === 0" class="m-0 rounded-3xl bg-mist p-5 text-ink-2">Every recommendation is done.</p>
          <p v-else-if="rows.length === 0" class="m-0 rounded-3xl bg-mist p-5 text-ink-2">No open recommendation is of this kind.</p>
          <template v-else>
            <div class="flex flex-wrap items-center gap-3">
              <span class="font-semibold">Sort by</span>
              <div role="group" aria-label="Sort by" class="flex flex-wrap gap-2">
                <button v-for="s in SORTS" :key="s.id" type="button" :aria-pressed="sort === s.id" :class="[pill, sort === s.id ? on : off]" @click="pickSort(s.id)">
                  {{ s.label }}<span v-if="sort === s.id" aria-hidden="true"> {{ direction === 'desc' ? '↓' : '↑' }}</span>
                </button>
              </div>
            </div>
            <ul v-scroll-hint aria-label="Recommendations" class="scroll-hint m-0 max-h-[31rem] list-none overflow-y-auto rounded-3xl bg-mist px-4 py-1">
              <li v-for="(r, i) in rows" :key="r.id" class="flex flex-col gap-2 border-b border-grey py-4 last:border-b-0">
                <RouterLink :to="`/recommendations/${r.id}`" class="text-lg font-semibold">{{ i + 1 }}. {{ r.title }}</RouterLink>
                <div class="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-[15px] text-ink-2">
                  <span class="rounded-md bg-grey px-2 py-0.5 text-sm font-medium text-black">{{ KIND_LABEL[kindOf(r)] }}</span>
                  <span>{{ whereText() }}</span>
                  <span>{{ scoreLine(r) }}</span>
                  <span>Improves {{ lcFirst(measureName(r.measure)) }}</span>
                </div>
              </li>
            </ul>
            <p v-if="scrollCue(rows.length)" class="m-0 text-sm text-muted">{{ scrollCue(rows.length) }}</p>
          </template>
        </RecSection>
      </div>

      <RecSection title="Done" heading-id="done-title">
        <details v-if="finished.length > 0">
          <summary class="cursor-pointer font-semibold text-orange-ink">Done ({{ finished.length }})</summary>
          <ul class="m-0 mt-3 list-none rounded-3xl bg-mist px-4 py-1">
            <li v-for="d in finished" :key="d.id" class="flex flex-col gap-1.5 border-b border-grey py-3.5 last:border-b-0">
              <RouterLink :to="`/recommendations/${d.id}`" class="font-semibold">{{ d.title }}</RouterLink>
              <div class="flex flex-wrap items-center gap-x-4 gap-y-1.5 text-[15px] text-ink-2">
                <span class="rounded-md bg-grey px-2 py-0.5 text-sm font-medium text-black">{{ KIND_LABEL[kindOf(d)] }}</span>
                <span>{{ doneLine(d) }}</span>
              </div>
            </li>
          </ul>
        </details>
        <p v-else class="m-0 text-muted">Nothing is marked done yet.</p>
      </RecSection>
    </template>
  </PageFrame>
</template>
