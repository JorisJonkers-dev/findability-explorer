import type { Page } from '@playwright/test'
import { expect, expectAccessible, test } from './fixtures'

const VIEWS = [
  { path: '/ai-readiness', title: 'AI readiness', heading: /./ },
  { path: '/ai-responses/all', title: 'Every response', heading: 'Every response' },
  { path: '/manage/prompt-test', title: 'Prompt test', heading: 'Prompt test' },
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
  await page.goto('/ai-readiness')
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

const RUN = '9d8e7f6a-5b4c-4d3e-8f2a-1b0c9d8e7f6a'
const NAMED = '00000000-0000-4000-8000-000000000001'

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
  model: 'openai/gpt-oss-20b', mode,
  text: failed ? '' : '## Welke band?\n**Twentebelt** en Tribelt maken oogbanden. <b>geen HTML</b>', citations: [],
  at: '2026-11-02T06:10:00Z', costUsd: 0.0014, inputTokens: 900, outputTokens: 300,
  ...(failed
    ? { failure: 'The model is rate limited.' }
    : {
        extraction: {
          failed: false,
          companies: [{ name: 'Twentebelt', competitorId: 'twentebelt', position: 1 }, { name: 'Tribelt B.V.', competitorId: 'twentebelt', position: 2 }],
          addresses: ['https://www.tribelt.nl/'],
          namesTribelt: true, citesTribelt: true, wrongClaims: ['Tribelt is in Hengelo.'], leftOut: [], costUsd: 0.0002,
        },
      }),
})

/** Answers the Prompt test with one run of two responses, as the e2e database has no run. */
async function oneRun(page: Page): Promise<void> {
  await page.route('**/api/v1/prompt-test', (route) => route.fulfill({ json: { promptSetVersion: 2, runs: [run], mentions: [], wrongClaims: [] } }))
  await page.route(`**/api/v1/prompt-runs/${RUN}/answers**`, (route) =>
    route.fulfill({ json: { run, items: [answer('web', NAMED, false), answer('knowledge', '00000000-0000-4000-8000-000000000002', true)] } }),
  )
}

test('the AI responses page shows its figures and sections, and leads on to every response', async ({ page }) => {
  await oneRun(page)
  await page.goto('/ai-responses')
  await expect(page.getByRole('heading', { level: 1, name: 'AI responses' })).toBeVisible()
  for (const name of ['Who the responses name', 'By model', 'The questions asked', 'By belt family and scope']) {
    await expect(page.getByRole('heading', { level: 2, name })).toBeVisible()
  }
  await expectAccessible(page)
  await page.getByRole('link', { name: 'See every response' }).click()
  await expect(page).toHaveURL('/ai-responses/all')
  await expect(page.getByRole('heading', { level: 1, name: 'Every response' })).toBeVisible()
})

test('Every response opens one response, and keeps its filters in the address', async ({ page }) => {
  await oneRun(page)
  await page.goto('/ai-responses/all')
  await expect(page.getByRole('heading', { level: 1, name: 'Every response' })).toBeVisible()
  await expect(page.getByText('2 responses shown.')).toBeVisible()
  await expectAccessible(page)
  await page.getByLabel('Search mode').selectOption('web')
  await expect(page).toHaveURL(/mode=web/)
  await page.reload()
  await expect(page.getByLabel('Search mode')).toHaveValue('web')
  await expect(page.getByText('1 response shown.')).toBeVisible()

  await page.getByRole('link', { name: 'OpenAI, gpt-oss-20b, with web search. Tribelt is named, number 2.' }).click()
  await expect(page).toHaveURL(`/ai-responses/${NAMED}?run=${RUN}`)
  await expect(page.getByRole('heading', { level: 1, name: 'OpenAI, gpt-oss-20b, with web search' })).toBeVisible()
  await expect(page.getByText('Tribelt is named, number 2.').first()).toBeVisible()
  await expect(page.getByRole('heading', { level: 4, name: 'Welke band?' })).toBeVisible()
  await expect(page.locator('[data-part="response"] b')).toHaveCount(0)
  await expect(page.getByRole('list', { name: 'Sources cited' })).toContainText('tribelt.nl')
  await expect(page.getByText('Tribelt is in Hengelo.')).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('link', { name: 'Back to Every response' }).click()
  await expect(page).toHaveURL(`/ai-responses/all?run=${RUN}`)
})

test("the AI readiness page shows a site's robots.txt when it is picked", async ({ page }) => {
  await page.goto('/ai-readiness')
  await expect(page.getByRole('region', { name: /^How many of the \d+ sites pass each check$/ })).toBeVisible()
  const sites = page.getByRole('region', { name: 'Every site, every column' })
  await sites.getByRole('rowgroup', { name: 'Competitors' }).getByRole('button').first().click()
  await expect(page).toHaveURL(/site=/)
  await expect(sites.getByText('What it means')).toBeVisible()
  await expectAccessible(page)
})
