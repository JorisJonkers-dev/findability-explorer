import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { competitor, summary } from '../../../testing/fixtures'
import { buttonNamed, manageApi, noContent } from '../../../testing/manageApi'
import { type Handler, json, mountApp, problem, settle } from '../../../testing/mountApp'
import { MODEL_DECIDER } from '../domain/candidates'

const fenner = competitor('Fenner', { country: 'UK', host: 'fenner.com' })
const added = competitor('Added Belts', { source: 'added', scans: 0, host: 'added.example.com', websiteUrl: 'https://added.example.com/', lastScan: summary() })
const rubber = competitor('Rubber Co', { archived: true, archivedBy: MODEL_DECIDER, archivedAt: '2026-10-27T09:00:00Z' })
const tribelt = competitor('Tribelt', { role: 'tribelt' })
const unsure = { competitorId: 'fenner', competitor: false, sure: false, reason: 'Makes rubber belting.', model: 'haiku', at: '2026-10-27T09:00:00Z' }
const archivedCheck = { competitorId: 'rubber-co', competitor: false, sure: true, reason: 'Sells rubber.', model: 'haiku', at: '2026-10-27T09:00:00Z' }

type Body = { method: string; path: string; body: unknown }
function api(sent: Body[] = [], extra: Record<string, Handler> = {}, companies = [fenner, added, rubber, tribelt]) {
  const record = (answer: Handler): Handler => async (request) => {
    const text = await request.text()
    sent.push({ method: request.method, path: new URL(request.url).pathname, body: text === '' ? null : JSON.parse(text) })
    return answer(request)
  }
  return manageApi(
    {
      'GET /api/v1/admin/competitor-checks': () => json({ items: [unsure, archivedCheck], unchecked: 2, review: { running: false } }),
      'POST /api/v1/admin/competitor-checks': record(() => json({ running: true }, 202)),
      'PATCH /api/v1/competitors/fenner': record(() => json(fenner)),
      'PATCH /api/v1/competitors/added-belts': record(() => json(added)),
      'PATCH /api/v1/competitors/rubber-co': record(() => json(rubber)),
      'DELETE /api/v1/competitors/added-belts': record(noContent),
      'POST /api/v1/competitors': record(() => json(competitor('New Belts', { source: 'added' }), 201)),
      ...extra,
    },
    companies,
  )
}
const widget = (wrapper: VueWrapper, title: string) => {
  const found = wrapper.findAll('section').find((s) => s.find('h2').exists() && s.get('h2').text() === title)
  if (!found) throw new Error(`no widget ${title}`)
  return found
}
const field = (scope: ReturnType<typeof widget>, label: string) => {
  const found = scope.findAll('label').find((l) => l.text().startsWith(label))
  if (!found) throw new Error(`no field ${label}`)
  return found.get('input')
}

describe('the Competitor list', () => {
  it('opens on the competitors, the waiting candidates, the checks now, and a card per subpage', async () => {
    const { wrapper } = await mountApp(api(), '/manage/competitors')
    expect(wrapper.get('h1').text()).toBe('Competitor list')
    expect(wrapper.get('header a[href="/competitors"]').text()).toBe('See the competitors')
    expect(widget(wrapper, 'Competitors').text()).toBe('Competitors21 archived')
    expect(widget(wrapper, 'Checks now').text()).toContain('No check is running.')
    expect(widget(wrapper, 'Check the list').text()).toContain('1archived by Haiku')
    expect(widget(wrapper, 'Companies').get('a').attributes('href')).toBe('/manage/competitors/companies')
  })

  it('says which checks run', async () => {
    const { wrapper } = await mountApp(api([], { 'GET /api/v1/admin/competitor-checks': () => json({ items: [], unchecked: 0, review: { running: true } }) }), '/manage/competitors')
    expect(widget(wrapper, 'Checks now').text()).toContain('Haiku is checking the list.')
  })
})

describe('the companies', () => {
  it('lists the competitors at full width, with archive, restore, and open on every row', async () => {
    const sent: Body[] = []
    const { wrapper } = await mountApp(api(sent), '/manage/competitors/companies')
    expect(wrapper.get('h1').text()).toBe('Companies')
    expect(widget(wrapper, 'The list').attributes('data-span')).toBe('12')
    expect(widget(wrapper, 'The list').text()).toContain('2 companies count as competitors and 1 is archived.')
    const rows = wrapper.findAll('table[aria-label="Companies"] tbody tr').map((r) => r.findAll('td').map((td) => td.text()))
    expect(rows).toEqual([
      ['Added Belts', 'added.example.com', 'NL', 'Competitor', 'Archive Added Belts Open Added Belts'],
      ['Fenner', 'fenner.com', 'UK', 'Competitor', 'Archive Fenner Open Fenner'],
      ['Rubber Co', 'twentebelt.com', 'NL', 'Archived by Haiku', 'Restore Rubber Co Open Rubber Co'],
    ])
    await buttonNamed(wrapper, 'Archive Fenner').trigger('click')
    await settle()
    expect(widget(wrapper, 'The list').text()).toContain('Fenner is archived. It no longer counts as a competitor.')
    await buttonNamed(wrapper, 'Restore Rubber Co').trigger('click')
    await settle()
    expect(widget(wrapper, 'The list').text()).toContain('Rubber Co is a competitor again.')
    expect(sent.map((s) => [s.path, s.body])).toEqual([
      ['/api/v1/competitors/fenner', { archived: true }],
      ['/api/v1/competitors/rubber-co', { archived: false }],
    ])
  })

  it('adds a company below the list, and checks it first', async () => {
    const sent: Body[] = []
    const { wrapper } = await mountApp(api(sent), '/manage/competitors/companies')
    const add = widget(wrapper, 'Add a company')
    await add.get('form').trigger('submit')
    expect(add.get('[role="alert"]').text()).toBe('Enter the company’s name.')
    await field(add, 'Name').setValue('New Belts')
    await field(add, 'Website').setValue('www.new-belts.example.com')
    await add.get('form').trigger('submit')
    await settle()
    expect(sent.map((s) => [s.method, s.path, s.body])).toEqual([['POST', '/api/v1/competitors', { name: 'New Belts', url: 'https://www.new-belts.example.com' }]])
    expect(add.text()).toContain('New Belts is added to the list.')
    expect(wrapper.findAll('section > div > h2').map((h) => h.text()).slice(0, 3)).toEqual(['The list', 'Add a company', 'Add several companies'])
  })
})

describe('one company', () => {
  it('saves its details, archives it, restores it, and shows where it shows', async () => {
    const sent: Body[] = []
    const { wrapper } = await mountApp(api(sent), '/manage/competitors/companies/added-belts')
    expect(wrapper.get('h1').text()).toBe('Added Belts')
    expect(wrapper.findAll('a').find((a) => a.text().includes('Back to Companies'))?.attributes('href')).toBe('/manage/competitors/companies')
    const details = widget(wrapper, 'Details')
    expect(field(details, 'Name').element.value).toBe('Added Belts')
    await details.get('form').trigger('submit')
    expect(wrapper.text()).toContain('Nothing changed.')
    await field(details, 'Name').setValue('Added Belts BV')
    await details.get('form').trigger('submit')
    await settle()
    expect(wrapper.text()).toContain('Added Belts BV is saved.')
    await buttonNamed(wrapper, 'Archive').trigger('click')
    await settle()
    expect(sent.map((s) => s.body)).toEqual([{ name: 'Added Belts BV' }, { archived: true }])
    expect(widget(wrapper, 'Where it shows').findAll('a').map((a) => a.attributes('href'))).toEqual([
      '/competitors/added-belts',
      '/ai-readiness',
      `/manage/scans/${summary().id}`,
    ])
  })

  it('removes an added company after asking, and goes back to the list', async () => {
    const sent: Body[] = []
    const { wrapper, router } = await mountApp(api(sent), '/manage/competitors/companies/added-belts')
    await buttonNamed(wrapper, 'Remove').trigger('click')
    expect(wrapper.text()).toContain('Delete this company with its 0 scans? This cannot be undone.')
    await buttonNamed(wrapper, 'Cancel').trigger('click')
    await buttonNamed(wrapper, 'Remove').trigger('click')
    await buttonNamed(wrapper, 'Delete with its 0 scans').trigger('click')
    await settle()
    expect(sent.map((s) => s.method)).toEqual(['DELETE'])
    expect(router.currentRoute.value.fullPath).toBe('/manage/competitors/companies')
  })

  it('restores an archived one, says who archived it, and checks the details', async () => {
    const sent: Body[] = []
    const { wrapper } = await mountApp(api(sent), '/manage/competitors/companies/rubber-co')
    expect(widget(wrapper, 'Archived by Haiku').text()).toContain('Archived by Claude Haiku on 27 October 2026.')
    await field(widget(wrapper, 'Details'), 'Name').setValue(' ')
    await widget(wrapper, 'Details').get('form').trigger('submit')
    expect(wrapper.get('[role="alert"]').text()).toBe('Enter the company’s name.')
    await buttonNamed(wrapper, 'Restore').trigger('click')
    await settle()
    expect(sent.map((s) => s.body)).toEqual([{ archived: false }])
    expect(wrapper.text()).toContain('Rubber Co is a competitor again.')
  })

  it('says when the company is not on the list, or the list could not be read', async () => {
    const gone = await mountApp(api(), '/manage/competitors/companies/nobody')
    expect(gone.wrapper.text()).toContain('This company is not on the list.')
    gone.wrapper.unmount()
    const broken = await mountApp(api([], { 'GET /api/v1/competitors': () => problem(500, 'Internal Server Error') }), '/manage/competitors/companies/fenner')
    expect(broken.wrapper.get('[role="alert"]').text()).toBe('Internal Server Error')
  })
})

describe('check the list', () => {
  it('lists the companies Haiku archived with Restore, the unsure ones with Keep and Archive, and starts a check', async () => {
    const sent: Body[] = []
    const { wrapper } = await mountApp(api(sent), '/manage/competitors/check')
    expect(wrapper.get('h1').text()).toBe('Check the list')
    expect(widget(wrapper, 'Archived by Haiku').text()).toContain('Sells rubber.')
    expect(widget(wrapper, 'Haiku is unsure of these').text()).toContain('Haiku is unsure. Makes rubber belting.')
    expect(widget(wrapper, 'Check the list').text()).toContain('Haiku has not checked 2 companies on the list yet.')
    await buttonNamed(wrapper, 'Restore Rubber Co').trigger('click')
    await settle()
    await buttonNamed(wrapper, 'Keep Fenner').trigger('click')
    await settle()
    await buttonNamed(wrapper, 'Check the list').trigger('click')
    await settle()
    expect(sent.map((s) => [s.method, s.path, s.body])).toEqual([
      ['PATCH', '/api/v1/competitors/rubber-co', { archived: false }],
      ['PATCH', '/api/v1/competitors/fenner', { kept: true }],
      ['POST', '/api/v1/admin/competitor-checks', null],
    ])
  })

  it('says when Haiku archived nothing, and when no Anthropic key is set up', async () => {
    const { wrapper } = await mountApp(
      api([], { 'GET /api/v1/admin/competitor-checks': () => json({ items: [], unchecked: 0, review: { running: false, problem: 'noKey' } }) }, [fenner]),
      '/manage/competitors/check',
    )
    expect(wrapper.text()).toContain('Haiku has not archived any company.')
    expect(wrapper.get('a[href="/manage/keys/anthropic"]').text()).toBe('Add an Anthropic key')
  })
})

describe('stopping Claude Haiku', () => {
  it('stops the check of the list while it runs', async () => {
    const sent: Body[] = []
    const { wrapper } = await mountApp(
      api(sent, {
        'GET /api/v1/admin/competitor-checks': () => json({ items: [], unchecked: 2, review: { running: true } }),
        'POST /api/v1/admin/competitor-checks/stop': (r) => (sent.push({ method: 'POST', path: new URL(r.url).pathname, body: null }), noContent()),
      }),
      '/manage/competitors/check',
    )
    await buttonNamed(wrapper, 'Stop the check').trigger('click')
    await settle()
    expect(sent.map((s) => s.path)).toEqual(['/api/v1/admin/competitor-checks/stop'])
  })

  it('stops classifying the candidates while Haiku reviews them, and says why it could not', async () => {
    const { wrapper } = await mountApp(
      api([], {
        'GET /api/v1/admin/competitor-candidates': () => json({ items: [], counts: { pending: 0, added: 0, dismissed: 0, unchecked: 0 }, review: { running: true } }),
        'POST /api/v1/admin/competitor-candidates/review/stop': () => problem(404, 'Not Found', 'No review runs.'),
      }),
      '/manage/competitors/candidates',
    )
    await buttonNamed(wrapper, 'Stop classifying').trigger('click')
    await settle()
    expect(wrapper.get('[role="alert"]').text()).toBe('No review runs.')
  })
})
