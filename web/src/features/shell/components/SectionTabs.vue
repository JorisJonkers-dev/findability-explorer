<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import type { Tab } from '../domain/sections'

withDefaults(defineProps<{ tabs: readonly Tab[]; label?: string }>(), { label: 'In this section' })
const route = useRoute()
</script>

<template>
  <nav :aria-label="label" class="flex flex-wrap gap-2 stage:justify-end stage:gap-3">
    <RouterLink
      v-for="tab in tabs"
      :key="tab.path"
      :to="tab.path"
      :aria-current="route.path === tab.path ? 'page' : undefined"
      class="inline-flex min-h-11 items-center gap-2 rounded-full border-2 border-solid px-4 text-[15px] no-underline stage:min-h-[72px] stage:px-8 stage:text-[28px]"
      :class="
        route.path === tab.path
          ? 'border-orange bg-orange font-semibold text-black hover:text-black'
          : 'border-silver font-medium text-ink-2 hover:border-ink-2 hover:text-black'
      "
    >
      {{ tab.label }}
      <span
        v-if="tab.count !== undefined && tab.count > 0"
        class="rounded-full bg-black px-2 text-sm font-semibold text-white stage:px-3 stage:text-[24px]"
        :aria-label="`${String(tab.count)} waiting`"
      >{{ tab.count }}</span>
    </RouterLink>
  </nav>
</template>
