<script setup lang="ts">
import { computed } from 'vue'
import { percent } from '../../../shared/percent'
import { tierFill } from '../../../shared/tiers'
import PillButton from '../../../shared/ui/PillButton.vue'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import SectionView from '../../shell/components/SectionView.vue'
import { useTermMap } from '../composables/useTermSearch'
import { useTermView } from '../composables/useTermView'
import type { TermGroup } from '../domain/filters'
import { highlight } from '../domain/fold'
import { cellBackground, chips, filterTerms, GROUP_LABELS, heatmap, languageRows, rankTerms, summary, tierRows } from '../domain/terms'

const { terms, total, error } = useTermMap()
const { view, draft, toggle, clear, toggleGap } = useTermView()

const languages = computed(() => languageRows(terms.value))
const tiers = computed(() => tierRows(terms.value))
const heat = computed(() => heatmap(terms.value))
const ranked = computed(() => rankTerms(filterTerms(terms.value, view.value), terms.value))
const activeChips = computed(() => chips(view.value))
const title = computed(() =>
  total.value > 0 ? `${String(total.value)} ways to search for a metal belt.` : 'Ways to search for a metal belt.',
)
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
  'grid w-full min-h-10 grid-cols-[6.5rem_minmax(0,1fr)_2.5rem] items-center gap-3 rounded-lg px-2 py-1 text-left text-sm hover:bg-mist stage:min-h-14 stage:grid-cols-[200px_minmax(0,1fr)_70px] stage:gap-4 stage:rounded-xl stage:px-3 stage:text-[26px]'
// Mobile: company and six narrow cells; the stage: the design's columns, a gap between the halves.
const heatGrid =
  'grid grid-cols-[minmax(0,1fr)_repeat(3,1.75rem)_0.25rem_repeat(2,1.75rem)_2rem] gap-1 stage:grid-cols-[minmax(0,1fr)_repeat(3,120px)_24px_repeat(2,120px)_100px] stage:gap-2'
const rankGrid = 'stage:grid-cols-[64px_minmax(0,1fr)_210px_150px_170px] stage:gap-4'
</script>

<template>
  <SectionView section="terms" :title="title" :source="['Terminology map, desk research', '22 October 2026']">
    <div class="flex flex-col gap-12 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,0.85fr)_minmax(0,1.1fr)_minmax(0,1.25fr)] stage:gap-24">
      <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-5 stage:col-span-3">{{ error }}</p>

      <section aria-labelledby="terms-charts-title" class="flex flex-col gap-6 stage:min-h-0 stage:gap-8">
        <SectionHeading id="terms-charts-title">Where Tribelt's words fall short</SectionHeading>
        <span class="flex flex-wrap gap-x-6 gap-y-2 text-sm text-muted stage:gap-8 stage:text-2xl">
          <span class="inline-flex items-center gap-2 stage:gap-3"><span aria-hidden="true" class="size-4 bg-orange stage:size-6"></span>Tribelt's site uses it</span>
          <span class="inline-flex items-center gap-2 stage:gap-3"><span aria-hidden="true" class="size-4 bg-silver stage:size-6"></span>Only others use it</span>
        </span>

        <div v-for="chart in [{ key: 'language', title: 'By language', rows: languages }, { key: 'tier', title: 'By kind of search', rows: tiers }] as const" :key="chart.key" role="group" :aria-label="chart.title" class="flex flex-col gap-1 stage:gap-2.5">
          <span class="font-semibold stage:text-[28px]">{{ chart.title }}</span>
          <button
            v-for="row in chart.rows"
            :key="row.value"
            type="button"
            :aria-pressed="view[chart.key] === row.value"
            :aria-label="row.description"
            :class="[chartRowClass, view[chart.key] === row.value ? 'bg-mist' : '']"
            @click="toggle(chart.key, row.value)"
          >
            <span :class="view[chart.key] === row.value ? 'font-semibold' : ''">{{ row.label }}</span>
            <span aria-hidden="true" class="flex h-4 stage:h-7">
              <span class="h-full bg-orange" :style="{ width: percent(row.tribeltShare) }"></span>
              <span class="h-full bg-silver" :style="{ width: percent(row.otherShare) }"></span>
            </span>
            <span class="text-right font-semibold">{{ row.count }}</span>
          </button>
        </div>

        <p class="m-0 rounded-2xl bg-mist p-5 stage:mt-auto stage:rounded-[28px] stage:px-10 stage:py-9 stage:text-[30px]">
          A specifier who knows only the installed belt searches with its maker's name.
          <strong class="font-semibold">Recommendation 3:</strong> every Tribelt belt page names its competitor equivalents, in
          Dutch, German and English.
        </p>
      </section>

      <section aria-labelledby="terms-heat-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
        <div class="flex flex-col gap-2">
          <SectionHeading id="terms-heat-title">Marketing and technical terms, per company</SectionHeading>
          <span class="text-sm text-ink-2 stage:text-[26px]">
            Marketing terms name a belt or its use. Technical terms are what engineers type: specifications and problems.
            Select a company to list its terms.
          </span>
        </div>
        <div class="flex flex-col stage:min-h-0 stage:flex-1">
          <div aria-hidden="true" :class="heatGrid" class="px-1 pb-1 text-center text-xs text-muted stage:px-3 stage:pb-2 stage:text-[22px]">
            <span></span>
            <span class="col-span-3 border-b-2 border-silver pb-1 stage:pb-1.5">Marketing</span>
            <span></span>
            <span class="col-span-2 border-b-2 border-orange pb-1 stage:pb-1.5">Technical</span>
            <span></span>
          </div>
          <div
            aria-hidden="true"
            :class="heatGrid"
            class="items-end border-b-[3px] border-black px-1 pb-2 text-center text-[10px] font-semibold stage:px-3 stage:pb-3.5 stage:text-[22px]"
          >
            <span class="text-left text-xs font-medium text-muted stage:text-[22px]">Company</span>
            <span>Broad</span><span>Prod.</span><span>Appl.</span><span></span><span>Spec.</span><span>Probl.</span><span>Total</span>
          </div>
          <ul aria-label="Terms per company and kind" class="m-0 max-h-[28rem] list-none overflow-y-auto p-0 stage:max-h-none stage:min-h-0 stage:flex-1">
            <li v-for="row in heat" :key="row.company" class="border-b border-grey">
              <button
                type="button"
                :aria-pressed="view.user === row.company"
                :aria-label="row.description"
                :class="[heatGrid, view.user === row.company ? 'bg-mist' : 'hover:bg-mist']"
                class="min-h-10 w-full items-center px-1 py-1 text-left stage:min-h-[58px] stage:px-3"
                @click="toggle('user', row.company)"
              >
                <span
                  class="truncate text-sm stage:text-[28px]"
                  :class="view.user === row.company || row.company === 'Tribelt' ? 'font-semibold' : ''"
                >{{ row.company }}</span>
                <template v-for="(cell, index) in row.cells" :key="cell.tier">
                  <span v-if="index === 3"></span>
                  <span
                    class="flex h-8 items-center justify-center text-xs font-semibold stage:h-[46px] stage:text-2xl"
                    :class="cell.count === 0 ? 'bg-mist' : ''"
                    :style="{ backgroundColor: cellBackground(cell) }"
                  >{{ cell.count > 0 ? cell.count : '' }}</span>
                </template>
                <span class="text-center text-sm font-semibold stage:text-[26px]">{{ row.total }}</span>
              </button>
            </li>
          </ul>
        </div>
      </section>

      <section aria-labelledby="terms-ranked-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
        <SectionHeading id="terms-ranked-title">Ranked by how many makers use them</SectionHeading>
        <div class="flex flex-wrap items-center gap-2 stage:gap-3">
          <div role="group" aria-label="Terms" class="flex flex-wrap gap-2 stage:gap-2.5">
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
            class="h-11 min-w-0 flex-[1_1_12rem] rounded-full border-2 border-grey bg-white px-4 text-sm stage:h-[76px] stage:min-w-[300px] stage:flex-1 stage:border-[3px] stage:px-7 stage:text-[26px]"
          />
        </div>
        <div class="flex min-h-9 flex-wrap items-center gap-2 stage:min-h-14 stage:gap-3">
          <span aria-live="polite" class="text-sm text-ink-2 stage:text-[26px]">{{ status }}</span>
          <button
            v-for="chip in activeChips"
            :key="chip.key"
            type="button"
            :aria-label="`Remove the filter ${chip.label}`"
            class="inline-flex min-h-9 items-center gap-2 rounded-full bg-black px-3 text-sm text-white hover:bg-ink-2 stage:min-h-14 stage:gap-3 stage:px-[22px] stage:text-2xl"
            @click="clear(chip.key)"
          >
            {{ chip.label }}
            <svg aria-hidden="true" viewBox="0 0 16 16" class="size-3.5 stage:size-5" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 4l8 8M12 4l-8 8" stroke-linecap="round" />
            </svg>
          </button>
        </div>
        <div class="flex flex-col stage:min-h-0 stage:flex-1">
          <div aria-hidden="true" :class="rankGrid" class="hidden border-b-[3px] border-black pb-3 text-[22px] font-semibold stage:grid">
            <span>#</span><span>Term</span><span>Kind</span><span>Makers</span><span>Tribelt</span>
          </div>
          <ol
            aria-label="Ranked terms"
            tabindex="0"
            class="m-0 max-h-[36rem] list-none overflow-y-auto border-t-2 border-black p-0 stage:max-h-none stage:min-h-0 stage:flex-1 stage:border-t-0"
          >
            <li
              v-for="row in ranked"
              :key="`${row.term}-${row.meta}`"
              :class="rankGrid"
              class="grid grid-cols-[1.75rem_minmax(0,1fr)_4.5rem] items-center gap-x-2 gap-y-0.5 border-b border-grey py-2 text-sm stage:min-h-[72px] stage:py-1.5 stage:text-[26px]"
            >
              <span class="col-start-1 row-span-2 row-start-1 self-start text-muted stage:col-start-auto stage:row-span-1 stage:row-start-auto stage:self-center">{{ row.rank }}</span>
              <span class="col-start-2 row-start-1 flex min-w-0 flex-col stage:col-start-auto stage:row-start-auto">
                <span class="truncate font-medium stage:text-[28px]">
                  <template v-for="(part, index) in highlight(row.term, view.q)" :key="index">
                    <mark v-if="part.match" class="bg-orange/30 text-black">{{ part.text }}</mark>
                    <template v-else>{{ part.text }}</template>
                  </template>
                </span>
                <span class="truncate text-xs text-muted stage:text-[22px]">{{ row.meta }}</span>
              </span>
              <span class="col-start-2 row-start-2 inline-flex items-center gap-2 text-xs stage:col-start-auto stage:row-start-auto stage:gap-2.5 stage:text-[26px]">
                <span aria-hidden="true" class="size-2.5 flex-none rounded-full stage:size-[18px]" :class="tierFill(row.tier)"></span>{{ row.kind }}
              </span>
              <span class="col-start-3 row-start-1 flex items-center gap-2 stage:col-start-auto stage:row-start-auto stage:gap-2.5">
                <span class="w-5 text-right font-semibold stage:w-8">{{ row.makers }}<span class="sr-only"> makers</span></span>
                <span aria-hidden="true" class="h-2 flex-1 bg-mist stage:h-3">
                  <span class="block h-full bg-black" :style="{ width: percent(row.share) }"></span>
                </span>
              </span>
              <span
                class="col-start-3 row-start-2 text-xs font-medium stage:col-start-auto stage:row-start-auto stage:text-[26px]"
                :class="row.tribelt ? 'text-ink-2' : 'text-orange-ink'"
              >{{ row.tribelt ? 'Uses it' : 'Lacks it' }}</span>
            </li>
          </ol>
        </div>
      </section>
    </div>
  </SectionView>
</template>
