<script setup lang="ts">
import { useTemplateRef } from 'vue'
import { RouterLink } from 'vue-router'
import AccountBar from '../../shell/components/AccountBar.vue'
import SectionEyebrow from '../../shell/components/SectionEyebrow.vue'
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
  <div class="mx-auto flex max-w-5xl flex-col gap-12 px-4 pt-5 pb-20 sm:px-8 lg:pt-10">
    <header class="flex flex-col gap-8">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <RouterLink
          to="/"
          class="inline-flex min-h-11 items-center gap-2 rounded-full bg-black px-5 font-medium text-white no-underline hover:bg-ink-2 hover:text-white"
        >
          <svg aria-hidden="true" viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M15 5 8 12l7 7" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          Overview
        </RouterLink>
        <AccountBar tone="light" />
      </div>
      <div class="flex flex-col gap-4">
        <SectionEyebrow>Settings</SectionEyebrow>
        <h1 class="m-0 font-display text-4xl leading-[0.95] font-semibold tracking-[-0.03em] sm:text-6xl">Language model</h1>
        <p class="m-0 max-w-[62ch] text-lg text-ink-2">
          The model that matches a scanned website's products to Tribelt's belt families. API keys are checked with
          their provider, stored sealed, and never shown again.
        </p>
      </div>
    </header>

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
  </div>
</template>
