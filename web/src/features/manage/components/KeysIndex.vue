<script setup lang="ts">
import { computed } from 'vue'
import BudgetMeter from '../../settings/components/BudgetMeter.vue'
import { useLlmSettings } from '../../settings/composables/useLlmSettings'
import { KEY_PAGES, type KeyProvider, keysOf, providerStatus } from '../../settings/domain/keys'
import { FORBIDDEN_MESSAGE } from '../../settings/domain/problems'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import IndexCard from './widgets/IndexCard.vue'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/** Keys and spending: this month's tokens, a card per key, the budget, and the events. */
const { settings, isLoading, forbidden, error } = useLlmSettings()
const credentials = computed(() => settings.value?.credentials ?? [])
// Mistral has a card only while a Mistral key is stored; the menu names the other two.
const providers = computed<KeyProvider[]>(() => ['openrouter', 'anthropic', ...(keysOf(credentials.value, 'mistral').length > 0 ? (['mistral'] as const) : [])])
const fingerprint = (p: KeyProvider) => keysOf(credentials.value, p)[0]?.fingerprint
</script>

<template>
  <ManageLayout title="Keys and spending" purpose="The keys that pay for the work, and the monthly spending limit.">
    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">{{ FORBIDDEN_MESSAGE }}</p>
      <p class="m-0 text-ink-2">Ask an administrator for the admin role in auth-api, then sign in again.</p>
    </div>
    <p v-else-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the settings…</p>
    <WidgetGrid v-else-if="settings">
      <ManageWidget title="This month's tokens" :span="12">
        <BudgetMeter :budget="settings.budget" />
      </ManageWidget>
      <IndexCard
        v-for="p in providers"
        :key="p"
        :title="KEY_PAGES[p].title"
        :text="`${providerStatus(credentials, p)}. ${KEY_PAGES[p].pays}`"
        :to="`/manage/keys/${p}`"
        :link="fingerprint(p) ? 'Open the key' : 'Add the key'"
        :figures="fingerprint(p) ? [{ value: fingerprint(p) ?? '', label: 'fingerprint' }] : []"
        :span="3"
      />
      <IndexCard
        title="Budgets"
        text="Every job that asks a language model counts against one monthly budget in tokens. Work stops when it runs out."
        to="/manage/keys/budgets"
        link="See the budget"
        :span="3"
      />
      <IndexCard
        title="Events"
        text="Key changes and tests, newest first."
        to="/manage/keys/events"
        link="See the events"
        :figures="[{ value: String(settings.events.length), label: 'events' }]"
        :span="3"
      />
    </WidgetGrid>
  </ManageLayout>
</template>
