import { expect, expectAccessible, test } from './fixtures'

test('the recommendations are ranked, and an administrator marks one as done and undoes it', async ({ page }, info) => {
  // The projects share one database, so each marks a recommendation of its own.
  const mine = info.project.name === 'phone' ? { id: 'family-pages', title: 'One page per belt family in each language' } : { id: 'listings', title: 'Get listed where answers find sources' }
  await page.goto('/recommendations')
  await expect(page.getByRole('heading', { level: 1, name: 'Start with the changes that help Tribelt most.' })).toBeVisible()
  // The best value comes first; the others follow, and each dot of the chart is a button.
  await expect(page.getByText('Do this first')).toBeVisible()
  await expect(page.getByRole('list', { name: 'Recommendations' }).locator(':scope > li').first()).toBeVisible()
  await expect(page.getByRole('group', { name: 'Benefit against effort' }).getByRole('button').first()).toBeVisible()
  await page.getByRole('button', { name: 'Biggest benefit' }).click()
  await expect(page.getByRole('button', { name: 'Biggest benefit' })).toHaveAttribute('aria-pressed', 'true')
  await expectAccessible(page)

  await page.goto(`/recommendations/${mine.id}`)
  await expect(page).toHaveURL(`/recommendations/${mine.id}`)
  await expect(page.getByRole('heading', { level: 1, name: mine.title })).toBeVisible()
  await expect(page.locator('ol button[aria-pressed]').first()).toBeVisible()
  await expectAccessible(page)

  await page.getByRole('button', { name: 'Mark as done' }).click()
  await expect(page.getByRole('status')).toContainText('Marked as done')
  await page.reload()
  await expect(page.getByRole('button', { name: 'Undo, it is not done' })).toBeVisible()
  await page.getByRole('button', { name: 'Undo, it is not done' }).click()
  await expect(page.getByRole('button', { name: 'Mark as done' })).toBeVisible()

  // A scan decides an automatic one; it cannot be marked by hand.
  await page.goto('/recommendations/llms-files')
  await expect(page.getByText('The latest scan of tribelt.nl checks this one.')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Mark as done' })).toHaveCount(0)
})
