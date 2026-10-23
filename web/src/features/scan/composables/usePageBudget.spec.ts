import { afterEach, describe, expect, it } from 'vitest'
import { usePageBudget } from './usePageBudget'

afterEach(() => {
  usePageBudget().reset()
})

describe('usePageBudget', () => {
  it('shares one budget, kept within the contract', () => {
    const first = usePageBudget()
    expect(first.pages.value).toBe(60)
    first.pages.value = 250
    expect(usePageBudget().pages.value).toBe(100)
    first.pages.value = 12
    expect(usePageBudget().pages.value).toBe(12)
  })
})
