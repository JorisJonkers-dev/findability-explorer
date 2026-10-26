<script setup lang="ts">
import { RouterLink } from 'vue-router'
import SectionView from '../../shell/components/SectionView.vue'
import { useReadinessTable } from '../composables/useReadiness'
import ReadinessTable from './ReadinessTable.vue'

const { rows, isLoading, error } = useReadinessTable()
</script>

<template>
  <SectionView
    section="ai-search"
    title="Who lets search engines and AI assistants in."
    :source="['Each company\'s newest scan, live from this app', 'Held against the report\'s recommendations']"
  >
    <div class="flex flex-col gap-6 stage:grid stage:min-h-0 stage:grid-rows-[auto_minmax(0,1fr)] stage:gap-10">
      <p class="m-0 max-w-4xl text-ink-2 stage:max-w-none stage:text-[30px]">
        Every scan first reads robots.txt, the sitemaps, llms.txt, llms-full.txt and the home page. Tribelt's own site is the
        baseline; the mirror of tribelt.nl shows the recommendations in practice. A company not scanned since shows "Not checked".
        <RouterLink to="/ai-search" class="font-medium">Back to AI search</RouterLink>
      </p>
      <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
      <p v-else-if="isLoading" class="m-0 text-muted">Loading the AI readiness…</p>
      <ReadinessTable v-else :rows="rows" />
    </div>
  </SectionView>
</template>
