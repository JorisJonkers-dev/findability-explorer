import { expect, expectAccessible, test } from './fixtures'

test('the plan ticks an improvement off and back, and keeps the tick', async ({ page }, info) => {
  // The projects share one database, so each ticks an improvement of its own.
  const title = info.project.name === 'phone' ? /One page per belt family/ : /Get listed where answers find sources/
  await page.goto('/plan')
  const improvements = page.getByRole('list', { name: 'Improvements' })
  await expect(improvements.getByRole('listitem')).toHaveCount(8)
  await expectAccessible(page)
  const box = page.getByRole('checkbox', { name: title })
  await box.check()
  await expect(page.getByText('Ticked off by e2e.')).toBeVisible()
  await page.reload()
  await expect(box).toBeChecked()
  await box.uncheck()
  await expect(box).not.toBeChecked()
  // A scan decides an automatic one; it cannot be ticked by hand.
  await expect(page.getByRole('checkbox', { name: /Add llms.txt and llms-full.txt/ })).toBeDisabled()
})

test('Catching up lists the open improvements', async ({ page }) => {
  await page.goto('/catching-up')
  await expect(page.getByRole('region', { name: 'What to do, in order' }).getByRole('listitem')).toHaveCount(5)
})
