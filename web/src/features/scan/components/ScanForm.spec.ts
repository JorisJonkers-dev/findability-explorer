import type { VueWrapper } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { analysis, competitor, me, termSearch, term } from '../../../testing/fixtures'
import { type Handler, json, problem, routes, settle } from '../../../testing/mountApp'
import { mountAt } from '../../../testing/mountAt'
import ScanForm from './ScanForm.vue'

// The form that starts a scan, in Manage, Website scans: it checks the address, suggests known companies, and names a new one.
const NEW_ID = '6c1d2b0f-3e4a-4f6b-9c7d-8e9f0a1b2c3d'
const twentebelt = competitor('Twentebelt', { scans: 2 })
const martens = competitor('Martens', { country: 'DE', websiteUrl: 'https://www.martens-belts.com/', host: 'martens-belts.com', website: 'https://www.martens-belts.com' })
const tribelt = competitor('Tribelt', { role: 'tribelt', groupCompany: true, websiteUrl: 'https://www.tribelt.nl/', host: 'tribelt.nl', website: 'https://www.tribelt.nl' })

function api(extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/me': () => json(me({ isAdmin: true })),
    'GET /api/v1/competitors': () => json({ items: [twentebelt, martens, tribelt] }),
    'GET /api/v1/terms': () => json(termSearch([term({ usedBy: ['Twentebelt'] })])),
    'GET /api/v1/analyses': () => json({ items: [] }),
    'POST /api/v1/analyses': () => json(analysis({}, { id: NEW_ID, status: 'queued', pagesCrawled: 0 }), 202),
    ...extra,
  })
}

const combobox = (wrapper: VueWrapper) => wrapper.get('input[role="combobox"]')

async function type(wrapper: VueWrapper, text: string) {
  await combobox(wrapper).setValue(text)
  await combobox(wrapper).trigger('input')
}

async function submit(wrapper: VueWrapper) {
  await wrapper.get('form[aria-label="Scan a website"]').trigger('submit')
  await settle()
}

describe('the scan form', () => {
  it('checks the address before sending it, and clears the message on typing', async () => {
    const { wrapper, requests } = await mountAt(ScanForm, '/manage/scans', api())
    await type(wrapper, 'https://intranet')
    await submit(wrapper)
    expect(wrapper.get('#scan-url-error').text()).toBe('Enter an address with a full host name, like https://www.example.com.')
    expect(combobox(wrapper).attributes('aria-invalid')).toBe('true')
    expect(combobox(wrapper).attributes('aria-describedby')).toBe('scan-url-error')
    expect(requests().filter((r) => r.startsWith('POST'))).toEqual([])
    await type(wrapper, 'https://www.')
    expect(wrapper.find('#scan-url-error').exists()).toBe(false)
  })

  it("puts the server's refusal under the address or under the form", async () => {
    let status = 422
    const refuse = { 'POST /api/v1/analyses': () => problem(status, 'Refused', status === 422 ? 'The host is not public.' : 'A scan of this company is still queued or running.') }
    const { wrapper } = await mountAt(ScanForm, '/manage/scans', api(refuse))
    await type(wrapper, 'https://www.example.com')
    await submit(wrapper)
    expect(wrapper.get('#scan-url-error').text()).toBe('The host is not public.')
    status = 409
    await submit(wrapper)
    expect(wrapper.find('#scan-url-error').exists()).toBe(false)
    expect(wrapper.get('[role="alert"]').text()).toBe('A scan of this company is still queued or running.')
  })

  it('suggests known companies as you type, and picks one with the keyboard', async () => {
    const { wrapper } = await mountAt(ScanForm, '/manage/scans', api())
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('false')
    await type(wrapper, 'twente')
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('true')
    expect(wrapper.findAll('[role="option"]').map((o) => o.text())).toEqual(['Twentebelttwentebelt.com'])
    await combobox(wrapper).trigger('keydown', { key: 'ArrowDown' })
    expect(combobox(wrapper).attributes('aria-activedescendant')).toBe('scan-url-option-0')
    await combobox(wrapper).trigger('keydown', { key: 'ArrowUp' })
    await combobox(wrapper).trigger('keydown', { key: 'ArrowUp' })
    expect(combobox(wrapper).attributes('aria-activedescendant')).toBe('scan-url-option-0')
    await combobox(wrapper).trigger('keydown', { key: 'Enter' })
    expect((combobox(wrapper).element as HTMLInputElement).value).toBe('https://www.twentebelt.com/')
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('false')
    expect(wrapper.get('#scan-url-company').text()).toBe('Twentebelt, a known competitor. Scanned 2 times.')
  })

  it('closes the suggestions on Escape or blur, picks one with the mouse, and ignores keys with nothing to suggest', async () => {
    const { wrapper } = await mountAt(ScanForm, '/manage/scans', api())
    await type(wrapper, 'zzz')
    await combobox(wrapper).trigger('keydown', { key: 'ArrowDown' })
    await combobox(wrapper).trigger('keydown', { key: 'ArrowUp' })
    await combobox(wrapper).trigger('keydown', { key: 'Enter' })
    await combobox(wrapper).trigger('keydown', { key: 'Escape' })
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('false')
    await type(wrapper, 'tribelt')
    await combobox(wrapper).trigger('keydown', { key: 'Escape' })
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('false')
    await combobox(wrapper).trigger('keydown', { key: 'ArrowDown' })
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('true')
    await combobox(wrapper).trigger('blur')
    expect(combobox(wrapper).attributes('aria-expanded')).toBe('false')
    await type(wrapper, 'märtens')
    await wrapper.get('[role="option"]').trigger('mousedown')
    expect((combobox(wrapper).element as HTMLInputElement).value).toBe('https://www.martens-belts.com/')
    await type(wrapper, 'https://www.tribelt.nl')
    expect(wrapper.get('#scan-url-company').text()).toContain("Tribelt, in Tribelt's own group.")
  })

  it('names an unknown company after its host, lets the name be changed, and sends it', async () => {
    const sentBodies: unknown[] = []
    const { wrapper } = await mountAt(
      ScanForm,
      '/manage/scans',
      api({
        'POST /api/v1/analyses': async (request) => {
          sentBodies.push(await request.json())
          return json(analysis({}, { id: NEW_ID, status: 'queued' }), 202)
        },
      }),
    )
    await type(wrapper, 'www.example-belts.de')
    const name = wrapper.get('#scan-url-company input')
    expect((name.element as HTMLInputElement).value).toBe('Example Belts')
    expect(wrapper.text()).toContain('The scan adds it as a new competitor.')
    await name.setValue('Example Belts GmbH')
    await name.trigger('input')
    await type(wrapper, 'www.example-belts.de/')
    expect((name.element as HTMLInputElement).value).toBe('Example Belts GmbH')
    await submit(wrapper)
    expect(sentBodies).toEqual([{ url: 'https://www.example-belts.de/', maxPages: 60, companyName: 'Example Belts GmbH' }])
  })
})
