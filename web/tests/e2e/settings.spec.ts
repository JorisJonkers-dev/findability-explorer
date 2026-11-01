import { expect, expectAccessible, test } from './fixtures'

test('the Prompt test models show the prices and the providers, and take a new provider; Questions shows the published set', async ({ page }, info) => {
  await page.goto('/manage/prompt-test/models')
  await expect(page.getByRole('heading', { level: 1, name: 'Models' })).toBeVisible()
  // The e2e run has no OpenRouter key, so the page says so and has no prices to go by.
  await expect(page.getByRole('alert').filter({ hasText: 'The OpenRouter key does not work' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'What runs cost' })).toContainText('No prices yet.')
  await expectAccessible(page)
  const providers = page.getByRole('list', { name: 'Providers' })
  await expect(providers.getByRole('listitem')).not.toHaveCount(0)
  // Adding is not saved here, so the projects sharing one database do not meet.
  const id = info.project.name === 'phone' ? 'cohere' : 'ai21'
  await page.getByLabel('Add a provider by its OpenRouter id').fill(id)
  await page.getByRole('button', { name: 'Add', exact: true }).click()
  await expect(providers).toContainText(id)
  const menu = page.getByRole('navigation', { name: 'Manage' })
  await expect(menu.getByRole('link', { name: /^Models/ })).toHaveAttribute('aria-current', 'page')

  await menu.getByRole('link', { name: /^Questions/ }).click()
  await expect(page).toHaveURL('/manage/prompt-test/questions')
  await expect(page.getByRole('heading', { level: 2, name: /^Version \d+ is published$/ })).toBeVisible()
  await expect(page.getByRole('region', { name: 'The questions' }).getByRole('heading', { level: 3 })).not.toHaveCount(0)
  await expectAccessible(page)
})
