<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { AiReadinessRow } from '../../../infrastructure/api/types.gen'
import { percent } from '../../../shared/percent'
import { vScrollHint } from '../../../shared/scrollHint'
import { directionOf, nextSort, type Sort } from '../../../shared/sort'
import SortButton from '../../../shared/ui/SortButton.vue'
import { hostOf } from '../../prompt-test/domain/grid'
import SectionView from '../../shell/components/SectionView.vue'
import { AI_SEARCH_TABS, companyPath } from '../../shell/domain/sections'
import { useReadinessFiles, useReadinessTable } from '../composables/useReadiness'
import { accessMark, checkSummary, readinessHeadline, scanned, scanRange } from '../domain/overview'
import { BAND_LABELS, BOTS, COLUMNS, type Gap, GAPS, markedLines, policySummary, type ReadinessColumn, tableRows } from '../domain/readiness'

const route = useRoute()
const router = useRouter()
const { rows, isLoading, error } = useReadinessTable()

// Sorted by a column while one is picked; Tribelt and the mirror stay on top whatever the sort.
const sort = ref<Sort<ReadinessColumn>>(null)
const gap = ref<Gap | ''>('')
const bands = computed(() => tableRows(rows.value, sort.value, gap.value === '' ? null : gap.value))

function sortBy(column: ReadinessColumn): void {
  const c = COLUMNS.find((x) => x.key === column)
  sort.value = nextSort(sort.value, column, c?.first ?? ('desc'))
}

/** A column head that sorts the table. */
const HEADS: { key: ReadinessColumn; label: string }[] = [
  { key: 'sitemaps', label: 'Sitemap' },
  { key: 'llmsTxt', label: 'llms.txt' },
  { key: 'llmsFullTxt', label: 'llms-full' },
  { key: 'markdown', label: 'Markdown' },
  { key: 'product', label: 'Structured data and platform' },
]
const summary = computed(() => checkSummary(rows.value))
const sites = computed(() => scanned(rows.value).length)
// The site picked lives in the address; Tribelt's own until another is picked.
const selected = computed<AiReadinessRow | undefined>(
  () => rows.value.find((r) => r.competitorId === route.query.site) ?? rows.value.find((r) => r.role === 'tribelt'),
)
const { files, isLoading: filesLoading, missing } = useReadinessFiles(
  computed(() => selected.value?.scan?.id),
  toRef(() => true),
)
const robots = computed(() => markedLines(files.value?.robotsTxt ?? ''))

const TONE = { ok: 'text-orange-ink', partly: 'text-orange-ink', blocked: 'font-bold text-black', unknown: 'text-muted' } as const

function hostOfRow(r: AiReadinessRow): string | undefined {
  return r.scan ? hostOf(r.scan.url) : undefined
}

function pick(r: AiReadinessRow): void {
  void router.replace({ query: { ...route.query, site: r.competitorId } })
}
</script>

<template>
  <SectionView section="ai-search" :title="isLoading || error ? 'Who lets search engines and AI assistants in.' : readinessHeadline(rows)" :tabs="AI_SEARCH_TABS">
    <template v-if="!isLoading && !error" #aside>
      <p class="m-0 text-sm text-muted stage:text-right stage:text-[26px]">
        {{ sites }} sites.<template v-if="scanRange(rows)"> {{ scanRange(rows) }}.</template>
      </p>
    </template>

    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the AI readiness…</p>
    <div v-else class="flex flex-col gap-12 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,0.7fr)_minmax(0,2fr)] stage:gap-24">
      <section aria-labelledby="sum-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-7">
        <h2 id="sum-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">How many of the {{ sites }} are set up for AI</h2>
        <table class="w-full border-collapse text-sm stage:text-[26px]">
          <caption class="sr-only">For each check, how many sites pass it, and whether Tribelt does</caption>
          <thead>
            <tr class="border-b-2 border-black">
              <th scope="col" class="py-2 text-left font-medium text-muted">Check</th>
              <th scope="col" class="py-2 pl-3 text-right font-medium text-muted">Sites</th>
              <th scope="col" class="py-2 pl-3 text-right font-medium text-muted">Tribelt</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in summary" :key="c.label" class="border-b border-grey">
              <th scope="row" class="py-2 text-left font-normal stage:py-3">
                {{ c.label }}
                <span aria-hidden="true" class="mt-1 block h-2 bg-mist stage:h-3">
                  <span class="block h-full bg-blue-grey" :style="{ width: percent(c.sites === 0 ? 0 : c.passing / c.sites) }"></span>
                </span>
              </th>
              <td class="py-2 pl-3 text-right font-semibold stage:py-3">{{ c.passing }}</td>
              <td class="py-2 pl-3 text-right font-semibold stage:py-3" :class="c.tribelt === false ? 'text-orange-ink' : ''">
                {{ c.tribelt === null ? 'Not scanned' : c.tribelt ? '● Yes' : '✕ No' }}
              </td>
            </tr>
          </tbody>
        </table>
      </section>

      <section aria-labelledby="sites-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
        <div class="flex flex-wrap items-baseline justify-between gap-4">
          <h2 id="sites-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Site by site</h2>
          <ul aria-label="Legend" class="m-0 flex list-none flex-wrap gap-x-5 gap-y-1 p-0 text-sm text-ink-2 stage:gap-x-8 stage:text-[24px]">
            <li><span aria-hidden="true" class="text-orange-ink">●</span> let in</li>
            <li><span aria-hidden="true" class="text-orange-ink">◐</span> partly kept out</li>
            <li><span aria-hidden="true" class="font-bold">✕</span> blocked</li>
            <li><span aria-hidden="true" class="text-muted">?</span> robots.txt could not be read</li>
          </ul>
        </div>
        <label class="flex flex-col gap-1.5 self-start text-sm text-ink-2 stage:gap-2 stage:text-[24px]">
          Show the competitors with a gap
          <select v-model="gap" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[64px] stage:px-5 stage:text-[24px]">
            <option value="">All companies</option>
            <option v-for="g in GAPS" :key="g.value" :value="g.value">{{ g.label }}</option>
          </select>
        </label>
        <div v-scroll-hint class="scroll-hint relative overflow-auto stage:min-h-0 stage:flex-1">
          <table class="w-max min-w-full border-collapse text-sm stage:text-[22px]">
            <caption class="sr-only">AI readiness per site: each crawler's access, the sitemap, llms.txt, llms-full.txt, Markdown pages, structured data and platform</caption>
            <thead>
              <tr class="border-b-2 border-black">
                <th scope="col" class="sticky left-0 z-10 bg-white px-2 py-2 text-left font-semibold">
                  <SortButton label="Company" :direction="directionOf(sort, 'name')" @sort="sortBy('name')">Company</SortButton>
                </th>
                <th v-for="b in BOTS" :key="b.token" scope="col" class="px-1 py-2 text-center align-bottom text-xs font-semibold stage:text-[18px]">
                  <SortButton :label="`${b.token} (${b.owner})`" :direction="directionOf(sort, `bot:${b.token}`)" @sort="sortBy(`bot:${b.token}`)">{{ b.token }}</SortButton>
                </th>
                <th v-for="h in HEADS" :key="h.key" scope="col" class="px-2 py-2 text-left font-semibold">
                  <SortButton :label="h.label" :direction="directionOf(sort, h.key)" @sort="sortBy(h.key)">{{ h.label }}</SortButton>
                </th>
              </tr>
            </thead>
            <tbody v-for="band in bands" :key="band.band" :aria-label="BAND_LABELS[band.band]">
              <tr>
                <th :colspan="BOTS.length + 6" scope="colgroup" class="sticky left-0 bg-mist px-2 py-1.5 text-left text-xs font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[20px]">
                  {{ BAND_LABELS[band.band] }}
                </th>
              </tr>
              <tr v-for="r in band.rows" :key="r.competitorId" class="border-b border-grey" :class="selected?.competitorId === r.competitorId ? 'bg-orange-pale' : ''">
                <th scope="row" class="sticky left-0 z-10 bg-inherit px-2 py-1.5 text-left font-medium">
                  <span class="flex items-center gap-2 bg-white px-1" :class="selected?.competitorId === r.competitorId ? 'bg-orange-pale' : ''">
                    <button
                      type="button"
                      :aria-pressed="selected?.competitorId === r.competitorId"
                      class="min-h-9 cursor-pointer border-0 bg-transparent p-0 text-left font-medium whitespace-nowrap"
                      @click="pick(r)"
                    >
                      {{ r.name }}<span v-if="hostOfRow(r)" class="block text-xs font-normal text-muted stage:text-[18px]">{{ hostOfRow(r) }}</span>
                    </button>
                    <a v-if="r.scan" :href="r.scan.url" target="_blank" rel="noopener" :aria-label="`Open ${hostOfRow(r)} in a new tab`" class="text-ink-2">
                      <svg aria-hidden="true" width="16" height="16" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M5 2H2v8h8V7M7 2h3v3M10 2 5.5 6.5" /></svg>
                    </a>
                  </span>
                </th>
                <td v-for="b in BOTS" :key="b.token" class="px-1 py-1.5 text-center" :class="TONE[accessMark(r.readiness, b.token).tone]">
                  <span aria-hidden="true">{{ accessMark(r.readiness, b.token).mark }}</span><span class="sr-only">{{ accessMark(r.readiness, b.token).text }}</span>
                </td>
                <template v-if="r.readiness">
                  <td class="px-2 py-1.5 whitespace-nowrap">
                    {{ r.readiness.sitemap.files > 0 ? `${r.readiness.sitemap.addresses.toLocaleString('en-US')} pages` : 'None' }}
                    <span class="block text-xs text-muted stage:text-[18px]">{{ r.readiness.sitemap.languages.map((l) => l.toUpperCase()).join(' ') || 'No language' }}</span>
                  </td>
                  <td class="px-2 py-1.5">{{ r.readiness.llmsTxt.present ? 'Yes' : 'No' }}</td>
                  <td class="px-2 py-1.5">{{ r.readiness.llmsFullTxt ? 'Yes' : 'No' }}</td>
                  <td class="px-2 py-1.5">{{ r.readiness.markdown ? 'Yes' : 'No' }}</td>
                  <td class="px-2 py-1.5">
                    {{ r.readiness.homeRead ? r.readiness.structuredData.join(', ') || 'None' : 'Unknown' }}
                    <span class="block text-xs text-muted stage:text-[18px]">{{ r.readiness.stack.cms ?? 'Platform unknown' }}</span>
                  </td>
                </template>
                <td v-else colspan="5" class="px-2 py-1.5 text-muted">Not scanned yet</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="selected" aria-live="polite" class="flex flex-col gap-5 rounded-2xl bg-mist p-5 stage:grid stage:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)] stage:gap-12 stage:rounded-[28px] stage:p-10">
          <div class="flex min-w-0 flex-col gap-2">
            <span class="text-sm font-semibold tracking-[0.04em] text-orange-ink uppercase stage:text-[22px]">robots.txt of {{ selected.name }}</span>
            <p v-if="!selected.scan || missing" class="m-0 text-ink-2">No scan of it recorded its robots.txt yet.</p>
            <p v-else-if="filesLoading" class="m-0 text-muted">Loading…</p>
            <p v-else-if="robots.length === 0" class="m-0 text-ink-2">The site serves no robots.txt.</p>
            <pre v-else class="m-0 max-h-64 overflow-auto rounded-xl bg-white p-3 text-xs stage:max-h-[360px] stage:p-5 stage:text-[20px]"><code><span
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
              All its raw files, and how they changed
            </RouterLink>
          </div>
        </div>
      </section>
    </div>
  </SectionView>
</template>
