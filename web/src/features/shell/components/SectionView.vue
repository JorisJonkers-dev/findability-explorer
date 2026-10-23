<script setup lang="ts">
import { computed } from 'vue'
import { usePreviousPath } from '../composables/navigation'
import { section as sectionById, type SectionId } from '../domain/sections'
import AccountBar from './AccountBar.vue'
import BackButton from './BackButton.vue'
import SectionEyebrow from './SectionEyebrow.vue'
import StageFrame from './StageFrame.vue'

const props = defineProps<{
  section: SectionId
  title: string
  /** Where the figures come from, a line each. */
  source: readonly string[]
}>()

const meta = computed(() => sectionById(props.section))
// Opened from the poster, the view settles in as the poster's zoom fades out; loaded directly, it just shows.
const fromPoster = usePreviousPath().value === '/'
</script>

<template>
  <StageFrame>
    <article
      class="flex flex-col gap-10 px-4 pt-5 pb-14 sm:px-8 stage:grid stage:h-full stage:grid-rows-[auto_minmax(0,1fr)] stage:gap-18 stage:p-28 stage:text-[34px] stage:leading-[1.4]"
      :class="{ 'detail-in': fromPoster }"
      :aria-labelledby="`${section}-title`"
    >
      <header class="flex flex-col gap-5 stage:grid stage:grid-cols-[auto_minmax(0,1fr)_auto] stage:items-end stage:gap-18">
        <BackButton />
        <div class="flex flex-col gap-3 stage:gap-4">
          <SectionEyebrow>{{ meta.number }} {{ meta.label }}</SectionEyebrow>
          <h1
            :id="`${section}-title`"
            class="m-0 font-display text-4xl leading-[0.95] font-semibold tracking-[-0.03em] text-balance sm:text-5xl stage:text-[128px]"
          >
            {{ title }}
          </h1>
        </div>
        <div class="flex flex-col gap-3 stage:items-end stage:gap-8 stage:self-stretch stage:justify-between">
          <AccountBar tone="light" />
          <p class="m-0 text-sm text-muted stage:text-right stage:text-[26px]">
            <template v-for="(line, index) in source" :key="line"><br v-if="index > 0" />{{ line }}</template>
          </p>
        </div>
      </header>
      <slot />
    </article>
  </StageFrame>
</template>
