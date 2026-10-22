import { computed } from 'vue'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import { creditedCompanies } from '../domain/matrix'
import { type Company, withShortNames } from '../domain/company'
import { useCompetitors } from './useCompetitors'

/**
 * Every known company with its short name, the one the terminology map credits: the list and the
 * whole map are two cached requests that every page shares.
 */
export function useCompanies() {
  const { competitors, isLoading, error } = useCompetitors()
  const map = useTermMap()
  const companies = computed<Company[]>(() => withShortNames(competitors.value, creditedCompanies(map.terms.value)))
  return {
    companies,
    isLoading,
    error,
    /** The company with this id, once the list has loaded. */
    byId: (id: string | undefined): Company | undefined => companies.value.find((c) => c.id === id),
  }
}
