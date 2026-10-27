<script setup lang="ts">
import SectionView from '../../shell/components/SectionView.vue'
import { COMPETITOR_TABS } from '../../shell/domain/sections'
</script>

<template>
  <SectionView section="competitors" title="Every competitor." :tabs="COMPETITOR_TABS">
    <p class="m-0 text-ink-2">Every competitor appears here.</p>
  </SectionView>
</template>
