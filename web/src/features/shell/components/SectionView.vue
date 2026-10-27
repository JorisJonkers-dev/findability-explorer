<script setup lang="ts">
import { computed } from 'vue'
import { usePreviousPath } from '../composables/navigation'
import { section as sectionById, type SectionId, type Tab } from '../domain/sections'
import SectionEyebrow from './SectionEyebrow.vue'
import SectionTabs from './SectionTabs.vue'
import StageFrame from './StageFrame.vue'
import TopBar from './TopBar.vue'

const props = withDefaults(
  defineProps<{
    section: SectionId
    title: string
    /** Where the figures come from, a line each. */
    source?: readonly string[]
    /** The section's pages, shown as tabs beside the title. */
    tabs?: readonly Tab[]
  }>(),
  { source: () => [], tabs: () => [] },
)

const meta = computed(() => sectionById(props.section))
const eyebrow = computed(() => [meta.value.number, meta.value.label].filter(Boolean).join(' '))
// Opened from the poster, the view settles in as the poster's zoom fades out; loaded directly, it just shows.
const fromPoster = usePreviousPath().value === '/'
</script>

<template>
  <StageFrame>
    <div
      class="flex flex-col gap-8 px-4 pt-3 pb-14 sm:px-8 stage:grid stage:h-full stage:grid-rows-[auto_auto_minmax(0,1fr)] stage:gap-14 stage:px-28 stage:pt-16 stage:pb-28 stage:text-[34px] stage:leading-[1.4]"
      :class="{ 'detail-in': fromPoster }"
    >
      <TopBar />
      <header class="flex flex-col gap-5 stage:grid stage:grid-cols-[minmax(0,1fr)_auto] stage:items-end stage:gap-18">
        <div class="flex flex-col gap-3 stage:gap-4">
          <SectionEyebrow>{{ eyebrow }}</SectionEyebrow>
          <h1
            :id="`${section}-title`"
            class="m-0 font-display text-4xl leading-[0.95] font-semibold tracking-[-0.03em] text-balance sm:text-5xl stage:text-[112px]"
          >
            {{ title }}
          </h1>
        </div>
        <div
          v-if="tabs.length > 0 || source.length > 0 || $slots.aside"
          class="flex flex-col gap-3 stage:items-end stage:gap-5 stage:pb-3"
        >
          <SectionTabs v-if="tabs.length > 0" :tabs="tabs" />
          <slot name="aside" />
          <p v-if="source.length > 0" class="m-0 text-sm text-muted stage:text-right stage:text-[26px]">
            <template v-for="(line, index) in source" :key="line"><br v-if="index > 0" />{{ line }}</template>
          </p>
        </div>
      </header>
      <!-- One element fills the stage below the header; it lays out its own columns. -->
      <article class="flex flex-col gap-8 stage:grid stage:min-h-0 stage:grid-rows-[minmax(0,1fr)]" :aria-labelledby="`${section}-title`">
        <slot />
      </article>
    </div>
  </StageFrame>
</template>
