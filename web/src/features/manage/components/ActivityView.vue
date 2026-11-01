<script setup lang="ts">
import { RouterLink } from 'vue-router'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { MANAGE_MENU } from '../../shell/domain/manageMenu'

/** The start of Manage, until Activity shows what runs and waits: a card per section, with what it holds. */
const sections = MANAGE_MENU.slice(1)
const activity = MANAGE_MENU[0]
</script>

<template>
  <ManageLayout title="Activity" :purpose="activity?.summary ?? ''">
    <ul aria-label="The sections of Manage" class="m-0 grid list-none gap-4 p-0 sm:grid-cols-2 xl:grid-cols-3">
      <li v-for="s in sections" :key="s.to" class="flex">
        <RouterLink :to="s.to" class="flex w-full flex-col gap-2 rounded-3xl bg-mist p-6 text-black no-underline hover:bg-grey hover:text-black">
          <span class="font-display text-xl font-semibold">{{ s.label }}</span>
          <span class="text-ink-2">{{ s.summary }}</span>
        </RouterLink>
      </li>
    </ul>
  </ManageLayout>
</template>
