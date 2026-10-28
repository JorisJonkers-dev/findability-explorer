import { expect, expectAccessible, test } from './fixtures'

const SECTIONS = [
  { path: '/buyers', title: 'Buyers', heading: 'Paid search buys traffic, not buyers.' },
  { path: '/ai-search', title: 'AI search', heading: 'An AI assistant can only recommend what it has read.' },
  { path: '/competitors', title: 'Competitors', heading: /^\d+ competitors are on our list\. The same belts, other names\.$/ },
  { path: '/terms', title: 'Search terms', heading: /ways to search for a metal belt\.$/ },
  { path: '/recommendations', title: 'Recommendations', heading: 'Check every Monday. Fix one thing a week. Watch the numbers move.' },
] as const

test('the overview renders its numbers and section cards, and is accessible', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  await expect(page).toHaveTitle('Overview · Findability Explorer')
  await expect(page.getByRole('heading', { level: 2, name: 'Numbers to watch' })).toBeVisible()
  await expect(page.locator('a[href^="/numbers/"]').first()).toBeVisible()
  const cards = page.locator('section[aria-labelledby="sections-title"] li a')
  await expect(cards).toHaveCount(4)
  await expect(cards.filter({ hasText: 'Open Search terms' })).toBeVisible()
  await expect(page.locator('a[href="/buyers"]')).toBeVisible()
  await expectAccessible(page)
})

test('the overview opens a number and goes back', async ({ page }) => {
  await page.goto('/')
  await page.locator('a[href="/numbers/aiShareOfVoice"]').click()
  await expect(page).toHaveURL('/numbers/aiShareOfVoice')
  await expect(page.getByRole('heading', { level: 2, name: 'Tribelt against the field' })).toBeVisible()
  await expectAccessible(page)
  await page.getByRole('link', { name: 'Back to Overview' }).click()
  await expect(page).toHaveURL('/')
  await expect(page.getByRole('heading', { level: 2, name: 'Numbers to watch' })).toBeVisible()
})

test('the overview opens Buyers and goes back', async ({ page }) => {
  await page.goto('/')
  await page.locator('a[href="/buyers"]').click()
  await expect(page).toHaveURL('/buyers')
  await expect(page.getByRole('heading', { level: 1, name: 'Paid search buys traffic, not buyers.' })).toBeVisible()
  await page.getByRole('link', { name: 'Back to Overview' }).click()
  await expect(page).toHaveURL('/')
})

test('the footer is on every page', async ({ page }) => {
  for (const path of ['/', '/buyers', '/numbers/aiShareOfVoice', '/terms', '/settings']) {
    await page.goto(path)
    await expect(page.locator('footer').first(), path).toBeVisible()
    await expect(page.locator('footer').first().getByRole('img', { name: 'University of Twente' }), path).toBeVisible()
  }
})

for (const section of SECTIONS) {
  test(`${section.title} can be opened directly and is accessible`, async ({ page }) => {
    await page.goto(section.path)
    await expect(page.getByRole('heading', { level: 1, name: section.heading })).toBeVisible()
    await expect(page).toHaveTitle(`${section.title} · Findability Explorer`)
    await expectAccessible(page)
  })
}

test('a section opens from the top bar, and Overview links back', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Recommendations', exact: true }).click()
  await expect(page).toHaveURL('/recommendations')
  await expect(page.getByRole('heading', { level: 1, name: 'Check every Monday. Fix one thing a week. Watch the numbers move.' })).toBeVisible()

  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Overview', exact: true }).click()
  await expect(page).toHaveURL('/')
  await expect(page.getByRole('heading', { level: 2, name: 'Numbers to watch' })).toBeVisible()
  // The browser's own back agrees with the links.
  await page.goBack()
  await expect(page).toHaveURL('/recommendations')
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
  await page.goto('/recommendations')
  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Settings' }).click()
  await expect(page).toHaveURL('/settings')
  await expect(page.getByRole('heading', { level: 1, name: 'Language models' })).toBeVisible()
  await expect(page).toHaveTitle('Settings · Findability Explorer')
  await expect(page.getByLabel('API key', { exact: true })).toHaveAttribute('type', 'password')
  await expectAccessible(page)
})

test('unknown routes go to the overview, and old addresses to their new views', async ({ page }) => {
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
