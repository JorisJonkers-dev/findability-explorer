import { expect, test } from './fixtures'

test.skip(({ isMobile }) => isMobile, 'the stage is for large windows; the phone project checks the stacked page')

const VIEWS = ['/', '/ai-search', '/ai-search/answers', '/ai-search/readiness', '/competitors', '/terms', '/terms/all', '/terms/families/eyelink', '/recommendations', '/buyers']

// The stage grows to the window's aspect, so a window that is not 16:9 has no white bands.
for (const viewport of [
  { width: 2560, height: 1300 },
  { width: 1920, height: 1080 },
]) {
  test.describe(`at ${String(viewport.width)}×${String(viewport.height)}`, () => {
    test.use({ viewport })

    test('the poster’s footer spans the window, and no view scrolls sideways', async ({ page }) => {
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
        const stage = await page.locator('.stage-focus').boundingBox()
        expect(stage?.width ?? 0, path).toBeCloseTo(viewport.width, 0)
        expect(stage?.height ?? 0, path).toBeCloseTo(viewport.height, 0)
      }
    })
  })
}
