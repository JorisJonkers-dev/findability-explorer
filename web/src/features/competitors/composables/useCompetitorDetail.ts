import { computed, ref, type Ref } from 'vue'
import { useStandings } from '../../catching-up/composables/useStandings'
import { useOverview } from '../../home/composables/useOverview'
import { usePromptAnswers, usePromptResults } from '../../prompt-test/composables/usePromptTest'
import { latestRun } from '../../prompt-test/domain/results'
import { useReadinessTable } from '../../readiness/composables/useReadiness'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import { detailHeadline, measuresOf, readinessScore } from '../domain/compare'
import { companyTerms } from '../domain/company'
import { aheadLines, competitorPurpose, detailDataLine, modelShares, theirTerms } from '../domain/detail'
import { beltRows, whereLine } from '../domain/list'
import { shareOf, CHIP_COUNT } from '../domain/roster'
import { useCompanies } from './useCompanies'
import { useCompany } from './useCompany'

/** One competitor set against Tribelt: its headline, its measures, its belts and its newest finished scan. */
export function useCompetitorDetail(id: Ref<string>) {
  const { companies, byId, isLoading } = useCompanies()
  const { detail, notFound, error } = useCompany(id)
  const { results } = usePromptResults()
  const { rowOf } = useReadinessTable()
  const { readinessChecks } = useStandings()
  const { overview } = useOverview()
  const map = useTermMap()

  const company = computed(() => byId(id.value))
  const tribelt = computed(() => companies.value.find((c) => c.role === 'tribelt'))
  const outOf = computed(() => (readinessChecks.value > 0 ? readinessChecks.value : CHIP_COUNT))
  const share = computed(() => shareOf(results.value, id.value))
  const tribeltShare = computed(() => (tribelt.value ? shareOf(results.value, tribelt.value.id) : undefined))
  const ready = computed(() => readinessScore(rowOf(id.value)?.readiness))
  const tribeltReady = computed(() => (tribelt.value ? readinessScore(rowOf(tribelt.value.id)?.readiness) : undefined))
  const coverage = (companyId: string | undefined): number | undefined =>
    companyId === undefined ? undefined : rowOf(companyId)?.readiness?.sitemap.coverage?.share
  const termsOf = (c: { shortName: string } | undefined): number | undefined => (c ? companyTerms(map.terms.value, c.shortName).length : undefined)

  // The newest run's responses, for how often each model names the company.
  const run = computed(() => latestRun(results.value))
  const { answers } = usePromptAnswers(
    computed(() => run.value?.id),
    ref({}),
  )

  const measures = computed(() =>
    company.value
      ? measuresOf({
          name: company.value.shortName,
          share: share.value,
          tribeltShare: tribeltShare.value,
          ready: ready.value,
          tribeltReady: tribeltReady.value,
          outOf: outOf.value,
          sitemap: coverage(id.value),
          tribeltSitemap: coverage(tribelt.value?.id),
          // A site no scan read has no terms found on it.
          terms: company.value.scans > 0 ? termsOf(company.value) : undefined,
          tribeltTerms: termsOf(tribelt.value),
          mapTerms: overview.value?.terms.total ?? map.terms.value.length,
        })
      : [],
  )

  return {
    company,
    purpose: computed(() => (company.value ? competitorPurpose(company.value) : '')),
    dataLine: computed(() => {
      const c = company.value
      if (!c) return ''
      const scan = c.lastScan
      return detailDataLine(c.shortName, run.value?.finishedAt ?? run.value?.startedAt, scan ? (scan.finishedAt ?? scan.createdAt) : undefined)
    }),
    ahead: computed(() => aheadLines(measures.value)),
    terms: computed(() => theirTerms(map.terms.value, id.value)),
    modelBars: computed(() => modelShares(answers.value?.items ?? [], id.value)),
    tribelt,
    isLoading,
    notFound,
    error,
    outOf,
    share,
    ready,
    title: computed(() => (company.value ? detailHeadline(company.value.shortName, share.value, tribeltShare.value) : 'A competitor.')),
    where: computed(() => (company.value ? whereLine(company.value) : '')),
    belts: computed(() => (company.value ? beltRows(company.value) : [])),
    /** The newest scan that finished, which the scan page opens. */
    latestScan: computed(() => detail.value?.scans.find((s) => s.status === 'done')),
    measures,
  }
}
