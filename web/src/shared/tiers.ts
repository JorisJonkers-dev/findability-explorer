import type { IntentTier } from '../infrastructure/api/types.gen'
import { tierLabel } from './vocabulary'

/**
 * Each tier's heat-tint, as whole Tailwind classes: Tailwind finds the classes it builds by
 * reading the source, so a class put together at run time would never be built.
 */
const DOT: Record<IntentTier, string> = {
  spec: 'bg-tier-spec',
  product: 'bg-tier-product',
  application: 'bg-tier-application',
  problem: 'bg-tier-problem',
  broad: 'bg-tier-broad',
}

const BORDER: Record<IntentTier, string> = {
  spec: 'border-tier-spec',
  product: 'border-tier-product',
  application: 'border-tier-application',
  problem: 'border-tier-problem',
  broad: 'border-tier-broad',
}

/** The background class that fills a tier's dot or bar segment. */
export function tierFill(tier: IntentTier): string {
  return DOT[tier]
}

/** The border class that outlines a tier's pressed filter. */
export function tierBorder(tier: IntentTier): string {
  return BORDER[tier]
}

/**
 * Each kind of buyer intent, how far along a buyer is when they ask: its label, and one plain
 * sentence that says what it means, for wherever buyer intent is shown.
 */
export const BUYER_INTENT: Record<IntentTier, { label: string; help: string }> = {
  spec: { label: tierLabel('spec'), help: 'The buyer already knows the exact belt and its sizes.' },
  product: { label: tierLabel('product'), help: 'The buyer names the type of belt.' },
  application: { label: tierLabel('application'), help: 'The buyer describes what the belt is for, such as a bakery oven.' },
  problem: { label: tierLabel('problem'), help: 'The buyer describes a problem to solve.' },
  broad: { label: tierLabel('broad'), help: 'A general search about conveyor belts.' },
}
