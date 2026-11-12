import { expect, expectAccessible, test } from './fixtures'

// The terms added and removed by hand are answered here, in the page, so this spec writes nothing
// to the database other specs share.
const AT = '2026-11-01T09:00:00Z'
const TERM = 'e2e hand-added roestvrijstalen band'

type Term = { term: string; language: string; family: string; tier: string; usedBy: string[]; source: string; origin: string }

test('an administrator adds a search term by hand, removes it, and puts it back', async ({ page }) => {
  const added: { term: Term; by: string; at: string }[] = []
  const removed: { term: Term; by: string; at: string }[] = []
  await page.route('**/api/v1/admin/terms/hand', async (route) => {
    if (route.request().method() === 'POST') {
      const body = route.request().postDataJSON() as Omit<Term, 'usedBy' | 'source' | 'origin'>
      const term = { ...body, usedBy: [], source: '', origin: 'hand' }
      added.push({ term, by: 'e2e', at: AT })
      await route.fulfill({ status: 201, json: term })
      return
    }
    await route.fulfill({ json: { added, removed } })
  })
  await page.route('**/api/v1/admin/terms/removed**', async (route) => {
    if (route.request().method() === 'POST') {
      const entry = added.find((a) => a.term.term === TERM)
      if (entry) removed.push(entry)
    } else {
      removed.splice(0, removed.length)
    }
    await route.fulfill({ status: 204 })
  })

  await page.goto('/manage/terms')
  await page.getByRole('link', { name: 'Change the list' }).click()
  await expect(page).toHaveURL('/manage/terms/edit')
  await expect(page.getByRole('heading', { level: 1, name: 'Add or remove terms' })).toBeVisible()
  await expectAccessible(page)

  await page.getByLabel('Search term', { exact: true }).fill(TERM)
  await page.getByLabel('Language').selectOption('nl')
  await page.getByRole('button', { name: 'Add the search term' }).click()
  await expect(page.getByText(`${TERM} (Dutch) is on the list. Scans look for it from the next scan on.`)).toBeVisible()
  const table = page.getByRole('table', { name: 'Search terms added by hand' })
  await expect(table).toContainText(TERM)
  await expect(table).toContainText('Added by hand')

  await table.getByRole('button', { name: `Remove from the list: ${TERM}` }).click()
  await expect(page.getByText(`${TERM} (Dutch) is off the list. It is no longer counted or looked for.`)).toBeVisible()
  const gone = page.getByRole('list', { name: 'Removed search terms' })
  await expect(gone).toContainText(TERM)
  await gone.getByRole('button', { name: 'Put back' }).click()
  await expect(page.getByText(`${TERM} (Dutch) is back on the list.`)).toBeVisible()
  await expectAccessible(page)
})
