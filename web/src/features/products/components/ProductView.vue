<script setup lang="ts">
import { computed } from 'vue'
import { familyLabel } from '../../../shared/vocabulary'
import SectionView from '../../shell/components/SectionView.vue'
import { familyOfSlug, PRODUCT_TABS } from '../../shell/domain/sections'

const props = defineProps<{ slug: string }>()
const family = computed(() => familyOfSlug(props.slug))
</script>

<template>
  <SectionView section="products" :title="family ? familyLabel(family) : 'No such belt family.'" :tabs="PRODUCT_TABS">
    <p class="m-0 text-ink-2">The family's search terms, sellers and AI answers appear here.</p>
  </SectionView>
</template>
