import { expect, expectAccessible, test } from './fixtures'

const SECTIONS = [
  { label: 'AI search', path: '/ai-search' },
  { label: 'Competitors', path: '/competitors' },
  { label: 'Search terms', path: '/terms' },
  { label: 'Recommendations', path: '/recommendations' },
] as const

test('the top bar reaches every section and marks the one shown', async ({ page }) => {
  await page.goto('/terms')
  const bar = page.getByRole('navigation', { name: 'Main' })
  for (const section of SECTIONS) {
    await bar.getByRole('link', { name: section.label, exact: true }).click()
    await expect(page).toHaveURL(section.path)
    await expect(bar.getByRole('link', { name: section.label, exact: true })).toHaveAttribute('aria-current', 'page')
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await expectAccessible(page)
  }
  for (const gone of ['Catching up', 'Products', 'Buyers', 'The plan']) {
    await expect(bar.getByRole('link', { name: gone, exact: true })).toHaveCount(0)
  }
  await bar.getByRole('link', { name: 'Overview' }).click()
  await expect(page).toHaveURL('/')
})

test('detail pages link back, and the old addresses redirect', async ({ page }) => {
  await page.goto('/ai-search/answers')
  await expect(page.getByRole('navigation', { name: 'In this section' })).toHaveCount(0)
  await page.getByRole('link', { name: 'Back to AI search' }).click()
  await expect(page).toHaveURL('/ai-search')

  await page.goto('/ai-search/readiness')
  await page.getByRole('link', { name: 'Back to AI search' }).click()
  await expect(page).toHaveURL('/ai-search')

  const moves = [
    ['/catching-up', '/'],
    ['/plan', '/recommendations'],
    ['/products', '/terms'],
    ['/products/open-spiral', '/terms/families/open-spiral'],
    ['/ai-search/prompt-test', '/ai-search/answers'],
    ['/competitors/all', '/competitors'],
  ] as const
  for (const [from, to] of moves) {
    await page.goto(from)
    await expect(page).toHaveURL(to)
  }
})

test('the old candidates review goes to Competitor review', async ({ page }) => {
  await page.goto('/settings/candidates')
  await expect(page).toHaveURL('/settings/competitors')
  await expect(page.getByRole('heading', { level: 1, name: 'Competitor review' })).toBeVisible()
  await expectAccessible(page)
})

test.describe('at phone width', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  test('the bar wraps, and no new page scrolls sideways', async ({ page }) => {
    for (const path of ['/', '/terms', '/terms/families/eyelink', '/competitors', '/ai-search', '/recommendations']) {
      await page.goto(path)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
      expect(overflow, path).toBe(0)
    }
  })
})
