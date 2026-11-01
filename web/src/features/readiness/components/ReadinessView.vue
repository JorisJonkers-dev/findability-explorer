<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { percent } from '../../../shared/percent'
import { directionOf, nextSort, type Sort } from '../../../shared/sort'
import PillButton from '../../../shared/ui/PillButton.vue'
import SortButton from '../../../shared/ui/SortButton.vue'
import { formatValue } from '../../catching-up/domain/standings'
import { usePlan } from '../../plan/composables/usePlan'
import PageFrame from '../../shell/components/PageFrame.vue'
import { useReadinessTable } from '../composables/useReadiness'
import {
  CHECK_NAMES,
  CHECK_WHY,
  checkFix,
  checkStates,
  compareChecks,
  compareIntro,
  everySite,
  platformOf,
  readinessDataLine,
  SCORED_CHECKS,
  type SiteColumn,
  type SiteFilter,
  siteName,
  siteScore,
  STATE_WORDS,
} from '../domain/checks'
import CheckMark from './CheckMark.vue'
import ReadinessSection from './ReadinessSection.vue'

/** AI readiness: Tribelt's eight checks, tribelt.nl against the competitors check by check, and every site. Checking again is in Manage. */
const router = useRouter()
const { rows, isLoading, error } = useReadinessTable()
const { plan } = usePlan()

const tribelt = computed(() => rows.value.find((r) => r.role === 'tribelt'))
const tribeltStates = computed(() => (tribelt.value?.readiness ? checkStates(tribelt.value.readiness) : undefined))
const tribeltScore = computed(() => (tribelt.value?.readiness ? formatValue('readinessChecks', siteScore(tribelt.value.readiness)) : ''))
const fixes = computed(() => SCORED_CHECKS.map((c) => (tribelt.value && tribeltStates.value?.[c] !== 'yes' ? checkFix(c, plan.value, tribelt.value.competitorId) : undefined)))
const compared = computed(() => compareChecks(rows.value))

// Every site: the filter, the search, the sort, and whether every row shows.
const LIMIT = 15
const filter = ref<SiteFilter>('competitors')
const query = ref('')
const sort = ref<Sort<SiteColumn>>({ key: 'score', direction: 'desc' })
const showAll = ref(false)
const sites = computed(() => everySite(rows.value, filter.value, sort.value, query.value))
const shown = computed(() => (showAll.value ? sites.value : sites.value.slice(0, LIMIT)))

/** The columns of Every site: what each is called, and the direction it sorts first. */
const HEADS: { key: SiteColumn; label: string; first: 'asc' | 'desc'; check: boolean }[] = [
  { key: 'name', label: 'Site', first: 'asc', check: false },
  { key: 'score', label: 'Checks passed', first: 'desc', check: false },
  ...SCORED_CHECKS.map((c) => ({ key: c, label: CHECK_NAMES[c], first: 'desc' as const, check: true })),
  { key: 'platform', label: 'Platform', first: 'asc', check: false },
]

function sortBy(key: SiteColumn, first: 'asc' | 'desc'): void {
  sort.value = nextSort(sort.value, key, first)
}

function open(id: string): void {
  void router.push(`/ai-readiness/${id}`)
}

const FILTERS: { value: SiteFilter; label: string }[] = [
  { value: 'competitors', label: 'Competitors' },
  { value: 'all', label: 'All sites' },
]

/** A column with a mark per check, which a narrow window leaves out so the table fits its box. */
const CHECK_COLUMN = 'hidden lg:table-cell'
</script>

<template>
  <PageFrame
    section="ai-readiness"
    title="AI readiness"
    purpose="How well search engines and AI assistants read, understand, and cite each website, scored out of eight checks."
    :data-line="isLoading || error ? undefined : readinessDataLine(rows)"
    :manage="{ to: '/manage/scans', label: 'Manage the scans' }"
    help-topic="websites"
  >
    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the AI readiness…</p>
    <template v-else>
      <div class="grid gap-10 min-[1100px]:grid-cols-2">
        <ReadinessSection id="tribelt-checks" title="Tribelt’s checks" :intro="tribeltStates ? `tribelt.nl passes ${tribeltScore} of the 8 checks.` : undefined">
          <p v-if="!tribeltStates" class="m-0 rounded-2xl bg-mist p-5 text-ink-2">tribelt.nl has not been scanned yet.</p>
          <ul v-else class="m-0 flex list-none flex-col gap-4 rounded-2xl bg-mist p-5">
            <li v-for="(c, i) in SCORED_CHECKS" :key="c" class="flex gap-3">
              <CheckMark :state="tribeltStates[c]" />
              <div class="flex min-w-0 flex-col gap-0.5">
                <p class="m-0"><strong>{{ CHECK_NAMES[c] }}</strong> <span class="text-sm text-muted">{{ STATE_WORDS[tribeltStates[c]] }}</span></p>
                <p class="m-0 text-sm text-ink-2">{{ CHECK_WHY[c] }}</p>
                <RouterLink v-if="fixes[i]" :to="fixes[i].to" class="self-start text-sm font-medium">{{ fixes[i].label }}</RouterLink>
              </div>
            </li>
          </ul>
        </ReadinessSection>

        <ReadinessSection id="compare" title="How tribelt.nl compares with the competitors" :intro="compared ? compareIntro(compared) : undefined">
          <p v-if="!compared" class="m-0 rounded-2xl bg-mist p-5 text-ink-2">tribelt.nl has not been scanned yet.</p>
          <div v-else class="rounded-2xl bg-mist p-3 sm:p-5">
            <table class="w-full border-collapse text-sm">
              <caption class="sr-only">Per check, whether tribelt.nl passes it, and how many competitors do</caption>
              <thead>
                <tr class="border-b border-silver text-left text-ink-2">
                  <th scope="col" class="py-2 pr-2 font-semibold">Check</th>
                  <th scope="col" class="px-2 py-2 font-semibold">tribelt.nl</th>
                  <th scope="col" class="py-2 pl-2 text-right font-semibold">Competitors that pass</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in compared.rows" :key="r.check" class="border-b border-grey last:border-b-0">
                  <th scope="row" class="py-2.5 pr-2 text-left font-medium">{{ r.name }}</th>
                  <td class="px-2 py-2.5">
                    <span class="rounded-full px-2.5 py-0.5 text-xs font-semibold" :class="r.tribelt === 'yes' ? 'bg-white text-black' : 'bg-orange-pale text-orange-ink'">{{
                      r.tribelt === 'yes' ? 'Passes' : 'Fails'
                    }}</span>
                  </td>
                  <td class="py-2.5 pl-2">
                    <span class="flex items-center justify-end gap-3">
                      <span aria-hidden="true" data-bar class="relative hidden h-2 w-20 overflow-hidden rounded-full bg-white sm:block">
                        <span class="absolute inset-y-0 left-0 rounded-full bg-orange" :style="{ width: percent(r.share) }"></span>
                      </span>
                      <span class="whitespace-nowrap tabular-nums">{{ r.passing }} of {{ compared.competitors }}</span>
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </ReadinessSection>
      </div>

      <ReadinessSection id="every-site" title="Every site" intro="Each site’s newest scan. Tribelt stays at the top. Click a row to open that site.">
        <div class="flex flex-wrap items-end gap-4">
          <label class="flex flex-col gap-1 text-sm text-ink-2">
            Search
            <input
              v-model="query"
              type="search"
              :placeholder="`Search these ${sites.length} rows`"
              class="h-11 w-60 max-w-full rounded-xl border-2 border-grey bg-white px-3 text-base text-black hover:border-black"
            />
          </label>
          <div role="group" aria-label="Show" class="flex flex-wrap items-center gap-2">
            <span class="text-sm text-ink-2">Show</span>
            <PillButton v-for="f in FILTERS" :key="f.value" size="sm" :pressed="filter === f.value" @click="filter = f.value">{{ f.label }}</PillButton>
          </div>
        </div>
        <div data-box="every-site" class="overflow-x-auto rounded-2xl bg-mist p-2 sm:p-4">
          <table class="w-full border-collapse text-sm">
            <caption class="sr-only">Each site: the checks it passes, a mark per check, and its platform</caption>
            <thead>
              <tr class="border-b border-silver text-left align-bottom text-xs text-ink-2">
                <th
                  v-for="h in HEADS"
                  :key="h.key"
                  scope="col"
                  class="px-1.5 py-2 font-semibold [&_button>span]:break-words [&_button>span]:whitespace-normal"
                  :class="[h.check ? CHECK_COLUMN : '', h.key === 'platform' ? 'hidden sm:table-cell' : '', h.key === 'score' ? 'text-right' : '']"
                >
                  <SortButton :label="h.label" :align="h.key === 'score' ? 'end' : 'start'" :direction="directionOf(sort, h.key)" class="text-left" @sort="sortBy(h.key, h.first)">{{ h.label }}</SortButton>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="s in shown"
                :key="s.competitorId"
                class="cursor-pointer border-b border-grey last:border-b-0 hover:bg-white"
                :class="s.role === 'tribelt' ? 'bg-orange-pale' : ''"
                @click="open(s.competitorId)"
              >
                <th scope="row" class="px-1.5 py-2.5 text-left font-medium">
                  <RouterLink :to="`/ai-readiness/${s.competitorId}`" @click.stop>{{ siteName(s) }}</RouterLink>
                </th>
                <td class="px-1.5 py-2.5">
                  <span class="flex items-center justify-end gap-2">
                    <span aria-hidden="true" class="relative hidden h-1.5 w-12 overflow-hidden rounded-full bg-white sm:block">
                      <span class="absolute inset-y-0 left-0 rounded-full bg-orange" :style="{ width: percent(siteScore(s.readiness) / 8) }"></span>
                    </span>
                    <strong class="tabular-nums">{{ formatValue('readinessChecks', siteScore(s.readiness)) }}/8</strong>
                  </span>
                </td>
                <td v-for="c in SCORED_CHECKS" :key="c" class="px-1.5 py-2.5" :class="CHECK_COLUMN">
                  <CheckMark :state="checkStates(s.readiness)[c]" :label="`${CHECK_NAMES[c]}, ${STATE_WORDS[checkStates(s.readiness)[c]]}`" />
                </td>
                <td class="hidden px-1.5 py-2.5 sm:table-cell">{{ platformOf(s.readiness) }}</td>
              </tr>
              <tr v-if="shown.length === 0">
                <td :colspan="HEADS.length" class="px-1.5 py-4 text-ink-2">No sites match this filter.</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="sites.length > LIMIT" class="m-0 flex flex-wrap items-center gap-3 text-sm text-ink-2">
          <span>Showing {{ shown.length }} of {{ sites.length }}.</span>
          <button type="button" class="cursor-pointer border-0 bg-transparent p-0 font-medium text-orange-ink underline underline-offset-4" @click="showAll = !showAll">
            {{ showAll ? 'Show fewer' : 'Show all' }}
          </button>
        </p>
      </ReadinessSection>
    </template>
  </PageFrame>
</template>
