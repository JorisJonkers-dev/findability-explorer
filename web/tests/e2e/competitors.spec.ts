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
