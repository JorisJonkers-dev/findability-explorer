<script setup lang="ts">
import ManageLayout from '../../shell/components/ManageLayout.vue'
import ScanPanel from './ScanPanel.vue'

/** Website scans in Manage: without a scan, the form and the recent scans; with one, what the crawl found, while it runs and once it is done. */
withDefaults(defineProps<{ scanId?: string }>(), { scanId: undefined })
</script>

<template>
  <ManageLayout
    :title="scanId ? 'What the scan found.' : 'Website scans'"
    :purpose="scanId ? 'What a scan read on a website, and the search terms its pages are built around.' : 'Scan a website, and open the recent scans.'"
    data-line="Scans, live from this app"
    :back="scanId ? { to: '/manage/scans', label: 'Back to Website scans' } : undefined"
    :results="{ to: '/ai-readiness', label: 'See AI readiness' }"
  >
    <ScanPanel :scan-id="scanId" is-admin />
  </ManageLayout>
</template>
