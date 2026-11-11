import { expect, expectAccessible, test } from './fixtures'

// The dev user of the default server is an administrator (playwright.config.ts). Starting a run asks
// Claude, so the start is answered here: this spec writes nothing to the shared database.
test('an administrator sees what a Recommendation run would do, and starts one', async ({ page }) => {
  let started = false
  await page.route('**/api/v1/admin/recommendation-runs', async (route) => {
    if (route.request().method() !== 'POST') return route.fallback()
    started = true
    const response = await route.fetch({ method: 'GET' })
    const body = (await response.json()) as Record<string, unknown>
    return route.fulfill({ status: 202, json: { ...body, running: true } })
  })
  await page.goto('/manage/recommendations')
  await expect(page.getByRole('heading', { level: 1, name: 'Recommendations' })).toBeVisible()
  const start = page.getByRole('region', { name: 'Start a Recommendation run' })
  await expect(start).toContainText('Reads the newest Prompt run')
  await expect(start).toContainText('Those you marked as done or edited stay.')
  await expect(page.getByRole('region', { name: 'Past runs' })).toBeVisible()
  await expectAccessible(page)
  const button = start.getByRole('button', { name: 'Start a Recommendation run' })
  // Without an Anthropic key, or with no gap to write, there is nothing to start.
  if ((await button.count()) > 0 && (await button.isEnabled())) {
    await button.click()
    await expect(page.getByText('The Recommendation run started.')).toBeVisible()
    expect(started).toBe(true)
  }
})
