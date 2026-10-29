<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { useManageCounts } from '../composables/useManageCounts'
import { badgePlacement, isOpen, MANAGE_MENU, type MenuGroup } from '../domain/manageMenu'

/**
 * The Manage menu down the side: a section per process, the open one folded out to its subpages,
 * and a section's count on the section while folded and on its subpage while open.
 */
const route = useRoute()
const { counts } = useManageCounts()

function count(group: MenuGroup): number {
  return counts.value[group.to] ?? 0
}

const LINK = 'flex min-h-11 items-center justify-between gap-3 rounded-xl px-3 no-underline'
const CURRENT = 'bg-black font-semibold text-white hover:text-white'
const OTHER = 'text-ink-2 hover:bg-mist hover:text-black'
const BADGE = 'rounded-full px-2 text-sm font-semibold'
</script>

<template>
  <nav aria-label="Manage" class="self-start">
    <ul class="m-0 flex list-none flex-col gap-1 p-0">
      <li v-for="g in MANAGE_MENU" :key="g.to" :data-open="isOpen(g, route.path) ? '' : undefined">
        <RouterLink
          :to="g.to"
          :aria-current="route.path === g.to ? 'page' : undefined"
          :class="[LINK, 'font-medium', route.path === g.to ? CURRENT : OTHER]"
        >
          {{ g.label }}
          <span
            v-if="badgePlacement(g, isOpen(g, route.path)) === 'group' && count(g) > 0"
            :class="[BADGE, route.path === g.to ? 'bg-white text-black' : 'bg-black text-white']"
          >{{ count(g) }}<span class="sr-only">{{ ' ' + (g.countLabel ?? '') }}</span></span>
        </RouterLink>
        <ul v-if="isOpen(g, route.path) && g.children.length > 0" class="m-0 mt-1 flex list-none flex-col gap-1 border-l-2 border-grey p-0 pl-3">
          <li v-for="c in g.children" :key="c.to">
            <RouterLink
              :to="c.to"
              :aria-current="route.path === c.to ? 'page' : undefined"
              :class="[LINK, 'text-[15px]', route.path === c.to ? CURRENT : OTHER]"
            >
              {{ c.label }}
              <span
                v-if="badgePlacement(g, true) === 'child' && c.to === g.countOn && count(g) > 0"
                :class="[BADGE, route.path === c.to ? 'bg-white text-black' : 'bg-black text-white']"
              >{{ count(g) }}<span class="sr-only">{{ ' ' + (g.countLabel ?? '') }}</span></span>
            </RouterLink>
          </li>
        </ul>
      </li>
    </ul>
  </nav>
</template>
