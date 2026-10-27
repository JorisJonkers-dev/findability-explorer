<script setup lang="ts">
import SectionView from '../../shell/components/SectionView.vue'
import { PRODUCT_TABS } from '../../shell/domain/sections'
</script>

<template>
  <SectionView section="products" title="Tribelt's belt families against the field." :tabs="PRODUCT_TABS">
    <p class="m-0 text-ink-2">One card per belt family appears here.</p>
  </SectionView>
</template>
