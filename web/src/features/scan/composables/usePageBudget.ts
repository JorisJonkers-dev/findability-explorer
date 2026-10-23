import { computed, ref } from 'vue'
import { clampPages, SCAN_PAGES } from '../domain/newScan'

// Client state: the page budget an administrator set, shared by the scan form and the re-scans
// for the rest of the visit.
const budget = ref(SCAN_PAGES)

/** The page budget scans and re-scans read, kept within the contract's limits. */
export function usePageBudget() {
  return {
    pages: computed({
      get: () => budget.value,
      set: (value: number) => {
        budget.value = clampPages(value)
      },
    }),
    reset: () => {
      budget.value = SCAN_PAGES
    },
  }
}
