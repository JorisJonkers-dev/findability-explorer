<script setup lang="ts">
import SectionView from '../../shell/components/SectionView.vue'

/** Placeholder: the detail page of one recommendation. */
defineProps<{ id: string }>()
</script>

<template>
  <SectionView section="recommendations" title="Recommendation" :back="{ to: '/recommendations', label: 'Back to Recommendations' }">
    <p class="m-0 text-ink-2">This recommendation's page is coming.</p>
  </SectionView>
</template>
