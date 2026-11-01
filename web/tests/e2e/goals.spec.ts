import { expect, expectAccessible, test } from './fixtures'

// Made-up figures: a partial run of version 1, its key refused, then a run of version 2.
const history = {
  versions: [
    { version: 1, publishedAt: '2026-10-26T15:04:51Z', questions: 90, note: 'A supplier, an application, and a problem question for each belt family, in six scopes.' },
    { version: 2, publishedAt: '2026-10-27T06:35:33Z', questions: 20, note: 'Kept the application question a buyer most often asks, in four scopes.' },
  ],
  runs: [
    { runId: '0b6a1c52-3e0f-4c1e-9a8d-6f1e2d3c4b5a', startedAt: '2026-10-27T06:03:12Z', promptSetVersion: 1, tasks: 1260, answered: 62, read: 47, partial: true, failure: 'OpenRouter answered 401: User not found.', all: 0.4, twente: 0.5 },
    { runId: '7c2d9e41-5a6b-4f70-8e1d-2b3c4d5e6f70', startedAt: '2026-10-27T08:36:32Z', promptSetVersion: 2, tasks: 200, answered: 200, read: 183, partial: false, all: 0.25, twente: 0.3 },
  ],
}

test('the Overview opens a goal, which says who is ahead and draws its history by Prompt set version', async ({ page }) => {
  await page.route('**/api/v1/prompt-test/history', (route) => route.fulfill({ json: history }))
  await page.goto('/')
  await page.getByRole('region', { name: 'Goals' }).getByRole('link', { name: /^AI responses that name Tribelt/ }).click()
  await expect(page).toHaveURL('/goals/aiShareOfVoice')
  await expect(page.getByRole('heading', { level: 1, name: 'AI responses that name Tribelt' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Tribelt against the competitors' })).toBeVisible()
  const overTime = page.getByRole('region', { name: 'Over time' })
  await expect(overTime.getByRole('img', { name: 'Over time' })).toBeVisible()
  await expectAccessible(page)
  await overTime.getByRole('group', { name: 'Prompt set' }).getByRole('button', { name: 'Version 1' }).click()
  await expect(page).toHaveURL(/version=1/)
  await expect(overTime.getByText(/a partial run of version 1\. Only 62 of its 1,260 requests got a response/)).toBeVisible()
  await page.getByRole('link', { name: 'Back to Overview' }).click()
  await expect(page).toHaveURL('/')
})

test('a goal the scans measure lists what it is broken down by', async ({ page }) => {
  await page.goto('/goals/readinessChecks')
  await expect(page.getByRole('heading', { level: 1, name: 'AI readiness checks passed' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Broken down' })).toBeVisible()
  await expectAccessible(page)
})

test.describe('at phone width', () => {
  test.use({ viewport: { width: 390, height: 844 } })

  test('the Overview and a goal never scroll sideways', async ({ page }) => {
    for (const path of ['/', '/goals/aiShareOfVoice', '/goals/termsUsed']) {
      await page.goto(path)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
      expect(overflow, path).toBe(0)
    }
  })
})
