import { expect, expectAccessible, test } from './fixtures'

test("a competitor's page sets it against Tribelt, with its belts, and links to its history", async ({ page }) => {
  await page.goto('/competitors/twentebelt')
  await expect(page).toHaveTitle('Competitor · Findability Explorer')
  const path = new URL(page.url()).pathname
  await expect(page.getByRole('link', { name: 'Back to Competitors' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Key figures' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Twentebelt against Tribelt' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Which Tribelt belts it sells' })).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('link', { name: 'See its scan history' }).click()
  await expect(page).toHaveURL(`${path}/history`)
  await page.getByRole('link', { name: 'Back to the latest scan' }).click()
  await expect(page).toHaveURL(path)
})

test('picking a measure says what it means', async ({ page }) => {
  await page.goto('/competitors/twentebelt')
  const chart = page.getByRole('region', { name: 'Twentebelt against Tribelt' })
  await chart.getByRole('button', { name: /^AI readiness checks passed/ }).click()
  await expect(chart.getByRole('button', { name: /^AI readiness checks passed/ })).toHaveAttribute('aria-pressed', 'true')
  await expect(chart.getByRole('link', { name: 'Open the AI readiness table' })).toBeVisible()
})
