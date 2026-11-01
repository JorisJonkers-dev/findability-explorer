<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import BigFigure from './BigFigure.vue'
import ManageWidget from './ManageWidget.vue'
import { SECONDARY } from './look'

/** A card on a section's index page: what a subpage holds, its figures, and the link to it. */
defineProps<{
  title: string
  text: string
  to: RouteLocationRaw
  link: string
  figures?: readonly { value: string; label: string }[]
  span?: 3 | 4 | 6
}>()
</script>

<template>
  <ManageWidget :title="title" :span="span ?? 4">
    <p class="m-0 text-ink-2">{{ text }}</p>
    <div v-if="figures && figures.length > 0" class="flex flex-wrap gap-8">
      <BigFigure v-for="f in figures" :key="f.label" :value="f.value" :label="f.label" />
    </div>
    <template #foot>
      <RouterLink :to="to" :class="SECONDARY">{{ link }}</RouterLink>
    </template>
  </ManageWidget>
</template>
