import { expect, expectAccessible, test } from './fixtures'

// This file runs only in the viewer project: a signed-in account without ROLE_ADMIN.

test('an account that is no administrator sees the competitors, but no controls', async ({ page }) => {
  await page.goto('/competitors')
  await expect(page.getByRole('region', { name: 'The competitors' })).toBeVisible()
  await expect(page.getByRole('combobox', { name: 'Website address' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Scan website' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Re-scan all companies with data' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Add a company' })).toHaveCount(0)
  await expect(page.getByRole('heading', { name: 'Manage the list' })).toHaveCount(0)
  await expectAccessible(page)
})

test('a competitor’s page shows an account that is no administrator no way to scan or change it', async ({ page }) => {
  await page.goto('/competitors/twentebelt')
  await expect(page.getByRole('heading', { level: 2, name: 'Twentebelt against Tribelt' })).toBeVisible()
  await expect(page.getByRole('button', { name: /^(Scan again|Scan this website)$/ })).toHaveCount(0)
  await expect(page.getByRole('group', { name: 'Manage this company' })).toHaveCount(0)
  await expect(page.getByRole('link', { name: 'See its scan history' })).toBeVisible()
  await expectAccessible(page)
})

test('the term review is closed to an account that is no administrator', async ({ page, consoleErrors }) => {
  await page.goto('/settings/terms')
  await expect(page.getByRole('heading', { level: 1, name: 'Term review' })).toBeVisible()
  await expect(page.getByRole('alert')).toContainText('Only administrators can review terms.')
  await expectAccessible(page)
  // The browser logs the API's 403 itself; that refusal is what this test expects.
  const refused = consoleErrors.filter((e) => e.includes('403'))
  expect(refused.length).toBeGreaterThan(0)
  consoleErrors.splice(0, consoleErrors.length, ...consoleErrors.filter((e) => !e.includes('403')))
})
