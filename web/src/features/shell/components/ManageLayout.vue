<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, type RouteLocationRaw } from 'vue-router'
import { useMe } from '../composables/useMe'
import { groupOf } from '../domain/manageMenu'
import FrameBody from './FrameBody.vue'
import ManageMenu from './ManageMenu.vue'
import TopBar from './TopBar.vue'

/**
 * The frame of every Manage page: the top bar, the Manage menu down the side, and beside it the
 * same parts as a Results page, with a link to its Results page. Manage is for administrators only;
 * anyone else is told so, and nothing behind it is asked of the server.
 */
defineProps<{
  title: string
  purpose: string
  dataLine?: string
  back?: { to: RouteLocationRaw; label: string }
  results?: { to: RouteLocationRaw; label: string }
  helpTopic?: string
}>()

const route = useRoute()
const { isAdmin } = useMe()
const eyebrow = computed(() => {
  const group = groupOf(route.path)
  return group ? `Manage, ${group.label}` : 'Manage'
})
</script>

<template>
  <div class="mx-auto flex w-full max-w-[1360px] flex-col gap-10 px-4 pt-5 pb-16 sm:px-10">
    <TopBar />
    <div v-if="isAdmin" class="grid gap-10 min-[1000px]:grid-cols-[15rem_minmax(0,1fr)]">
      <ManageMenu />
      <div class="flex min-w-0 flex-col gap-10">
        <FrameBody
          :eyebrow="eyebrow"
          :title="title"
          :purpose="purpose"
          :data-line="dataLine"
          :back="back"
          :link="results"
          :help-topic="helpTopic"
        >
          <template v-if="$slots.aside" #aside><slot name="aside" /></template>
          <slot />
        </FrameBody>
      </div>
    </div>
    <section v-else aria-labelledby="manage-closed" class="flex max-w-[64rem] flex-col gap-4">
      <h1 id="manage-closed" class="m-0 font-display text-4xl leading-[1.04] font-semibold tracking-[-0.02em] sm:text-6xl">Manage</h1>
      <p class="m-0 text-lg text-ink-2">Only administrators can start, stop, and change the work behind the results.</p>
      <RouterLink to="/" data-part="to-overview" class="self-start font-medium">Go to the Overview</RouterLink>
    </section>
  </div>
</template>
