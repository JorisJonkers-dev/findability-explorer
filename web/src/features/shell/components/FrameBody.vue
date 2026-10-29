<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import BackLink from '../../../shared/ui/BackLink.vue'
import SectionEyebrow from './SectionEyebrow.vue'

type Link = { to: RouteLocationRaw; label: string }

/**
 * The parts every page has, in the spec's order: the way back, the title, one sentence on what the
 * page shows, where its figures come from, the link to its other view, its sections, and its Help topic.
 */
defineProps<{
  eyebrow?: string
  title: string
  purpose: string
  dataLine?: string
  back?: Link
  /** On a Results page its Manage page, for administrators; on a Manage page its Results page. */
  link?: Link
  helpTopic?: string
}>()
</script>

<template>
  <BackLink v-if="back" :to="back.to" :label="back.label" />
  <header class="flex flex-col gap-5 md:flex-row md:items-start md:justify-between">
    <div class="flex max-w-[64rem] min-w-0 flex-col gap-3.5">
      <SectionEyebrow v-if="eyebrow">{{ eyebrow }}</SectionEyebrow>
      <h1 class="m-0 font-display text-4xl leading-[1.04] font-semibold tracking-[-0.02em] text-balance sm:text-6xl">{{ title }}</h1>
      <p data-part="purpose" class="m-0 max-w-[62ch] text-lg text-ink-2">{{ purpose }}</p>
      <p v-if="dataLine" data-part="data-line" class="m-0 self-start rounded-full bg-mist px-3.5 py-1 text-sm text-ink-2">{{ dataLine }}</p>
    </div>
    <div v-if="link || $slots.aside" class="flex flex-wrap items-center gap-3 md:justify-end">
      <slot name="aside" />
      <RouterLink
        v-if="link"
        :to="link.to"
        class="inline-flex min-h-11 items-center rounded-full border-2 border-solid border-silver px-4 font-medium text-black no-underline hover:border-black hover:text-black"
      >
        {{ link.label }}
      </RouterLink>
    </div>
  </header>
  <slot />
  <p v-if="helpTopic" class="m-0">
    <RouterLink :to="`/help/${helpTopic}`" class="font-medium">How this is measured, in Help</RouterLink>
  </p>
</template>
