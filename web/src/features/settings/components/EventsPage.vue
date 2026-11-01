<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import BigFigure from '../../manage/components/widgets/BigFigure.vue'
import DataTable from '../../manage/components/widgets/DataTable.vue'
import type { Column } from '../../manage/domain/table'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { SECONDARY } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useLlmSettings } from '../composables/useLlmSettings'
import { type EventLine, eventLines } from '../domain/events'

/** The log of key changes and tests, newest first. */
const { settings, isLoading, error } = useLlmSettings()
const lines = computed(() => eventLines(settings.value?.events ?? []))
const COLUMNS: readonly Column<EventLine>[] = [
  { key: 'when', label: 'When', value: (l) => l.key },
  { key: 'text', label: 'What happened', value: (l) => l.text },
  { key: 'actor', label: 'By', value: (l) => l.actor },
  { key: 'ok', label: 'Result', value: (l) => (l.ok === null ? '' : l.ok ? 'Accepted' : 'Refused') },
]
</script>

<template>
  <ManageLayout title="Events" purpose="Key changes and tests, newest first." :back="{ to: '/manage/keys', label: 'Back to Keys and spending' }">
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the events…</p>
    <WidgetGrid v-else>
      <ManageWidget :title="`The last ${String(lines.length)} events`" :span="8">
        <DataTable :rows="lines" :columns="COLUMNS" :row-key="(l) => l.key" label="Events" :search-text="(l) => `${l.text} ${l.actor}`" empty="Nothing has changed yet.">
          <template #cell-when="{ row }"><span class="whitespace-nowrap">{{ row.when }}</span></template>
        </DataTable>
      </ManageWidget>
      <ManageWidget title="Events so far" :span="4">
        <BigFigure :value="String(lines.length)" label="events" />
        <p class="m-0 text-ink-2">Every key change and key test is written here.</p>
        <template #foot>
          <RouterLink to="/manage/keys" :class="SECONDARY">Open Keys and spending</RouterLink>
        </template>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
