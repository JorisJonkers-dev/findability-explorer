<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useAiSearchSummary } from '../../ai-search/composables/useAiSearchSummary'
import PageFrame from '../../shell/components/PageFrame.vue'
import { companyPath } from '../../shell/domain/sections'
import { useReadinessTable } from '../composables/useReadiness'
import { CHECK_NAMES, checkStates, SCORED_CHECKS, siteName, STATE_WORDS } from '../domain/checks'
import { policySummary } from '../domain/readiness'
import { builtWith, crawlerRows, evidence, filesAsFound, scanDataLine, sitePurpose, siteTitle, structuredData } from '../domain/site'
import CheckMark from './CheckMark.vue'
import ReadinessSection from './ReadinessSection.vue'

/**
 * One site's AI readiness: the eight checks with their evidence, the structured data and the files
 * the scan found, beside the crawler policy.
 */
const props = defineProps<{ companyId: string }>()

const { rowOf, isLoading, error } = useReadinessTable()
const { summary } = useAiSearchSummary()
const row = computed(() => rowOf(props.companyId))
const r = computed(() => row.value?.readiness)
const states = computed(() => (r.value ? checkStates(r.value) : undefined))
const crawlers = computed(() => (r.value ? crawlerRows(r.value, summary.value?.crawlers ?? []) : []))
const name = computed(() => (row.value ? siteName(row.value) : ''))
const back = { to: '/ai-readiness', label: 'AI readiness' }
const LINK = 'inline-flex min-h-11 items-center rounded-full border-2 border-silver px-4 font-medium text-black no-underline hover:border-black hover:text-black'
</script>

<template>
  <PageFrame
    v-if="row"
    section="ai-readiness"
    :title="siteTitle(row)"
    :purpose="sitePurpose(row)"
    :data-line="scanDataLine(row) || undefined"
    :back="back"
    :manage="{ to: '/manage/scans', label: 'Manage the scans' }"
    help-topic="websites"
  >
    <div v-if="r && states" class="grid gap-10 min-[1100px]:grid-cols-2">
      <div data-column="left" class="flex min-w-0 flex-col gap-10">
        <ReadinessSection id="site-checks" title="The eight checks" :intro="builtWith(r) || undefined">
          <ul class="m-0 flex list-none flex-col gap-3 rounded-2xl bg-mist p-5">
            <li v-for="c in SCORED_CHECKS" :key="c" class="flex gap-3">
              <CheckMark :state="states[c]" :label="STATE_WORDS[states[c]]" />
              <div class="flex min-w-0 flex-col gap-0.5">
                <strong>{{ CHECK_NAMES[c] }}</strong>
                <p class="m-0 text-sm text-ink-2">{{ evidence(c, r) }}</p>
              </div>
            </li>
          </ul>
        </ReadinessSection>

        <ReadinessSection id="site-structured" title="The structured data found">
          <p class="m-0">{{ structuredData(r).found }}</p>
          <p class="m-0 text-sm text-muted">{{ structuredData(r).product }}</p>
        </ReadinessSection>

        <ReadinessSection id="site-files" title="The files as found" intro="Each scan stores the files it fetched. Open them to read, search, copy, or download them.">
          <template #aside>
            <RouterLink
              :to="`/ai-readiness/${companyId}/files`"
              class="inline-flex min-h-11 items-center rounded-full bg-black px-4 font-semibold text-white no-underline hover:bg-orange-ink hover:text-white"
            >
              Read the files
            </RouterLink>
          </template>
          <dl class="m-0 flex flex-col rounded-2xl bg-mist px-5 py-2">
            <div v-for="f in filesAsFound(r)" :key="f.name" class="flex flex-col gap-0.5 border-b border-grey py-3 last:border-b-0">
              <dt class="font-semibold">{{ f.name }}</dt>
              <dd class="m-0 text-sm text-ink-2">{{ f.text }}</dd>
            </div>
          </dl>
        </ReadinessSection>
      </div>

      <div data-column="right" class="flex min-w-0 flex-col gap-10">
        <ReadinessSection id="site-crawlers" title="Crawler policy" intro="Each search and AI crawler, what it does for the company, and whether robots.txt allows it.">
          <p class="m-0 font-semibold">{{ policySummary(r) }}</p>
          <div v-if="crawlers.length > 0" class="rounded-2xl bg-mist p-3 sm:p-5">
            <table class="w-full border-collapse text-sm">
              <caption class="sr-only">Each crawler, what it does for {{ name }}, and what robots.txt says to it</caption>
              <thead>
                <tr class="border-b border-silver text-left text-ink-2">
                  <th scope="col" class="py-2 pr-2 font-semibold">Crawler</th>
                  <th scope="col" class="px-2 py-2 font-semibold">What it does for {{ name }}</th>
                  <th scope="col" class="py-2 pl-2 font-semibold">robots.txt</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="c in crawlers" :key="c.bot" class="border-b border-grey align-top last:border-b-0">
                  <th scope="row" class="py-2.5 pr-2 text-left font-bold break-words">{{ c.bot }}</th>
                  <td class="px-2 py-2.5">
                    {{ c.gets }}<span class="block text-xs text-muted">{{ c.lab }}</span>
                  </td>
                  <td class="py-2.5 pl-2" :class="c.blocked ? 'font-semibold text-orange-ink' : ''">
                    {{ c.access }}<span v-if="c.note" class="block text-xs font-normal text-muted">{{ c.note }}</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </ReadinessSection>
      </div>
    </div>

    <nav aria-label="More about this site" class="flex flex-wrap gap-3">
      <RouterLink v-if="row.role === 'competitor'" :to="companyPath(companyId)" :class="LINK">{{ name }}’s competitor page</RouterLink>
      <RouterLink v-if="row.scan" :to="`${companyPath(companyId)}/history`" :class="LINK">Scan history</RouterLink>
    </nav>
  </PageFrame>

  <PageFrame v-else-if="isLoading" section="ai-readiness" title="AI readiness" purpose="Loading the site…" :back="back" />
  <PageFrame v-else-if="error" section="ai-readiness" title="AI readiness" purpose="The site could not be loaded." :back="back">
    <p role="alert" class="m-0 text-orange-ink">{{ error }}</p>
  </PageFrame>
  <PageFrame v-else section="ai-readiness" title="Site not found" purpose="No scan has this address." :back="back">
    <p class="m-0 rounded-2xl bg-mist p-5 text-ink-2">No scan has been read for this site. Go back to AI readiness and pick one.</p>
  </PageFrame>
</template>
