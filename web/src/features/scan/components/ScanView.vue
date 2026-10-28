<script setup lang="ts">
import SectionView from '../../shell/components/SectionView.vue'
import { useMe } from '../../shell/composables/useMe'
import ScanPanel from './ScanPanel.vue'

/** One scan: what the crawl found on a website, while it runs and once it is done. */
defineProps<{ scanId: string }>()

const { isAdmin } = useMe()
</script>

<template>
  <SectionView section="competitors" title="What the scan found." :back="{ to: '/competitors', label: 'Back to Competitors' }" :source="['Scans: live, from this app']">
    <ScanPanel :scan-id="scanId" :is-admin="isAdmin" />
  </SectionView>
</template>
