import { expect, expectAccessible, test } from './fixtures'

const SECTIONS = [
  { path: '/buyers', title: 'Buyers', heading: 'Paid search buys traffic, not buyers.' },
  { path: '/ai-search', title: 'AI search', heading: 'An AI assistant can only recommend what it has read.' },
  { path: '/competitors', title: 'Competitors', heading: /competitors, the same belts, other names\.$/ },
  { path: '/terms', title: 'Search terms', heading: /ways to search for a metal belt\.$/ },
  { path: '/plan', title: 'The plan', heading: 'Measure first. Then build. Then expand.' },
] as const

test('the poster renders and is accessible', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1, name: /^One belt,\s*five names\.$/ })).toBeVisible()
  await expect(page).toHaveTitle('Overview · Findability Explorer')
  const sections = page.getByRole('navigation', { name: 'Sections' })
  await expect(sections.getByRole('link')).toHaveCount(5)
  // The live figures: the research's 44 competitors and the map's 90 terms in Dutch, German and English.
  await expect(sections.getByRole('link', { name: /^Competitors: 44 competitors/ })).toBeVisible()
  await expect(sections.getByRole('link', { name: /^Search terms: 90 search terms in 3 languages\. Tribelt's own website uses 30 of them\./ })).toBeVisible()
  await expectAccessible(page)
})

for (const section of SECTIONS) {
  test(`${section.title} can be opened directly and is accessible`, async ({ page }) => {
    await page.goto(section.path)
    await expect(page.getByRole('heading', { level: 1, name: section.heading })).toBeVisible()
    await expect(page).toHaveTitle(`${section.title} · Findability Explorer`)
    await expectAccessible(page)
  })
}

test('a tile zooms into its section, and Overview zooms back out', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('navigation', { name: 'Sections' }).getByRole('link', { name: /^The plan:/ }).click()
  await expect(page).toHaveURL('/plan')
  await expect(page.getByRole('heading', { level: 1, name: 'Measure first. Then build. Then expand.' })).toBeVisible()

  await page.getByRole('navigation', { name: 'Main' }).getByRole('button', { name: 'Overview' }).click()
  await expect(page).toHaveURL('/')
  await expect(page.getByRole('heading', { level: 1, name: /^One belt/ })).toBeVisible()
  // The browser's own back and forward agree with the zoom.
  await page.goForward()
  await expect(page).toHaveURL('/plan')
})

test('the company picker suggests a known company, and names an unknown one', async ({ page }) => {
  await page.goto('/competitors')
  const field = page.getByRole('combobox', { name: 'Website address' })
  await field.fill('twente')
  const options = page.getByRole('listbox', { name: 'Known companies' })
  await expect(options.getByRole('option').filter({ hasText: 'Twentebelt' })).toBeVisible()
  await expect(field).toHaveAttribute('aria-expanded', 'true')
  await expectAccessible(page)
  await field.press('ArrowDown')
  await field.press('Enter')
  await expect(field).toHaveValue(/twentebelt\.com/)
  await expect(field).toHaveAttribute('aria-expanded', 'false')

  await field.fill('https://www.example-belts.de')
  await expect(page.getByLabel('New company')).toHaveValue('Example Belts')
  await expect(page.getByText('The scan adds it as a new competitor.')).toBeVisible()
  await expectAccessible(page)
})

test('picking a company shows it, and keeps it in the address', async ({ page }) => {
  await page.goto('/competitors')
  await page.getByRole('list', { name: 'Companies' }).getByRole('button', { name: /^Märtens/ }).click()
  await expect(page).toHaveURL(/[?&]company=/)
  await expect(page.getByRole('heading', { level: 2, name: 'Märtens' })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('heading', { level: 2, name: 'Märtens' })).toBeVisible()
})

test('the scan form refuses an address it cannot scan', async ({ page }) => {
  await page.goto('/competitors')
  const field = page.getByRole('combobox', { name: 'Website address' })
  await field.fill('https://intranet')
  await page.getByRole('button', { name: 'Scan website' }).click()
  await expect(page.getByRole('alert')).toHaveText('Enter an address with a full host name, like https://www.example.com.')
  await expect(field).toHaveAttribute('aria-invalid', 'true')
  await expect(page).toHaveURL('/competitors')
})

test('the terms view filters by language and survives a reload', async ({ page }) => {
  await page.goto('/terms')
  await page.getByRole('button', { name: /^German: \d+ terms/ }).click()
  await expect(page).toHaveURL(/[?&]language=de(&|$)/)
  await expect(page.getByRole('button', { name: /^Remove the filter German/ })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('button', { name: /^German: \d+ terms/ })).toHaveAttribute('aria-pressed', 'true')
  await expectAccessible(page)
})

test('the settings page is reachable for an administrator, and accessible', async ({ page }) => {
  await page.goto('/')
  await page.goto('/plan')
  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Settings' }).click()
  await expect(page).toHaveURL('/settings')
  await expect(page.getByRole('heading', { level: 1, name: 'Language models' })).toBeVisible()
  await expect(page).toHaveTitle('Settings · Findability Explorer')
  await expect(page.getByLabel('API key', { exact: true })).toHaveAttribute('type', 'password')
  await expectAccessible(page)
})

test('unknown routes go to the poster, and old addresses to their new views', async ({ page }) => {
  await page.goto('/no/such/page')
  await expect(page).toHaveURL('/')
  await page.goto('/terminology?q=spiral')
  await expect(page).toHaveURL('/terms?q=spiral')
  await page.goto('/analyses')
  await expect(page).toHaveURL('/competitors')
})

test.describe('at phone width', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  test('no view scrolls sideways', async ({ page }) => {
    for (const path of ['/', ...SECTIONS.map((s) => s.path), '/settings']) {
      await page.goto(path)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
      expect(overflow, path).toBe(0)
    }
  })
})
