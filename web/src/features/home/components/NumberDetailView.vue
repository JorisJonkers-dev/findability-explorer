<script setup lang="ts">
import { computed } from 'vue'
import SectionView from '../../shell/components/SectionView.vue'

/** Placeholder: the detail page of one number on the overview. */
const props = defineProps<{ measure: string }>()
const title = computed(() => props.measure.replaceAll('-', ' '))
</script>

<template>
  <SectionView section="overview" :title="title" :back="{ to: '/', label: 'Back to Overview' }">
    <p class="m-0 text-ink-2">This number's page is coming.</p>
  </SectionView>
</template>
