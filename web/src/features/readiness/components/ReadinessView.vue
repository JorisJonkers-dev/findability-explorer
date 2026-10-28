<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { AiReadiness, AiReadinessRow, ReadinessCheckKind } from '../../../infrastructure/api/types.gen'
import { percent } from '../../../shared/percent'
import { vScrollHint } from '../../../shared/scrollHint'
import { directionOf, nextSort, type Sort } from '../../../shared/sort'
import SortButton from '../../../shared/ui/SortButton.vue'
import { hostOf } from '../../prompt-test/domain/grid'
import { formatValue } from '../../catching-up/domain/standings'
import SectionView from '../../shell/components/SectionView.vue'
import { companyPath } from '../../shell/domain/sections'
import { useAiSearchSummary } from '../../ai-search/composables/useAiSearchSummary'
import { usePlan } from '../../plan/composables/usePlan'
import { useMe } from '../../shell/composables/useMe'
import { useReadinessCheck, useReadinessFiles, useReadinessTable } from '../composables/useReadiness'
import { ACCESS_LEGEND, accessMark, hasMark, type MarkTone, readinessCheckLine, scanNote } from '../domain/overview'
import { bestSites, CHECK_WHAT, checkGroups, fixFor, LETS_IN, readinessFigures, readinessTitle, siteChecks, STATE_WORDS, tribeltStatus } from '../domain/checks'
import { BAND_LABELS, BOTS, COLUMNS, type Gap, GAPS, markedLines, policySummary, type ReadinessColumn, sitemapCoverage, tableRows } from '../domain/readiness'

const route = useRoute()
const router = useRouter()
const { rows, check, isLoading: tableLoading, error: tableError } = useReadinessTable()
const { isAdmin } = useMe()
const { isStarting, error: checkError, start: startCheck } = useReadinessCheck()
const checkLine = computed(() => readinessCheckLine(check.value))
// The headline, the checks summary and the count of sites are the server's, from the AI search summary.
const { summary: aiSearch, isLoading: summaryLoading, error: summaryError } = useAiSearchSummary()
const isLoading = computed(() => tableLoading.value || summaryLoading.value)
const error = computed(() => tableError.value ?? summaryError.value)

// Sorted by a column while one is picked; Tribelt and the mirror stay on top whatever the sort.
const sort = ref<Sort<ReadinessColumn>>(null)
const gap = ref<Gap | ''>('')
const bands = computed(() => tableRows(rows.value, sort.value, gap.value === '' ? null : gap.value))

function sortBy(column: ReadinessColumn): void {
  const c = COLUMNS.find((x) => x.key === column)
  sort.value = nextSort(sort.value, column, c?.first ?? ('desc'))
}

/** A column head that sorts the table, and whether its cells are symbols, centred, or text. */
const HEADS: { key: ReadinessColumn; label: string; mark: boolean }[] = [
  { key: 'sitemaps', label: 'Sitemap', mark: false },
  { key: 'llmsTxt', label: 'llms.txt', mark: true },
  { key: 'llmsFullTxt', label: 'llms-full', mark: true },
  { key: 'markdown', label: 'Markdown', mark: true },
  { key: 'product', label: 'Structured data and platform', mark: false },
]

/**
 * A sortable head: a soft mist patch on hover, orange while the table is sorted by it. A long
 * head wraps, at a crawler's hyphen, rather than widening its column.
 */
function headClass(column: ReadinessColumn, centred = false): string {
  const on = directionOf(sort.value, column) !== null
  return `-mx-1.5 rounded-lg px-1.5 py-1 hover:bg-mist hover:text-black [&>span]:whitespace-normal stage:-mx-3 stage:rounded-xl stage:px-3 stage:py-2 ${centred ? 'text-center' : 'text-left'} ${on ? 'text-orange-ink' : ''}`
}
const { plan } = usePlan()
const sites = computed(() => aiSearch.value?.sites ?? 0)
const title = computed(() => (aiSearch.value && !error.value ? readinessTitle(aiSearch.value.ranking, aiSearch.value.readinessChecks) : 'How ready the sites are for AI assistants'))
const figures = computed(() => (aiSearch.value ? readinessFigures(aiSearch.value) : []))
const groups = computed(() => checkGroups(aiSearch.value, aiSearch.value?.readinessChecks ?? 0))
const best = computed(() => bestSites(aiSearch.value?.ranking ?? []))
const others = computed(() => Math.max(0, sites.value - best.value.sites.length))

// The check picked in the bar list, and the sites whose five checks are open; the first check Tribelt fails is open to start with.
const picked = ref<ReadinessCheckKind | undefined>(undefined)
const allChecks = computed(() => groups.value.flatMap((g) => g.checks))
const pickedCheck = computed(() => allChecks.value.find((c) => c.kind === picked.value) ?? allChecks.value.find((c) => c.tribelt === false) ?? allChecks.value[0])
const fix = computed(() => (pickedCheck.value ? fixFor(pickedCheck.value.kind, plan.value) : undefined))
const open = ref<Record<string, boolean>>({})
const rowOf = (id: string): AiReadinessRow | undefined => rows.value.find((r) => r.competitorId === id)
// The site picked lives in the address; Tribelt's own until another is picked.
const selected = computed<AiReadinessRow | undefined>(
  () => rows.value.find((r) => r.competitorId === route.query.site) ?? rows.value.find((r) => r.role === 'tribelt'),
)
const { files, isLoading: filesLoading, missing } = useReadinessFiles(
  computed(() => selected.value?.scan?.id),
  toRef(() => true),
)
const robots = computed(() => markedLines(files.value?.robotsTxt ?? ''))

/** The square a symbol sits in, the same size in every column so the rows line up. */
const CHIP = 'inline-flex size-7 items-center justify-center rounded-md align-middle stage:size-12 stage:rounded-lg'

/**
 * How each symbol looks. Its shape says what it means and the colour only supports it: a calm
 * tick for the many sites that let a crawler in, so a partial or full block stands out.
 */
const TONE: Record<MarkTone, string> = {
  ok: 'text-orange-ink',
  partly: 'border-2 border-orange-soft bg-white text-orange-ink',
  blocked: 'bg-black text-white',
  no: 'bg-grey text-ink-2',
  unknown: 'text-muted',
}

/** Each symbol drawn as a stroke, so it looks the same in every font. */
const STROKE: Record<string, string> = {
  '✓': 'M3.5 8.5 6.5 11.5 12.5 4.5',
  '~': 'M3 9.5C4.5 6.5 6.5 6.5 8 8s3.5 1.5 5-1.5',
  '✕': 'M4.5 4.5 11.5 11.5M11.5 4.5 4.5 11.5',
  '–': 'M5 8H11',
}

/** The file columns that show a tick or a cross, after the sitemap. */
const FILES = [
  { key: 'llmsTxt', what: 'llms.txt', has: (r: AiReadiness) => r.llmsTxt.present },
  { key: 'llmsFullTxt', what: 'llms-full.txt', has: (r: AiReadiness) => r.llmsFullTxt },
  { key: 'markdown', what: 'Markdown pages', has: (r: AiReadiness) => r.markdown },
] as const

function hostOfRow(r: AiReadinessRow): string | undefined {
  return r.scan ? hostOf(r.scan.url) : undefined
}

/** Every head stays on top while the table scrolls down, on white so the rows slide under it. */
const HEAD = 'sticky top-0 border-b-2 border-b-black bg-white py-2 align-bottom text-[13px] leading-tight font-semibold text-ink-2 stage:py-3 stage:text-[21px]'

/** Every body cell: the same height, its line below, its content centred on the line. */
const CELL = 'h-14 border-b border-b-grey py-1.5 align-middle stage:h-[88px] stage:py-2'

function isPicked(r: AiReadinessRow): boolean {
  return selected.value?.competitorId === r.competitorId
}

function pick(r: AiReadinessRow): void {
  void router.replace({ query: { ...route.query, site: r.competitorId } })
}
</script>

<template>
  <SectionView eyebrow="02 AI search, AI readiness" :title="title" :back="{ to: '/ai-search', label: 'Back to AI search' }">
    <template #lede>
      <p class="m-0 max-w-[62ch] text-base text-ink-2 text-pretty sm:text-lg stage:max-w-[2300px] stage:text-[40px]">Each check is something a website can do to help search engines and AI assistants read it. The newest scan checked the sites of Tribelt and its competitors.</p>
    </template>
    <template v-if="!isLoading && !error" #aside>
      <div class="flex flex-col gap-3 stage:items-end stage:gap-5">
        <p class="m-0 max-w-[60ch] text-sm text-muted text-pretty stage:max-w-[1500px] stage:text-right stage:text-[26px]">{{ scanNote(rows, sites) }}</p>
        <div v-if="isAdmin" class="flex flex-wrap items-center gap-3 stage:flex-row-reverse stage:gap-6">
          <button
            type="button"
            class="inline-flex min-h-11 items-center rounded-full border-2 border-black bg-white px-4 text-sm font-semibold hover:bg-black hover:text-white disabled:opacity-60 disabled:hover:bg-white disabled:hover:text-black stage:min-h-16 stage:px-7 stage:text-[24px]"
            :disabled="isStarting || check?.running"
            @click="startCheck"
          >
            Check AI readiness again
          </button>
          <p v-if="checkLine" class="m-0 text-sm font-medium stage:text-[24px]" aria-live="polite">{{ checkLine }}</p>
          <p v-if="checkError" role="alert" class="m-0 text-sm font-medium text-orange-ink stage:text-[24px]">{{ checkError }}</p>
        </div>
      </div>
    </template>

    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the AI readiness…</p>
    <div v-else class="flex flex-col gap-14 stage:min-h-0 stage:gap-20 stage:overflow-y-auto stage:pr-6">
      <dl aria-label="Key figures" class="m-0 grid gap-4 sm:grid-cols-3 stage:gap-8">
        <div
          v-for="f in figures"
          :key="f.text"
          class="flex items-baseline gap-4 rounded-[20px] px-6 py-5 stage:gap-8 stage:rounded-[32px] stage:px-12 stage:py-8"
          :class="f.accent ? 'bg-orange-pale' : 'bg-mist'"
        >
          <dd class="order-1 m-0 font-display text-4xl font-semibold stage:text-[96px]" :class="f.accent ? 'text-orange-ink' : ''">{{ f.big }}</dd>
          <dt class="order-2 font-semibold stage:text-[34px]">{{ f.text }}</dt>
        </div>
      </dl>

      <div class="grid gap-8 stage:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] stage:gap-20">
        <section aria-labelledby="checks-title" class="flex flex-col gap-4 rounded-[28px] border-2 border-grey p-6 stage:gap-7 stage:rounded-[48px] stage:border-[3px] stage:p-14">
          <h2 id="checks-title" class="m-0 font-display text-2xl font-semibold stage:text-[52px]">How many of the {{ sites }} sites pass each check</h2>
          <p class="m-0 text-ink-2 stage:text-[30px]">Pick a check to see what it means for Tribelt.</p>
          <div v-for="g in groups" :key="g.title" role="group" :aria-label="g.aria" class="flex flex-col gap-1 stage:gap-2">
            <span class="px-2.5 pt-2 text-[13px] font-bold tracking-[0.06em] text-muted uppercase stage:px-5 stage:pt-4 stage:text-[24px]">{{ g.title }}</span>
            <button
              v-for="c in g.checks"
              :key="c.kind"
              type="button"
              :aria-pressed="pickedCheck?.kind === c.kind"
              :aria-label="c.aria"
              class="grid min-h-12 w-full cursor-pointer grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_2.5rem] items-center gap-3 rounded-xl border-2 bg-transparent px-2.5 py-1 text-left text-black hover:bg-mist sm:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_2.5rem_9.5rem] stage:min-h-[96px] stage:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_100px_360px] stage:gap-8 stage:rounded-[24px] stage:border-[3px] stage:px-5"
              :class="pickedCheck?.kind === c.kind ? 'border-black bg-white' : 'border-transparent'"
            >
              <span class="font-semibold stage:text-[32px]">{{ c.label }}</span>
              <span aria-hidden="true" class="relative block h-4 overflow-hidden rounded-lg bg-grey stage:h-8 stage:rounded-2xl">
                <span class="absolute inset-y-0 left-0 rounded-lg" :class="pickedCheck?.kind === c.kind ? 'bg-orange' : 'bg-blue-grey'" :style="{ width: c.passing > 0 ? percent(Math.max(c.share, 0.015)) : '0%' }"></span>
              </span>
              <span class="text-right text-base font-extrabold stage:text-[36px]">{{ c.passing }}</span>
              <span class="hidden text-right text-sm font-bold sm:block stage:text-[28px]" :class="c.tribelt === false ? 'text-orange-ink' : c.tribelt === null ? 'text-muted' : ''">{{ c.status }}</span>
            </button>
          </div>
        </section>

        <aside v-if="pickedCheck" aria-live="polite" class="flex flex-col gap-4 self-start rounded-[28px] bg-orange-pale p-6 stage:gap-7 stage:rounded-[48px] stage:p-14">
          <span class="text-sm font-bold tracking-[0.06em] text-orange-ink uppercase stage:text-[26px]">The check you picked</span>
          <h2 class="m-0 font-display text-3xl leading-tight font-semibold stage:text-[60px]">{{ pickedCheck.label }}</h2>
          <p class="m-0 text-lg leading-relaxed stage:text-[34px]">{{ CHECK_WHAT[pickedCheck.kind] }}</p>
          <span
            class="self-start rounded-full bg-white px-4 py-2 font-bold stage:px-8 stage:py-4 stage:text-[30px]"
            :class="pickedCheck.tribelt === false ? 'text-orange-ink' : ''"
          >
            {{ tribeltStatus(pickedCheck.tribelt) }}{{ pickedCheck.tribelt === null ? '. tribelt.nl has not been scanned yet.' : '.' }}
          </span>
          <RouterLink
            v-if="fix"
            :to="fix.to"
            class="inline-flex min-h-12 items-center self-start rounded-full bg-black px-6 font-bold text-white no-underline hover:bg-orange-ink hover:text-white stage:min-h-[96px] stage:px-12 stage:text-[32px]"
          >
            {{ fix.label }}
          </RouterLink>
        </aside>
      </div>

      <div class="grid gap-10 stage:grid-cols-[minmax(0,1fr)_minmax(0,1.15fr)] stage:gap-20">
        <section aria-labelledby="best-title" class="flex flex-col gap-4 stage:gap-6">
          <h2 id="best-title" class="m-0 font-display text-2xl font-semibold stage:text-[52px]">The sites with the best score</h2>
          <ul class="m-0 list-none border-t-2 border-black p-0 stage:border-t-4">
            <li v-for="s in best.sites" :key="s.competitorId" class="border-b border-grey">
              <div class="grid min-h-14 grid-cols-[minmax(0,1fr)_5rem_auto] items-center gap-3 px-2 py-1 stage:min-h-[104px] stage:grid-cols-[minmax(0,1fr)_200px_360px] stage:gap-6 stage:px-4">
                <span class="text-lg stage:text-[38px]" :class="s.role === 'tribelt' ? 'font-extrabold text-orange-ink' : 'font-semibold'">{{ s.name }}</span>
                <span class="text-right text-lg font-extrabold stage:text-[38px]">{{ formatValue('readinessChecks', s.score) }} of {{ aiSearch?.readinessChecks }}</span>
                <button
                  type="button"
                  :aria-expanded="open[s.competitorId] === true"
                  :aria-label="`${open[s.competitorId] ? 'Hide' : 'Show'} the checks for ${s.name}`"
                  class="min-h-11 cursor-pointer justify-self-end rounded-full border-2 border-silver bg-white px-4 text-sm font-semibold hover:bg-mist stage:min-h-[80px] stage:border-[3px] stage:px-8 stage:text-[28px]"
                  @click="open = { ...open, [s.competitorId]: !open[s.competitorId] }"
                >
                  {{ open[s.competitorId] ? 'Hide the checks' : 'Show the checks' }}
                </button>
              </div>
              <ul v-if="open[s.competitorId]" class="m-0 flex list-none flex-col gap-1.5 px-4 pt-0 pb-4 stage:gap-3 stage:px-8 stage:pb-8 stage:text-[30px]">
                <li v-for="c in siteChecks(rowOf(s.competitorId) ?? { readiness: undefined })" :key="c.label" class="flex items-center gap-3">
                  <span class="w-20 font-extrabold stage:w-36" :class="c.state === 'no' ? 'text-orange-ink' : c.state === 'unknown' ? 'text-muted' : ''">{{ STATE_WORDS[c.state] }}</span>
                  {{ c.label }}
                </li>
              </ul>
            </li>
          </ul>
          <p v-if="others > 0" class="m-0 text-sm text-muted stage:text-[26px]">
            The other {{ others }} {{ others === 1 ? 'site passes' : 'sites pass' }} fewer. Every site, with every column, is further down.
          </p>
        </section>

        <section aria-labelledby="crawlers-title" class="flex flex-col gap-4 stage:gap-6">
          <h2 id="crawlers-title" class="m-0 font-display text-2xl font-semibold stage:text-[52px]">Which crawlers can read tribelt.nl</h2>
          <div class="overflow-x-auto">
            <table class="w-full border-collapse text-sm stage:text-[28px]">
              <caption class="sr-only">Each crawler, what it does for Tribelt, and whether tribelt.nl lets it in</caption>
              <thead>
                <tr class="border-b-2 border-black">
                  <th scope="col" class="py-2 text-left font-semibold text-muted">Crawler</th>
                  <th scope="col" class="py-2 pl-3 text-left font-semibold text-muted">What it does for Tribelt</th>
                  <th scope="col" class="py-2 pl-3 text-right font-semibold text-muted">tribelt.nl</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="c in aiSearch?.crawlers ?? []" :key="c.bot" class="border-b border-grey">
                  <th scope="row" class="py-2 text-left font-bold stage:py-4">{{ c.bot }}<span class="block text-[13px] font-normal text-muted stage:text-[22px]">{{ c.lab }}</span></th>
                  <td class="py-2 pl-3 stage:py-4">
                    <span class="inline-flex items-center gap-2 stage:gap-4">
                      <span aria-hidden="true" class="size-3 flex-none rounded-full stage:size-5" :class="c.feeds === 'today' ? 'bg-orange' : 'bg-blue-grey'"></span>
                      {{ c.gets }}<span class="sr-only">{{ c.feeds === 'today' ? ', shows up in answers today' : ', teaches future models' }}</span>
                    </span>
                  </td>
                  <td class="py-2 pl-3 text-right font-bold stage:py-4" :class="c.access === 'blocked' ? 'text-orange-ink' : ''">{{ LETS_IN[c.access ?? 'unknown'] }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <ul aria-label="Legend" class="m-0 flex list-none flex-wrap gap-x-5 gap-y-2 p-0 text-sm text-ink-2 stage:gap-x-10 stage:text-[26px]">
            <li class="inline-flex items-center gap-2"><span aria-hidden="true" class="size-3 rounded-full bg-orange stage:size-5"></span>Shows up in answers today</li>
            <li class="inline-flex items-center gap-2"><span aria-hidden="true" class="size-3 rounded-full bg-blue-grey stage:size-5"></span>Teaches future models</li>
          </ul>
        </section>
      </div>

      <section aria-labelledby="sites-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
        <div class="flex flex-col gap-4 sm:flex-row sm:flex-wrap sm:items-end sm:justify-between">
          <h2 id="sites-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[52px]">Every site, every column</h2>
          <label class="flex flex-col gap-1.5 text-sm text-ink-2 sm:flex-row sm:items-center sm:gap-3 stage:text-[24px]">
            Show
            <select v-model="gap" class="h-11 w-full cursor-pointer truncate rounded-full sm:w-80 stage:w-[760px] border-2 border-grey bg-white px-4 text-base text-black hover:border-black stage:h-16 stage:px-6 stage:text-[24px]">
              <option value="">All companies</option>
              <option v-for="g in GAPS" :key="g.value" :value="g.value">{{ g.label }}</option>
            </select>
          </label>
        </div>
        <ul aria-label="What the crawler columns say" class="m-0 flex list-none flex-wrap gap-x-6 gap-y-2 rounded-xl bg-mist p-0 px-3 py-2.5 text-sm text-ink-2 stage:gap-x-12 stage:rounded-2xl stage:px-6 stage:py-4 stage:text-[22px]">
          <li v-for="l in ACCESS_LEGEND" :key="l.mark" class="flex items-center gap-2 stage:gap-3">
            <span aria-hidden="true" :data-mark="l.mark" :class="[CHIP, TONE[l.tone], 'flex-none']"><svg viewBox="0 0 16 16" class="size-4 stage:size-8" fill="none" stroke="currentColor" stroke-width="2.25" stroke-linecap="round" stroke-linejoin="round"><path :d="STROKE[l.mark]" /></svg></span>
            <span>{{ l.meaning }}</span>
          </li>
        </ul>
        <div v-scroll-hint class="scroll-hint relative max-h-[75svh] overflow-auto stage:max-h-none stage:min-h-0 stage:flex-1">
          <table class="w-full border-separate border-spacing-0 text-sm stage:text-[22px]">
            <caption class="sr-only">AI readiness per site: each crawler's access, the sitemap, llms.txt, llms-full.txt, Markdown pages, structured data and platform</caption>
            <thead>
              <tr>
                <th scope="col" :class="[HEAD, 'left-0 z-30 border-r border-r-silver pl-3 text-left stage:pl-5']">
                  <SortButton label="Company" :direction="directionOf(sort, 'name')" :class="headClass('name')" @sort="sortBy('name')">Company</SortButton>
                </th>
                <th v-for="b in BOTS" :key="b.token" scope="col" :class="[HEAD, 'z-20 px-1 text-center stage:px-1.5']">
                  <SortButton :label="`${b.token} (${b.owner})`" align="center" :direction="directionOf(sort, `bot:${b.token}`)" :class="headClass(`bot:${b.token}`, true)" @sort="sortBy(`bot:${b.token}`)">{{ b.token }}</SortButton>
                </th>
                <th v-for="h in HEADS" :key="h.key" scope="col" :class="[HEAD, 'z-20 px-3 stage:px-3.5', h.mark ? 'text-center' : 'text-left']">
                  <SortButton :label="h.label" :align="h.mark ? 'center' : 'start'" :direction="directionOf(sort, h.key)" :class="headClass(h.key, h.mark)" @sort="sortBy(h.key)">{{ h.label }}</SortButton>
                </th>
              </tr>
            </thead>
            <tbody v-for="band in bands" :key="band.band" :aria-label="BAND_LABELS[band.band]">
              <tr>
                <th :colspan="BOTS.length + 6" scope="colgroup" class="border-b border-grey bg-mist p-0 text-left">
                  <span class="sticky left-0 inline-block px-3 py-2 text-xs font-semibold tracking-[0.06em] text-orange-ink uppercase stage:px-5 stage:py-3 stage:text-[20px]">{{ BAND_LABELS[band.band] }}</span>
                </th>
              </tr>
              <tr
                v-for="r in band.rows"
                :key="r.competitorId"
                class="group cursor-pointer"
                :class="isPicked(r) ? 'bg-orange-pale' : 'bg-white hover:bg-mist'"
                @click="pick(r)"
              >
                <th
                  scope="row"
                  :class="[CELL, 'sticky left-0 z-10 max-w-44 border-r border-r-silver bg-inherit pr-2 pl-3 text-left font-medium sm:max-w-none stage:pr-4 stage:pl-5', isPicked(r) ? 'shadow-[inset_4px_0_0_var(--color-orange)] stage:shadow-[inset_8px_0_0_var(--color-orange)]' : '']"
                >
                  <span class="flex items-center gap-2 stage:gap-3">
                    <button
                      type="button"
                      :aria-pressed="isPicked(r)"
                      class="min-w-0 cursor-pointer rounded border-0 bg-transparent p-0 text-left font-medium decoration-orange decoration-2 underline-offset-4 group-hover:underline hover:text-orange-ink sm:whitespace-nowrap"
                      @click.stop="pick(r)"
                    >
                      {{ r.name }}<span v-if="hostOfRow(r)" class="block truncate text-xs font-normal text-muted stage:text-[18px]">{{ hostOfRow(r) }}</span>
                    </button>
                    <a
                      v-if="r.scan"
                      :href="r.scan.url"
                      target="_blank"
                      rel="noopener"
                      :aria-label="`Open ${hostOfRow(r)} in a new tab`"
                      :title="`Open ${hostOfRow(r)} in a new tab`"
                      class="inline-flex size-7 flex-none items-center justify-center rounded-full text-ink-2 hover:bg-white hover:text-black stage:size-11"
                      @click.stop
                    >
                      <svg aria-hidden="true" class="size-3.5 stage:size-6" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M5 2H2v8h8V7M7 2h3v3M10 2 5.5 6.5" /></svg>
                    </a>
                  </span>
                </th>
                <td v-for="b in BOTS" :key="b.token" :class="[CELL, 'px-1 text-center stage:px-1.5']" :title="accessMark(r.readiness, b.token).text">
                  <span aria-hidden="true" :data-mark="accessMark(r.readiness, b.token).mark" :class="[CHIP, TONE[accessMark(r.readiness, b.token).tone]]"><svg viewBox="0 0 16 16" class="size-4 stage:size-8" fill="none" stroke="currentColor" stroke-width="2.25" stroke-linecap="round" stroke-linejoin="round"><path :d="STROKE[accessMark(r.readiness, b.token).mark]" /></svg></span><span class="sr-only">{{ accessMark(r.readiness, b.token).text }}</span>
                </td>
                <template v-if="r.readiness">
                  <td :class="[CELL, 'px-3 whitespace-nowrap stage:px-3.5']">
                    {{ r.readiness.sitemap.files > 0 ? sitemapCoverage(r.readiness) : 'None' }}
                    <span class="block text-xs text-muted stage:text-[18px]">{{ r.readiness.sitemap.languages.map((l) => l.toUpperCase()).join(' ') || 'No language' }}</span>
                  </td>
                  <td v-for="f in FILES" :key="f.key" :class="[CELL, 'px-3 text-center stage:px-3.5']" :title="hasMark(f.has(r.readiness), f.what).text">
                    <span aria-hidden="true" :data-mark="hasMark(f.has(r.readiness), f.what).mark" :class="[CHIP, TONE[hasMark(f.has(r.readiness), f.what).tone]]"><svg viewBox="0 0 16 16" class="size-4 stage:size-8" fill="none" stroke="currentColor" stroke-width="2.25" stroke-linecap="round" stroke-linejoin="round"><path :d="STROKE[hasMark(f.has(r.readiness), f.what).mark]" /></svg></span><span class="sr-only">{{ hasMark(f.has(r.readiness), f.what).text }}</span>
                  </td>
                  <td :class="[CELL, 'min-w-48 px-3 stage:min-w-[300px] stage:px-3.5']">
                    {{ r.readiness.homeRead ? r.readiness.structuredData.join(', ') || 'No structured data' : 'Could not read the home page' }}
                    <span class="block text-xs text-muted stage:text-[18px]">{{ r.readiness.stack.cms ?? 'Unknown platform' }}</span>
                  </td>
                </template>
                <td v-else colspan="5" :class="[CELL, 'px-3 text-muted stage:px-3.5']">Not scanned yet</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="selected" aria-live="polite" class="flex flex-col gap-5 rounded-2xl bg-mist p-5 stage:grid stage:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] stage:gap-12 stage:rounded-[28px] stage:p-10">
          <div class="flex min-w-0 flex-col gap-2">
            <span class="text-sm font-semibold tracking-[0.04em] text-orange-ink uppercase stage:text-[22px]">{{ selected.name }}'s robots.txt</span>
            <p v-if="!selected.scan || missing" class="m-0 text-ink-2">No scan has recorded its robots.txt yet.</p>
            <p v-else-if="filesLoading" class="m-0 text-muted">Loading…</p>
            <p v-else-if="robots.length === 0" class="m-0 text-ink-2">The site serves no robots.txt.</p>
            <pre v-else tabindex="0" class="m-0 max-h-64 overflow-auto rounded-xl bg-white p-3 text-xs stage:max-h-[360px] stage:p-5 stage:text-[20px]"><code><span
              v-for="(line, i) in robots"
              :key="i"
              :class="line.ai ? 'bg-orange/25 font-semibold' : ''"
              class="block"
            >{{ line.text || ' ' }}</span></code></pre>
          </div>
          <div class="flex flex-col gap-3">
            <span class="text-sm font-semibold tracking-[0.04em] text-orange-ink uppercase stage:text-[22px]">What it means</span>
            <p class="m-0 stage:text-[28px]">{{ policySummary(selected.readiness) }}</p>
            <RouterLink :to="`${companyPath(selected.competitorId)}/history`" class="self-start font-medium underline underline-offset-4 stage:text-[24px]">
              See all its raw files and how they changed
            </RouterLink>
          </div>
        </div>
      </section>
    </div>
  </SectionView>
</template>
