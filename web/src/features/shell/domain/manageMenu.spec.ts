import { describe, expect, it } from 'vitest'
import { badgePlacement, groupOf, isOpen, MANAGE_MENU, type MenuGroup } from './manageMenu'

const named = (label: string): MenuGroup => {
  const group = MANAGE_MENU.find((g) => g.label === label)
  if (!group) throw new Error(`no section ${label}`)
  return group
}

describe('the Manage menu', () => {
  it('lists the seven sections in order, each at its address', () => {
    expect(MANAGE_MENU.map((g) => [g.label, g.to])).toEqual([
      ['Activity', '/manage'],
      ['Prompt test', '/manage/prompt-test'],
      ['Website scans', '/manage/scans'],
      ['Competitor list', '/manage/competitors'],
      ['Search term list', '/manage/terms'],
      ['Recommendations', '/manage/recommendations'],
      ['Keys and spending', '/manage/keys'],
    ])
  })

  it('lists only the subpages that exist, each under its section', () => {
    expect(MANAGE_MENU.flatMap((g) => g.children.map((c) => [g.label, c.label, c.to]))).toEqual([
      ['Website scans', 'Scan queue', '/manage/scans/queue'],
      ['Website scans', 'Recent scans', '/manage/scans/recent'],
      ['Website scans', 'Scan settings', '/manage/scans/settings'],
      ['Website scans', 'Matching products', '/manage/claude-steps/matching'],
      ['Competitor list', 'Companies', '/manage/competitors/companies'],
      ['Competitor list', 'Candidates', '/manage/competitors/candidates'],
      ['Competitor list', 'Check the list', '/manage/competitors/check'],
      ['Search term list', 'Term review', '/manage/terms/review'],
      ['Search term list', 'Waiting terms', '/manage/terms/waiting'],
      ['Recommendations', 'Tick off yourself', '/manage/recommendations/tick-off'],
      ['Recommendations', 'Goals', '/manage/recommendations/goals'],
      ['Keys and spending', 'OpenRouter key', '/manage/keys/openrouter'],
      ['Keys and spending', 'Anthropic key', '/manage/keys/anthropic'],
      ['Keys and spending', 'Budgets', '/manage/keys/budgets'],
      ['Keys and spending', 'Events', '/manage/keys/events'],
      ['Keys and spending', 'Models and instructions', '/manage/claude-steps'],
    ])
    // A Claude step is listed under the section it works for, and all of them under Keys and spending.
    for (const group of MANAGE_MENU) {
      for (const child of group.children) expect(child.to.startsWith(`${group.to}/`) || child.to.startsWith('/manage/claude-steps')).toBe(true)
    }
  })

  it('says in a sentence what each section holds', () => {
    for (const group of MANAGE_MENU) expect(group.summary).toMatch(/^[A-Z].+\.$/)
  })

  it('finds the section of an index, a subpage, and a detail page under it', () => {
    expect(groupOf('/manage')?.label).toBe('Activity')
    expect(groupOf('/manage/prompt-test')?.label).toBe('Prompt test')
    expect(groupOf('/manage/competitors/candidates')?.label).toBe('Competitor list')
    expect(groupOf('/manage/scans/7f3c')?.label).toBe('Website scans')
    expect(groupOf('/manage/claude-steps/matching')?.label).toBe('Website scans')
    expect(groupOf('/manage/claude-steps')?.label).toBe('Keys and spending')
    expect(groupOf('/manage/no-such-section')?.label).toBe('Activity')
    expect(groupOf('/manageable')).toBeUndefined()
    expect(groupOf('/terms')).toBeUndefined()
  })

  it('opens the section the address is in, and only that one', () => {
    expect(isOpen(named('Competitor list'), '/manage/competitors/candidates')).toBe(true)
    expect(isOpen(named('Activity'), '/manage/competitors/candidates')).toBe(false)
    expect(isOpen(named('Activity'), '/manage')).toBe(true)
  })

  it('shows a count on the section while folded, and on its subpage while open', () => {
    expect(badgePlacement(named('Competitor list'), false)).toBe('group')
    expect(badgePlacement(named('Competitor list'), true)).toBe('child')
    expect(badgePlacement(named('Search term list'), true)).toBe('child')
  })

  it('keeps a count on the section when no subpage holds it', () => {
    expect(badgePlacement(named('Prompt test'), true)).toBe('group')
    expect(badgePlacement(named('Website scans'), false)).toBe('group')
    const orphan: MenuGroup = { label: 'Lost', to: '/manage/lost', summary: 'Lost.', countLabel: 'waiting', countOn: '/manage/lost/gone', children: [] }
    expect(badgePlacement(orphan, true)).toBe('group')
  })

  it('shows no count on a section that has none', () => {
    expect(badgePlacement(named('Keys and spending'), true)).toBe('none')
    expect(badgePlacement(named('Activity'), false)).toBe('none')
  })
})
