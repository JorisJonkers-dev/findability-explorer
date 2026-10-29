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
  // A scan's own page is in Manage, so it is not offered.
  await expect(page.getByRole('link', { name: 'Open its latest scan' })).toHaveCount(0)
  await expectAccessible(page)
})

test('Manage is closed to an account that is no administrator, with the way to the Overview', async ({ page }) => {
  await page.goto('/')
  const bar = page.getByRole('navigation', { name: 'Main' })
  await expect(bar.getByRole('link', { name: 'Help', exact: true })).toBeVisible()
  await expect(bar.getByRole('link', { name: 'Manage', exact: true })).toHaveCount(0)
  await page.goto('/manage')
  await expect(page.getByText('Only administrators can start, stop, and change the work behind the results.')).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Manage' })).toHaveCount(0)
  await expect(bar.getByRole('link', { name: 'Manage', exact: true })).toHaveCount(0)
  await expectAccessible(page)
  await page.getByRole('link', { name: 'Go to the Overview' }).click()
  await expect(page).toHaveURL('/')
})

test('the term review is closed to an account that is no administrator', async ({ page, consoleErrors }) => {
  await page.goto('/settings/terms')
  await expect(page).toHaveURL('/manage/terms/waiting')
  await expect(page.getByText('Only administrators can start, stop, and change the work behind the results.')).toBeVisible()
  await expect(page.getByRole('list', { name: 'Candidates' })).toHaveCount(0)
  await expectAccessible(page)
  // The page still asks for its list, and the browser logs the API's 403 itself; that refusal is expected.
  consoleErrors.splice(0, consoleErrors.length, ...consoleErrors.filter((e) => !e.includes('403')))
})
