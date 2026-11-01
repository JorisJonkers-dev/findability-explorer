import type { Page } from '@playwright/test'
import { expect, expectAccessible, test } from './fixtures'

// The dev user of the default server is an administrator (playwright.config.ts).

test('an administrator adds a company, sees it, and deletes it again', async ({ page }) => {
  const stamp = `${String(Date.now())}${String(Math.floor(Math.random() * 1000))}`
  const name = `E2E Belts ${stamp}`
  await page.goto('/competitors')
  await page.getByRole('button', { name: 'Add a company' }).click()
  const form = page.getByRole('form', { name: 'Add a company' })
  await form.getByLabel('Company name').fill(name)
  await form.getByLabel('Company website').fill(`https://e2e-${stamp}.example.com`)
  await form.getByRole('button', { name: 'Add company' }).click()
  // The added company opens on its own page.
  await expect(page).toHaveURL(new RegExp(`/competitors/e2e-belts-${stamp}$`))
  await expect(page.getByRole('heading', { level: 2, name: `${name} against Tribelt` })).toBeVisible()
  await expectAccessible(page)

  await page.getByRole('group', { name: 'Manage this company' }).getByRole('button', { name: 'Remove' }).click()
  await page.getByRole('button', { name: 'Delete with its 0 scans' }).click()
  await expect(page).toHaveURL('/competitors')
  await expect(page.getByRole('region', { name: 'The competitors' }).getByRole('link', { name: new RegExp(name) })).toHaveCount(0)
})

const SCAN_1 = '11111111-1111-4111-8111-111111111111'
const SCAN_2 = '22222222-2222-4222-8222-222222222222'
const GROUPS = ['eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other', 'broad', 'spec'] as const

function summary(id: string, number: number, createdAt: string) {
  return {
    id,
    number,
    competitorId: 'twentebelt',
    url: 'https://www.twentebelt.com/',
    status: 'done',
    maxPages: 60,
    pagesCrawled: 52,
    createdAt,
    finishedAt: createdAt,
    matchingModel: 'anthropic: claude-haiku-5-5',
  }
}

/** Two finished scans of Twentebelt, answered by the browser: a real one would crawl the site. */
async function stubHistory(page: Page): Promise<void> {
  const scans = [summary(SCAN_2, 2, '2026-10-23T09:30:00Z'), summary(SCAN_1, 1, '2026-10-22T09:30:00Z')]
  await page.route('**/api/v1/competitors/twentebelt', async (route) => {
    const real = await route.fetch()
    const detail = (await real.json()) as { competitor: unknown }
    await route.fulfill({ json: { competitor: detail.competitor, scans } })
  })
  await page.route('**/api/v1/competitors/twentebelt/profile', (route) =>
    route.fulfill({
      json: {
        competitorId: 'twentebelt',
        scan: scans[0],
        finishedScans: 2,
        languages: ['en', 'nl'],
        groups: GROUPS.map((group) => ({
          group,
          terms:
            group === 'tri_flex_turn_curve'
              ? [
                  { phrase: 'spiral belts', language: 'en', score: 40, pages: 9, tier: 'product', family: 'tri_flex_turn_curve', inMap: true, firstScan: 1 },
                  { phrase: 'spiraalbanden', language: 'nl', score: 12, pages: 3, tier: 'product', family: 'tri_flex_turn_curve', closestTerm: 'spiraalband', inMap: false, firstScan: 2 },
                ]
              : [],
        })),
      },
    }),
  )
  await page.route('**/api/v1/competitors/twentebelt/scans/compare**', (route) =>
    route.fulfill({
      json: {
        from: scans[1],
        to: scans[0],
        groups: GROUPS.map((group) => ({
          group,
          added: group === 'tri_flex_turn_curve' ? [{ phrase: 'spiraalbanden', language: 'nl', tier: 'product', inMap: false, toScore: 12, toPages: 3 }] : [],
          removed: group === 'broad' ? [{ phrase: 'transportbanden', language: 'nl', tier: 'broad', inMap: true, fromScore: 9, fromPages: 2 }] : [],
          moved: group === 'tri_flex_turn_curve' ? [{ phrase: 'spiral belts', language: 'en', tier: 'product', inMap: true, fromScore: 20, toScore: 40, fromPages: 4, toPages: 9 }] : [],
        })),
      },
    }),
  )
}

test('a company’s search terms, scan history and comparison render', async ({ page }) => {
  await page.goto('/competitors/twentebelt')
  // The history is stubbed once the page has finished loading: the stub's scans have no pages of their own to read,
  // so a company page still asking for its data would be answered with scans it cannot fetch (a 404).
  await page.waitForLoadState('networkidle')
  await stubHistory(page)
  await page.getByRole('link', { name: 'See its scan history' }).click()
  await expect(page).toHaveURL('/competitors/twentebelt/history')
  await expect(page).toHaveTitle('Over time · Findability Explorer')
  await expect(page.getByRole('heading', { level: 1, name: 'How Twentebelt changed, scan by scan.' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Back to the latest scan' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Tri-Flex (2)' })).toHaveAttribute('aria-pressed', 'true')
  await expect(page.getByRole('table', { name: 'English (1)' })).toContainText('spiral belts')
  await expect(page.getByRole('table', { name: 'Dutch (1)' })).toContainText('new in this scan')
  const scans = page.getByRole('list', { name: 'Scans, newest first' })
  await expect(scans.getByRole('listitem')).toHaveCount(2)
  await expect(scans).toContainText('52 of 60 pages')
  const compare = page.getByRole('region', { name: 'Compare two scans' })
  await expect(compare).toContainText('1 added')
  await expect(compare.getByRole('region', { name: 'Tri-Flex' })).toContainText('score 20 → 40, pages 4 → 9')
  await expect(compare.getByRole('region', { name: 'Broad' })).toContainText('Removed')
  await expectAccessible(page)
})

test('an administrator accepts a candidate into the search term list', async ({ page }) => {
  const candidate = {
    id: '3a7bd3e2360a9f01',
    phrase: 'spiral freezer belt',
    status: 'pending',
    languages: ['en'],
    companies: [{ competitorId: 'twentebelt', name: 'Twentebelt' }],
    scans: 2,
    bestScore: 42.5,
    maxPages: 6,
    suggestedTier: 'application',
    suggestedFamily: 'tri_flex_turn_curve',
    closestTerm: 'spiral belt',
    lastSeenAt: '2026-10-23T09:30:00Z',
  }
  let accepted: unknown = null
  await page.route('**/api/v1/admin/term-candidates?**', (route) =>
    route.fulfill({ json: { items: accepted === null ? [candidate] : [], counts: { pending: accepted === null ? 1 : 0, accepted: accepted === null ? 0 : 1, rejected: 0 }, review: { on: true, running: false, waiting: 0, monthTokens: 0, monthCostUsd: 0 } } }),
  )
  await page.route('**/api/v1/admin/term-candidates/3a7bd3e2360a9f01/accept', async (route) => {
    accepted = route.request().postDataJSON()
    await route.fulfill({ json: { ...candidate, status: 'accepted' } })
  })
  await page.goto('/manage')
  await page.getByRole('navigation', { name: 'Manage' }).getByRole('link', { name: /^Search term list/ }).click()
  await expect(page).toHaveURL('/manage/terms')
  await page.getByRole('link', { name: 'Review the terms' }).click()
  await expect(page).toHaveURL('/manage/terms/waiting')
  await expect(page.getByRole('heading', { level: 1, name: 'Waiting terms' })).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Manage' }).getByRole('link', { name: /^Waiting terms/ })).toHaveAttribute('aria-current', 'page')
  await expect(page.getByRole('heading', { level: 3, name: 'spiral freezer belt' })).toBeVisible()
  await expect(page.getByText('Claude Haiku has reviewed every search term.')).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('button', { name: 'Accept…' }).click()
  const form = page.getByRole('form', { name: 'Accept spiral freezer belt' })
  await form.getByLabel('Term').fill('spiral freezer belt')
  await form.getByLabel('Buyer intent').selectOption('product')
  await form.getByRole('button', { name: 'Add to the list' }).click()
  await expect(page.getByText('Added “spiral freezer belt” to the search term list.')).toBeVisible()
  expect(accepted).toEqual({ term: 'spiral freezer belt', language: 'en', family: 'tri_flex_turn_curve', tier: 'product', usedBy: ['Twentebelt'] })
  await expect(page.getByRole('button', { name: 'Accepted (1)' })).toBeVisible()
})

test('an administrator follows a Prompt run from Activity, stops it, and sees it stopped by hand', async ({ page }) => {
  const RUN = '9d8e7f6a-5b4c-4d3e-8f2a-1b0c9d8e7f6b'
  // The e2e server has no OpenRouter key, so no run can start; a running run is answered by the browser.
  const run = {
    id: RUN,
    promptSetVersion: 2,
    factSheetVersion: 1,
    trigger: 'manual',
    startedBy: 'e2e',
    startedAt: '2026-11-01T08:29:00Z',
    status: 'running',
    models: [{ provider: 'openai', model: 'openai/gpt-oss-20b', free: false }],
    tasks: 400,
    answered: 90,
    failed: 10,
    inputTokens: 0,
    outputTokens: 0,
    costUsd: 0.1,
    byProvider: [],
    cells: [],
  }
  let stopped = false
  await page.route('**/api/v1/prompt-test', (route) =>
    route.fulfill({
      json: {
        promptSetVersion: 2,
        runs: [stopped ? { ...run, status: 'stopped', finishedAt: '2026-11-01T08:40:00Z', failure: 'Stopped by hand. The responses so far are kept, and a retry finishes the run.' } : run],
        mentions: [],
        wrongClaims: [],
      },
    }),
  )
  const work = { id: RUN, kind: 'promptRun', startedAt: run.startedAt, done: 100, total: 400, failed: 10, openRouterCostUsd: 0.1, anthropicCostUsd: 0 }
  await page.route('**/api/v1/admin/activity', (route) =>
    route.fulfill({
      json: {
        running: stopped ? [] : [work],
        recent: stopped ? [{ ...work, finishedAt: '2026-11-01T08:40:00Z', ending: 'stoppedByHand' }] : [],
        spending: { since: '2026-11-01T00:00:00Z', lines: [{ kind: 'promptRuns', key: 'openRouter', costUsd: 0.1 }] },
      },
    }),
  )
  await page.route(`**/api/v1/admin/prompt-runs/${RUN}/stop`, async (route) => {
    stopped = true
    await route.fulfill({ status: 204 })
  })
  await page.route(`**/api/v1/admin/prompt-runs/${RUN}/retry-plan`, (route) =>
    route.fulfill({ json: { askTasks: 300, askCostUsd: 0.3, unknownTasks: 0, readTasks: 0, readCostUsd: 0, readingCostUsd: 0, answerCosts: [] } }),
  )

  await page.goto('/manage')
  await expect(page.getByRole('heading', { level: 1, name: 'Activity' })).toBeVisible()
  const running = page.getByRole('region', { name: 'Prompt run' })
  await expect(running).toContainText('100 of 400 responses')
  await expect(page.getByRole('region', { name: 'Recent work' })).toBeVisible()
  await expectAccessible(page)

  await page.getByRole('navigation', { name: 'Manage' }).getByRole('link', { name: /^Prompt test/ }).click()
  await expect(page).toHaveURL('/manage/prompt-test')
  await expect(page.getByRole('region', { name: 'Start a Prompt run' })).toContainText('A Prompt run is already running. Stop it or wait for it to finish.')
  await expectAccessible(page)

  await page.getByRole('link', { name: 'See the runs' }).click()
  await expect(page).toHaveURL('/manage/prompt-test/runs')
  const widget = page.getByRole('region', { name: 'Prompt run' })
  await widget.getByRole('button', { name: 'Stop the Prompt run' }).click()
  await expect(widget).toContainText('Stopping keeps the responses so far. You can finish the run later with a retry.')
  await expectAccessible(page)
  await widget.getByRole('button', { name: 'Stop the Prompt run' }).click()
  await expect(page.getByText('The Prompt run stops within a minute. Its responses so far are kept.')).toBeVisible()
  await expect(page.getByRole('region', { name: 'All runs' }).getByRole('row').nth(1)).toContainText('Stopped by hand')
  await expect(page.getByRole('region', { name: 'Run of 1 November 2026' }).getByRole('button', { name: 'Retry the 300 failed responses' })).toBeVisible()

  await page.getByRole('navigation', { name: 'Manage' }).getByRole('link', { name: 'Activity' }).click()
  await expect(page.getByRole('region', { name: 'Recent work' }).getByRole('row', { name: /Prompt run/ })).toContainText('Stopped by hand')
})
