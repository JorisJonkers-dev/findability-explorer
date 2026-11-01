import { expect, expectAccessible, test } from './fixtures'

// The dev user of the default server is an administrator (playwright.config.ts). The spec saves no
// version: the e2e database is shared, and a saved version cannot be taken back.
test('an administrator opens the Claude steps and edits matching products without saving', async ({ page }) => {
  await page.goto('/manage/keys')
  await page.getByRole('navigation', { name: 'Manage' }).getByRole('link', { name: 'Models and instructions' }).click()
  await expect(page).toHaveURL('/manage/claude-steps')
  await expect(page.getByRole('heading', { level: 1, name: 'Claude models and instructions' })).toBeVisible()
  await expectAccessible(page)

  await page.getByRole('table', { name: 'Claude steps' }).getByRole('link', { name: 'Matching products' }).click()
  await expect(page).toHaveURL('/manage/claude-steps/matching')
  await expect(page.getByRole('heading', { level: 1, name: 'Matching products' })).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Manage' }).getByRole('link', { name: 'Matching products' })).toHaveAttribute('aria-current', 'page')
  const box = page.getByLabel('Instructions', { exact: true })
  await expect(box).toHaveValue(/Tribelt/)
  await expectAccessible(page)

  await box.fill('Match only metal belts.')
  await expect(page.getByRole('button', { name: 'Save as a new version' })).toBeEnabled()
  await page.getByLabel('Model', { exact: true }).selectOption('claude-sonnet-5-5')
  await expect(page.locator('[data-part="cost"]')).toContainText('more than now')
  await page.getByRole('button', { name: 'Undo the changes' }).click()
  await expect(page.getByRole('button', { name: 'Save as a new version' })).toBeDisabled()
})
