<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useCompanies } from '../../competitors/composables/useCompanies'
import type { Company } from '../../competitors/domain/company'
import BigFigure from '../../manage/components/widgets/BigFigure.vue'
import DataTable, { type Column } from '../../manage/components/widgets/DataTable.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { SECONDARY } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { newestScans, pagesRead, scanDay, scanStamp } from '../domain/queue'

/** Each company's newest scan, newest first, opening its scan page. */
const { companies, isLoading, error } = useCompanies()
type Row = ReturnType<typeof newestScans<Company>>[number]
const rows = computed<Row[]>(() => newestScans(companies.value))
const last = computed(() => rows.value[0])
const COLUMNS: readonly Column<Row>[] = [
  { key: 'company', label: 'Company', value: (r) => r.company.name },
  { key: 'number', label: 'Scan number', value: (r) => r.scan.number ?? 1, num: true },
  { key: 'date', label: 'Date', value: (r) => r.scan.createdAt },
  { key: 'pages', label: 'Pages read', value: (r) => r.scan.pagesCrawled, num: true },
]
</script>

<template>
  <ManageLayout
    title="Recent scans"
    purpose="Each company's newest scan."
    :back="{ to: '/manage/scans', label: 'Back to Website scans' }"
    :results="{ to: '/ai-readiness', label: 'See AI readiness' }"
  >
    <WidgetGrid>
      <ManageWidget title="Scans" :span="8">
        <p class="m-0 text-ink-2">Newest first. Open one to see its details.</p>
        <p v-if="error" role="alert" class="m-0 font-medium text-orange-ink">{{ error }}</p>
        <p v-else-if="isLoading" class="m-0 text-muted">Loading the scans…</p>
        <DataTable v-else :rows="rows" :columns="COLUMNS" :row-key="(r) => r.scan.id" label="Scans" :search-text="(r) => r.company.name" empty="No company has been scanned yet.">
          <template #cell-company="{ row }">
            <RouterLink :to="`/manage/scans/${row.scan.id}`" class="font-semibold">{{ row.company.name }}</RouterLink>
          </template>
          <template #cell-date="{ row }">{{ scanStamp(row.scan.createdAt) }}</template>
          <template #cell-pages="{ row }">{{ pagesRead(row.scan) }}</template>
        </DataTable>
      </ManageWidget>
      <ManageWidget title="At a glance" :span="4">
        <BigFigure
          :value="String(rows.length)"
          label="companies scanned"
          :note="last ? `The newest scan is of ${last.company.name}, on ${scanDay(last.scan.createdAt)}.` : undefined"
        />
        <template #foot>
          <RouterLink to="/manage/scans" :class="SECONDARY">Start a scan</RouterLink>
        </template>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
