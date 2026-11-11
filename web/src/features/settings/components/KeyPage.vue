<script setup lang="ts">
import { computed, nextTick } from 'vue'
import { RouterLink } from 'vue-router'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useLlmSettings } from '../composables/useLlmSettings'
import { keyPage, keysOf } from '../domain/keys'
import { FORBIDDEN_MESSAGE } from '../domain/problems'
import AddCredentialForm from './AddCredentialForm.vue'
import BudgetMeter from './BudgetMeter.vue'
import CredentialItem from './CredentialItem.vue'

/**
 * One provider's key: the key (its name and fingerprint), Test the key, and Replace the key; what it
 * pays for; and, on the Anthropic page, this month's tokens and the optional Admin key.
 */
const props = defineProps<{ provider: string }>()
const page = computed(() => keyPage(props.provider))
const { settings, isLoading, forbidden, error } = useLlmSettings()
const keys = computed(() => (page.value && settings.value ? keysOf(settings.value.credentials, page.value.provider) : []))
const adminKeys = computed(() => (settings.value ? keysOf(settings.value.credentials, 'anthropic_admin') : []))
const headingId = computed(() => `keys-${props.provider}`)

/** A removed key's row is gone, so focus goes to the widget's heading rather than to nowhere. */
async function removed(): Promise<void> {
  await nextTick()
  document.getElementById(headingId.value)?.focus()
}
</script>

<template>
  <ManageLayout
    :title="page?.title ?? 'Key'"
    :purpose="page?.purpose ?? ''"
    :back="{ to: '/manage/keys', label: 'Back to Keys and spending' }"
  >
    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">{{ FORBIDDEN_MESSAGE }}</p>
      <p class="m-0 text-ink-2">Ask an administrator for the admin role in auth-api, then sign in again.</p>
    </div>
    <p v-else-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the keys…</p>
    <WidgetGrid v-else-if="page && settings">
      <ManageWidget :title="keys.length > 1 ? 'The keys' : 'The key'" :span="8" :heading-id="headingId">
        <p class="m-0 text-ink-2">API keys are checked with their provider, stored sealed, and never shown again.</p>
        <ul v-if="keys.length > 0" :aria-label="page.title" class="m-0 flex list-none flex-col gap-4 p-0">
          <CredentialItem v-for="k in keys" :key="k.id" :credential="k" @removed="removed" />
        </ul>
        <AddCredentialForm v-else :fixed="page.provider" :title="`Add the ${page.title}`" />
      </ManageWidget>

      <ManageWidget title="What it pays for" :span="4">
        <p class="m-0 text-ink-2">{{ page.pays }}</p>
      </ManageWidget>

      <ManageWidget v-if="page.provider === 'anthropic'" title="Spent this month" :span="6">
        <BudgetMeter :budget="settings.budget" />
        <template #foot>
          <RouterLink to="/manage/keys/budgets" class="font-medium">Change the budget</RouterLink>
        </template>
      </ManageWidget>

      <ManageWidget v-if="page.provider === 'anthropic'" title="Admin key for usage reports" :span="6">
        <p class="m-0 text-ink-2">An optional Anthropic Admin key reads Anthropic's own usage reports. It pays for nothing.</p>
        <ul v-if="adminKeys.length > 0" aria-label="Admin keys" class="m-0 flex list-none flex-col gap-4 p-0">
          <CredentialItem v-for="k in adminKeys" :key="k.id" :credential="k" />
        </ul>
        <AddCredentialForm v-else fixed="anthropic_admin" title="Add the Admin key" />
      </ManageWidget>
    </WidgetGrid>
    <p v-else class="m-0 rounded-2xl bg-mist p-6 text-lg">There is no key page for this provider.</p>
  </ManageLayout>
</template>
