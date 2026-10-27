// Tests apply.ts against a small fake of the admin API. Run with:
//   node --test scripts/competitor-map/

import type { AddressInfo } from 'node:net'
import type { Candidate, CompetitorMap, MapEntry, Status } from './apply.ts'
import assert from 'node:assert/strict'
import { mkdtempSync, readFileSync, writeFileSync } from 'node:fs'
import { createServer } from 'node:http'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { after, before, describe, it } from 'node:test'
import { fileURLToPath } from 'node:url'
import { host, match, buildIndex, normName, run } from './apply.ts'

function entry(e: Partial<MapEntry> & Pick<MapEntry, 'name' | 'verdict'>): MapEntry {
  return { otherNames: [], websites: [], confidence: 'high', reason: 'Test reason.', ...e }
}

const testMap: CompetitorMap = {
  date: '2026-10-27',
  companies: [
    entry({ name: 'George Lane', verdict: 'competitor', websites: ['georgelane.co.uk'], website: 'https://www.georgelane.co.uk' }),
    entry({ name: 'Hebei Risen', verdict: 'competitor', otherNames: ['Hebei Risen Conveyor Belt Products Co., Ltd.'], website: 'https://www.metalconveybelt.com' }),
    entry({ name: 'Kienzle GmbH', verdict: 'competitor', websites: ['kienzle.com'] }),
    entry({ name: 'wire-mesh.com', verdict: 'competitor', duplicateOf: 'Wire-Mesh Products, Inc.' }),
    entry({ name: 'Taken Site', verdict: 'competitor', website: 'https://www.twentebelt.com' }),
    entry({ name: 'Flexco', verdict: 'not', websites: ['flexco.com'] }),
    entry({ name: 'Dürlsch', verdict: 'not', nonexistent: true }),
    entry({ name: 'Kramer', verdict: 'unclear', websites: ['kramer-band.de'] }),
    entry({ name: 'Wrongly Added', verdict: 'not' }),
    entry({ name: 'Wrongly Dismissed', verdict: 'competitor', website: 'https://wrongly-dismissed.example.com' }),
  ],
}

let seq = 0
function cand(name: string, status: Status, extra: Partial<Candidate> = {}): Candidate {
  seq++
  return {
    id: seq.toString(16).padStart(16, '0'),
    name,
    status,
    answers: 1,
    addresses: [],
    ...(status === 'pending' ? {} : { decision: { status, by: 'joris', at: '2026-10-20T00:00:00Z' } }),
    ...extra,
  }
}

// Fake is an in-memory admin API with the endpoints apply.ts uses.
class Fake {
  candidates: Candidate[] = []
  competitors: { name: string, host: string }[] = [{ name: 'Twentebelt', host: 'twentebelt.com' }]
  mutations: string[] = []
  sawCookie = ''

  reset() {
    seq = 0
    this.mutations = []
    this.competitors = [{ name: 'Twentebelt', host: 'twentebelt.com' }]
    this.candidates = [
      cand('George Lane Ltd', 'pending', { sites: ['georgelane.co.uk'] }),
      cand('Hebei Risen Conveyor Belt Products Co., Ltd.', 'pending'),
      cand('Kienzle GmbH Drahtförderbänder', 'pending', { sites: ['kienzle.com'] }),
      cand('wire-mesh.com', 'pending'),
      cand('Taken Site', 'pending'),
      cand('Flexco Europe BV', 'pending', { addresses: ['https://www.flexco.com/de'] }),
      cand('Durlsch', 'pending'),
      cand('Kramer', 'pending'),
      cand('Unknown Belts', 'pending'),
      cand('Wrongly Added', 'added'),
      cand('Wrongly Dismissed', 'dismissed'),
      cand('Not In The Map', 'dismissed'),
    ]
  }

  find(id: string) {
    return this.candidates.find(c => c.id === id)
  }

  handle(method: string, url: URL, body: unknown): [number, unknown] {
    const p = url.pathname
    const base = '/api/v1/admin/competitor-candidates'
    if (method === 'GET' && p === base) {
      const s = url.searchParams.get('status')
      return [200, { items: this.candidates.filter(c => c.status === s), counts: {}, review: { running: false } }]
    }
    if (method === 'GET' && p === '/api/v1/competitors')
      return [200, { items: this.competitors }]
    this.mutations.push(`${method} ${p}`)
    if (method === 'POST' && p === `${base}/decisions`) {
      const b = body as { action: string, ids: string[] }
      const items = b.ids.map((id) => {
        const c = this.find(id)
        if (!c || c.status !== 'pending')
          return { id, outcome: 'skipped', reason: 'Decided already.' }
        c.status = 'dismissed'
        return { id, name: c.name, outcome: 'done' }
      })
      return [200, { items }]
    }
    const m = p.match(/^\/api\/v1\/admin\/competitor-candidates\/([0-9a-f]{16})\/(add|dismiss|decision)$/)
    const c = m && this.find(m[1])
    if (!m || !c)
      return [404, { title: 'Not found' }]
    if (method === 'POST' && m[2] === 'add') {
      const b = body as { name: string, url: string }
      const h = host(b.url)
      if (c.status !== 'pending')
        return [409, { detail: 'Decided already.' }]
      if (this.competitors.some(x => x.host === h))
        return [409, { detail: 'Another company has the website.' }]
      if (h === 'fails.example.com')
        return [500, { detail: 'Boom.' }]
      this.competitors.push({ name: b.name, host: h })
      c.status = 'added'
      return [200, c]
    }
    if (method === 'POST' && m[2] === 'dismiss') {
      if (c.status !== 'pending')
        return [409, { detail: 'Decided already.' }]
      c.status = 'dismissed'
      return [200, c]
    }
    if (method === 'DELETE' && m[2] === 'decision') {
      if (c.status === 'pending')
        return [404, { detail: 'No decision.' }]
      c.status = 'pending'
      return [204, undefined]
    }
    return [405, { title: 'Method not allowed' }]
  }
}

const fake = new Fake()
let baseUrl = ''
const server = createServer((req, res) => {
  let raw = ''
  req.on('data', (d) => { raw += d })
  req.on('end', () => {
    fake.sawCookie = String(req.headers.cookie ?? '')
    if (fake.sawCookie !== 'fe_session=secret-session') {
      res.writeHead(401, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ title: 'Unauthorized' }))
      return
    }
    const [status, json] = fake.handle(req.method ?? 'GET', new URL(req.url ?? '/', 'http://x'), raw ? JSON.parse(raw) : undefined)
    res.writeHead(status, { 'Content-Type': 'application/json' })
    res.end(json === undefined ? '' : JSON.stringify(json))
  })
})

const dir = mkdtempSync(join(tmpdir(), 'competitor-map-'))
const mapPath = join(dir, 'map.json')
writeFileSync(mapPath, JSON.stringify(testMap))

async function go(args: { apply?: boolean, fixDecided?: boolean }, cookie = 'secret-session') {
  const lines: string[] = []
  const code = await run({ apply: !!args.apply, fixDecided: !!args.fixDecided, mapPath }, { FE_BASE_URL: baseUrl, FE_COOKIE: cookie }, l => lines.push(l))
  return { code, out: lines.join('\n') }
}

const status = (name: string) => fake.candidates.find(c => c.name === name)?.status

before(async () => {
  await new Promise<void>(r => server.listen(0, '127.0.0.1', r))
  baseUrl = `http://127.0.0.1:${(server.address() as AddressInfo).port}`
})
after(() => server.close())

describe('matching', () => {
  it('folds names', () => {
    assert.equal(normName('Hebei Risen Conveyor Belt Products Co., Ltd.'), 'hebei risen conveyor belt products')
    assert.equal(normName('Dürlsch'), normName('Durlsch'))
    assert.equal(normName('Haver & Boecker OHG'), 'haver and boecker')
    assert.equal(normName('Bruynooghe nv'), 'bruynooghe')
  })
  it('finds hosts', () => {
    assert.equal(host('https://www.Flexco.com/de'), 'flexco.com')
    assert.equal(host('georgelane.co.uk'), 'georgelane.co.uk')
    assert.equal(host(''), '')
  })
  it('matches by website before name, and refuses a match with mixed verdicts', () => {
    const ix = buildIndex({ date: '', companies: [
      entry({ name: 'Same', verdict: 'competitor', websites: ['a.example'] }),
      entry({ name: 'Same Name', verdict: 'not' }),
      entry({ name: 'Same Name', verdict: 'competitor', otherNames: [] }),
    ] })
    assert.equal(match(cand('Same Name', 'pending', { sites: ['a.example'] }), ix)?.name, 'Same')
    assert.equal(match(cand('Same Name', 'pending'), ix), undefined)
  })
})

describe('apply', () => {
  it('only reports in a dry run, and never prints the cookie', async () => {
    fake.reset()
    const { code, out } = await go({})
    assert.equal(code, 0)
    assert.deepEqual(fake.mutations, [])
    assert.match(out, /Would add 2 candidates to the competitor list and dismiss 2\./)
    assert.match(out, /This was a dry run/)
    assert.doesNotMatch(out, /secret-session/)
  })

  it('adds and dismisses as the map says and leaves the rest', async () => {
    fake.reset()
    const { code, out } = await go({ apply: true })
    assert.equal(code, 0, out)
    assert.equal(status('George Lane Ltd'), 'added')
    assert.equal(status('Hebei Risen Conveyor Belt Products Co., Ltd.'), 'added')
    assert.ok(fake.competitors.some(c => c.host === 'metalconveybelt.com' && c.name === 'Hebei Risen'))
    assert.equal(status('Flexco Europe BV'), 'dismissed')
    assert.equal(status('Durlsch'), 'dismissed')
    for (const n of ['Kienzle GmbH Drahtförderbänder', 'wire-mesh.com', 'Taken Site', 'Kramer', 'Unknown Belts'])
      assert.equal(status(n), 'pending', n)
    assert.equal(status('Wrongly Added'), 'added')
    assert.equal(status('Wrongly Dismissed'), 'dismissed')
    assert.match(out, /Decided, but the map disagrees \(2\)/)
    assert.match(out, /Decided, not in the map \(1\)/)
    assert.match(out, /on the list already as Twentebelt/)
  })

  it('changes nothing the second time', async () => {
    const { code } = await go({ apply: true })
    assert.equal(code, 0)
    fake.mutations = []
    const again = await go({ apply: true })
    assert.equal(again.code, 0)
    assert.deepEqual(fake.mutations, [])
    assert.match(again.out, /Will add 0 candidates to the competitor list and dismiss 0\./)
  })

  it('fixes decided candidates only with --fix-decided', async () => {
    const { code, out } = await go({ apply: true, fixDecided: true })
    assert.equal(code, 0, out)
    assert.equal(status('Wrongly Added'), 'dismissed')
    assert.equal(status('Wrongly Dismissed'), 'added')
    assert.equal(status('Not In The Map'), 'dismissed')
    fake.mutations = []
    await go({ apply: true, fixDecided: true })
    assert.deepEqual(fake.mutations, [])
  })

  it('goes on after a failure and says so', async () => {
    fake.reset()
    const map = structuredClone(testMap)
    map.companies[0].website = 'https://fails.example.com'
    writeFileSync(mapPath, JSON.stringify(map))
    try {
      const { code, out } = await go({ apply: true })
      assert.equal(code, 1)
      assert.match(out, /FAILED to add George Lane Ltd/)
      assert.equal(status('Hebei Risen Conveyor Belt Products Co., Ltd.'), 'added')
      assert.match(out, /Done, with 1 failures\./)
    }
    finally {
      writeFileSync(mapPath, JSON.stringify(testMap))
    }
  })

  it('stops when the session is refused', async () => {
    fake.reset()
    await assert.rejects(go({}, 'wrong'), /FE_COOKIE is the session of an administrator/)
  })
})

describe('the map file', () => {
  const real = JSON.parse(readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'competitor-map.json'), 'utf8')) as CompetitorMap & { counts: Record<string, number> }

  it('is well formed', () => {
    const names = new Set<string>()
    for (const e of real.companies) {
      assert.ok(!names.has(e.name), `duplicate entry ${e.name}`)
      names.add(e.name)
      assert.ok(['competitor', 'not', 'unclear'].includes(e.verdict), e.name)
      assert.ok(['high', 'medium', 'low'].includes(e.confidence), e.name)
      assert.ok(e.reason && (e as unknown as { makes: string }).makes, e.name)
      assert.ok((e as unknown as { evidence: string[] }).evidence.length > 0, e.name)
      if (e.nonexistent)
        assert.equal(e.verdict, 'not', e.name)
      if (e.website)
        assert.match(e.website, /^https:\/\//, e.name)
    }
    assert.equal(real.counts.companies, real.companies.length)
  })

  it('gives every new competitor a website, or says why not', () => {
    for (const e of real.companies) {
      if (e.verdict === 'competitor' && !e.duplicateOf && !e.website)
        assert.match(e.reason, /website/i, e.name)
    }
  })
})
