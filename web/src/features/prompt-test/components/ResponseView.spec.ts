import { describe, expect, it } from 'vitest'
import type { PromptAnswer } from '../../../infrastructure/api/types.gen'
import { promptAnswer, promptResults, promptRun, RUN_ID } from '../../../testing/fixtures'
import { type Handler, json, mountApp, problem, routes } from '../../../testing/mountApp'
import { words } from '../../../testing/words'

const OLDER = '1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d'
const ID = '00000000-0000-4000-8000-0000000000c1'

const text = [
  '## Welke band?',
  'Voor de oven is een **metalen band** de juiste keuze. Zie [Tribelt](https://www.tribelt.nl/).',
  '- Spiraalband',
  '- Draadband',
  '<script>alert("x")</script> <img src=x onerror=alert(1)>',
  '[klik](javascript:alert(1))',
].join('\n')

const web = (change: Partial<PromptAnswer> = {}, companies = ['Twentebelt:twentebelt', 'Tribelt B.V.:ashworth', 'Esfo']) =>
  promptAnswer({ id: ID, mode: 'web', text, citations: [], ...change }, companies, {
    addresses: ['https://www.tribelt.nl/belts', 'https://nl.kompass.com/c/twentebelt/', 'ftp://odd.test/'],
    citesTribelt: true,
    wrongClaims: ['Tribelt is in Hengelo.'],
    leftOut: ['Tribelt makes eyelink belts to measure.'],
  })

function api(answer: PromptAnswer, extra: Record<string, Handler> = {}) {
  return routes({
    'GET /api/v1/prompt-test': () => json(promptResults()),
    [`GET /api/v1/prompt-runs/${RUN_ID}/answers`]: () => json({ run: promptRun(), items: [answer] }),
    ...extra,
  })
}

const section = (wrapper: Awaited<ReturnType<typeof mountApp>>['wrapper'], title: string) => {
  const heading = wrapper.findAll('h2').find((h) => h.text() === title)
  if (!heading) throw new Error(`no section ${title}`)
  return wrapper.get(`section[aria-labelledby="${String(heading.attributes('id'))}"]`)
}

describe('One response', () => {
  it('has the frame: the model and mode, the question, Tribelt’s place, and the way back', async () => {
    const { wrapper } = await mountApp(api(web()), `/ai-responses/${ID}?run=${RUN_ID}`)
    expect(wrapper.get('h1').text()).toBe('OpenAI, gpt-oss-20b, with web search')
    expect(wrapper.get('[data-part="purpose"]').text()).toBe(
      'The response to “We bouwen een doorloopoven voor een industriële bakkerij. Welke fabrikanten in de regio Twente?” in the Prompt run of 1 November 2026.',
    )
    expect(wrapper.get('[data-part="data-line"]').text()).toBe('Tribelt is named, number 2. Received on 1 November 2026.')
    expect(wrapper.findAll(`a[href="/ai-responses/all?run=${RUN_ID}"]`).map((a) => a.text())).toContain('Back to Every response')
    expect(wrapper.text()).not.toMatch(/\banswers?\b/i)
  })

  it('renders the response’s Markdown, and never its HTML', async () => {
    const { wrapper } = await mountApp(api(web()), `/ai-responses/${ID}?run=${RUN_ID}`)
    const response = wrapper.get('[data-part="response"]')
    expect(response.attributes('lang')).toBe('nl')
    expect(response.get('h4').text()).toBe('Welke band?')
    expect(response.get('strong').text()).toBe('metalen band')
    expect(response.findAll('li').map((li) => li.text())).toEqual(['Spiraalband', 'Draadband'])
    expect(response.get('a').attributes('href')).toBe('https://www.tribelt.nl/')
    expect(response.findAll('a')).toHaveLength(1)
    expect(response.find('script').exists()).toBe(false)
    expect(response.find('img').exists()).toBe(false)
    expect(response.text()).toContain('<script>alert("x")</script>')
    expect(response.text()).toContain('[klik](javascript:alert(1))')
  })

  it('lists the companies named in order, with Tribelt’s place right when the reading files it under another company', async () => {
    const { wrapper } = await mountApp(api(web()), `/ai-responses/${ID}?run=${RUN_ID}`)
    const companies = section(wrapper, 'Companies named')
    expect(companies.text()).toContain('Tribelt is named, number 2.')
    const rows = companies.findAll('li')
    expect(rows.map(words)).toEqual(['1 Twentebelt', '2 Tribelt B.V. Tribelt', '3 Esfo'])
    expect(rows[1]?.attributes('data-tribelt')).toBe('true')
    expect(rows[1]?.find('a').exists()).toBe(false)
    expect(rows[0]?.get('a').attributes('href')).toBe('/competitors/twentebelt')
  })

  it('lists the sources cited with tribelt.nl marked, and links only web addresses', async () => {
    const { wrapper } = await mountApp(api(web()), `/ai-responses/${ID}?run=${RUN_ID}`)
    const sources = section(wrapper, 'Sources cited').findAll('li')
    expect(sources.map(words)).toEqual(['https://www.tribelt.nl/belts tribelt.nl', 'https://nl.kompass.com/c/twentebelt/', 'ftp://odd.test/'])
    expect(sources[2]?.find('a').exists()).toBe(false)
  })

  it('shows what the fact sheet check found, and who checked it', async () => {
    const { wrapper } = await mountApp(api(web()), `/ai-responses/${ID}?run=${RUN_ID}`)
    const check = section(wrapper, 'Checked against the fact sheet')
    expect(check.findAll('h3').map((h) => h.text())).toEqual(['Wrong about Tribelt', 'Left out'])
    expect(check.text()).toContain('Tribelt is in Hengelo.')
    expect(check.text()).toContain('Tribelt makes eyelink belts to measure.')
    expect(section(wrapper, 'Checked by').text()).toContain('A model read this response to find the companies it names and to check it against the fact sheet.')
  })

  it('says when the check found nothing, and when Tribelt is not named', async () => {
    const clean = promptAnswer({ id: ID, mode: 'knowledge' }, ['Tribelt:tribelt'])
    const first = await mountApp(api(clean), `/ai-responses/${ID}`)
    expect(section(first.wrapper, 'Checked against the fact sheet').text()).toContain('Nothing in this response contradicts the fact sheet.')
    expect(section(first.wrapper, 'Checked against the fact sheet').text()).toContain('The response left out none of the facts on the fact sheet.')
    expect(section(first.wrapper, 'Sources cited').text()).toContain('This response came from the model’s own knowledge, so it cites no sources.')
    expect(first.wrapper.get('[data-part="data-line"]').text()).toMatch(/^Tribelt is named first\./)
    const none = await mountApp(api(promptAnswer({ id: ID, mode: 'web' }, []), {}), `/ai-responses/${ID}`)
    expect(section(none.wrapper, 'Companies named').text()).toContain('This response names no company.')
    expect(section(none.wrapper, 'Sources cited').text()).toContain('This response cites no sources.')
    expect(section(none.wrapper, 'Checked against the fact sheet').text()).toContain('The response does not name Tribelt, so no facts count as left out.')
  })

  it('says when no model has checked the response yet', async () => {
    const { wrapper } = await mountApp(api(promptAnswer({ id: ID, extraction: undefined })), `/ai-responses/${ID}`)
    expect(section(wrapper, 'Checked by').text()).toContain('No model has checked this response yet.')
    expect(section(wrapper, 'Checked against the fact sheet').text()).toContain('No model has checked this response yet.')
    expect(section(wrapper, 'Companies named').text()).toContain('No model has read this response yet')
  })

  it('says when the response failed', async () => {
    const { wrapper } = await mountApp(api(promptAnswer({ id: ID, failure: 'The model is rate limited.', extraction: undefined, text: '' })), `/ai-responses/${ID}`)
    expect(section(wrapper, 'The response').text()).toContain('This response failed, so the model gave no text. The model is rate limited.')
    expect(section(wrapper, 'Companies named').text()).toContain('This response failed, so no companies were read from it.')
    expect(section(wrapper, 'Checked against the fact sheet').text()).toContain('A failed response is not checked against the fact sheet.')
  })

  it('says so when no response has the address, and reports failures', async () => {
    const missing = await mountApp(api(web()), '/ai-responses/00000000-0000-4000-8000-0000000000ff')
    expect(missing.wrapper.get('h1').text()).toBe('Response not found')
    expect(missing.wrapper.text()).toContain('No response has this address. Go back to Every response and pick one.')
    const gone = await mountApp(api(web(), { [`GET /api/v1/prompt-runs/${OLDER}/answers`]: () => problem(500, 'Internal Server Error', 'Gone.') }), `/ai-responses/${ID}?run=${OLDER}`)
    expect(gone.wrapper.get('[role="alert"]').text()).toContain('Gone.')
  })
})
