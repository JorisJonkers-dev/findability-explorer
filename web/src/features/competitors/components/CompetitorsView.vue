<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import type { BeltFamily } from '../../../infrastructure/api/types.gen'
import { directionOf, nextSort, type Sort, type SortDirection } from '../../../shared/sort'
import FrameSection from '../../../shared/ui/FrameSection.vue'
import SortButton from '../../../shared/ui/SortButton.vue'
import { TABLE } from '../../../shared/ui/table'
import { PRODUCT_FAMILIES } from '../../../shared/vocabulary'
import ScanForm from '../../scan/components/ScanForm.vue'
import PageFrame from '../../shell/components/PageFrame.vue'
import { useMe } from '../../shell/composables/useMe'
import { companyPath } from '../../shell/domain/sections'
import { familyName } from '../../terminology/domain/searchTerms'
import { useCompanies } from '../composables/useCompanies'
import { useCompetitorBoard } from '../composables/useCompetitorBoard'
import { oxford, scoreText, shareText } from '../domain/compare'
import { bestCompetitor, type CompetitorColumn, competitorTable, countriesOf, familiesCell, scanDate, summaryLine } from '../domain/table'
import AddCompetitors from './AddCompetitors.vue'
import AdminTools from './AdminTools.vue'
import ListUpkeep from './ListUpkeep.vue'

/** Competitors: one table of Tribelt and every competitor, each opening its own page. */
const router = useRouter()
const { isAdmin } = useMe()
const { companies } = useCompanies()
const { tableRows, newestScan, count, outOf, group, isLoading, error } = useCompetitorBoard()
// The scanning tools work on the companies on the list, not on the archived ones.
const active = computed(() => companies.value.filter((c) => !c.archived && c.role !== 'mirror'))

const q = ref('')
const family = ref<BeltFamily | ''>('')
const country = ref('')
const archived = ref(false)
const sort = ref<Sort<CompetitorColumn>>(null)
const rows = computed(() => competitorTable(tableRows.value, { q: q.value, family: family.value, country: country.value, archived: archived.value }, sort.value))
const countries = computed(() => countriesOf(tableRows.value))
const archivedCount = computed(() => tableRows.value.filter((r) => r.archived).length)
const summary = computed(() => summaryLine(tableRows.value.find((r) => r.tribelt), bestCompetitor(tableRows.value)))
const dataLine = computed(() => {
  const list = `${String(count.value)} ${count.value === 1 ? 'competitor' : 'competitors'} on the competitor list.`
  return newestScan.value === undefined ? `${list} No site has been scanned yet.` : `${list} The newest scan is from ${scanDate(newestScan.value)}.`
})
const groupNames = computed(() => group.value.map((c) => c.shortName))

const COLUMNS: readonly { key: CompetitorColumn; label: string; first: SortDirection; end?: boolean }[] = [
  { key: 'name', label: 'Name', first: 'asc' },
  { key: 'country', label: 'Country', first: 'asc' },
  { key: 'families', label: 'Belt families sold', first: 'desc' },
  { key: 'share', label: 'Named in AI responses', first: 'desc', end: true },
  { key: 'ready', label: 'AI readiness', first: 'desc', end: true },
  { key: 'terms', label: 'Search terms used', first: 'desc', end: true },
  { key: 'scan', label: 'Last scanned', first: 'desc' },
]
const archivedLabel = computed(() => `${archived.value ? 'Hide' : 'Show'} ${String(archivedCount.value)} ${archivedCount.value === 1 ? 'archived company' : 'archived companies'}`)

/** A started scan opens on its own page. */
function started(id: string): void {
  void router.push(`/manage/scans/${id}`)
}
</script>

<template>
  <PageFrame
    section="competitors"
    title="Competitors"
    purpose="The companies that make metal conveyor belts Tribelt's buyers could choose instead, and how each compares with Tribelt on AI responses, AI readiness, and search terms."
    :data-line="dataLine"
    :manage="{ to: '/manage/competitors', label: 'Manage the competitor list' }"
    help-topic="competitors"
  >
    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the competitors…</p>
    <template v-else>
      <p data-part="summary" class="m-0 max-w-[72ch] text-lg">{{ summary }}</p>

      <FrameSection id="list" title="The competitors">
        <div class="flex flex-wrap items-end gap-3">
          <label class="flex min-w-0 flex-[1_1_14rem] flex-col gap-1 text-sm font-medium">
            Search by name
            <input v-model="q" type="search" class="h-11 rounded-full border-2 border-solid border-grey bg-white px-4 text-base" />
          </label>
          <select v-model="family" aria-label="Belt family" class="h-11 rounded-full border-2 border-solid border-grey bg-white px-3 text-sm">
            <option value="">All belt families</option>
            <option v-for="f in PRODUCT_FAMILIES" :key="f" :value="f">{{ familyName(f) }}</option>
          </select>
          <select v-model="country" aria-label="Country" class="h-11 rounded-full border-2 border-solid border-grey bg-white px-3 text-sm">
            <option value="">All countries</option>
            <option v-for="c in countries" :key="c" :value="c">{{ c }}</option>
          </select>
        </div>
        <div :class="TABLE.box">
          <table :class="TABLE.table">
            <thead>
              <tr>
                <th v-for="c in COLUMNS" :key="c.key" scope="col" :class="[TABLE.head, c.end ? 'text-right' : '']">
                  <SortButton :label="c.label" :direction="directionOf(sort, c.key)" :align="c.end ? 'end' : 'start'" @sort="sort = nextSort(sort, c.key, c.first)" />
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in rows" :key="r.id" :class="r.tribelt ? 'bg-orange-pale' : ''">
                <th scope="row" :class="[TABLE.cell, 'font-normal']">
                  <template v-if="r.tribelt">
                    <strong class="font-semibold">{{ r.name }}</strong>
                    <span class="ml-2 rounded-full bg-white px-2 py-0.5 text-xs text-ink-2">For comparison</span>
                  </template>
                  <template v-else>
                    <RouterLink :to="companyPath(r.id)">{{ r.name }}</RouterLink>
                    <span v-if="r.archived" class="block text-xs text-muted">Archived</span>
                  </template>
                </th>
                <td :class="TABLE.cell">{{ r.country || 'Not recorded' }}</td>
                <td :class="TABLE.cell">
                  <span v-if="r.families.length > 0" :title="familiesCell(r).title">{{ familiesCell(r).text }}</span>
                  <template v-else>{{ familiesCell(r).text }}</template>
                </td>
                <td :class="[TABLE.cell, 'text-right']">{{ shareText(r.share) }}</td>
                <td :class="[TABLE.cell, 'text-right']">{{ r.ready === undefined ? 'Not checked' : scoreText(r.ready, outOf) }}</td>
                <td :class="[TABLE.cell, 'text-right']">{{ r.terms }}</td>
                <td :class="TABLE.cell">{{ scanDate(r.scannedAt) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p v-if="rows.length === 1" class="m-0 text-ink-2">No competitor matches these filters.</p>
        <p class="m-0 text-sm text-muted">
          <template v-if="archivedCount > 0">
            <button type="button" :aria-pressed="archived" class="cursor-pointer border-0 bg-transparent p-0 font-medium text-orange-ink underline" @click="archived = !archived">
              {{ archivedLabel }}
            </button>
            <span>Archived companies are left out of every figure in the tool.</span>
          </template>
          <template v-else>No company is archived.</template>
          <template v-if="groupNames.length > 0">
            {{ oxford(groupNames) }} {{ groupNames.length === 1 ? 'is' : 'are' }} left out:
            {{ groupNames.length === 1 ? 'it is a group company' : 'they are group companies' }} of Tribelt, not
            {{ groupNames.length === 1 ? 'a competitor' : 'competitors' }}.
          </template>
        </p>
      </FrameSection>

      <section v-if="isAdmin" aria-labelledby="admin-tools-title" class="flex flex-col gap-8 rounded-3xl bg-mist p-6">
        <h2 id="admin-tools-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em]">Manage the list</h2>
        <ScanForm @started="started" />
        <AdminTools :companies="active" @added="(id: string) => router.push(`/competitors/${id}`)" />
        <AddCompetitors :companies="active" />
        <ListUpkeep />
      </section>
    </template>
  </PageFrame>
</template>
