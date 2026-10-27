import type { Product, ProductStandout } from '../../../infrastructure/api/types.gen'
import { familyLabel } from '../../../shared/vocabulary'
import { inWords } from '../../catching-up/domain/standings'

/** The share of a family's answers naming Tribelt, or undefined before any run asked about it. */
export function aiShare(p: Pick<Product, 'answers' | 'named'>): number | undefined {
  return p.answers === 0 ? undefined : p.named / p.answers
}

/** A share as the cards write it: "30%", or "–" without answers. */
export function shareText(share: number | undefined): string {
  return share === undefined ? '–' : `${String(Math.round(share * 100))}%`
}

/** The most used term of a family, or undefined. */
export function topTerm(p: Pick<Product, 'terms'>): string | undefined {
  return p.terms[0]?.term
}

/** The languages a family's terms are in, Dutch, German and English first. */
export function languagesOf(p: Pick<Product, 'terms'>): string[] {
  const order = ['nl', 'de', 'en']
  const all = [...new Set(p.terms.map((t) => (t.language === 'en-US' ? 'en' : t.language)))]
  return all.sort((a, b) => (order.includes(a) ? order.indexOf(a) : 9) - (order.includes(b) ? order.indexOf(b) : 9) || a.localeCompare(b))
}

/** A family page's headline, from who sells it and the languages buyers search in. */
export function productHeadline(p: Pick<Product, 'family' | 'sellers' | 'terms'>): string {
  const sellers = p.sellers.length === 1 ? 'one competitor sells it' : `${inWords(p.sellers.length)} competitors sell it`
  const languages = languagesOf(p).length
  const search = languages === 0 ? '' : `, and buyers search for it in ${inWords(languages)} ${languages === 1 ? 'language' : 'languages'}`
  return `${familyLabel(p.family)}. ${sellers.charAt(0).toUpperCase()}${sellers.slice(1)}${search}.`
}

/** What a standout says: its lead in bold, then the figures behind it. */
export function standoutText(s: ProductStandout, products: readonly Product[]): { lead: string; rest: string } {
  const p = products.find((x) => x.family === s.family)
  const name = familyLabel(s.family)
  const terms = p ? `Tribelt uses ${p.termsUsed === 0 ? 'none' : String(p.termsUsed)} of its ${String(p.terms.length)} search terms.` : ''
  const sellers = p?.sellers.length ?? 0
  switch (s.finding) {
    case 'crowded':
      return { lead: `Most competitors sell ${name}.`, rest: `${String(sellers)} ${sellers === 1 ? 'competitor sells' : 'competitors sell'} it. ${terms}`.trim() }
    case 'fewestTerms':
      return { lead: `Tribelt uses the fewest search terms for ${name}.`, rest: terms }
    case 'bestInAI':
      return { lead: `AI answers name Tribelt most for ${name}.`, rest: `${shareText(p ? aiShare(p) : undefined)} of the answers about it name Tribelt.` }
  }
}
