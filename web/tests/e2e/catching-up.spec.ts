import { expect, expectAccessible, test } from './fixtures'

test('Catching up compares Tribelt on every measure and is accessible', async ({ page }) => {
  await page.goto('/catching-up')
  await expect(page).toHaveTitle('Catching up · Findability Explorer')
  const field = page.getByRole('region', { name: 'Tribelt against the field' })
  await expect(field.getByRole('figure')).toHaveCount(5)
  await expect(field.getByRole('figure').first()).toContainText('How often AI answers name the company')
  await expect(field.getByRole('figure').first()).toContainText('Typical competitor')
  await expectAccessible(page)
})
