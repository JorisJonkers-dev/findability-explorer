<script setup lang="ts">
import { vScrollHint } from '../../../shared/scrollHint'
import { computed, ref, watch } from 'vue'
import type { CompetitorProfile, ProfileGroup } from '../../../infrastructure/api/types.gen'
import { tierFill } from '../../../shared/tiers'
import { tierLabel } from '../../../shared/vocabulary'
import LanguageFilter from '../../../shared/ui/LanguageFilter.vue'
import PillButton from '../../../shared/ui/PillButton.vue'
import { languageChoices, ofLanguage } from '../../terminology/domain/languages'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import { directionOf, nextSort, type Sort, type SortDirection, sortRows } from '../../../shared/sort'
import SortButton from '../../../shared/ui/SortButton.vue'
import {
  byLanguage,
  firstGroupWithTerms,
  groupLabel,
  groupTabLabel,
  mapLabel,
  type ProfileColumn,
  profileSortValue,
  scanDate,
  scanName,
  sinceLabel,
} from '../domain/history'

const props = defineProps<{ profile: CompetitorProfile }>()

const picked = ref<ProfileGroup | undefined>(firstGroupWithTerms(props.profile.groups))
watch(
  () => props.profile.competitorId,
  () => {
    picked.value = firstGroupWithTerms(props.profile.groups)
  },
)
const group = computed(() => props.profile.groups.find((g) => g.group === picked.value))
// The language filter offers every language of the profile, so it holds as the family changes.
const language = ref<string | null>(null)
const choices = computed(() => languageChoices(props.profile.groups.flatMap((g) => g.terms.map((t) => t.language))))
const languages = computed(() => byLanguage(ofLanguage(group.value?.terms ?? [], language.value)))
const latest = computed(() => props.profile.scan?.number)

// The tables sort by a column while one is picked; each language's table keeps its own rows.
const sort = ref<Sort<ProfileColumn>>(null)
function sortBy(column: ProfileColumn, first: SortDirection): void {
  sort.value = nextSort(sort.value, column, first)
}
function ariaSort(column: ProfileColumn): 'ascending' | 'descending' | 'none' {
  const d = directionOf(sort.value, column)
  return d === null ? 'none' : d === 'asc' ? 'ascending' : 'descending'
}
const PROFILE_COLUMNS: readonly { key: ProfileColumn; label: string; first: SortDirection; numeric: boolean }[] = [
  { key: 'phrase', label: 'Term', first: 'asc', numeric: false },
  { key: 'score', label: 'Score', first: 'desc', numeric: true },
  { key: 'pages', label: 'Pages', first: 'desc', numeric: true },
  { key: 'tier', label: 'Tier', first: 'asc', numeric: false },
  { key: 'map', label: 'Search term list', first: 'desc', numeric: false },
  { key: 'found', label: 'Found', first: 'desc', numeric: false },
]
</script>

<template>
  <section aria-labelledby="profile-title" class="flex flex-col gap-4">
    <div class="flex flex-col gap-1.5">
      <SectionHeading id="profile-title">Search terms by belt family</SectionHeading>
      <p v-if="profile.scan" class="m-0 text-sm text-muted">
        From {{ scanName(profile.scan) }} of {{ scanDate(profile.scan) }}, the latest of {{ profile.finishedScans }} finished
        {{ profile.finishedScans === 1 ? 'scan' : 'scans' }}.
      </p>
    </div>
    <p v-if="!profile.scan" class="m-0 text-ink-2">No scan of this company has finished yet.</p>
    <template v-else>
      <div role="group" aria-label="Belt family" class="flex flex-wrap gap-2">
        <PillButton v-for="g in profile.groups" :key="g.group" size="sm" :pressed="picked === g.group" @click="picked = g.group">
          {{ groupTabLabel(g) }}
        </PillButton>
      </div>
      <LanguageFilter :choices="choices" :selected="language" @select="(l) => (language = l)" />
      <div v-scroll-hint class="scroll-hint flex flex-col gap-5">
        <p v-if="languages.length === 0" class="m-0 text-ink-2">
          The scan found no {{ group ? groupLabel(group.group).toLowerCase() : '' }} terms.
        </p>
        <table v-for="lang in languages" :key="lang.language" class="w-full border-collapse text-sm">
          <caption class="pb-1.5 text-left font-semibold">{{ lang.label }} ({{ lang.terms.length }})</caption>
          <thead>
            <tr class="text-left text-xs text-muted">
              <th
                v-for="column in PROFILE_COLUMNS"
                :key="column.key"
                scope="col"
                :aria-sort="ariaSort(column.key)"
                class="pb-1 pr-3 font-medium"
                :class="column.numeric ? 'text-right' : ''"
              >
                <SortButton
                  :label="column.label"
                  :direction="directionOf(sort, column.key)"
                  :align="column.numeric ? 'end' : 'start'"
                  @sort="sortBy(column.key, column.first)"
                />
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in sortRows(lang.terms, sort, profileSortValue)" :key="t.phrase" class="border-t border-grey align-baseline">
              <th scope="row" class="py-1.5 pr-3 text-left font-medium">{{ t.phrase }}</th>
              <td class="py-1.5 pr-3 text-right tabular-nums">{{ t.score }}</td>
              <td class="py-1.5 pr-3 text-right tabular-nums text-ink-2">{{ t.pages }}&nbsp;p.</td>
              <td class="py-1.5 pr-3">
                <span class="inline-flex items-center gap-1.5">
                  <span aria-hidden="true" class="size-2.5 flex-none rounded-full" :class="tierFill(t.tier)"></span>{{ tierLabel(t.tier) }}
                </span>
              </td>
              <td class="py-1.5 pr-3 text-ink-2">{{ mapLabel(t) }}</td>
              <td class="py-1.5 text-muted" :class="t.firstScan === latest ? 'font-semibold text-orange-ink' : ''">{{ sinceLabel(t.firstScan, latest) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </section>
</template>
