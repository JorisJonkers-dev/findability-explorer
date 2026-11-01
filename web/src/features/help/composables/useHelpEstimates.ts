import { useQuery } from '@tanstack/vue-query'
import { computed, type Ref } from 'vue'
import { getPromptTestSettingsOptions } from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { Estimates } from '../domain/topics'

/**
 * Server state for the figures Help's topics say: what a Prompt run costs, and how often it runs.
 * Only administrators may read the Prompt test's settings, so a reader's topics leave them out.
 */
export function useHelpEstimates(isAdmin: Ref<boolean>) {
  const query = useQuery(computed(() => ({ ...getPromptTestSettingsOptions(), enabled: isAdmin.value })))
  const estimates = computed<Estimates | undefined>(() => {
    const s = query.data.value
    return s ? { runUsd: s.prices.runUsd, cadence: s.cadence } : undefined
  })
  return { estimates }
}
