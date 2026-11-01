<script setup lang="ts">
import { vScrollHint } from '../../../shared/scrollHint'
import { computed, ref } from 'vue'
import { percent } from '../../../shared/percent'
import { tierFill } from '../../../shared/tiers'
import PillButton from '../../../shared/ui/PillButton.vue'
import FrameSection from '../../../shared/ui/FrameSection.vue'
import PageFrame from '../../shell/components/PageFrame.vue'
import { RouterLink } from 'vue-router'
import { useTermMap } from '../composables/useTermSearch'
import { useTermView } from '../composables/useTermView'
import type { TermGroup } from '../domain/filters'
import { highlight } from '../domain/fold'
import { directionOf, nextSort, type Sort, type SortDirection, sortRows } from '../../../shared/sort'
import SortButton from '../../../shared/ui/SortButton.vue'
import {
  cellBackground,
  chips,
  filterTerms,
  GROUP_LABELS,
  type HeatColumn,
  heatmap,
  heatSortValue,
  languageRows,
  type RankColumn,
  rankSortValue,
  rankTerms,
  summary,
  tierRows,
  usedLine,
} from '../domain/terms'
import { useOverview } from '../../home/composables/useOverview'

const { terms, total, error } = useTermMap()
const { view, draft, toggle, clear, toggleGap } = useTermView()

// Each column can be widened or minimised on the stage; the browser remembers the layout.
// The charts and the count of what tribelt.nl uses are the server's, as on the home page.
const { overview } = useOverview()
const languages = computed(() => languageRows(overview.value?.terms))
const tiers = computed(() => tierRows(overview.value?.terms))
const heat = computed(() => heatmap(terms.value))
const ranked = computed(() => rankTerms(filterTerms(terms.value, view.value), terms.value))

// Each table sorts by a column while one is picked: names A to Z first, counts highest first.
const heatSort = ref<Sort<HeatColumn>>(null)
const sortedHeat = computed(() => sortRows(heat.value, heatSort.value, heatSortValue))
const rankSort = ref<Sort<RankColumn>>(null)
const sortedRanked = computed(() => sortRows(ranked.value, rankSort.value, rankSortValue))
function sortHeat(column: HeatColumn, first: SortDirection): void {
  heatSort.value = nextSort(heatSort.value, column, first)
}
function sortRank(column: RankColumn, first: SortDirection): void {
  rankSort.value = nextSort(rankSort.value, column, first)
}

/** The per-company table's columns of kinds of search, with their short headers. */
const HEAT_COLUMNS: readonly { key: HeatColumn; label: string; short: string }[] = [
  { key: 'broad', label: 'Broad', short: 'Broad' },
  { key: 'product', label: 'Product', short: 'Prod.' },
  { key: 'application', label: 'Application', short: 'Appl.' },
  { key: 'spec', label: 'Specification', short: 'Spec.' },
  { key: 'problem', label: 'Problem', short: 'Probl.' },
]
const activeChips = computed(() => chips(view.value))
const dataLine = computed(() => `${String(ranked.value.length)} of ${String(total.value)} terms shown. From the search term list and each site's newest scan.`)
const status = computed(() => (terms.value.length === 0 ? 'Loading…' : summary(ranked.value.length, terms.value.length)))

const GROUPS: readonly { value: TermGroup | null; label: string }[] = [
  { value: null, label: 'All' },
  { value: 'marketing', label: GROUP_LABELS.marketing },
  { value: 'technical', label: GROUP_LABELS.technical },
]

/** The group pills pick one, like radio buttons: pressing the pressed one keeps it. */
function pickGroup(group: TermGroup | null): void {
  if (group === view.value.group) return
  if (group === null) clear('group')
  else toggle('group', group)
}

const chartRowClass =
  'grid w-full min-h-10 grid-cols-[6.5rem_minmax(0,1fr)_2.5rem] items-center gap-3 rounded-lg px-2 py-1 text-left text-sm hover:bg-mist'
// Company and six narrow cells, with a gap between the two halves.
const heatGrid =
  'grid grid-cols-[minmax(0,1fr)_repeat(3,1.75rem)_0.25rem_repeat(2,1.75rem)_2rem] gap-1'
</script>

<template>
  <PageFrame
    section="terms"
    title="Every term"
    purpose="Every term in the search term list, with the competitors that use it and whether tribelt.nl does."
    :data-line="dataLine"
    :back="{ to: '/terms', label: 'Back to Search terms' }"
    :manage="{ to: '/manage/terms', label: 'Manage the search term list' }"
    help-topic="terms"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-5">{{ error }}</p>
    <FrameSection id="terms-ranked" title="Ranked by how many competitors use them">
      <div class="flex flex-col gap-4">
        <div role="group" aria-label="Language" class="flex flex-wrap gap-2">
          <PillButton size="sm" :pressed="view.language === null" @click="clear('language')">All languages</PillButton>
          <PillButton
            v-for="row in languages"
            :key="row.value"
            size="sm"
            :pressed="view.language === row.value"
            @click="view.language !== row.value && toggle('language', row.value)"
          >
            {{ row.label }}
          </PillButton>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <div role="group" aria-label="Terms" class="flex flex-wrap gap-2">
            <PillButton v-for="group in GROUPS" :key="group.label" size="sm" :pressed="view.group === group.value" @click="pickGroup(group.value)">
              {{ group.label }}
            </PillButton>
          </div>
          <PillButton size="sm" :pressed="view.gap" @click="toggleGap">Only terms Tribelt lacks</PillButton>
          <label for="terms-q" class="sr-only">Search terms</label>
          <input
            id="terms-q"
            v-model="draft"
            type="search"
            placeholder="Search terms"
            spellcheck="false"
            autocomplete="off"
            class="h-11 min-w-0 flex-[1_1_12rem] rounded-full border-2 border-grey bg-white px-4 text-sm"
          />
        </div>
        <div class="flex min-h-9 flex-wrap items-center gap-2">
          <span aria-live="polite" class="text-sm text-ink-2">{{ status }}</span>
          <button
            v-for="chip in activeChips"
            :key="chip.key"
            type="button"
            :aria-label="`Remove the filter ${chip.label}`"
            class="inline-flex min-h-9 items-center gap-2 rounded-full bg-black px-3 text-sm text-white hover:bg-ink-2"
            @click="clear(chip.key)"
          >
            {{ chip.label }}
            <svg aria-hidden="true" viewBox="0 0 16 16" class="size-3.5" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 4l8 8M12 4l-8 8" stroke-linecap="round" />
            </svg>
          </button>
        </div>
        <div class="flex flex-col">
          <div
            role="group"
            aria-label="Sort the terms"
            class="flex flex-wrap gap-x-4 gap-y-1 border-b-2 border-black pb-2 text-xs font-semibold"
          >
            <SortButton label="Rank" :direction="directionOf(rankSort, 'rank')" @sort="sortRank('rank', 'asc')">#</SortButton>
            <SortButton label="Term" :direction="directionOf(rankSort, 'term')" @sort="sortRank('term', 'asc')" />
            <SortButton label="Buyer intent" :direction="directionOf(rankSort, 'kind')" @sort="sortRank('kind', 'asc')" />
            <SortButton label="Competitors" :direction="directionOf(rankSort, 'makers')" @sort="sortRank('makers', 'desc')" />
            <SortButton label="Tribelt's term" :direction="directionOf(rankSort, 'tribelt')" @sort="sortRank('tribelt', 'desc')" />
          </div>
          <ol
            v-scroll-hint
            aria-label="Ranked terms"
            tabindex="0"
            class="scroll-hint m-0 max-h-[36rem] list-none overflow-y-auto border-t-2 border-black p-0"
          >
            <li
              v-for="row in sortedRanked"
              :key="`${row.term}-${row.meta}`"
              class="grid grid-cols-[1.75rem_minmax(0,1fr)_4.5rem] items-center gap-x-2 gap-y-0.5 border-b border-grey py-2 text-sm"
            >
              <span class="col-start-1 row-span-2 row-start-1 self-start text-muted">{{ row.rank }}</span>
              <span class="col-start-2 row-start-1 flex min-w-0 flex-col">
                <span class="truncate font-medium">
                  <RouterLink :to="row.path">
                    <template v-for="(part, index) in highlight(row.term, view.q)" :key="index">
                      <mark v-if="part.match" class="bg-orange/30 text-black">{{ part.text }}</mark>
                      <template v-else>{{ part.text }}</template>
                    </template>
                  </RouterLink>
                </span>
                <span class="truncate text-xs text-muted">{{ row.meta }}</span>
              </span>
              <span class="col-start-2 row-start-2 inline-flex items-center gap-2 text-xs">
                <span aria-hidden="true" class="size-2.5 flex-none rounded-full" :class="tierFill(row.tier)"></span>{{ row.kind }}
              </span>
              <span class="col-start-3 row-start-1 flex items-center gap-2">
                <span class="w-5 text-right font-semibold">{{ row.makers }}<span class="sr-only"> competitors</span></span>
                <span aria-hidden="true" class="h-2 flex-1 bg-mist">
                  <span class="block h-full bg-black" :style="{ width: percent(row.share) }"></span>
                </span>
              </span>
              <span
                class="col-span-2 col-start-2 row-start-3 truncate text-xs font-medium"
                :class="row.tribelt ? 'text-ink-2' : row.tribeltTerm === 'Lacks it' ? 'text-orange-ink' : 'text-black'"
                :title="row.tribelt ? undefined : row.tribeltTerm"
              ><span class="">Tribelt: </span>{{ row.tribeltTerm }}</span>
            </li>
          </ol>
        </div>
      </div>
    </FrameSection>
    <div class="grid grid-cols-1 items-start gap-10 min-[1100px]:grid-cols-2">
      <FrameSection id="terms-charts" title="Terms tribelt.nl does not use">
        <div class="flex flex-col gap-4">
          <p v-if="usedLine(overview?.terms)" class="m-0 text-base font-semibold">
            {{ usedLine(overview?.terms) }}
          </p>
          <span class="flex flex-wrap gap-x-6 gap-y-2 text-sm text-muted">
            <span class="inline-flex items-center gap-2"><span aria-hidden="true" class="size-4 bg-orange"></span>tribelt.nl uses it</span>
            <span class="inline-flex items-center gap-2"><span aria-hidden="true" class="size-4 bg-silver"></span>Only others use it</span>
          </span>

          <div v-for="chart in [{ key: 'language', title: 'By language', rows: languages }, { key: 'tier', title: 'By buyer intent', rows: tiers }] as const" :key="chart.key" role="group" :aria-label="chart.title" class="flex flex-col gap-1">
            <span class="font-semibold">{{ chart.title }}</span>
            <button
              v-for="row in chart.rows"
              :key="row.value"
              type="button"
              :aria-pressed="view[chart.key] === row.value"
              :aria-label="row.description"
              :class="[chartRowClass, view[chart.key] === row.value ? 'bg-mist' : '']"
              class="group relative"
              @click="toggle(chart.key, row.value)"
            >
              <span :class="view[chart.key] === row.value ? 'font-semibold' : ''">{{ row.label }}</span>
              <span aria-hidden="true" class="relative flex h-4">
                <span class="h-full bg-orange" :style="{ width: percent(row.tribeltShare) }"></span>
                <span class="h-full bg-silver" :style="{ width: percent(row.otherShare) }"></span>
                <!-- Above the bar while it is hovered or focused: what tribelt.nl uses, and what only others still have. -->
                <span
                  class="pointer-events-none absolute bottom-full left-0 z-10 mb-1.5 hidden whitespace-nowrap rounded-lg bg-black px-2.5 py-1.5 text-xs font-medium text-white shadow group-hover:block group-focus-visible:block"
                  data-testid="bar-info"
                >
                  tribelt.nl uses {{ row.tribelt }}, only others use {{ row.count - row.tribelt }}
                </span>
              </span>
              <span class="text-right font-semibold">{{ row.tribelt }}/{{ row.count }}</span>
            </button>
          </div>

          <p class="m-0 rounded-2xl bg-mist p-5">
            A buyer who only knows the belt they already have searches using the competitor's name for it.
            <strong class="font-semibold">Recommendation 3:</strong> every Tribelt belt page names its competitor equivalents, in
            Dutch, German and English.
          </p>
        </div>
      </FrameSection>
      <FrameSection id="terms-heat" title="Marketing and technical terms, per company">
        <div class="flex flex-col gap-4">
          <div class="flex flex-col gap-2">
            <span class="text-sm text-ink-2">
              Marketing terms name a belt or its use. Technical terms are what engineers type: specifications and problems.
              Select a company to list its terms.
            </span>
          </div>
          <div class="flex flex-col">
            <div aria-hidden="true" :class="heatGrid" class="px-1 pb-1 text-center text-xs text-muted">
              <span></span>
              <span class="col-span-3 border-b-2 border-silver pb-1">Marketing</span>
              <span></span>
              <span class="col-span-2 border-b-2 border-orange pb-1">Technical</span>
              <span></span>
            </div>
            <div
              role="group"
              aria-label="Sort the companies"
              :class="heatGrid"
              class="items-end border-b-[3px] border-black px-1 pb-2 text-center text-[10px] font-semibold"
            >
              <SortButton label="Company" :direction="directionOf(heatSort, 'company')" class="text-xs font-medium text-muted" @sort="sortHeat('company', 'asc')" />
              <template v-for="(column, index) in HEAT_COLUMNS" :key="column.key">
                <span v-if="index === 3"></span>
                <SortButton :label="column.label" :direction="directionOf(heatSort, column.key)" align="center" @sort="sortHeat(column.key, 'desc')">{{ column.short }}</SortButton>
              </template>
              <SortButton label="Total" :direction="directionOf(heatSort, 'total')" align="center" @sort="sortHeat('total', 'desc')" />
            </div>
            <ul v-scroll-hint aria-label="Terms per company and kind" class="scroll-hint m-0 max-h-[28rem] list-none overflow-y-auto p-0">
              <li v-for="row in sortedHeat" :key="row.company" class="border-b border-grey">
                <button
                  type="button"
                  :aria-pressed="view.user === row.company"
                  :aria-label="row.description"
                  :class="[heatGrid, view.user === row.company ? 'bg-mist' : 'hover:bg-mist']"
                  class="min-h-10 w-full items-center px-1 py-1 text-left"
                  @click="toggle('user', row.company)"
                >
                  <span
                    class="truncate text-sm"
                    :class="view.user === row.company || row.company === 'Tribelt' ? 'font-semibold' : ''"
                  >{{ row.company }}</span>
                  <template v-for="(cell, index) in row.cells" :key="cell.tier">
                    <span v-if="index === 3"></span>
                    <span
                      class="flex h-8 items-center justify-center text-xs font-semibold"
                      :class="cell.count === 0 ? 'bg-mist' : ''"
                      :style="{ backgroundColor: cellBackground(cell) }"
                    >{{ cell.count > 0 ? cell.count : '' }}</span>
                  </template>
                  <span class="text-center text-sm font-semibold">{{ row.total }}</span>
                </button>
              </li>
            </ul>
          </div>
        </div>
      </FrameSection>
    </div>
    <p data-part="intent-note" class="m-0 text-sm text-muted">
      Buyer intent is what the buyer already knows when they search. <RouterLink to="/terms">See what each kind means</RouterLink>.
    </p>
  </PageFrame>
</template>
