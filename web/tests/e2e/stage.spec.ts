import { expect, test } from './fixtures'

test.skip(({ isMobile }) => isMobile, 'these are large windows; the phone project checks the stacked page')

// Every page, in Results and in Manage, scrolls like a page.
const VIEWS = [
  '/',
  '/goals/aiShareOfVoice',
  '/ai-responses',
  '/ai-responses/all',
  '/ai-readiness',
  '/terms',
  '/terms/all',
  '/terms/families/eyelink',
  '/competitors',
  '/recommendations',
  '/help',
  '/manage',
  '/manage/prompt-test',
  '/manage/scans',
  '/manage/competitors/candidates',
  '/manage/terms/waiting',
  '/manage/recommendations',
  '/manage/keys',
]

for (const viewport of [
  { width: 2560, height: 1300 },
  { width: 1920, height: 1080 },
]) {
  test.describe(`at ${String(viewport.width)}×${String(viewport.height)}`, () => {
    test.use({ viewport })

    test('the footer spans the window, and no view scrolls sideways', async ({ page }) => {
      await page.goto('/')
      const footer = page.locator('footer').first()
      await expect(footer).toBeVisible()
      const box = await footer.boundingBox()
      expect(box?.x ?? -1).toBeCloseTo(0, 0)
      expect((box?.x ?? 0) + (box?.width ?? 0)).toBeCloseTo(viewport.width, 0)
      for (const path of VIEWS) {
        await page.goto(path)
        await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
        expect(overflow, path).toBe(0)
      }
    })
  })
}
