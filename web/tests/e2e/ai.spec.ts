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
  finishedAt: '2026-11-02T06:20:00Z', status: 'done', complete: true, models: [{ provider: 'openai', model: 'openai/gpt-oss-20b', free: false }],
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

test('Over time on AI responses switches Prompt set versions and keeps the choice in the address', async ({ page }) => {
  // Made-up figures: a partial run of version 1, its key refused, then a run of version 2.
  await page.route('**/api/v1/prompt-test/history', (route) =>
    route.fulfill({
      json: {
        versions: [
          { version: 1, publishedAt: '2026-10-26T15:04:51Z', questions: 90, note: 'A supplier, an application, and a problem question for each belt family, in six scopes.' },
          { version: 2, publishedAt: '2026-10-27T06:35:33Z', questions: 20, note: 'Kept the application question a buyer most often asks, in four scopes.' },
        ],
        runs: [
          { runId: '0b6a1c52-3e0f-4c1e-9a8d-6f1e2d3c4b5a', startedAt: '2026-10-27T06:03:12Z', promptSetVersion: 1, tasks: 1260, answered: 62, read: 47, partial: true, failure: 'OpenRouter answered 401: User not found.', all: 0.4, dutch: 0.48 },
          { runId: '7c2d9e41-5a6b-4f70-8e1d-2b3c4d5e6f70', startedAt: '2026-10-27T08:36:32Z', promptSetVersion: 2, tasks: 200, answered: 200, read: 183, partial: false, all: 0.25, dutch: 0.41, english: 0.1 },
        ],
      },
    }),
  )
  await oneRun(page)
  await page.goto('/ai-responses')
  const overTime = page.getByRole('region', { name: 'Over time' })
  await expect(overTime.getByText('The questions changed with each version, so the line breaks between versions.', { exact: false })).toBeVisible()
  await expect(overTime.getByRole('img', { name: 'Over time' })).toBeVisible()
  await expectAccessible(page)
  const versions = overTime.getByRole('group', { name: 'Prompt set' })
  await versions.getByRole('button', { name: 'Version 1' }).click()
  await expect(page).toHaveURL(/version=1/)
  await expect(versions.getByRole('button', { name: 'Version 1' })).toHaveAttribute('aria-pressed', 'true')
  await expect(overTime.getByText(/^Version 1, published 26 October 2026: 90 questions\..*It has one run so far, so there is no line yet\.$/)).toBeVisible()
  await expect(overTime.getByText(/a partial run of version 1\. Only 62 of its 1,260 requests got a response/)).toBeVisible()
  await page.reload()
  await expect(versions.getByRole('button', { name: 'Version 1' })).toHaveAttribute('aria-pressed', 'true')
  await expectAccessible(page)
})
