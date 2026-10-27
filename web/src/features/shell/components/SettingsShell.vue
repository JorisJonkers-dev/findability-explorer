<script setup lang="ts">
import { computed, ref } from 'vue'
import { useTermCandidates } from '../../review/composables/useTermReview'
import { useMe } from '../composables/useMe'
import { useStage } from '../composables/useStage'
import type { Tab } from '../domain/sections'
import SectionView from './SectionView.vue'

/** A settings page in the same frame as every section: the top bar, the title, the settings as tabs. */
defineProps<{ title: string }>()
const { me } = useMe()
// On the stage the forms are drawn at the window's own sizes, as on a page, whatever the scale.
const { pageZoom } = useStage()
// How many terms wait in Term review: its tab shows it on every settings page.
const { counts } = useTermCandidates(ref('pending'))

const tabs = computed<Tab[]>(() => [
  { label: 'Language models', path: '/settings' },
  { label: 'Prompt test', path: '/settings/prompt-test' },
  { label: 'Term review', path: '/settings/terms', ...(counts.value ? { count: counts.value.pending } : {}) },
])
</script>

<template>
  <SectionView eyebrow="Settings" :title="title" :tabs="tabs" tabs-label="Settings">
    <template v-if="me" #aside>
      <p class="m-0 text-sm text-muted stage:text-right stage:text-[26px]">Signed in as {{ me.name }}{{ me.isAdmin ? ' (administrator)' : '' }}</p>
    </template>
    <!-- The settings are forms that run longer than the stage, so they scroll inside it. -->
    <div class="stage:min-h-0 stage:overflow-y-auto">
      <div class="mx-auto flex w-full max-w-4xl min-w-0 flex-col gap-10 text-base leading-normal stage:pb-8" :style="pageZoom ? { zoom: pageZoom } : undefined">
        <slot name="lead" />
        <slot />
      </div>
    </div>
  </SectionView>
</template>
