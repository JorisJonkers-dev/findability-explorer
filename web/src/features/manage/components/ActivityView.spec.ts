import { describe, expect, it } from 'vitest'
import type { Activity, Work } from '../../../infrastructure/api/types.gen'
import { me, RUN_ID, summary } from '../../../testing/fixtures'
import { manageCounts } from '../../../testing/manageFixtures'
import { type Handler, json, mountApp, routes, settle } from '../../../testing/mountApp'
import { llmSettings, storedCredential } from '../../settings/fixtures'
import { promptSettings } from '../fixtures'

const SCAN = '00000000-0000-4000-9000-000000000001'
const QUEUED = '00000000-0000-4000-9000-000000000002'
const DONE = '00000000-0000-4000-9000-000000000003'
const STOPPED = '00000000-0000-4000-8000-000000000009'
const CHECK = '00000000-0000-4000-9000-000000000004'

const work = (overrides: Partial<Work>): Work => ({ id: RUN_ID, kind: 'promptRun', startedAt: '2026-11-01T08:29:00Z', done: 0, total: 0, failed: 0, openRouterCostUsd: 0, anthropicCostUsd: 0, ...overrides })

function activity(overrides: Partial<Activity> = {}): Activity {
  return {
    running: [
      work({ done: 100, total: 400, failed: 10, openRouterCostUsd: 0.1 }),
      work({ id: SCAN, kind: 'scan', done: 15, total: 60, startedAt: '2026-11-01T08:20:00Z' }),
      work({ id: CHECK, kind: 'readinessCheck', done: 12, total: 48 }),
    ],
    recent: [
      work({ id: STOPPED, ending: 'stoppedByHand', done: 40, total: 200, finishedAt: '2026-10-28T09:37:00Z', openRouterCostUsd: 0.05 }),
      work({ id: DONE, kind: 'scan', ending: 'done', done: 60, total: 60, finishedAt: '2026-10-27T09:47:00Z', anthropicCostUsd: 0.02 }),
    ],
    spending: {
      since: '2026-11-01T00:00:00Z',
      lines: [
        { kind: 'promptRuns', key: 'openRouter', costUsd: 2.5 },
        { kind: 'scans', key: 'anthropic', costUsd: 0.4 },
        { kind: 'termReview', key: 'anthropic', costUsd: 0.08 },
      ],
    },
    ...overrides,
  }
}

function server(extra: Record<string, Handler> = {}) {
  const sent: string[] = []
  const handle = routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    ...manageCounts({ terms: 3, candidates: 1 }),
    'GET /api/v1/admin/prompt-test': () => json(promptSettings()),
    'GET /api/v1/admin/llm': () => json(llmSettings()),
    'GET /api/v1/admin/activity': () => json(activity()),
    'GET /api/v1/analyses': () => json({ items: [summary({ id: QUEUED, status: 'queued' })] }),
    [`POST /api/v1/admin/prompt-runs/${RUN_ID}/stop`]: () => (sent.push('stop run'), new Response(null, { status: 204 })),
    'DELETE /api/v1/admin/analyses/queue': () => (sent.push('empty queue'), new Response(null, { status: 204 })),
    ...extra,
  })
  return { handle, sent }
}

const widget = (wrapper: Awaited<ReturnType<typeof mountApp>>['wrapper'], title: string) => {
  const heading = wrapper.findAll('section > div > h2').find((h) => h.text() === title)
  if (!heading) throw new Error(`no widget ${title}`)
  return wrapper.get(`section[aria-labelledby="${heading.attributes('id') ?? ''}"]`)
}

describe('Activity', () => {
  it('opens Manage with its title, purpose, and the way to the Overview', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage')
    expect(wrapper.get('h1').text()).toBe('Activity')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe('See what is running, what comes next, and what is waiting for you.')
    expect(wrapper.get('header a[href="/"]').text()).toBe('See the Overview')
    expect(wrapper.get('nav[aria-label="Manage"] a[aria-current="page"]').text()).toContain('Activity')
  })

  it('shows each piece of running work with its meter, cost, and Stop, which says what it keeps first', async () => {
    const { handle, sent } = server()
    const { wrapper } = await mountApp(handle, '/manage')
    const run = widget(wrapper, 'Prompt run')
    expect(run.text()).toContain('100 of 400 responses')
    expect(run.text()).toContain('25%')
    expect(run.text()).toContain('$0.10 so far')
    expect(run.text()).toContain('$0.15 to go')
    expect(widget(wrapper, 'Scan').text()).toContain('15 of 60 pages')
    expect(widget(wrapper, 'AI readiness check').text()).toContain('12 of 48 sites')
    await run.findAll('button').find((b) => b.text() === 'Stop the Prompt run')?.trigger('click')
    expect(run.get('[data-part="confirm-stop"]').text()).toContain('Stopping keeps the responses so far. You can finish the run later with a retry.')
    await run.get('[data-part="confirm-stop"]').findAll('button').find((b) => b.text() === 'Stop the Prompt run')?.trigger('click')
    await settle()
    expect(sent).toEqual(['stop run'])
    expect(wrapper.text()).toContain('The Prompt run stops within a minute. Its responses so far are kept.')
  })

  it('keeps running when Keep running is chosen', async () => {
    const { handle, sent } = server()
    const { wrapper } = await mountApp(handle, '/manage')
    const scan = widget(wrapper, 'Scan')
    await scan.findAll('button').find((b) => b.text() === 'Stop the scan')?.trigger('click')
    expect(scan.text()).toContain('Stopping keeps the pages read so far.')
    await scan.findAll('button').find((b) => b.text() === 'Keep running')?.trigger('click')
    expect(scan.find('[data-part="confirm-stop"]').exists()).toBe(false)
    expect(sent).toEqual([])
  })

  it('says what comes next, and takes the waiting scans off the queue', async () => {
    const { handle, sent } = server()
    const { wrapper } = await mountApp(handle, '/manage')
    const coming = widget(wrapper, 'Coming up')
    expect(coming.get('[data-part="next-run"]').text()).toContain('1 December 2026, 06:00')
    expect(coming.get('[data-part="next-run"]').text()).toContain('About $0.25 a run, monthly')
    expect(coming.get('[data-part="term-review"]').text()).toContain('On, so waiting search terms are reviewed automatically.')
    expect(coming.get('[data-part="scan-queue"]').text()).toContain('1 scan waits in the queue.')
    await coming.findAll('button').find((b) => b.text() === 'Remove the waiting scans')?.trigger('click')
    await settle()
    expect(sent).toEqual(['empty queue'])
  })

  it('says the term review is off, and how many search terms wait for an administrator', async () => {
    const off = { items: [], counts: { pending: 3, accepted: 0, rejected: 0 }, review: { on: false, running: false, waiting: 3, monthTokens: 0, monthCostUsd: 0 } }
    const { wrapper } = await mountApp(server({ 'GET /api/v1/admin/term-candidates': () => json(off) }).handle, '/manage')
    expect(widget(wrapper, 'Coming up').get('[data-part="term-review"]').text()).toContain('Off, so 3 search terms wait for you.')
  })

  it('lists what waits for an administrator, and says so when nothing does', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage')
    const waiting = widget(wrapper, 'Waiting for you')
    expect(waiting.classes()).toContain('bg-orange-pale')
    expect(waiting.findAll('a').map((a) => a.text())).toEqual(['3 search terms wait for the term review', '1 competitor candidate waits'])
    wrapper.unmount()
    const quiet = await mountApp(
      server({ ...manageCounts(), 'GET /api/v1/admin/llm': () => json(llmSettings({ credentials: [storedCredential()] })) }).handle,
      '/manage',
    )
    const none = widget(quiet.wrapper, 'Waiting for you')
    expect(none.findAll('a').map((a) => a.text())).toEqual(['The Anthropic key is missing'])
    quiet.wrapper.unmount()
    const empty = await mountApp(server({ ...manageCounts() }).handle, '/manage')
    expect(widget(empty.wrapper, 'Waiting for you').text()).toContain('Nothing is waiting for you.')
  })

  it('shows both budgets and the spending by kind of work', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage')
    const open = widget(wrapper, 'OpenRouter budget')
    expect(open.text()).toContain('$2.50 of $10.00')
    expect(open.get('[data-part="openrouter-left"]').text()).toBe('$7.50 left25% used')
    expect(widget(wrapper, 'Anthropic budget').text()).toContain('$0.48 of $15.00')
    expect(widget(wrapper, 'Anthropic budget').get('[data-part="anthropic-left"]').text()).toBe('$14.52 left3% used')
    const spending = widget(wrapper, 'Spending by kind of work')
    expect(spending.findAll('tbody tr').map((r) => r.text())).toEqual(['Prompt runsOpenRouter$2.50', 'ScansAnthropic$0.40', 'Term reviewAnthropic$0.08'])
    expect(spending.get('tfoot').text()).toBe('All work$2.98')
  })

  it('lists the recent work, newest first, with how each ended', async () => {
    const { wrapper } = await mountApp(server().handle, '/manage')
    const rows = widget(wrapper, 'Recent work').findAll('tbody tr')
    expect(rows.map((r) => r.findAll('th, td').map((c) => c.text()))).toEqual([
      ['28 October 2026, 09:37', 'Prompt run', 'Stopped by hand', '40 of 200 responses', '$0.05', 'Open Prompt run, 28 October 2026, 09:37'],
      ['27 October 2026, 09:47', 'Scan', 'Done', '60 of 60 pages', '$0.02', 'Open Scan, 27 October 2026, 09:47'],
    ])
    expect(rows[1]?.get('a').attributes('href')).toBe(`/manage/scans/${DONE}`)
  })

  it('says when nothing runs and no work has ended', async () => {
    const { wrapper } = await mountApp(
      server({
        'GET /api/v1/admin/activity': () => json(activity({ running: [], recent: [] })),
        'GET /api/v1/analyses': () => json({ items: [] }),
      }).handle,
      '/manage',
    )
    expect(widget(wrapper, 'Running now').text()).toContain('Nothing is running.')
    expect(widget(wrapper, 'Recent work').text()).toContain('No work has ended yet.')
    expect(widget(wrapper, 'Coming up').get('[data-part="scan-queue"]').text()).toContain('No scans are waiting.')
  })

  it('tells someone who is no administrator that Manage is for administrators, with the way to the Overview', async () => {
    const { wrapper } = await mountApp(routes({}), '/manage')
    expect(wrapper.text()).toContain('Only administrators can start, stop, and change the work behind the results.')
    expect(wrapper.get('a[data-part="to-overview"]').attributes('href')).toBe('/')
  })
})
