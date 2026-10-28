<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useMe } from '../composables/useMe'
import { LOGOUT_ACTION } from '../domain/auth'
import { NAV_SECTIONS, sectionOfPath } from '../domain/sections'

const route = useRoute()
const { isAdmin } = useMe()
const current = computed(() => sectionOfPath(route.path)?.id)
const inSettings = computed(() => route.path === '/settings' || route.path.startsWith('/settings/'))

const LINK =
  'inline-flex min-h-11 items-center rounded-full px-3 text-[15px] font-medium no-underline stage:min-h-[72px] stage:px-7 stage:text-[30px]'
</script>

<template>
  <nav
    aria-label="Main"
    class="flex flex-wrap items-center gap-x-1 gap-y-1 border-b-2 border-grey pb-3 stage:gap-1.5 stage:pb-7"
  >
    <RouterLink
      to="/"
      class="mr-3 inline-flex min-h-11 items-center gap-2.5 font-display text-lg font-semibold whitespace-nowrap text-black no-underline hover:text-black stage:mr-8 stage:gap-4 stage:text-[34px]"
    >
      <span aria-hidden="true" class="size-6 rounded-md bg-orange stage:size-10 stage:rounded-[10px]"></span>
      Findability Explorer
    </RouterLink>
    <RouterLink
      v-for="s in NAV_SECTIONS"
      :key="s.id"
      :to="s.path"
      :aria-current="current === s.id ? 'page' : undefined"
      :class="[LINK, current === s.id ? 'bg-black font-semibold text-white hover:text-white' : 'text-ink-2 hover:text-black']"
    >
      {{ s.label }}
    </RouterLink>
    <span class="ml-auto inline-flex flex-wrap items-center gap-1 stage:gap-3">
      <RouterLink
        v-if="isAdmin"
        to="/settings"
        :aria-current="inSettings ? 'page' : undefined"
        :class="[LINK, inSettings ? 'border-black bg-black text-white hover:text-white' : 'border-silver text-black hover:text-black']"
        class="gap-2 border-2 border-solid stage:gap-3"
      >
        <svg aria-hidden="true" viewBox="0 0 24 24" class="size-4 stage:size-[30px]" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3" />
          <path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M4.9 19.1 7 17M17 7l2.1-2.1" stroke-linecap="round" />
        </svg>
        Settings
      </RouterLink>
      <form method="post" :action="LOGOUT_ACTION" class="m-0 inline">
        <button type="submit" :class="LINK" class="border-0 bg-transparent font-sans text-ink-2 underline underline-offset-4 hover:text-black">
          Sign out
        </button>
      </form>
    </span>
  </nav>
</template>
