import type { Page } from '@playwright/test'
import { expect, expectAccessible, test } from './fixtures'

const SCAN = '6c1d2b0f-3e4a-4b5c-9d6e-7f8091a2b3c4'
const TOKENS = ['Googlebot', 'Google-Extended', 'Bingbot', 'OAI-SearchBot', 'ChatGPT-User', 'GPTBot', 'Claude-SearchBot', 'ClaudeBot', 'PerplexityBot', 'CCBot']

const readiness = (blocked: string[] = [], overrides: Record<string, unknown> = {}) => ({
  robots: 'found',
  bots: TOKENS.map((bot) => ({ bot, access: blocked.includes(bot) ? 'blocked' : 'allowed', named: true, followsAdvice: !blocked.includes(bot) })),
  sitemap: { declared: true, files: 1, addresses: 120, languages: ['nl', 'de', 'en'] },
  llmsTxt: { present: true, bytes: 2048, links: 12 },
  llmsFullTxt: false,
  markdown: false,
  homeRead: true,
  structuredData: ['Organization'],
  languages: ['nl', 'de', 'en'],
  stack: { cms: 'WordPress' },
  advice: ['sitemapDeclared', 'sitemapLanguages', 'llmsTxt', 'markdown', 'organization', 'product', 'languages'].map((check) => ({ check, ok: check !== 'markdown' && check !== 'product' })),
  gaps: 2,
  ...overrides,
})

const row = (id: string, name: string, role: string, scanId: string, r: object, url = `https://www.${id}.example`) => ({
  competitorId: id,
  name,
  role,
  scan: { id: scanId, competitorId: id, number: 2, url, status: 'done', maxPages: 15, pagesCrawled: 15, createdAt: '2026-10-26T09:00:00Z', finishedAt: '2026-10-26T09:02:00Z' },
  readiness: r,
})

const table = {
  items: [
    row('tribelt', 'Tribelt B.V.', 'tribelt', SCAN, readiness(['GPTBot']), 'https://www.tribelt.nl'),
    ...Array.from({ length: 20 }, (_, i) =>
      row(`belts-${String(i)}`, `Märtens Transportbänder Belts Number ${String(i + 1)} GmbH`, i === 19 ? 'group' : 'competitor', `00000000-0000-4000-8000-0000000000${String(10 + i)}`, readiness([], { markdown: i % 3 === 0 })),
    ),
  ],
  check: { running: false, checked: 0, failed: 0, total: 0 },
}

const ROBOTS = 'User-agent: *\nDisallow: /admin/\n\nUser-agent: GPTBot\nDisallow: /\n\nSitemap: https://www.tribelt.nl/sitemap.xml'

async function mockReadiness(page: Page): Promise<void> {
  await page.route('**/api/v1/ai-readiness', (route) => route.fulfill({ json: table }))
  await page.route(`**/api/v1/analyses/${SCAN}/readiness-files`, (route) =>
    route.fulfill({ json: { robotsTxt: ROBOTS, robotsTruncated: false, llmsTxt: '# Tribelt\n> Metal conveyor belts.', llmsTruncated: false, llmsFullTxt: '', llmsFullTruncated: false } }),
  )
}

test("AI readiness leads to tribelt.nl's page and its files, which a search highlights", async ({ page }) => {
  await mockReadiness(page)
  await page.goto('/ai-readiness')
  await expect(page.getByRole('heading', { level: 1, name: 'AI readiness' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Tribelt’s checks' }).getByRole('listitem')).toHaveCount(8)
  await expectAccessible(page)

  await page.getByRole('region', { name: 'Every site' }).getByRole('link', { name: 'Tribelt', exact: true }).click()
  await expect(page).toHaveURL('/ai-readiness/tribelt')
  await expect(page.getByRole('heading', { level: 1, name: 'Tribelt’s AI readiness' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Crawler policy' })).toContainText('Blocks GPTBot.')
  await expectAccessible(page)

  await page.getByRole('link', { name: 'Read the files' }).click()
  await expect(page).toHaveURL('/ai-readiness/tribelt/files')
  await expect(page.getByRole('heading', { level: 1, name: 'Files found on tribelt.nl' })).toBeVisible()
  await expect(page.getByRole('tab', { name: 'robots.txt' })).toHaveAttribute('aria-selected', 'true')
  await page.getByRole('searchbox', { name: 'Search in the file' }).fill('gptbot')
  const panel = page.getByRole('tabpanel')
  await expect(panel.locator('mark')).toHaveCount(1)
  await expect(panel).toContainText('1 match')
  await expectAccessible(page)

  await page.getByRole('tab', { name: 'llms-full.txt' }).click()
  await expect(panel).toContainText('The scan found no llms-full.txt on tribelt.nl.')
  await page.getByRole('tab', { name: 'sitemap.xml' }).click()
  await expect(panel).toContainText("This release does not keep the sitemap's text yet.")
})

for (const width of [390, 800, 1024, 1440]) {
  test(`Every site fits its box at ${String(width)} pixels`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 })
    await mockReadiness(page)
    await page.goto('/ai-readiness')
    const box = page.locator('[data-box="every-site"]')
    await expect(box.locator('tbody tr')).toHaveCount(15)
    const overflow = await box.evaluate((el) => el.scrollWidth - el.clientWidth)
    expect(overflow).toBeLessThanOrEqual(0)
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBe(0)
  })
}
