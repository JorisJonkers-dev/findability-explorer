<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import FrameSection from '../../../shared/ui/FrameSection.vue'
import { useCompanies } from '../../competitors/composables/useCompanies'
import { useOverview } from '../../home/composables/useOverview'
import { usePlan } from '../../plan/composables/usePlan'
import PageFrame from '../../shell/components/PageFrame.vue'
import { companyPath } from '../../shell/domain/sections'
import { useTermMap } from '../composables/useTermSearch'
import { findTerm, relatedTerms, termPath, termRecommendations, termSentence } from '../domain/searchTerms'

/** One search term: what it is in a sentence, who uses it, whether tribelt.nl does, related terms, and its recommendation. */
const props = defineProps<{ id: string }>()

const { terms, isLoading, error } = useTermMap()
const { byId } = useCompanies()
const { overview } = useOverview()
const { plan } = usePlan()

const term = computed(() => findTerm(terms.value, props.id))
const competitors = computed(() => overview.value?.competitors.makers ?? 0)
const purpose = computed(() => (term.value ? termSentence(term.value, competitors.value) : 'One term of the search term list.'))
const makers = computed(() =>
  (term.value?.makers ?? [])
    .map((m) => byId(m.competitorId))
    .filter((c) => c !== undefined)
    .sort((a, b) => a.shortName.localeCompare(b.shortName)),
)
const source = computed(() => (term.value?.source.startsWith('http') ? term.value.source : undefined))
const related = computed(() => (term.value ? relatedTerms(terms.value, term.value) : []))
const recommendations = computed(() => termRecommendations(plan.value?.improvements ?? []))
const plural = (n: number) => `${String(n)} ${n === 1 ? 'competitor' : 'competitors'}`
</script>

<template>
  <PageFrame
    section="terms"
    :title="term?.term ?? 'Search term'"
    :purpose="purpose"
    data-line="From the search term list and each site's newest scan."
    :back="{ to: '/terms/all', label: 'Back to Every term' }"
    :manage="{ to: '/manage/terms', label: 'Manage the search term list' }"
    help-topic="terms"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-5">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the search term list…</p>
    <p v-else-if="!term" role="alert" class="m-0 rounded-2xl bg-mist p-5">There is no term with this address.</p>
    <div v-else data-part="columns" class="grid grid-cols-1 items-start gap-10 min-[1100px]:grid-cols-2">
      <FrameSection id="where" title="Where competitors use it">
        <div class="flex flex-col gap-3 rounded-3xl bg-mist p-6">
          <p v-if="makers.length === 0" class="m-0 text-ink-2">No competitor on the list uses this term.</p>
          <ul v-else class="m-0 flex list-none flex-col gap-1.5 p-0">
            <li v-for="c in makers" :key="c.id"><RouterLink :to="companyPath(c.id)">{{ c.shortName }}</RouterLink></li>
          </ul>
          <p v-if="source" class="m-0 text-sm text-muted">
            The term was first recorded on <a :href="source" target="_blank" rel="noopener noreferrer">{{ source }}</a>.
          </p>
        </div>
      </FrameSection>
      <FrameSection id="tribelt" title="On tribelt.nl">
        <p class="m-0 rounded-3xl bg-mist p-6 text-ink-2">{{ term.tribeltUses ? 'tribelt.nl uses this term.' : 'No tribelt.nl page uses this term.' }}</p>
      </FrameSection>
      <FrameSection id="related" title="Related terms" intro="Other terms of the same belt family and language.">
        <div class="rounded-3xl bg-mist p-6">
          <p v-if="related.length === 0" class="m-0 text-ink-2">No other term of this belt family and language is in the list.</p>
          <ul v-else class="m-0 flex list-none flex-col gap-2 p-0">
            <li v-for="r in related" :key="termPath(r)" class="flex items-baseline gap-2">
              <span class="w-4 flex-none font-semibold" :class="r.tribeltUses ? 'text-black' : 'text-muted'">
                <span aria-hidden="true">{{ r.tribeltUses ? '✓' : '·' }}</span>
                <span class="sr-only">{{ r.tribeltUses ? 'Used on tribelt.nl' : 'Not used on tribelt.nl' }}</span>
              </span>
              <RouterLink :to="termPath(r)">{{ r.term }}</RouterLink>
              <span class="text-sm text-muted">{{ plural(r.usedBy.length) }}</span>
            </li>
          </ul>
        </div>
      </FrameSection>
      <FrameSection id="recommendation" title="The recommendation">
        <div class="rounded-3xl bg-mist p-6">
          <p v-if="recommendations.length === 0" class="m-0 text-ink-2">No recommendation for search terms is open.</p>
          <ul v-else class="m-0 flex list-none flex-col gap-3 p-0">
            <li v-for="r in recommendations" :key="r.id" class="flex flex-col gap-0.5">
              <RouterLink :to="`/recommendations/${r.id}`" class="font-semibold">{{ r.title }}</RouterLink>
              <span class="text-sm text-muted">Benefit {{ r.benefit }} of 5, effort {{ r.effort }} of 5</span>
            </li>
          </ul>
        </div>
      </FrameSection>
    </div>
  </PageFrame>
</template>
