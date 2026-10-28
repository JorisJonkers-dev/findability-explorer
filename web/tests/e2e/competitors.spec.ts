import { expect, expectAccessible, test } from './fixtures'

test("a company's latest scan page shows its belts and AI readiness, with a link to its history", async ({ page }) => {
  await page.goto('/competitors/tribelt')
  await expect(page).toHaveTitle('Latest scan · Findability Explorer')
  const path = new URL(page.url()).pathname
  await expect(page.getByRole('link', { name: 'Back to Competitors' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Which Tribelt belts it sells' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Ready for AI' })).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('link', { name: 'See how it changed over time' }).click()
  await expect(page).toHaveURL(`${path}/history`)
  await page.getByRole('link', { name: 'Back to the latest scan' }).click()
  await expect(page).toHaveURL(path)
})
