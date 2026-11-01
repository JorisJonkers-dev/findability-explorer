import { expect, expectAccessible, test } from './fixtures'

const SECTIONS = [
  { label: 'AI responses', path: '/ai-responses' },
  { label: 'AI readiness', path: '/ai-readiness' },
  { label: 'Search terms', path: '/terms' },
  { label: 'Competitors', path: '/competitors' },
  { label: 'Recommendations', path: '/recommendations' },
  { label: 'Help', path: '/help' },
] as const

test('the top bar reaches every Results section and marks the one shown', async ({ page }) => {
  await page.goto('/terms')
  const bar = page.getByRole('navigation', { name: 'Main' })
  for (const section of SECTIONS) {
    await bar.getByRole('link', { name: section.label, exact: true }).click()
    await expect(page).toHaveURL(section.path)
    await expect(bar.getByRole('link', { name: section.label, exact: true })).toHaveAttribute('aria-current', 'page')
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await expectAccessible(page)
  }
  for (const gone of ['Catching up', 'Products', 'Buyers', 'The plan', 'AI search', 'Settings']) {
    await expect(bar.getByRole('link', { name: gone, exact: true })).toHaveCount(0)
  }
  await bar.getByRole('link', { name: 'Overview' }).click()
  await expect(page).toHaveURL('/')
})

test('an administrator opens Manage from the top bar and moves through its menu', async ({ page }) => {
  await page.goto('/')
  const bar = page.getByRole('navigation', { name: 'Main' })
  await bar.getByRole('link', { name: 'Manage', exact: true }).click()
  await expect(page).toHaveURL('/manage')
  await expect(bar.getByRole('link', { name: 'Manage', exact: true })).toHaveAttribute('aria-current', 'page')
  const menu = page.getByRole('navigation', { name: 'Manage' })
  await expect(menu.getByRole('link', { name: 'Activity' })).toHaveAttribute('aria-current', 'page')
  await expect(page.getByRole('heading', { level: 1, name: 'Activity' })).toBeVisible()
  await expectAccessible(page)

  await menu.getByRole('link', { name: /^Prompt test/ }).click()
  await expect(page).toHaveURL('/manage/prompt-test')
  await expect(page.getByRole('heading', { level: 1, name: 'Prompt test' })).toBeVisible()
  await expect(menu.getByRole('link', { name: /^Prompt test/ })).toHaveAttribute('aria-current', 'page')

  // A section whose only page so far is a subpage opens on it, folded out.
  await menu.getByRole('link', { name: /^Competitor list/ }).click()
  await expect(page).toHaveURL('/manage/competitors/candidates')
  await expect(menu.getByRole('link', { name: /^Candidates/ })).toHaveAttribute('aria-current', 'page')
  await expect(menu.getByRole('link', { name: /^Waiting terms/ })).toHaveCount(0)
  await expect(page.getByRole('heading', { level: 1, name: 'Candidates' })).toBeVisible()
  await expectAccessible(page)

  await menu.getByRole('link', { name: /^Website scans/ }).click()
  await expect(page).toHaveURL('/manage/scans')
  await expect(page.getByRole('heading', { level: 1, name: 'Website scans' })).toBeVisible()
  await expect(menu.getByRole('link', { name: /^Candidates/ })).toHaveCount(0)
  await expectAccessible(page)

  await page.getByRole('link', { name: 'See AI readiness' }).click()
  await expect(page).toHaveURL('/ai-readiness')
  await expect(page.getByRole('navigation', { name: 'Manage' })).toHaveCount(0)
})

test('detail pages link back, and the old addresses redirect', async ({ page }) => {
  await page.goto('/ai-responses/all')
  await expect(page.getByRole('navigation', { name: 'In this section' })).toHaveCount(0)
  await page.getByRole('link', { name: 'Back to AI responses' }).click()
  await expect(page).toHaveURL('/ai-responses')

  const moves = [
    ['/catching-up', '/'],
    ['/plan', '/recommendations'],
    ['/products', '/terms'],
    ['/products/open-spiral', '/terms/families/open-spiral'],
    ['/ai-search/prompt-test', '/ai-responses/all'],
    ['/competitors/all', '/competitors'],
    ['/numbers/aiShareOfVoice', '/goals/aiShareOfVoice'],
    ['/ai-search', '/ai-responses'],
    ['/ai-search/answers', '/ai-responses/all'],
    ['/ai-search/readiness', '/ai-readiness'],
    ['/buyers', '/'],
    ['/settings', '/manage/keys'],
    ['/settings/prompt-test', '/manage/prompt-test'],
    ['/settings/terms', '/manage/terms/waiting'],
    ['/settings/competitors', '/manage/competitors/candidates'],
    ['/how-it-works', '/help'],
  ] as const
  for (const [from, to] of moves) {
    await page.goto(from)
    await expect(page).toHaveURL(to)
  }
})

test('the old candidates review goes to Candidates in Manage', async ({ page }) => {
  await page.goto('/settings/candidates')
  await expect(page).toHaveURL('/manage/competitors/candidates')
  await expect(page.getByRole('heading', { level: 1, name: 'Candidates' })).toBeVisible()
  await expectAccessible(page)
})

test.describe('at phone width', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  test('the bar wraps, and no new page scrolls sideways', async ({ page }) => {
    for (const path of ['/', '/terms', '/terms/all', '/terms/families/eyelink', '/competitors', '/ai-responses', '/ai-readiness', '/recommendations', '/help', '/manage']) {
      await page.goto(path)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
      expect(overflow, path).toBe(0)
    }
  })
})
