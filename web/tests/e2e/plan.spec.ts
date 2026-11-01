import { expect, expectAccessible, test } from './fixtures'

test('the recommendations are ranked, a bubble narrows them, and an administrator marks one as done and undoes it', async ({ page }, info) => {
  // The projects share one database, so each marks a recommendation of its own.
  const mine = info.project.name === 'phone' ? { id: 'family-pages', title: 'One page per belt family in each language' } : { id: 'listings', title: 'Get listed where AI responses find sources' }
  await page.goto('/recommendations')
  await expect(page.getByRole('heading', { level: 1, name: 'Recommendations' })).toBeVisible()
  await expect(page.getByRole('heading', { level: 2, name: 'Benefit against effort' })).toBeVisible()
  await expect(page.getByRole('heading', { level: 2, name: 'Ranked list' })).toBeVisible()
  await expect(page.getByRole('heading', { level: 2, name: 'Done', exact: true })).toBeVisible()
  const list = page.getByRole('list', { name: 'Recommendations' }).locator(':scope > li')
  await expect(list.first()).toBeVisible()
  const all = await list.count()

  // A bubble lists only its own, and Show every bubble lists them all again.
  const bubble = page.getByRole('group', { name: 'Benefit against effort' }).getByRole('button').first()
  await bubble.click()
  await expect(bubble).toHaveAttribute('aria-pressed', 'true')
  await expect(list).toHaveCount(Number(await bubble.textContent()))
  await page.getByRole('button', { name: 'Show every bubble' }).click()
  await expect(list).toHaveCount(all)

  const effort = page.getByRole('group', { name: 'Sort by' }).getByRole('button', { name: 'Effort' })
  await effort.click()
  await expect(effort).toHaveAttribute('aria-pressed', 'true')
  await expectAccessible(page)

  await page.goto(`/recommendations/${mine.id}`)
  await expect(page.getByRole('heading', { level: 1, name: mine.title })).toBeVisible()
  for (const name of ['Where, and how', 'Why it ranks here', 'The goal it improves', 'Steps', 'When it counts as done']) {
    await expect(page.getByRole('heading', { level: 2, name })).toBeVisible()
  }
  await expectAccessible(page)

  await page.getByRole('button', { name: 'Mark as done' }).click()
  await expect(page.getByRole('status')).toContainText('Marked done')
  await page.reload()
  await expect(page.getByRole('button', { name: 'Undo, it is not done' })).toBeVisible()
  await page.getByRole('button', { name: 'Undo, it is not done' }).click()
  await expect(page.getByRole('button', { name: 'Mark as done' })).toBeVisible()

  // A scan decides an automatic one; it cannot be marked by hand.
  await page.goto('/recommendations/llms-files')
  await expect(page.getByText('The latest scan of tribelt.nl checks this one.')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Mark as done' })).toHaveCount(0)
})
