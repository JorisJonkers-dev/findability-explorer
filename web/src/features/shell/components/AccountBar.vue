<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useMe } from '../composables/useMe'
import { LOGOUT_ACTION } from '../domain/auth'

defineProps<{ tone: 'light' | 'dark' }>()
const { me } = useMe()
</script>

<template>
  <nav
    v-if="me"
    aria-label="Account"
    class="flex flex-wrap items-center gap-x-4 gap-y-1 text-sm stage:gap-x-10 stage:text-[26px]"
    :class="tone === 'dark' ? 'text-silver' : 'text-muted'"
  >
    <span>Signed in as <span class="font-medium" :class="tone === 'dark' ? 'text-white' : 'text-black'">{{ me.name }}</span></span>
    <RouterLink
      v-if="me.isAdmin"
      to="/settings"
      class="font-medium"
      :class="tone === 'dark' ? 'text-white hover:text-silver' : ''"
    >
      Settings
    </RouterLink>
    <form method="post" :action="LOGOUT_ACTION" class="m-0 inline">
      <button
        type="submit"
        class="cursor-pointer border-0 bg-transparent p-0 font-medium underline underline-offset-4"
        :class="tone === 'dark' ? 'text-white hover:text-silver' : 'text-orange-ink hover:text-black'"
      >
        Sign out
      </button>
    </form>
  </nav>
</template>
