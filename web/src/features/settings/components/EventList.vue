<script setup lang="ts">
import { computed } from 'vue'
import type { LlmEvent } from '../../../infrastructure/api/types.gen'
import { eventLines } from '../domain/events'

const props = defineProps<{ events: readonly LlmEvent[] }>()
const lines = computed(() => eventLines(props.events))
</script>

<template>
  <section aria-labelledby="events-title" class="flex flex-col gap-4">
    <h2 id="events-title" class="m-0 font-display text-3xl font-semibold tracking-[-0.01em]">Recent changes</h2>
    <ol v-if="lines.length > 0" class="m-0 flex max-h-[28rem] list-none flex-col overflow-y-auto p-0" tabindex="0" aria-labelledby="events-title">
      <li v-for="line in lines" :key="line.key" class="flex flex-col gap-1 border-t-2 border-grey py-3">
        <span class="flex flex-wrap items-center gap-x-3 text-sm text-muted">
          <span>{{ line.when }}</span>
          <span>by {{ line.actor }}</span>
          <span v-if="line.ok !== null" class="font-medium" :class="line.ok ? 'text-black' : 'text-orange-ink'">
            {{ line.ok ? 'Accepted' : 'Refused' }}
          </span>
        </span>
        <span>{{ line.text }}</span>
      </li>
    </ol>
    <p v-else class="m-0 text-ink-2">Nothing has changed yet.</p>
  </section>
</template>
