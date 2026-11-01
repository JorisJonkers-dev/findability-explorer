import { expect, expectAccessible, test } from './fixtures'

test('Help finds a topic across every group, and its link leads to where the work is done', async ({ page }) => {
  await page.goto('/help')
  await expect(page.getByRole('heading', { level: 1, name: 'Help' })).toBeVisible()
  const groups = page.getByRole('group', { name: 'Topics' })
  await expect(groups.getByRole('button', { name: /^Getting started/ })).toHaveAttribute('aria-pressed', 'true')
  await expectAccessible(page)

  await page.getByRole('searchbox', { name: 'Search Help' }).fill('buyer intent')
  await expect(groups.getByRole('button', { name: /^Everything/ })).toHaveAttribute('aria-pressed', 'true')
  const topic = page.getByRole('article').filter({ has: page.getByRole('heading', { level: 3, name: 'Buyer intent' }) })
  await expect(topic).toContainText('The buyer names the type of belt.')
  await expectAccessible(page)

  await topic.getByRole('link', { name: /See the search terms/ }).click()
  await expect(page).toHaveURL('/terms')
})

test("a topic's address opens its group with the topic in view, and an administrator sees its Manage links", async ({ page }) => {
  await page.goto('/help/stop')
  await expect(page.getByRole('group', { name: 'Topics' }).getByRole('button', { name: /^The Prompt test/ })).toHaveAttribute('aria-pressed', 'true')
  const topic = page.locator('#help-stop')
  await expect(topic).toBeInViewport()
  await expect(topic.getByRole('link', { name: /Activity/ })).toHaveAttribute('href', '/manage')
  await page.getByRole('group', { name: 'Topics' }).getByRole('button', { name: /^Words used in this tool/ }).click()
  await expect(page.getByRole('term').filter({ hasText: 'Typical competitor' })).toBeVisible()
  await expectAccessible(page)
})
