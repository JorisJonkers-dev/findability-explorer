<script setup lang="ts">
import { useTemplateRef } from 'vue'
import SettingsShell from '../../shell/components/SettingsShell.vue'
import { useLlmSettings } from '../composables/useLlmSettings'
import { FORBIDDEN_MESSAGE } from '../domain/problems'
import AddCredentialForm from './AddCredentialForm.vue'
import BudgetPanel from './BudgetPanel.vue'
import CredentialItem from './CredentialItem.vue'
import EventList from './EventList.vue'
import ModelChooser from './ModelChooser.vue'

const { settings, isLoading, forbidden, error } = useLlmSettings()
const keysTitle = useTemplateRef<HTMLElement>('keysTitle')

/** A removed key's row is gone, so focus goes to the list's heading rather than to nowhere. */
function removed(): void {
  keysTitle.value?.focus()
}
</script>

<template>
  <SettingsShell title="Language models">
    <template #lead>
      <p class="m-0 max-w-[62ch] text-lg text-ink-2">
        The model that matches a scanned website's products to Tribelt's belt families. API keys are checked with
        their provider, stored sealed, and never shown again.
      </p>
    </template>

    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">{{ FORBIDDEN_MESSAGE }}</p>
      <p class="m-0 text-ink-2">Ask an administrator for the admin role in auth-api, then sign in again.</p>
    </div>
    <p v-else-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the settings…</p>
    <template v-else-if="settings">
      <ModelChooser :settings="settings" />

      <section aria-labelledby="keys-title" class="flex flex-col gap-5">
        <h2 id="keys-title" ref="keysTitle" tabindex="-1" class="m-0 font-display text-3xl font-semibold tracking-[-0.01em] focus:outline-none">
          API keys
        </h2>
        <ul v-if="settings.credentials.length > 0" class="m-0 flex list-none flex-col gap-4 p-0">
          <CredentialItem v-for="credential in settings.credentials" :key="credential.id" :credential="credential" @removed="removed" />
        </ul>
        <p v-else class="m-0 text-ink-2">No keys yet. Add one below.</p>
      </section>

      <AddCredentialForm />

      <div class="grid gap-12 lg:grid-cols-2">
        <BudgetPanel :budget="settings.budget" />
        <EventList :events="settings.events" />
      </div>
    </template>
  </SettingsShell>
</template>
