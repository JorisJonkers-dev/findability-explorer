import { describe, expect, it } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { overviewRoutes } from '../../../testing/homeFixtures'
import { promptResults } from '../../../testing/fixtures'
import { json, problem, routes, settle } from '../../../testing/mountApp'
import { mountAt } from '../../../testing/mountAt'
import { useGoalPage } from './useGoalPage'

async function mountGoal(measure: string, extra: Parameters<typeof overviewRoutes>[0] = {}) {
  const slug = ref(measure)
  let state: ReturnType<typeof useGoalPage> | undefined
  const Probe = defineComponent({ setup: () => ((state = useGoalPage(slug)), () => h('div')) })
  const mounted = await mountAt(Probe, `/goals/${measure}`, routes(overviewRoutes(extra)))
  if (!state) throw new Error('the composable did not mount')
  return { ...mounted, state, slug }
}

describe('useGoalPage', () => {
  it('builds a Prompt test goal: its page, the newest run, its breakdown, and its line over time', async () => {
    const { state } = await mountGoal('aiShareOfVoice')
    expect(state.isLoading.value).toBe(false)
    expect(state.error.value).toBeNull()
    expect(state.missing.value).toBe(false)
    expect(state.page.value?.name).toBe('AI responses that name Tribelt')
    expect(state.dataLine.value).toMatch(/^From the Prompt run of 1 November 2026: 90 questions to 2 models/)
    expect(state.history.value?.label).toBe('Tribelt')
    expect(state.helpTopic.value).toBe('prompt-test')
    const b = state.breakdown.value
    expect(b?.kind).toBe('groups')
    if (b?.kind === 'groups') expect(b.groups.map((g) => g.title)).toEqual(['By model', 'By scope', 'By belt family'])
    expect(state.breakdownError.value).toBeNull()
  })

  it('lists the readiness checks, and the search terms per language, for the goals the scans measure', async () => {
    const checks = await mountGoal('readinessChecks')
    expect(checks.state.dataLine.value).toBe('From the newest scan of each site.')
    expect(checks.state.history.value).toBeUndefined()
    expect(checks.state.helpTopic.value).toBe('readiness')
    expect(checks.state.breakdown.value?.kind).toBe('checks')
    const terms = await mountGoal('termsUsed')
    expect(terms.state.breakdown.value?.kind).toBe('terms')
    expect(terms.state.helpTopic.value).toBe('terms')
  })

  it('says when there is no goal at the address', async () => {
    const { state } = await mountGoal('nothing')
    expect(state.missing.value).toBe(true)
    expect(state.page.value).toBeUndefined()
    expect(state.dataLine.value).toBeUndefined()
    expect(state.breakdown.value).toBeUndefined()
    expect(state.breakdownError.value).toBeNull()
    expect(state.helpTopic.value).toBeUndefined()
    expect(state.history.value).toBeUndefined()
  })

  it('reports a failure of the goals, and of the figures a breakdown needs', async () => {
    const down = await mountGoal('aiShareOfVoice', { 'GET /api/v1/plan': () => problem(500, 'Internal Server Error', 'Plan down.') })
    expect(down.state.error.value).toContain('Plan down.')
    expect(down.state.missing.value).toBe(false)
    const results = await mountGoal('citationShare', { 'GET /api/v1/prompt-test': () => problem(500, 'Internal Server Error', 'Runs down.') })
    expect(results.state.breakdownError.value).toContain('Runs down.')
    expect(results.state.breakdown.value).toBeUndefined()
    expect(results.state.dataLine.value).toBeUndefined()
    const summary = await mountGoal('readinessChecks', { 'GET /api/v1/ai-search': () => problem(500, 'Internal Server Error', 'Summary down.') })
    expect(summary.state.breakdownError.value).toContain('Summary down.')
    expect(summary.state.breakdown.value).toBeUndefined()
    const overview = await mountGoal('termsUsed', { 'GET /api/v1/overview': () => problem(500, 'Internal Server Error', 'Overview down.') })
    expect(overview.state.breakdownError.value).toContain('Overview down.')
    expect(overview.state.breakdown.value).toBeUndefined()
  })

  it('has no breakdown to show before the first Prompt run', async () => {
    const { state } = await mountGoal('aiShareOfVoice', { 'GET /api/v1/prompt-test': () => json(promptResults({ runs: [] })) })
    expect(state.breakdown.value).toEqual({ kind: 'groups', groups: [] })
    expect(state.dataLine.value).toBe('No Prompt run has finished yet.')
  })

  it('follows the address to another goal', async () => {
    const { state, slug } = await mountGoal('aiShareOfVoice')
    slug.value = 'twenteShareOfVoice'
    await settle()
    expect(state.page.value?.name).toBe('AI responses about the Twente region name Tribelt')
  })
})
