import { expect, expectAccessible, test } from './fixtures'

test('a belt family opens from Search terms, with its terms, sellers and a way back', async ({ page }) => {
  await page.goto('/terms/families/eyelink')
  await expect(page).toHaveTitle('Belt family · Findability Explorer')
  await expect(page.getByRole('heading', { level: 1 })).toContainText('Eyelink.')
  await expect(page.getByRole('table').getByRole('rowheader')).not.toHaveCount(0)
  await expect(page.getByRole('list', { name: 'Sellers' }).getByRole('listitem')).not.toHaveCount(0)
  await expectAccessible(page)
  await page.getByRole('link', { name: 'Back to Search terms' }).click()
  await expect(page).toHaveURL('/terms')
})

test('the old product addresses go to Search terms', async ({ page }) => {
  await page.goto('/products')
  await expect(page).toHaveURL('/terms')
  await page.goto('/products/eyelink')
  await expect(page).toHaveURL('/terms/families/eyelink')
})
