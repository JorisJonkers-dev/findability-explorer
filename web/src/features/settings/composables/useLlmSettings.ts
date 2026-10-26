import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, type Ref, ref } from 'vue'
import {
  addLlmCredentialMutation,
  clearActiveLlmMutation,
  deleteLlmCredentialMutation,
  getLlmKeyUsageOptions,
  getLlmSettingsOptions,
  getLlmSettingsQueryKey,
  listLlmModelsOptions,
  setActiveLlmMutation,
  testLlmCredentialMutation,
} from '../../../infrastructure/api/@tanstack/vue-query.gen'
import type { ActiveLlm, LlmKeyUsage, LlmModel, LlmSettings, NewLlmCredential } from '../../../infrastructure/api/types.gen'
import { testOutcome } from '../domain/credentials'
import { isForbidden, type SettingsAction, settingsProblem } from '../domain/problems'

/** Server state for the language-model settings; a 403 means the account is no administrator. */
export function useLlmSettings() {
  const query = useQuery(getLlmSettingsOptions())
  return {
    settings: computed<LlmSettings | undefined>(() => query.data.value),
    isLoading: query.isPending,
    forbidden: computed(() => isForbidden(query.error.value)),
    error: computed(() => (query.error.value ? settingsProblem(query.error.value, 'load') : null)),
  }
}

/** Asks for the settings again, after a change that does not answer with them. */
function useRefresh(): () => void {
  const queryClient = useQueryClient()
  return () => {
    void queryClient.invalidateQueries({ queryKey: getLlmSettingsQueryKey() })
  }
}

/** The models a key may use, asked of its provider once a key is picked (id not empty). */
export function useLlmModels(credentialId: Ref<string>) {
  const query = useQuery(
    computed(() => ({
      ...listLlmModelsOptions({ path: { credentialId: credentialId.value } }),
      enabled: credentialId.value !== '',
      // A provider's model list changes rarely; picking the key again need not ask again.
      staleTime: 5 * 60_000,
    })),
  )
  return {
    models: computed<LlmModel[]>(() => (query.data.value ? query.data.value.items : [])),
    isLoading: computed(() => query.isFetching.value && query.data.value === undefined),
    error: computed(() => (query.error.value ? settingsProblem(query.error.value, 'models') : null)),
  }
}

/**
 * What the provider reports a key used, asked only once shown is true: each look asks the
 * provider, so the figures are current but never fetched for nothing.
 */
export function useKeyUsage(credentialId: Ref<string>, shown: Ref<boolean>) {
  const query = useQuery(
    computed(() => ({
      ...getLlmKeyUsageOptions({ path: { credentialId: credentialId.value } }),
      enabled: shown.value,
      staleTime: 60_000,
    })),
  )
  return {
    usage: computed<LlmKeyUsage | undefined>(() => query.data.value),
    isLoading: computed(() => query.isFetching.value && query.data.value === undefined),
    error: computed(() => (query.error.value ? settingsProblem(query.error.value, 'usage') : null)),
  }
}

/** Adds a key; the provider checks it first, and a refused key is not stored (422). */
export function useAddCredential() {
  const refresh = useRefresh()
  const add = useMutation(addLlmCredentialMutation())
  return {
    isAdding: add.isPending,
    error: computed(() => (add.error.value ? settingsProblem(add.error.value, 'add') : null)),
    add: (body: NewLlmCredential, onAdded: () => void) => {
      add.mutate(
        { body },
        {
          onSuccess: () => {
            refresh()
            onAdded()
          },
        },
      )
    },
  }
}

/** Tests one key with its provider; the outcome is also recorded on the key and in the events. */
export function useCredentialTest() {
  const refresh = useRefresh()
  const test = useMutation(testLlmCredentialMutation())
  return {
    isTesting: test.isPending,
    outcome: computed(() => (test.data.value ? testOutcome(test.data.value) : null)),
    error: computed(() => (test.error.value ? settingsProblem(test.error.value, 'test') : null)),
    test: (credentialId: string) => {
      test.mutate({ path: { credentialId } }, { onSettled: refresh })
    },
  }
}

/** Removes one stored key. */
export function useRemoveCredential() {
  const refresh = useRefresh()
  const remove = useMutation(deleteLlmCredentialMutation())
  return {
    isRemoving: remove.isPending,
    error: computed(() => (remove.error.value ? settingsProblem(remove.error.value, 'remove') : null)),
    remove: (credentialId: string, onRemoved: () => void) => {
      remove.mutate(
        { path: { credentialId } },
        {
          onSuccess: () => {
            refresh()
            onRemoved()
          },
        },
      )
    },
  }
}

/** Chooses the key and model product matching uses, or turns matching off. Both answer with the settings. */
export function useActiveModel() {
  const queryClient = useQueryClient()
  const set = useMutation(setActiveLlmMutation())
  const clear = useMutation(clearActiveLlmMutation())
  const last = ref<Extract<SettingsAction, 'save' | 'turnOff'> | null>(null)
  const notice = ref<string | null>(null)

  function store(settings: LlmSettings, message: string): void {
    queryClient.setQueryData(getLlmSettingsQueryKey(), settings)
    notice.value = message
  }

  return {
    isSaving: computed(() => set.isPending.value || clear.isPending.value),
    /** What the last change did, for a live region. */
    notice,
    error: computed(() => {
      const action = last.value
      const error = action === 'turnOff' ? clear.error.value : set.error.value
      return action !== null && error ? settingsProblem(error, action) : null
    }),
    save: (body: ActiveLlm) => {
      last.value = 'save'
      notice.value = null
      set.mutate({ body }, { onSuccess: (settings) => { store(settings, `Product matching now uses ${body.model}.`) } })
    },
    turnOff: () => {
      last.value = 'turnOff'
      notice.value = null
      clear.mutate({}, { onSuccess: (settings) => { store(settings, 'Product matching is off.') } })
    },
  }
}
