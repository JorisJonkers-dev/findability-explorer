import { expect, expectAccessible, test } from './fixtures'

test('the Prompt test models show the Model list and where to add one; Questions shows the published set', async ({ page }) => {
  await page.goto('/manage/prompt-test/models')
  await expect(page.getByRole('heading', { level: 1, name: 'Models' })).toBeVisible()
  // The e2e run has no OpenRouter key, so the page says so and reads no catalogue.
  await expect(page.getByRole('alert').filter({ hasText: 'The OpenRouter key does not work' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'The model list' })).toContainText('These models are asked in every run')
  await expect(page.getByRole('region', { name: 'Add a model' })).toContainText('OpenRouter’s models can be listed once the key works.')
  // Nothing has changed, so there is nothing to save.
  await expect(page.getByRole('button', { name: 'Save the model list' })).toBeDisabled()
  await expectAccessible(page)
  const menu = page.getByRole('navigation', { name: 'Manage' })
  await expect(menu.getByRole('link', { name: /^Models/ })).toHaveAttribute('aria-current', 'page')

  await menu.getByRole('link', { name: /^Questions/ }).click()
  await expect(page).toHaveURL('/manage/prompt-test/questions')
  await expect(page.getByRole('heading', { level: 2, name: /^Version \d+ is published$/ })).toBeVisible()
  await expect(page.getByRole('region', { name: 'The questions' }).getByRole('heading', { level: 3 })).not.toHaveCount(0)
  await expectAccessible(page)
})
