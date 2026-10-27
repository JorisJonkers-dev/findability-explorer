import { expect, expectAccessible, test } from './fixtures'

const VIEWS = [
  { path: '/ai-search/readiness', title: 'AI readiness', heading: 'Who lets search engines and AI assistants in.' },
  { path: '/ai-search/prompt-test', title: 'Prompt test', heading: 'Do AI assistants name Tribelt?' },
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
