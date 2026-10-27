<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { Competitor } from '../../../infrastructure/api/types.gen'
import { percent } from '../../../shared/percent'
import { vScrollHint } from '../../../shared/scrollHint'
import { usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { useReadinessTable } from '../../readiness/composables/useReadiness'
import SectionView from '../../shell/components/SectionView.vue'
import { useMe } from '../../shell/composables/useMe'
import { companyPath, COMPETITOR_TABS } from '../../shell/domain/sections'
import { useRescan } from '../composables/useCompanyAdmin'
import AddCompetitors from './AddCompetitors.vue'
import { useCompetitorChecks, useListDecision } from '../composables/useCompetitorChecks'
import { useCompetitors } from '../composables/useCompetitors'
import { rescanSummary, unscannedCandidates } from '../domain/admin'
import { haikuArchived, isFlagged, listCheckLine, verdictLabel, verdictReason } from '../domain/checks'
import { hostOf } from '../domain/company'
import {
  ADMIN_FILTERS,
  chipsOf,
  keeps,
  lastScanOf,
  parseRosterFilter,
  placeOf,
  ROSTER_FILTERS,
  type RosterFilter,
  roleLabel,
  rosterOrder,
  shareOf,
} from '../domain/roster'

const route = useRoute()
const router = useRouter()
const { isAdmin } = useMe()
const { competitors, isLoading, error } = useCompetitors()
const { rowOf } = useReadinessTable()
const { results } = usePromptResults()
const { isQueuing, batch, error: rescanError, rescan } = useRescan()
const { checks, unchecked, review, isStarting, error: checkError, start: startCheck } = useCompetitorChecks(isAdmin)
const { isSaving, error: decisionError, notice, keep, archive, restore } = useListDecision()

const filter = computed(() => parseRosterFilter(route.query.show, isAdmin.value))
const filters = computed(() => (isAdmin.value ? [...ROSTER_FILTERS, ...ADMIN_FILTERS] : ROSTER_FILTERS))
const active = computed(() => competitors.value.filter((c) => !c.archived && c.role !== 'mirror'))
const share = (c: Pick<Competitor, 'id'>) => shareOf(results.value, c.id)
const flagged = (c: Competitor) => isFlagged(c, checks.value.get(c.id))
const rows = computed(() => rosterOrder(active.value.filter((c) => keeps(c, filter.value, flagged(c))), share))
const archived = computed(() => haikuArchived(competitors.value))
const counts = computed<Partial<Record<RosterFilter, number>>>(() => ({ unsure: active.value.filter(flagged).length, archived: archived.value.length }))
const most = computed(() => Math.max(0.01, ...active.value.map((c) => share(c) ?? 0)))
const never = computed(() => unscannedCandidates(active.value).length)
const competitorCount = computed(() => active.value.filter((c) => c.role === 'competitor').length)

function show(f: RosterFilter): void {
  void router.replace({ query: f === 'all' ? {} : { show: f } })
}
</script>

<template>
  <SectionView
    section="competitors"
    :title="competitorCount > 0 ? `${String(competitorCount)} competitors, and Tribelt among them.` : 'Every competitor.'"
    :tabs="COMPETITOR_TABS"
  >
    <div class="flex flex-col gap-12 stage:grid stage:min-h-0 stage:gap-24" :class="isAdmin ? 'stage:grid-cols-[minmax(0,1.7fr)_minmax(0,1fr)]' : 'stage:grid-cols-1'">
      <section aria-labelledby="list-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <h2 id="list-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">All competitors</h2>
          <div role="group" aria-label="Show" class="flex flex-wrap gap-1.5">
            <button
              v-for="f in filters"
              :key="f.value"
              type="button"
              :aria-pressed="filter === f.value"
              class="inline-flex min-h-11 items-center rounded-full border-2 px-4 text-sm font-medium stage:min-h-16 stage:px-7 stage:text-[24px]"
              :class="filter === f.value ? 'border-black bg-black text-white' : 'border-silver bg-white text-ink-2 hover:border-black'"
              @click="show(f.value)"
            >
              {{ f.label }}<template v-if="counts[f.value] !== undefined"> ({{ counts[f.value] }})</template>
            </button>
          </div>
        </div>
        <div v-if="isAdmin" class="flex flex-wrap items-center gap-3">
          <button
            type="button"
            class="inline-flex min-h-11 items-center rounded-full border-2 border-black bg-white px-4 text-sm font-semibold hover:bg-mist disabled:opacity-60 stage:min-h-16 stage:px-7 stage:text-[24px]"
            :disabled="isQueuing || never === 0"
            @click="rescan({ unscanned: true })"
          >
            Scan the {{ never }} never scanned
          </button>
          <button
            type="button"
            class="inline-flex min-h-11 items-center rounded-full border-2 border-silver bg-white px-4 text-sm font-medium hover:border-black disabled:opacity-60 stage:min-h-16 stage:px-7 stage:text-[24px]"
            :disabled="isQueuing"
            @click="rescan({ all: true })"
          >
            Scan all again
          </button>
          <p v-if="batch" class="m-0 text-sm font-medium stage:text-[24px]" aria-live="polite">{{ rescanSummary(batch) }}</p>
          <p v-if="rescanError" role="alert" class="m-0 text-sm font-medium text-orange-ink stage:text-[24px]">{{ rescanError }}</p>
        </div>
        <section v-if="isAdmin && review" aria-label="Haiku's check of the list" class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-2xl bg-mist p-4 stage:p-6">
          <p class="m-0 font-medium stage:text-[24px]">{{ listCheckLine(review, unchecked) }}</p>
          <RouterLink v-if="review.problem === 'noKey'" to="/settings" class="font-medium stage:text-[24px]">Add one in Language models</RouterLink>
          <button
            v-else
            type="button"
            class="inline-flex min-h-11 items-center rounded-full border-0 bg-black px-5 text-sm font-semibold text-white hover:bg-ink-2 disabled:opacity-60 stage:min-h-16 stage:px-7 stage:text-[24px]"
            :disabled="review.running || isStarting"
            @click="startCheck"
          >
            Check the list with Haiku
          </button>
        </section>
        <p v-if="isAdmin && checkError" role="alert" class="m-0 text-sm font-medium text-orange-ink stage:text-[24px]">{{ checkError }}</p>
        <p v-if="isAdmin && decisionError" role="alert" class="m-0 text-sm font-medium text-orange-ink stage:text-[24px]">{{ decisionError }}</p>
        <p v-if="isAdmin && notice" class="m-0 text-sm font-medium stage:text-[24px]" aria-live="polite">{{ notice }}</p>
        <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
        <p v-else-if="isLoading" class="m-0 text-muted">Loading the competitors…</p>
        <section v-else-if="filter === 'archived'" aria-labelledby="archived-title" class="flex flex-col gap-3 stage:min-h-0 stage:flex-1 stage:overflow-auto">
          <h3 id="archived-title" class="m-0 text-lg font-semibold stage:text-[32px]">Archived by Haiku</h3>
          <p v-if="archived.length === 0" class="m-0 text-muted">Haiku has archived no company.</p>
          <ul v-else aria-label="Archived by Haiku" class="m-0 flex list-none flex-col gap-2 p-0">
            <li v-for="c in archived" :key="c.id" class="flex flex-wrap items-center justify-between gap-3 border-b border-grey py-2 stage:py-3">
              <span class="flex min-w-0 flex-col">
                <RouterLink :to="companyPath(c.id)" class="font-semibold text-black stage:text-[26px]">{{ c.name }}</RouterLink>
                <span class="text-sm text-ink-2 stage:text-[22px]">{{ verdictReason(checks.get(c.id)) }}</span>
              </span>
              <button
                type="button"
                class="min-h-9 rounded-full border-2 border-black bg-white px-4 text-xs font-semibold whitespace-nowrap hover:bg-black hover:text-white disabled:opacity-60 stage:min-h-14 stage:px-5 stage:text-[20px]"
                :disabled="isSaving"
                @click="restore(c)"
              >
                Restore<span class="sr-only">{{ ' ' + c.name }}</span>
              </button>
            </li>
          </ul>
        </section>
        <div v-else v-scroll-hint class="scroll-hint relative overflow-auto stage:min-h-0 stage:flex-1">
          <table class="w-full min-w-[720px] border-collapse text-sm stage:text-[26px]">
            <caption class="sr-only">Every competitor, Tribelt and group company, with its website, AI readiness, share of AI answers and last scan</caption>
            <thead>
              <tr class="border-b-2 border-black">
                <th scope="col" class="py-2 text-left font-medium text-muted">Company</th>
                <th scope="col" class="py-2 pl-3 text-left font-medium text-muted">Website</th>
                <th scope="col" class="py-2 pl-3 text-left font-medium text-muted">AI readiness</th>
                <th scope="col" class="py-2 pl-3 text-left font-medium text-muted">Share of AI answers</th>
                <th scope="col" class="py-2 pl-3 text-left font-medium text-muted">Last scan</th>
                <th v-if="isAdmin" scope="col" class="py-2 pl-3 text-left font-medium text-muted">Haiku's check</th>
                <th v-if="isAdmin" scope="col" class="py-2 pl-3"><span class="sr-only">Scan</span></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="c in rows" :key="c.id" class="border-b border-grey" :class="c.role === 'tribelt' ? 'bg-orange-pale' : ''">
                <th scope="row" class="py-2 text-left font-normal stage:py-3">
                  <RouterLink :to="companyPath(c.id)" class="font-semibold text-black" :class="c.role === 'tribelt' ? 'font-bold' : ''">{{ c.name }}</RouterLink>
                  <span class="block text-xs text-muted stage:text-[20px]">{{ [roleLabel(c), placeOf(c)].filter(Boolean).join('. ') }}</span>
                </th>
                <td class="py-2 pl-3 stage:py-3">
                  <a v-if="c.websiteUrl" :href="c.websiteUrl" target="_blank" rel="noopener" :title="`Open ${hostOf(c.websiteUrl) ?? ''} in a new tab`">{{ hostOf(c.websiteUrl) }}</a>
                  <span v-else class="text-muted">No website</span>
                </td>
                <td class="py-2 pl-3 stage:py-3">
                  <ul v-if="chipsOf(rowOf(c.id)?.readiness).length > 0" :aria-label="`AI readiness of ${c.name}`" class="m-0 flex list-none flex-wrap gap-1 p-0">
                    <li
                      v-for="chip in chipsOf(rowOf(c.id)?.readiness)"
                      :key="chip.label"
                      :title="chip.tip"
                      class="inline-flex items-center gap-1 rounded-full border-2 px-2 text-xs whitespace-nowrap stage:px-3 stage:text-[18px]"
                      :class="chip.state === 'no' ? 'border-grey bg-white text-muted' : 'border-orange-soft bg-orange-pale text-orange-ink'"
                    >
                      <span aria-hidden="true">{{ chip.mark }}</span>{{ chip.label }}<span class="sr-only">: {{ chip.tip }}</span>
                    </li>
                  </ul>
                  <span v-else class="text-muted">Not checked</span>
                </td>
                <td class="py-2 pl-3 stage:py-3">
                  <span v-if="share(c) === undefined" class="text-muted">No run yet</span>
                  <span v-else class="flex items-center gap-2">
                    <span aria-hidden="true" class="relative h-3 w-20 bg-mist stage:h-4 stage:w-40">
                      <span class="absolute inset-y-0 left-0" :class="c.role === 'tribelt' ? 'bg-orange' : 'bg-blue-grey'" :style="{ width: percent((share(c) ?? 0) / most) }"></span>
                    </span>
                    <span class="font-semibold">{{ Math.round((share(c) ?? 0) * 100) }}%</span>
                  </span>
                </td>
                <td class="py-2 pl-3 stage:py-3" :class="lastScanOf(c).attention ? 'font-semibold text-orange-ink' : 'text-ink-2'">{{ lastScanOf(c).text }}</td>
                <td v-if="isAdmin" class="max-w-80 py-2 pl-3 stage:py-3">
                  <template v-if="checks.get(c.id)">
                    <span class="block font-semibold" :class="flagged(c) ? 'text-orange-ink' : ''">{{ verdictLabel(checks.get(c.id)) }}</span>
                    <span class="block text-xs text-ink-2 stage:text-[20px]">{{ verdictReason(checks.get(c.id)) }}</span>
                    <span v-if="flagged(c)" role="group" :aria-label="`Decide on ${c.name}`" class="mt-1 flex flex-wrap gap-1.5">
                      <button
                        type="button"
                        class="min-h-9 rounded-full border-2 border-black bg-white px-3 text-xs font-semibold hover:bg-black hover:text-white disabled:opacity-60 stage:min-h-14 stage:px-5 stage:text-[20px]"
                        :disabled="isSaving"
                        @click="keep(c)"
                      >
                        Keep<span class="sr-only">{{ ' ' + c.name }}</span>
                      </button>
                      <button
                        type="button"
                        class="min-h-9 rounded-full border-2 border-silver bg-white px-3 text-xs font-medium hover:border-black disabled:opacity-60 stage:min-h-14 stage:px-5 stage:text-[20px]"
                        :disabled="isSaving"
                        @click="archive(c)"
                      >
                        Archive<span class="sr-only">{{ ' ' + c.name }}</span>
                      </button>
                    </span>
                  </template>
                  <span v-else-if="c.keptBy" class="text-ink-2">Kept on the list</span>
                  <span v-else-if="c.role === 'competitor'" class="text-muted">Not checked</span>
                </td>
                <td v-if="isAdmin" class="py-2 pl-3 text-right stage:py-3">
                  <button
                    v-if="c.websiteUrl"
                    type="button"
                    class="min-h-9 rounded-full border-2 border-silver bg-white px-3 text-xs font-medium whitespace-nowrap hover:border-black disabled:opacity-60 stage:min-h-14 stage:px-5 stage:text-[20px]"
                    :disabled="isQueuing"
                    @click="rescan({ competitorIds: [c.id] })"
                  >
                    {{ c.scans === 0 ? 'Scan' : 'Scan again' }}<span class="sr-only">{{ ' ' + c.name }}</span>
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
      <AddCompetitors v-if="isAdmin" :companies="active" />
    </div>
  </SectionView>
</template>
