import { expect, expectAccessible, test } from './fixtures'

test('All competitors lists the field, filters it, and opens a company', async ({ page }) => {
  await page.goto('/competitors/all')
  const list = page.getByRole('region', { name: 'All competitors' })
  await expect(list.getByRole('rowheader').first()).toContainText('Tribelt')
  await expectAccessible(page)
  await page.getByRole('group', { name: 'Show' }).getByRole('button', { name: 'Tribelt and group' }).click()
  await expect(page).toHaveURL(/show=ours/)
  await expect(list.getByRole('rowheader')).not.toHaveCount(0)
  await list.getByRole('rowheader').first().getByRole('link').click()
  await expect(page).toHaveURL(/\/competitors/)
})

test('an administrator picks companies into the tray and adds them in one go', async ({ page }) => {
  let sent: unknown = null
  await page.route('**/api/v1/admin/competitors/batch', async (route) => {
    sent = route.request().postDataJSON()
    await route.fulfill({ json: { items: [{ name: 'Example Belts', outcome: 'added', competitorId: 'example-belts' }], scans: 0 } })
  })
  await page.goto('/competitors/all')
  const field = page.getByRole('combobox', { name: 'Name or website' })
  await field.fill('example-belts.de')
  await expect(page.getByRole('option', { name: /Add example-belts\.de/ })).toBeVisible()
  await expectAccessible(page)
  await field.press('ArrowDown')
  await field.press('ArrowUp')
  await field.press('Enter')
  await expect(page.getByRole('list', { name: 'To add' })).toContainText('Example Belts')
  await page.getByLabel('Scan them straight away').uncheck()
  await page.getByRole('button', { name: 'Add 1 competitor' }).click()
  await expect(page.getByText('Added 1 competitor.')).toBeVisible()
  expect(sent).toEqual({ items: [{ name: 'Example Belts', url: 'https://example-belts.de/' }], scan: false })
})

test("a company's latest scan page shows its belts and AI readiness, with tabs to its history", async ({ page }) => {
  await page.goto('/competitors/all?show=competitor')
  await page.getByRole('region', { name: 'All competitors' }).getByRole('rowheader').first().getByRole('link').click()
  await expect(page).toHaveTitle('Latest scan · Findability Explorer')
  const path = new URL(page.url()).pathname
  await expect(page.getByRole('region', { name: 'Which Tribelt belts it sells' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Ready for AI' })).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('navigation', { name: 'In this section' }).getByRole('link', { name: 'Over time' }).click()
  await expect(page).toHaveURL(`${path}/history`)
})
