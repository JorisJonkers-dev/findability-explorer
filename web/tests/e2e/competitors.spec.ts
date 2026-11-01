import { expect, expectAccessible, test } from './fixtures'

test('Competitors is one table with Tribelt pinned, each competitor opening its page', async ({ page }) => {
  await page.goto('/competitors')
  await expect(page.getByRole('heading', { level: 1, name: 'Competitors' })).toBeVisible()
  const table = page.getByRole('region', { name: 'The competitors' }).getByRole('table')
  await expect(table.getByRole('columnheader')).toHaveCount(7)
  await expect(table.getByRole('row').nth(1)).toContainText('For comparison')
  await expectAccessible(page)
  await table.getByRole('button', { name: /^Name, / }).click()
  await table.getByRole('link').first().click()
  await expect(page).toHaveURL(/\/competitors\/[^/]+$/)
  await expect(page.getByRole('region', { name: 'Against Tribelt' })).toBeVisible()
})

test("a competitor's page sets it against Tribelt, with what it sells, and links to its history", async ({ page }) => {
  await page.goto('/competitors/twentebelt')
  await expect(page).toHaveTitle('Competitor · Findability Explorer')
  const path = new URL(page.url()).pathname
  await expect(page.getByRole('link', { name: 'Back to Competitors' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Against Tribelt' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Where it is ahead of Tribelt' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'What it sells and what it calls it' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'In AI responses' })).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('link', { name: 'See its scan history' }).click()
  await expect(page).toHaveURL(`${path}/history`)
  await expect(page.getByRole('heading', { level: 1, name: 'Twentebelt, scan history' })).toBeVisible()
  await page.getByRole('link', { name: 'Back to Twentebelt' }).click()
  await expect(page).toHaveURL(path)
})

test('picking a measure says what it means', async ({ page }) => {
  await page.goto('/competitors/twentebelt')
  const chart = page.getByRole('region', { name: 'Against Tribelt' })
  await chart.getByRole('button', { name: /^AI readiness checks passed/ }).click()
  await expect(chart.getByRole('button', { name: /^AI readiness checks passed/ })).toHaveAttribute('aria-pressed', 'true')
  await expect(chart.getByRole('link', { name: 'Open the AI readiness table' })).toBeVisible()
})
