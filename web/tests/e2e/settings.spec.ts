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
  await expect(page.getByRole('heading', { level: 2, name: /^Publish as version \d+$/ })).toBeVisible()
  await expect(page.getByRole('region', { name: 'The questions' }).getByRole('heading', { level: 3 })).not.toHaveCount(0)
  await expectAccessible(page)
})

test('an administrator adds a question in a new scope to the draft and publishes it', async ({ page }) => {
  // Publishing is answered by the browser, so the projects sharing one database keep the imported versions.
  type Draft = { questions: { family: string; scope: string; text: string }[]; scopes?: { id: string; name: string }[] }
  const sent: Draft[] = []
  await page.route('**/api/v1/admin/prompt-sets', async (route) => {
    sent.push(route.request().postDataJSON() as Draft)
    await route.fulfill({ status: 201, json: { version: 99, publishedAt: '2026-11-01T12:00:00Z', note: 'Adds 1 question.', questions: [] } })
  })
  await page.goto('/manage/prompt-test/questions')
  const publish = page.getByRole('region', { name: /^Publish as version \d+$/ })
  await expect(publish).toContainText('No changes to publish yet.')
  const eyelink = page.getByRole('region', { name: /^Eyelink/ })
  await eyelink.getByRole('button', { name: '+ Add a question to Eyelink' }).click()
  const form = eyelink.getByRole('form', { name: 'A new question for Eyelink' })
  await form.getByLabel('Scope').selectOption({ label: 'A new scope…' })
  await form.getByLabel('Language').selectOption('fr')
  await form.getByLabel('Region, if any').fill('Lyon')
  await form.getByLabel('Question').fill('Quel fabricant de bandes à œillets recommandez-vous pour une boulangerie à Lyon ?')
  await expectAccessible(page)
  await form.getByRole('button', { name: 'Add the question' }).click()
  await expect(page.getByText('French, Lyon is a new scope, and the question is added to the draft.')).toBeVisible()
  await expect(publish).toContainText('Adds 1 question (1 in French)')
  await expect(publish).toContainText('Adds 1 scope (French, Lyon).')
  await expectAccessible(page)
  await publish.getByRole('button', { name: /^Publish as version \d+$/ }).click()
  await expect(page.getByText('Version 99 is published. The charts mark the date the questions changed.')).toBeVisible()
  expect(sent[0]?.scopes).toEqual([{ id: 'fr-lyon', name: 'French, Lyon', language: 'fr', region: 'Lyon' }])
  expect(sent[0]?.questions.filter((q) => q.scope === 'fr-lyon')).toEqual([
    expect.objectContaining({ family: 'eyelink', scope: 'fr-lyon', text: 'Quel fabricant de bandes à œillets recommandez-vous pour une boulangerie à Lyon ?' }),
  ])
})
