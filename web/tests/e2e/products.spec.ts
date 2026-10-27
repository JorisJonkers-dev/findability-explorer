import { expect, expectAccessible, test } from './fixtures'

test('the products show a card per belt family and open each one', async ({ page }) => {
  await page.goto('/products')
  const families = page.getByRole('navigation', { name: 'Belt families' })
  await expect(families.getByRole('link')).toHaveCount(5)
  await expectAccessible(page)
  await families.getByRole('link', { name: /^Eyelink/ }).click()
  await expect(page).toHaveURL('/products/eyelink')
  await expect(page.getByRole('heading', { level: 1 })).toContainText('Eyelink.')
  await expect(page.getByRole('region', { name: 'How buyers search for it' }).getByRole('rowheader')).not.toHaveCount(0)
  await expect(page.getByRole('list', { name: 'Sellers' }).getByRole('listitem')).not.toHaveCount(0)
  await expectAccessible(page)
  await page.getByRole('navigation', { name: 'In this section' }).getByRole('link', { name: 'Open spiral' }).click()
  await expect(page.getByRole('heading', { level: 1 })).toContainText('Open spiral.')
})
