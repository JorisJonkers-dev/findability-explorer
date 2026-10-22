import type { IntentTier } from '../infrastructure/api/types.gen'

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
