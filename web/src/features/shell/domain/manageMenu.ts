/** A subpage in the Manage menu. */
export type MenuChild = { label: string; to: string }

/**
 * A section of the Manage menu: its index address, one sentence on what it holds, its subpages,
 * and, when it has a count, what the count counts and the subpage the count belongs to.
 */
export type MenuGroup = {
  label: string
  to: string
  summary: string
  /** What the section's count counts, "waiting" or "running"; a section without one shows no count. */
  countLabel?: 'waiting' | 'running'
  /** The subpage the count moves to while the section is open. */
  countOn?: string
  children: readonly MenuChild[]
}

/**
 * The Manage menu, in the order of the spec's site map. A subpage is listed only once its page
 * exists; the others join the menu with the tickets that build them.
 */
export const MANAGE_MENU: readonly MenuGroup[] = [
  { label: 'Activity', to: '/manage', summary: 'Where to start, stop, and change the work behind the results.', children: [] },
  {
    label: 'Prompt test',
    to: '/manage/prompt-test',
    summary: 'The questions, the providers, the schedule, the budget, and the fact sheet of the Prompt test.',
    countLabel: 'running',
    countOn: '/manage/prompt-test/runs',
    children: [
      { label: 'Questions', to: '/manage/prompt-test/questions' },
      { label: 'Models', to: '/manage/prompt-test/models' },
      { label: 'Schedule', to: '/manage/prompt-test/schedule' },
      { label: 'Fact sheet', to: '/manage/prompt-test/fact-sheet' },
      { label: 'Runs', to: '/manage/prompt-test/runs' },
    ],
  },
  {
    label: 'Website scans',
    to: '/manage/scans',
    summary: 'Scan a website, and open the recent scans.',
    countLabel: 'running',
    countOn: '/manage/scans/queue',
    children: [
      { label: 'Scan queue', to: '/manage/scans/queue' },
      { label: 'Recent scans', to: '/manage/scans/recent' },
      { label: 'Scan settings', to: '/manage/scans/settings' },
    ],
  },
  {
    label: 'Competitor list',
    to: '/manage/competitors',
    summary: 'The companies the AI responses name that are not on the competitor list yet.',
    countLabel: 'waiting',
    countOn: '/manage/competitors/candidates',
    children: [
      { label: 'Companies', to: '/manage/competitors/companies' },
      { label: 'Candidates', to: '/manage/competitors/candidates' },
      { label: 'Check the list', to: '/manage/competitors/check' },
      { label: 'Competitor definition', to: '/manage/competitors/definition' },
    ],
  },
  {
    label: 'Search term list',
    to: '/manage/terms',
    summary: 'The search terms the scans found that wait for a decision.',
    countLabel: 'waiting',
    countOn: '/manage/terms/waiting',
    children: [
      { label: 'Term review', to: '/manage/terms/review' },
      { label: 'Waiting terms', to: '/manage/terms/waiting' },
    ],
  },
  {
    label: 'Recommendations',
    to: '/manage/recommendations',
    summary: 'Marking recommendations as done, and the goals.',
    children: [
      { label: 'Tick off yourself', to: '/manage/recommendations/tick-off' },
      { label: 'Goals', to: '/manage/recommendations/goals' },
    ],
  },
  {
    label: 'Keys and spending',
    to: '/manage/keys',
    summary: 'The API keys, the budget, and the events.',
    children: [
      { label: 'OpenRouter key', to: '/manage/keys/openrouter' },
      { label: 'Anthropic key', to: '/manage/keys/anthropic' },
      { label: 'Budgets', to: '/manage/keys/budgets' },
      { label: 'Events', to: '/manage/keys/events' },
    ],
  },
]

/** The section a Manage address belongs to: the one whose index address is the longest prefix of it. */
export function groupOf(path: string): MenuGroup | undefined {
  return MANAGE_MENU.filter((g) => path === g.to || path.startsWith(`${g.to}/`)).sort((a, b) => b.to.length - a.to.length)[0]
}

/** Whether a section is folded out at this address. */
export function isOpen(group: MenuGroup, path: string): boolean {
  return groupOf(path) === group
}

/** Where a section's count shows: on the section while folded, and on its subpage while open. */
export function badgePlacement(group: MenuGroup, open: boolean): 'group' | 'child' | 'none' {
  if (!group.countLabel) return 'none'
  if (open && group.children.some((c) => c.to === group.countOn)) return 'child'
  return 'group'
}
