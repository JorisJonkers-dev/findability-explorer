<script setup lang="ts">
import { computed } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { useMe } from '../composables/useMe'
import { section as sectionById, type SectionId } from '../domain/sections'
import FrameBody from './FrameBody.vue'
import TopBar from './TopBar.vue'

/**
 * The frame of every Results page: the top bar, then the way back, the title, one sentence on what
 * the page shows, where its figures come from, for administrators the link to its Manage page, its
 * sections, and its Help topic.
 */
const props = defineProps<{
  title: string
  purpose: string
  dataLine?: string
  back?: { to: RouteLocationRaw; label: string }
  manage?: { to: RouteLocationRaw; label: string }
  helpTopic?: string
  section?: SectionId
}>()

const { isAdmin } = useMe()
const eyebrow = computed(() => {
  if (!props.section) return undefined
  const s = sectionById(props.section)
  return `${s.number} ${s.label}`
})
</script>

<template>
  <div class="mx-auto flex w-full max-w-[1360px] flex-col gap-10 px-4 pt-5 pb-16 sm:px-10">
    <TopBar />
    <FrameBody
      :eyebrow="eyebrow"
      :title="title"
      :purpose="purpose"
      :data-line="dataLine"
      :back="back"
      :link="isAdmin ? manage : undefined"
      :help-topic="helpTopic"
    >
      <template v-if="$slots.aside" #aside><slot name="aside" /></template>
      <slot />
    </FrameBody>
  </div>
</template>
