import { expect, expectAccessible, test } from './fixtures'

test('the Prompt test settings show the prices, the providers and the questions, and take a new provider', async ({ page }, info) => {
  await page.goto('/settings/prompt-test')
  await expect(page.getByRole('heading', { level: 1, name: 'Prompt test' })).toBeVisible()
  // The e2e run has no OpenRouter key, so the page says so and has no prices to go by.
  await expect(page.getByRole('alert').filter({ hasText: 'The OpenRouter key does not work' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'What runs cost' })).toContainText('No prices yet.')
  await expect(page.getByRole('heading', { level: 2, name: 'The 20 buyer questions' })).toBeVisible()
  await expectAccessible(page)
  const providers = page.getByRole('list', { name: 'Providers' })
  await expect(providers.getByRole('listitem')).not.toHaveCount(0)
  // Adding is not saved here, so the projects sharing one database do not meet.
  const id = info.project.name === 'phone' ? 'cohere' : 'ai21'
  await page.getByLabel('Add a provider by its OpenRouter id').fill(id)
  await page.getByRole('button', { name: 'Add', exact: true }).click()
  await expect(providers).toContainText(id)
  await expect(page.getByRole('navigation', { name: 'Settings' }).getByRole('link', { name: 'Prompt test' })).toHaveAttribute('aria-current', 'page')
})
