import { expect, expectAccessible, test } from './fixtures'

// The dev user of the default server is an administrator (playwright.config.ts).

test('an administrator adds a company in Manage, opens it, and removes it again', async ({ page }) => {
  // A name and a site of its own, so no other spec, on this branch or another, meets it in the shared database.
  const stamp = `${String(Date.now())}${String(Math.floor(Math.random() * 1000))}`
  const name = `Manage Spec Belts ${stamp}`
  const site = `manage-spec-${stamp}.org`
  try {
    await page.goto('/manage/competitors')
    await expect(page.getByRole('heading', { level: 1, name: 'Competitor list' })).toBeVisible()
    await expectAccessible(page)
    await page.getByRole('link', { name: 'See the companies' }).click()
    await expect(page).toHaveURL('/manage/competitors/companies')
    const form = page.getByRole('form', { name: 'Add a company' })
    await form.getByLabel('Name').fill(name)
    await form.getByLabel('Website').fill(`https://www.${site}`)
    await form.getByRole('button', { name: 'Add the company' }).click()
    await expect(form).toContainText(`${name} is added to the list.`)
    // The list shows its first rows; the search finds the new one.
    await page.getByRole('searchbox', { name: 'Search' }).fill(name)
    await page.getByRole('table', { name: 'Companies' }).getByRole('link', { name, exact: true }).click()
    await expect(page.getByRole('heading', { level: 1, name })).toBeVisible()
    await expectAccessible(page)
    await page.getByRole('button', { name: 'Remove', exact: true }).click()
    await page.getByRole('button', { name: 'Delete with its 0 scans' }).click()
    await expect(page).toHaveURL('/manage/competitors/companies')
    await page.getByRole('searchbox', { name: 'Search' }).fill(name)
    await expect(page.getByRole('table', { name: 'Companies' }).getByRole('link', { name, exact: true })).toHaveCount(0)
  } finally {
    // Whatever happened above, the company leaves the shared database again.
    const list = (await (await page.request.get('/api/v1/competitors')).json()) as { items: { id: string; host?: string }[] }
    for (const c of list.items.filter((x) => x.host === site)) await page.request.delete(`/api/v1/competitors/${c.id}`)
  }
})

test('Website scans opens on the four ways to start, and each subpage is reached from the menu', async ({ page }) => {
  await page.goto('/manage/scans')
  await expect(page.getByRole('heading', { level: 1, name: 'Website scans' })).toBeVisible()
  for (const start of ['Scan tribelt.nl', 'Scan this site', 'Rescan these competitors', 'Check AI readiness again']) {
    await expect(page.getByRole('button', { name: start })).toBeVisible()
  }
  await expectAccessible(page)
  const menu = page.getByRole('navigation', { name: 'Manage' })
  for (const [label, path] of [
    ['Scan queue', '/manage/scans/queue'],
    ['Recent scans', '/manage/scans/recent'],
    ['Scan settings', '/manage/scans/settings'],
  ] as const) {
    await menu.getByRole('link', { name: new RegExp(`^${label}`) }).click()
    await expect(page).toHaveURL(path)
    await expect(page.getByRole('heading', { level: 1, name: label })).toBeVisible()
    await expectAccessible(page)
  }
})

test('the other Manage pages open, each in its place in the menu', async ({ page }) => {
  for (const [path, title] of [
    ['/manage/competitors/candidates', 'Candidates'],
    ['/manage/competitors/check', 'Check the list'],
    ['/manage/competitors/definition', 'Competitor definition'],
    ['/manage/terms', 'Search term list'],
    ['/manage/terms/review', 'Term review'],
    ['/manage/recommendations', 'Recommendations'],
    ['/manage/recommendations/tick-off', 'Tick off yourself'],
    ['/manage/recommendations/goals', 'Goals'],
    ['/manage/keys', 'Keys and spending'],
    ['/manage/keys/openrouter', 'OpenRouter key'],
    ['/manage/keys/anthropic', 'Anthropic key'],
    ['/manage/keys/budgets', 'Budgets'],
    ['/manage/keys/events', 'Events'],
  ] as const) {
    await page.goto(path)
    await expect(page.getByRole('heading', { level: 1, name: title, exact: true })).toBeVisible()
    await expect(page.getByRole('navigation', { name: 'Manage' }).locator('a[aria-current="page"]')).toHaveCount(1)
    await expectAccessible(page)
  }
})

test('no Manage page scrolls sideways on a phone', async ({ page }) => {
  await page.setViewportSize({ width: 400, height: 800 })
  for (const path of ['/manage/scans', '/manage/competitors/companies', '/manage/keys/openrouter', '/manage/recommendations/goals', '/manage/terms/waiting']) {
    await page.goto(path)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    const wide = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
    expect(wide, path).toBeLessThanOrEqual(0)
  }
})

test('an administrator edits the competitor definition, which saving does not check against', async ({ page }) => {
  // The version the code held is the first stored one. Saving is answered here, so this spec stores
  // no version in the shared database and leaves the definition in use as it was.
  let saved: unknown = null
  await page.route('**/api/v1/admin/competitor-definition', async (route) => {
    if (route.request().method() !== 'PUT') return route.fallback()
    saved = route.request().postDataJSON()
    const current = { version: 3, text: 'Makers of metal process belts compete.', savedAt: '2026-11-01T10:00:00Z', savedBy: 'e2e' }
    return route.fulfill({ json: { current, versions: [current], outdatedCompetitors: 0, outdatedCandidates: 0, recheckCostUsd: 0 } })
  })
  await page.goto('/manage/competitors/definition')
  await expect(page.getByRole('heading', { level: 1, name: 'Competitor definition' })).toBeVisible()
  const box = page.getByLabel('Definition', { exact: true })
  await expect(box).toHaveValue(/A competitor makes metal conveyor belts/)
  await expect(page.getByRole('list', { name: 'Versions of the definition' })).toContainText('Version 2')
  await expectAccessible(page)
  await box.fill('Makers of metal process belts compete.')
  await page.getByRole('button', { name: 'Save as a new version' }).click()
  await expect(page.getByText('The definition is saved as version 3. Nothing was rechecked yet.')).toBeVisible()
  expect(saved).toEqual({ text: 'Makers of metal process belts compete.' })
})
