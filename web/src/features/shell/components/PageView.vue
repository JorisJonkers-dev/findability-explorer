<script setup lang="ts">
import { computed } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import BackLink from '../../../shared/ui/BackLink.vue'
import { section as sectionById, type SectionId } from '../domain/sections'
import SectionEyebrow from './SectionEyebrow.vue'
import TopBar from './TopBar.vue'

const props = defineProps<{
  section: SectionId
  /** Names the eyebrow instead of the section, such as "Number to watch". */
  eyebrow?: string
  /** A detail page's way back, such as "Back to Overview". */
  back?: { to: RouteLocationRaw; label: string }
  title: string
}>()

const eyebrowText = computed(() => {
  if (props.eyebrow) return props.eyebrow
  const meta = sectionById(props.section)
  return `${meta.number} ${meta.label}`
})
</script>

<template>
  <div class="mx-auto flex w-full max-w-[1360px] flex-col gap-10 px-4 pt-5 pb-16 sm:px-10">
    <TopBar />
    <BackLink v-if="back" :to="back.to" :label="back.label" />
    <header class="flex max-w-[64rem] flex-col gap-3.5">
      <SectionEyebrow>{{ eyebrowText }}</SectionEyebrow>
      <h1 class="m-0 font-display text-4xl leading-[1.04] font-semibold tracking-[-0.02em] text-balance sm:text-6xl">{{ title }}</h1>
      <slot name="lede" />
    </header>
    <slot />
  </div>
</template>
