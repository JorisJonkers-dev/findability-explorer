import { expect, expectAccessible, test } from './fixtures'

test('a belt family opens from Search terms, with its terms, sellers and a way back', async ({ page }) => {
  await page.goto('/terms/families/eyelink')
  await expect(page).toHaveTitle('Belt family · Findability Explorer')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Eyelink belts')
  await expect(page.getByRole('region', { name: 'Terms by language' }).getByRole('table').getByRole('rowheader')).not.toHaveCount(0)
  await expect(page.getByRole('list', { name: 'Sellers' }).getByRole('listitem')).not.toHaveCount(0)
  await expect(page.getByRole('region', { name: 'AI responses about eyelink belts' })).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('link', { name: 'Back to Search terms' }).click()
  await expect(page).toHaveURL('/terms')
})

test('Search terms explains buyer intent, and opens a term the competitors use and tribelt.nl does not', async ({ page }) => {
  await page.goto('/terms')
  await expect(page.getByRole('heading', { level: 1, name: 'Search terms' })).toBeVisible()
  await page.getByText('What buyer intent means').click()
  await expect(page.getByText('The buyer names the type of belt.')).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('region', { name: 'Terms competitors use and tribelt.nl does not' }).getByRole('table').getByRole('link').first().click()
  await expect(page).toHaveURL(/\/terms\/[a-z-]+--/)
  await expect(page.getByRole('region', { name: 'Where competitors use it' })).toBeVisible()
  await expect(page.getByText(/Its buyer intent is/)).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('link', { name: 'Back to Every term' }).click()
  await expect(page).toHaveURL('/terms/all')
})

test('the old product addresses go to Search terms and its families', async ({ page }) => {
  await page.goto('/products')
  await expect(page).toHaveURL('/terms')
  await page.goto('/products/eyelink')
  await expect(page).toHaveURL('/terms/families/eyelink')
})
