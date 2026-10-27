// Applies the researched competitor map to the competitor review through the admin API.
//
// Run with Node 24 or later, which runs TypeScript as it is:
//   FE_BASE_URL=https://... FE_COOKIE=<fe_session value> node scripts/competitor-map/apply.ts
// It only reports what it would do. Add --apply to act, and --fix-decided to also change decided
// candidates the map disagrees with. See README.md.

import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

export type Verdict = 'competitor' | 'not' | 'unclear'

export interface MapEntry {
  name: string
  otherNames: string[]
  websites: string[]
  verdict: Verdict
  confidence: 'high' | 'medium' | 'low'
  reason: string
  website?: string
  nonexistent?: boolean
  duplicateOf?: string
}

export interface CompetitorMap {
  date: string
  companies: MapEntry[]
}

export type Status = 'pending' | 'added' | 'dismissed'

export interface Candidate {
  id: string
  name: string
  status: Status
  answers: number
  addresses: string[]
  sites?: string[]
  website?: string
  names?: { name: string }[]
  decision?: { status: Status, by: string, at: string }
}

export interface Competitor {
  name: string
  host?: string
}

export type Action =
  | { kind: 'add', candidate: Candidate, entry: MapEntry, url: string }
  | { kind: 'dismiss', candidate: Candidate, entry: MapEntry }
  | { kind: 'undo-add', candidate: Candidate, entry: MapEntry, url: string }
  | { kind: 'undo-dismiss', candidate: Candidate, entry: MapEntry }

export interface Note { candidate: Candidate, entry?: MapEntry, why: string }

export interface Plan {
  actions: Action[]
  // Pending candidates left alone, with why.
  untouched: Note[]
  // Decided candidates the map disagrees with.
  disagreements: Note[]
  // Decided candidates the map does not cover.
  uncovered: Candidate[]
  agreeing: number
}

const legalForms = new Set([
  'gmbh', 'bv', 'nv', 'co', 'kg', 'ltd', 'limited', 'inc', 'corp', 'corporation', 'llc', 'spa',
  'srl', 'aps', 'plc', 'ohg', 'ag', 'sa', 'the', 'company', 'group',
])

// normName folds a company name for matching: no case, accents, punctuation or legal form.
export function normName(name: string): string {
  return name
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/&/g, ' and ')
    .replace(/\b([a-z])\.(?=[a-z]\.)/g, '$1')
    .replace(/[^a-z0-9]+/g, ' ')
    .split(' ')
    .filter(w => w && !legalForms.has(w))
    .join(' ')
}

// host gives the host of an address or a bare domain, lower case and without "www.".
export function host(address: string): string {
  const s = address.trim().toLowerCase()
  if (!s)
    return ''
  try {
    const h = new URL(s.includes('://') ? s : `https://${s}`).hostname
    return h.replace(/^www\./, '')
  }
  catch {
    return ''
  }
}

function hostsOf(c: Candidate): string[] {
  const all = [...(c.sites ?? []), c.website ?? '', ...c.addresses].map(host).filter(Boolean)
  return [...new Set(all)]
}

function namesOf(c: Candidate): string[] {
  return [...new Set([c.name, ...(c.names ?? []).map(n => n.name)].map(normName).filter(Boolean))]
}

export interface Index {
  byHost: Map<string, MapEntry[]>
  byName: Map<string, MapEntry[]>
}

function push<K, V>(m: Map<K, V[]>, k: K, v: V) {
  const l = m.get(k)
  if (!l)
    m.set(k, [v])
  else if (!l.includes(v))
    l.push(v)
}

export function buildIndex(map: CompetitorMap): Index {
  const byHost = new Map<string, MapEntry[]>()
  const byName = new Map<string, MapEntry[]>()
  for (const e of map.companies) {
    for (const w of [...e.websites, e.website ?? '']) {
      const h = host(w)
      if (h)
        push(byHost, h, e)
    }
    for (const n of [e.name, ...e.otherNames]) {
      const k = normName(n)
      if (k)
        push(byName, k, e)
    }
  }
  return { byHost, byName }
}

// match finds the map entry for a candidate: by website host first, then by its names. A match
// that points at entries with different verdicts is no match.
export function match(c: Candidate, ix: Index): MapEntry | undefined {
  const pick = (found: MapEntry[]) => {
    const unique = [...new Set(found)]
    if (unique.length === 0)
      return undefined
    if (unique.length === 1)
      return unique[0]
    const same = unique.every(e => e.verdict === unique[0].verdict && e.duplicateOf === unique[0].duplicateOf)
    return same ? unique[0] : undefined
  }
  const byHost = pick(hostsOf(c).flatMap(h => ix.byHost.get(h) ?? []))
  if (byHost)
    return byHost
  return pick(namesOf(c).flatMap(n => ix.byName.get(n) ?? []))
}

// plan decides what to do with each candidate. listed holds the hosts already on the company list.
export function plan(candidates: Candidate[], map: CompetitorMap, listed: Map<string, string>, fixDecided: boolean): Plan {
  const ix = buildIndex(map)
  const p: Plan = { actions: [], untouched: [], disagreements: [], uncovered: [], agreeing: 0 }
  for (const c of candidates) {
    const e = match(c, ix)
    if (c.status === 'pending') {
      if (!e) {
        p.untouched.push({ candidate: c, why: 'The map has no entry for it.' })
        continue
      }
      if (e.verdict === 'unclear') {
        p.untouched.push({ candidate: c, entry: e, why: 'The map leaves it unclear.' })
        continue
      }
      if (e.verdict === 'not') {
        p.actions.push({ kind: 'dismiss', candidate: c, entry: e })
        continue
      }
      if (e.duplicateOf) {
        p.untouched.push({ candidate: c, entry: e, why: `It is the same company as ${e.duplicateOf}, which is on the list already. Merge it by hand or let the review merge it.` })
        continue
      }
      if (!e.website) {
        p.untouched.push({ candidate: c, entry: e, why: 'A competitor, but no working website was found to add it with. Add it by hand once it has one.' })
        continue
      }
      const owner = listed.get(host(e.website))
      if (owner) {
        p.untouched.push({ candidate: c, entry: e, why: `A competitor, but its website is on the list already as ${owner}.` })
        continue
      }
      p.actions.push({ kind: 'add', candidate: c, entry: e, url: e.website })
      continue
    }
    if (!e) {
      p.uncovered.push(c)
      continue
    }
    const wanted = e.verdict === 'competitor' && !e.duplicateOf ? 'added' : e.verdict === 'not' ? 'dismissed' : undefined
    if (e.verdict === 'unclear' || (e.verdict === 'competitor' && e.duplicateOf && c.status === 'dismissed')) {
      p.agreeing++
      continue
    }
    if (e.duplicateOf) {
      p.disagreements.push({ candidate: c, entry: e, why: `Added, but the map says it is the same company as ${e.duplicateOf}. Not changed; undo it by hand if it doubles that company.` })
      continue
    }
    if (wanted === c.status) {
      p.agreeing++
      continue
    }
    if (wanted === 'dismissed') {
      p.disagreements.push({ candidate: c, entry: e, why: `Added, but the map says it is not a competitor. ${e.reason}` })
      if (fixDecided)
        p.actions.push({ kind: 'undo-add', candidate: c, entry: e, url: '' })
      continue
    }
    if (!e.website) {
      p.disagreements.push({ candidate: c, entry: e, why: 'Dismissed, but the map says it is a competitor. It has no website to add it with, so it is not changed.' })
      continue
    }
    const owner = listed.get(host(e.website))
    if (owner) {
      p.disagreements.push({ candidate: c, entry: e, why: `Dismissed, but the map says it is a competitor. Its website is on the list already as ${owner}, so it is not changed.` })
      continue
    }
    p.disagreements.push({ candidate: c, entry: e, why: `Dismissed, but the map says it is a competitor. ${e.reason}` })
    if (fixDecided)
      p.actions.push({ kind: 'undo-dismiss', candidate: c, entry: e })
  }
  return p
}

export class Api {
  private readonly base: string
  private readonly cookie: string
  constructor(base: string, cookie: string) {
    this.base = base.replace(/\/+$/, '')
    this.cookie = cookie.includes('=') ? cookie : `fe_session=${cookie}`
  }

  async call(method: string, path: string, body?: unknown): Promise<{ status: number, json: unknown }> {
    const res = await fetch(this.base + path, {
      method,
      redirect: 'manual',
      headers: {
        'Cookie': this.cookie,
        'Accept': 'application/json',
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      signal: AbortSignal.timeout(60_000),
    })
    const text = await res.text()
    let json: unknown
    try {
      json = text ? JSON.parse(text) : undefined
    }
    catch {
      json = text
    }
    return { status: res.status, json }
  }

  async get<T>(path: string): Promise<T> {
    const r = await this.call('GET', path)
    if (r.status === 401 || r.status === 403)
      throw new Error(`The API refused ${path} with ${r.status}. Check that FE_COOKIE is the session of an administrator.`)
    if (r.status !== 200)
      throw new Error(`GET ${path} answered ${r.status}: ${problem(r.json)}`)
    return r.json as T
  }
}

function problem(json: unknown): string {
  if (json && typeof json === 'object') {
    const o = json as Record<string, unknown>
    return String(o.detail ?? o.title ?? JSON.stringify(o)).slice(0, 300)
  }
  return String(json ?? '').slice(0, 300)
}

const candidatesPath = '/api/v1/admin/competitor-candidates'

export async function fetchCandidates(api: Api): Promise<Candidate[]> {
  const all: Candidate[] = []
  for (const s of ['pending', 'added', 'dismissed'] as const) {
    const list = await api.get<{ items: Candidate[] }>(`${candidatesPath}?status=${s}`)
    all.push(...list.items.map(c => ({ ...c, status: c.status ?? s })))
  }
  return all
}

export async function fetchListed(api: Api): Promise<Map<string, string>> {
  const list = await api.get<{ items: Competitor[] }>('/api/v1/competitors')
  const m = new Map<string, string>()
  for (const c of list.items) {
    if (c.host)
      m.set(c.host.replace(/^www\./, ''), c.name)
  }
  return m
}

type Log = (line: string) => void

function label(c: Candidate) {
  return `${c.name} (${c.id})`
}

// execute carries the actions out, one by one, and goes on after a failure. It returns how many
// failed.
export async function execute(api: Api, actions: Action[], log: Log): Promise<number> {
  let failed = 0
  const ok = (r: { status: number }) => r.status >= 200 && r.status < 300
  const fail = (what: string, r: { status: number, json: unknown }) => {
    failed++
    log(`FAILED ${what}. The API answered ${r.status}: ${problem(r.json)}`)
  }
  const add = async (c: Candidate, e: MapEntry, url: string) => {
    const r = await api.call('POST', `${candidatesPath}/${c.id}/add`, { name: e.name, url })
    if (ok(r))
      log(`Added ${label(c)} as ${e.name} with ${url}.`)
    else fail(`to add ${label(c)}`, r)
  }
  const undo = async (c: Candidate) => {
    const r = await api.call('DELETE', `${candidatesPath}/${c.id}/decision`)
    if (ok(r)) {
      log(`Undid the decision on ${label(c)}.`)
      return true
    }
    fail(`to undo the decision on ${label(c)}`, r)
    return false
  }

  const dismissals = actions.filter(a => a.kind === 'dismiss')
  for (let i = 0; i < dismissals.length; i += 200) {
    const chunk = dismissals.slice(i, i + 200)
    const r = await api.call('POST', `${candidatesPath}/decisions`, { action: 'dismiss', ids: chunk.map(a => a.candidate.id) })
    if (!ok(r)) {
      fail(`to dismiss ${chunk.length} candidates at once`, r)
      continue
    }
    const items = (r.json as { items?: { id: string, outcome: string, reason?: string }[] })?.items ?? []
    const byId = new Map(items.map(it => [it.id, it]))
    for (const a of chunk) {
      const it = byId.get(a.candidate.id)
      if (it?.outcome === 'done') {
        log(`Dismissed ${label(a.candidate)}.`)
      }
      else {
        failed++
        log(`SKIPPED dismissing ${label(a.candidate)}. ${it?.reason ?? 'The API gave no outcome for it.'}`)
      }
    }
  }

  for (const a of actions) {
    if (a.kind === 'add') {
      await add(a.candidate, a.entry, a.url)
    }
    else if (a.kind === 'undo-add') {
      if (await undo(a.candidate)) {
        const r = await api.call('POST', `${candidatesPath}/${a.candidate.id}/dismiss`)
        if (ok(r))
          log(`Dismissed ${label(a.candidate)}.`)
        else fail(`to dismiss ${label(a.candidate)}`, r)
      }
    }
    else if (a.kind === 'undo-dismiss') {
      if (await undo(a.candidate) && a.entry.website)
        await add(a.candidate, a.entry, a.entry.website)
    }
  }
  return failed
}

export function report(p: Plan, applying: boolean, fixDecided: boolean, log: Log) {
  const count = (k: Action['kind']) => p.actions.filter(a => a.kind === k).length
  const verb = applying ? 'Will' : 'Would'
  log(`${verb} add ${count('add')} candidates to the competitor list and dismiss ${count('dismiss')}.`)
  if (fixDecided)
    log(`${verb} change ${count('undo-add') + count('undo-dismiss')} decided candidates the map disagrees with.`)
  log(`${p.untouched.length} waiting candidates are left alone, ${p.disagreements.length} decided ones disagree with the map, ${p.uncovered.length} decided ones are not in the map, and ${p.agreeing} decided ones agree with it.`)
  const section = (title: string, lines: string[]) => {
    if (lines.length === 0)
      return
    log('')
    log(`${title} (${lines.length})`)
    for (const l of lines) log(`  ${l}`)
  }
  section('To add', p.actions.filter(a => a.kind === 'add').map(a => `${label(a.candidate)} as ${a.entry.name} with ${(a as { url: string }).url}`))
  section('To dismiss', p.actions.filter(a => a.kind === 'dismiss').map(a => `${label(a.candidate)}${a.entry.nonexistent ? ', not a real company' : ''}`))
  section('Left alone', p.untouched.map(n => `${label(n.candidate)}. ${n.why}`))
  section('Decided, but the map disagrees', p.disagreements.map(n => `${label(n.candidate)} is ${n.candidate.status}. ${n.why}`))
  section('Decided, not in the map', p.uncovered.map(c => `${label(c)} is ${c.status}${c.decision ? ` by ${c.decision.by}` : ''}.`))
}

export interface Options { apply: boolean, fixDecided: boolean, mapPath: string }

export function parseArgs(argv: string[]): Options {
  const o: Options = { apply: false, fixDecided: false, mapPath: join(dirname(fileURLToPath(import.meta.url)), 'competitor-map.json') }
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i]
    if (a === '--apply')
      o.apply = true
    else if (a === '--fix-decided')
      o.fixDecided = true
    else if (a === '--map')
      o.mapPath = argv[++i]
    else throw new Error(`Unknown argument ${a}. Use --apply, --fix-decided or --map <file>.`)
  }
  return o
}

export async function run(o: Options, env: Record<string, string | undefined>, log: Log): Promise<number> {
  const base = env.FE_BASE_URL
  const cookie = env.FE_COOKIE
  if (!base || !/^https?:\/\//.test(base))
    throw new Error('Set FE_BASE_URL to the app\'s address, such as http://localhost:8181.')
  if (!cookie)
    throw new Error('Set FE_COOKIE to the value of an administrator\'s fe_session cookie.')
  const map = JSON.parse(readFileSync(o.mapPath, 'utf8')) as CompetitorMap
  const api = new Api(base, cookie)
  log(`Reading the candidates from ${base} and the map of ${map.date} (${map.companies.length} companies).`)
  const candidates = await fetchCandidates(api)
  const listed = await fetchListed(api)
  const p = plan(candidates, map, listed, o.fixDecided)
  report(p, o.apply, o.fixDecided, log)
  if (!o.apply) {
    log('')
    log('This was a dry run. Nothing was changed. Run again with --apply to act.')
    return 0
  }
  log('')
  const failed = await execute(api, p.actions, log)
  log(failed ? `Done, with ${failed} failures.` : 'Done.')
  return failed ? 1 : 0
}

if (import.meta.main) {
  try {
    process.exitCode = await run(parseArgs(process.argv.slice(2)), process.env, line => console.log(line))
  }
  catch (err) {
    console.error((err as Error).message)
    process.exitCode = 2
  }
}
