<script setup lang="ts">
import { toRef } from 'vue'
import { RouterLink } from 'vue-router'
import BarList from '../../../shared/ui/charts/BarList.vue'
import FrameSection from '../../../shared/ui/FrameSection.vue'
import PageFrame from '../../shell/components/PageFrame.vue'
import { companyPath, familySlug } from '../../shell/domain/sections'
import { languageLabel } from '../../../shared/vocabulary'
import { familyName, termPath } from '../../terminology/domain/searchTerms'
import { useCompetitorDetail } from '../composables/useCompetitorDetail'
import MeasureChart from './MeasureChart.vue'

/** One competitor: against Tribelt, where it is ahead, what it sells, its terms tribelt.nl lacks, and how often each model names it. */
const props = defineProps<{ companyId: string }>()

const id = toRef(() => props.companyId)
const { company, isLoading, notFound, error, purpose, dataLine, ahead, terms, modelBars, belts, measures } = useCompetitorDetail(id)
const button = 'inline-flex min-h-11 items-center rounded-full border-2 border-solid border-silver px-4 font-medium text-black no-underline hover:border-black hover:text-black'
</script>

<template>
  <PageFrame
    section="competitors"
    :title="company?.shortName ?? 'Competitor'"
    :purpose="purpose || 'A company on the competitor list.'"
    :data-line="dataLine || undefined"
    :back="{ to: '/competitors', label: 'Back to Competitors' }"
    :manage="{ to: `/manage/competitors/companies/${encodeURIComponent(companyId)}`, label: 'Manage this company' }"
    help-topic="competitors"
  >
    <template v-if="company" #aside>
      <a v-if="company.websiteUrl" :href="company.websiteUrl" target="_blank" rel="noopener noreferrer" :class="button">
        Visit its website<span class="sr-only"> (opens in a new tab)</span>
      </a>
      <RouterLink :to="`${companyPath(company.id)}/history`" :class="button">See its scan history</RouterLink>
    </template>

    <p v-if="notFound" role="alert" class="m-0 text-orange-ink">No company has this id.</p>
    <p v-else-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading || !company" class="m-0 text-muted">Loading the company…</p>
    <template v-else>
      <div class="grid grid-cols-1 items-start gap-10 min-[1100px]:grid-cols-2">
        <FrameSection id="compare" title="Against Tribelt">
          <MeasureChart :name="company.shortName" :measures="measures" />
        </FrameSection>

        <FrameSection id="ahead" title="Where it is ahead of Tribelt">
          <div class="rounded-3xl bg-mist p-6">
            <p v-if="ahead.length === 0" class="m-0 text-ink-2">{{ company.shortName }} is not ahead of Tribelt on any measure.</p>
            <ul v-else class="m-0 flex list-disc flex-col gap-1.5 pl-5">
              <li v-for="line in ahead" :key="line">{{ line }}</li>
            </ul>
          </div>
        </FrameSection>

        <FrameSection id="belts" title="What it sells and what it calls it">
          <ul class="m-0 flex list-none flex-col rounded-3xl bg-mist px-6 py-3">
            <li v-for="row in belts" :key="row.family" class="flex flex-col gap-0.5 border-b border-grey py-2.5 last:border-b-0">
              <RouterLink :to="`/terms/families/${familySlug(row.family)}`" class="font-semibold">{{ row.label }}</RouterLink>
              <span :class="row.offered ? 'text-black' : 'text-muted'">{{ row.name }}</span>
            </li>
          </ul>
        </FrameSection>

        <FrameSection id="their-terms" title="Search terms it uses that tribelt.nl does not">
          <div class="flex flex-col gap-3 rounded-3xl bg-mist p-6">
            <p v-if="terms.total === 0" class="m-0 text-ink-2">Every term this competitor uses is also on tribelt.nl.</p>
            <ul v-else class="m-0 flex list-none flex-col gap-1.5 p-0">
              <li v-for="t in terms.shown" :key="termPath(t)">
                <RouterLink :to="termPath(t)">{{ t.term }}</RouterLink>
                <span class="text-sm text-muted"> {{ languageLabel(t.language) }}, {{ familyName(t.family) }}</span>
              </li>
            </ul>
            <p v-if="terms.total > terms.shown.length" class="m-0 text-sm text-muted">
              Showing {{ terms.shown.length }} of {{ terms.total }}. <RouterLink :to="{ path: '/terms/all', query: { user: company.shortName } }" class="underline">See them all</RouterLink>.
            </p>
          </div>
        </FrameSection>

        <FrameSection id="models" title="In AI responses" intro="How often each model names it, in the newest Prompt run.">
          <div class="rounded-3xl bg-mist p-6">
            <p v-if="modelBars.length === 0" class="m-0 text-ink-2">The newest Prompt run has no read responses yet.</p>
            <BarList v-else :rows="modelBars" :max="1" :label="`How often each model names ${company.shortName}`" />
          </div>
          <RouterLink to="/ai-responses/all" class="self-start font-medium">See every response</RouterLink>
        </FrameSection>
      </div>
    </template>
  </PageFrame>
</template>
