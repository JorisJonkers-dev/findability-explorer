import { expect, expectAccessible, test } from './fixtures'

const VIEWS = [
  { path: '/ai-search/readiness', title: 'AI readiness', heading: /./ },
  { path: '/ai-search/prompt-test', title: 'Prompt test', heading: /./ },
  { path: '/settings/prompt-test', title: 'Prompt test settings', heading: 'Prompt test' },
] as const

for (const view of VIEWS) {
  test(`${view.title} renders and is accessible`, async ({ page }) => {
    await page.goto(view.path)
    await expect(page).toHaveTitle(`${view.title} · Findability Explorer`)
    await expect(page.getByRole('heading', { level: 1, name: view.heading })).toBeVisible()
    await expectAccessible(page)
  })
}

test('the AI readiness table pins Tribelt and the mirror on top', async ({ page }) => {
  await page.goto('/ai-search/readiness')
  const baseline = page.getByRole('rowgroup', { name: 'Tribelt today, and the target' })
  await expect(baseline.getByRole('rowheader')).toHaveText(['Tribelt B.V.', 'Mirror of tribelt.nl'])
})

test.describe('at phone width', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  test('no new view scrolls sideways', async ({ page }) => {
    for (const view of VIEWS) {
      await page.goto(view.path)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
      expect(overflow, view.path).toBe(0)
    }
  })
})

test('the AI search summary lists every crawler and leads on to its pages', async ({ page }) => {
  await page.goto('/ai-search')
  const labs = page.getByRole('region', { name: 'Which AI labs may read tribelt.nl' })
  await expect(labs.getByRole('rowheader')).toHaveCount(10)
  await expect(page.getByRole('region', { name: 'Ready for AI' })).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('link', { name: /^AI readiness Crawler access/ }).click()
  await expect(page).toHaveURL('/ai-search/readiness')
})

const RUN = '9d8e7f6a-5b4c-4d3e-8f2a-1b0c9d8e7f6a'

const run = {
  id: RUN, promptSetVersion: 2, factSheetVersion: 1, trigger: 'manual', startedBy: 'e2e', startedAt: '2026-11-02T06:00:00Z',
  finishedAt: '2026-11-02T06:20:00Z', status: 'done', models: [{ provider: 'openai', model: 'openai/gpt-oss-20b', free: false }],
  tasks: 2, answered: 1, failed: 1, inputTokens: 900, outputTokens: 300, costUsd: 0.0014,
  byProvider: [{ provider: 'openai', answered: 1, failed: 1, costUsd: 0.0014, models: ['openai/gpt-oss-20b'] }],
  cells: [{ provider: 'openai', scope: 'twente', family: 'eyelink', mode: 'web', answers: 1, named: 1, cited: 1 }],
}

const answer = (mode: 'knowledge' | 'web', id: string, failed: boolean) => ({
  id, promptId: 'eyelink-application-twente', family: 'eyelink', kind: 'application', scope: 'twente', language: 'nl',
  prompt: 'We bouwen een doorloopoven. Welke fabrikanten in de regio Twente?', promptSetVersion: 2, provider: 'openai',
  model: 'openai/gpt-oss-20b', mode, text: failed ? '' : 'Tribelt in Haaksbergen makes eyelink belts.', citations: [],
  at: '2026-11-02T06:10:00Z', costUsd: 0.0014, inputTokens: 900, outputTokens: 300,
  ...(failed
    ? { failure: 'The model is rate limited.' }
    : {
        extraction: {
          failed: false, companies: [{ name: 'Tribelt', competitorId: 'tribelt', position: 1 }], addresses: ['https://www.tribelt.nl/'],
          namesTribelt: true, citesTribelt: true, wrongClaims: [], costUsd: 0.0002,
        },
      }),
})

test('the Prompt test grid opens an answer and keeps its filters in the address', async ({ page }) => {
  // What asking again would cost comes from the server, per provider and mode.
  await page.route(`**/api/v1/admin/prompt-runs/${RUN}/retry-plan`, (route) =>
    route.fulfill({
      json: {
        askTasks: 1,
        askCostUsd: 0.0014,
        unknownTasks: 0,
        readTasks: 0,
        readCostUsd: 0,
        readingCostUsd: 0.0003,
        answerCosts: [
          { provider: 'openai', mode: 'knowledge', costUsd: 0.0014 },
          { provider: 'openai', mode: 'web', costUsd: 0.0214 },
        ],
      },
    }),
  )
  await page.route('**/api/v1/prompt-test', (route) => route.fulfill({ json: { promptSetVersion: 2, runs: [run], mentions: [], wrongClaims: [] } }))
  await page.route(`**/api/v1/prompt-runs/${RUN}/answers**`, (route) =>
    route.fulfill({ json: { run, items: [answer('web', '00000000-0000-4000-8000-000000000001', false), answer('knowledge', '00000000-0000-4000-8000-000000000002', true)] } }),
  )
  await page.goto('/ai-search/prompt-test')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('AI answers name Tribelt more than any competitor.')
  await page.getByRole('button', { name: /OpenAI, with web search, .*: Tribelt named first/ }).click()
  await expect(page).toHaveURL(/answer=/)
  await expect(page.getByRole('list', { name: 'Sources cited' })).toContainText('tribelt.nl')
  await expectAccessible(page)
  await page.getByRole('group', { name: 'Search mode' }).getByRole('button', { name: 'Own knowledge' }).click()
  await expect(page).toHaveURL(/mode=knowledge/)
  await page.reload()
  await expect(page.getByRole('group', { name: 'Search mode' }).getByRole('button', { name: 'Own knowledge' })).toHaveAttribute('aria-pressed', 'true')
  await page.getByRole('button', { name: /OpenAI, own knowledge, .*: No answer/ }).click()
  let asked: unknown = null
  await page.route(`**/api/v1/admin/prompt-runs/${RUN}/retry`, async (route) => {
    asked = route.request().postDataJSON()
    await route.fulfill({ status: 202, json: { run: { ...run, status: 'running' }, tasks: 1, estimatedCostUsd: 0.0014 } })
  })
  await page.getByRole('button', { name: 'Ask OpenAI again, about $0.0014' }).click()
  await expect(page.getByText('Asking one answer again, for about $0.00.')).toBeVisible()
  expect(asked).toEqual({ promptId: 'eyelink-application-twente', provider: 'openai', mode: 'knowledge' })
  await expectAccessible(page)
})

test("the AI readiness page shows a site's robots.txt when it is picked", async ({ page }) => {
  await page.goto('/ai-search/readiness')
  await expect(page.getByRole('region', { name: /set up for AI$/ })).toBeVisible()
  const sites = page.getByRole('region', { name: 'Site by site' })
  await sites.getByRole('rowgroup', { name: 'Competitors' }).getByRole('button').first().click()
  await expect(page).toHaveURL(/site=/)
  await expect(sites.getByText('What it means')).toBeVisible()
  await expectAccessible(page)
})
