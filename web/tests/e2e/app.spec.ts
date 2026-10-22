import { expect, expectAccessible, test } from './fixtures'

test('the overview renders and is accessible', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1, name: 'One belt, five names. Buyers search with all of them.' })).toBeVisible()
  const nav = page.getByRole('navigation', { name: 'Main' })
  await expect(nav.getByRole('link', { name: 'Overview' })).toHaveAttribute('aria-current', 'page')
  await expect(page.getByText('146 terms in 10 languages', { exact: false })).toBeVisible()
  await expect(page).toHaveTitle('Overview · Findability Explorer')
  await expectAccessible(page)
})

test('the scan form refuses an address it cannot scan', async ({ page }) => {
  await page.goto('/scan')
  await expect(page.getByRole('heading', { level: 1, name: 'Scan a competitor' })).toBeVisible()
  await page.getByLabel('Website address').fill('https://intranet')
  await page.getByRole('button', { name: 'Scan website' }).click()
  await expect(page.getByRole('alert')).toHaveText('Enter an address with a full host name, like https://www.example.com.')
  await expect(page.getByLabel('Website address')).toHaveAttribute('aria-invalid', 'true')
  await expect(page).toHaveURL('/scan')
  await expectAccessible(page)
})

test('searching the terminology for a product finds its company, and picking it filters by it', async ({ page }) => {
  await page.goto('/terminology')
  await page.getByLabel('Search terms and companies').fill('TwenteFlex')
  // The search waits for typing to pause, then puts the query in the URL.
  await expect(page).toHaveURL(/[?&]q=TwenteFlex(&|$)/)
  const table = page.getByRole('region', { name: 'Matching terms' })
  const row = table.getByRole('row').filter({ has: page.getByRole('button', { name: /^Twentebelt/ }) }).first()
  await expect(row).toBeVisible()
  await expectAccessible(page)

  await row.getByRole('button', { name: /^Twentebelt/ }).click()
  await expect(page).toHaveURL(/[?&]user=Twentebelt(&|$)/)
  await expect(page.getByText('Terms used by Twentebelt.')).toBeVisible()

  // The view survives a reload.
  await page.reload()
  await expect(page.getByLabel('Search terms and companies')).toHaveValue('TwenteFlex')
  await expect(table.getByRole('button', { name: /^Twentebelt/ }).first()).toHaveAttribute('aria-pressed', 'true')
})

test("the competitor matrix shows Twentebelt's name for Tri-Flex", async ({ page }) => {
  await page.goto('/competitors')
  const matrix = page.getByRole('region', { name: 'Competitors' })
  const row = matrix.getByRole('row').filter({ has: page.getByRole('rowheader', { name: /^Twentebelt/ }) })
  await expect(row).toContainText('TwenteFlex spiral belts')
  await expect(page.getByRole('heading', { name: "Tribelt's own group" })).toBeVisible()
  await expectAccessible(page)

  await row.getByRole('link', { name: 'Twentebelt' }).click()
  await expect(page).toHaveURL('/terminology?user=Twentebelt')
})

test('unknown routes go to the overview, and old addresses to their new pages', async ({ page }) => {
  await page.goto('/no/such/page')
  await expect(page).toHaveURL('/')
  await expect(page.getByRole('heading', { level: 1, name: /^One belt, five names/ })).toBeVisible()
  await page.goto('/keywords?q=spiral')
  await expect(page).toHaveURL('/terminology?q=spiral')
  await page.goto('/analyses')
  await expect(page).toHaveURL('/scan')
})

test('the widest page never scrolls sideways; its table scrolls in its own box', async ({ page }) => {
  await page.goto('/competitors')
  await expect(page.getByRole('region', { name: 'Competitors' }).getByRole('row').nth(1)).toBeVisible()
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
  expect(overflow).toBe(0)
})
