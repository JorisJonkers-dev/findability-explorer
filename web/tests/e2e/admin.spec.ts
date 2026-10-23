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
  await expect(form.getByText(`Added ${name}.`)).toBeVisible()
  await expect(page.getByRole('heading', { level: 2, name })).toBeVisible()
  await expect(page).toHaveURL(new RegExp(`company=e2e-belts-${stamp}`))
  await expectAccessible(page)

  await page.getByRole('group', { name: 'Manage this company' }).getByRole('button', { name: 'Remove' }).click()
  await page.getByRole('button', { name: 'Delete with its 0 scans' }).click()
  await expect(page.getByRole('heading', { level: 2, name })).toHaveCount(0)
  await expect(page).toHaveURL('/competitors')
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
  await stubHistory(page)
  await page.goto('/competitors?company=twentebelt')
  await page.getByRole('link', { name: 'Search terms and history' }).click()
  await expect(page).toHaveURL('/competitors/twentebelt/history')
  await expect(page).toHaveTitle('Search terms and history · Findability Explorer')
  await expect(page.getByRole('heading', { level: 1, name: 'Twentebelt: its search terms, scan by scan.' })).toBeVisible()
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

test('an administrator accepts a candidate into the terminology map', async ({ page }) => {
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
    route.fulfill({ json: { items: accepted === null ? [candidate] : [], counts: { pending: accepted === null ? 1 : 0, accepted: accepted === null ? 0 : 1, rejected: 0 } } }),
  )
  await page.route('**/api/v1/admin/term-candidates/3a7bd3e2360a9f01/accept', async (route) => {
    accepted = route.request().postDataJSON()
    await route.fulfill({ json: { ...candidate, status: 'accepted' } })
  })
  await page.goto('/settings')
  await page.getByRole('link', { name: 'Review the terms scans found' }).click()
  await expect(page).toHaveURL('/settings/terms')
  await expect(page.getByRole('heading', { level: 1, name: 'Term review' })).toBeVisible()
  await expect(page.getByRole('heading', { level: 3, name: 'spiral freezer belt' })).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('button', { name: 'Accept…' }).click()
  const form = page.getByRole('form', { name: 'Accept spiral freezer belt' })
  await form.getByLabel('Term').fill('spiral freezer belt')
  await form.getByLabel('Intent tier').selectOption('product')
  await form.getByRole('button', { name: 'Add to the map' }).click()
  await expect(page.getByText('Added “spiral freezer belt” to the terminology map.')).toBeVisible()
  expect(accepted).toEqual({ term: 'spiral freezer belt', language: 'en', family: 'tri_flex_turn_curve', tier: 'product', usedBy: ['Twentebelt'] })
  await expect(page.getByRole('button', { name: 'Accepted (1)' })).toBeVisible()
})
