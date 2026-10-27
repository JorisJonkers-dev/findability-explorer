import { expect, expectAccessible, test } from './fixtures'

const SECTIONS = [
  { label: 'Catching up', path: '/catching-up' },
  { label: 'AI search', path: '/ai-search' },
  { label: 'Competitors', path: '/competitors' },
  { label: 'Products', path: '/products' },
  { label: 'Search terms', path: '/terms' },
  { label: 'The plan', path: '/plan' },
  { label: 'Buyers', path: '/buyers' },
] as const

test('the top bar reaches every section and marks the one shown', async ({ page }) => {
  await page.goto('/catching-up')
  const bar = page.getByRole('navigation', { name: 'Main' })
  for (const section of SECTIONS) {
    await bar.getByRole('link', { name: section.label, exact: true }).click()
    await expect(page).toHaveURL(section.path)
    await expect(bar.getByRole('link', { name: section.label, exact: true })).toHaveAttribute('aria-current', 'page')
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await expectAccessible(page)
  }
  await bar.getByRole('link', { name: 'Overview' }).click()
  await expect(page).toHaveURL('/')
})

test("a section's tabs open its pages", async ({ page }) => {
  await page.goto('/ai-search')
  const tabs = page.getByRole('navigation', { name: 'In this section' })
  await tabs.getByRole('link', { name: 'Prompt test' }).click()
  await expect(page).toHaveURL('/ai-search/prompt-test')
  await expect(tabs.getByRole('link', { name: 'Prompt test' })).toHaveAttribute('aria-current', 'page')
  await tabs.getByRole('link', { name: 'AI readiness' }).click()
  await expect(page).toHaveURL('/ai-search/readiness')

  await page.goto('/competitors')
  await tabs.getByRole('link', { name: 'All competitors' }).click()
  await expect(page).toHaveURL('/competitors/all')
  await expectAccessible(page)

  await page.goto('/products')
  await tabs.getByRole('link', { name: 'Open spiral' }).click()
  await expect(page).toHaveURL('/products/open-spiral')
  await expectAccessible(page)
})

test('the old candidates review goes to All competitors', async ({ page }) => {
  await page.goto('/settings/candidates')
  await expect(page).toHaveURL('/competitors/all')
})

test.describe('at phone width', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  test('the bar and the tabs wrap, and no new page scrolls sideways', async ({ page }) => {
    for (const path of ['/catching-up', '/products', '/products/eyelink', '/competitors/all', '/ai-search']) {
      await page.goto(path)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
      expect(overflow, path).toBe(0)
    }
  })
})
