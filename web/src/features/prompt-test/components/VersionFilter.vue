<script setup lang="ts">
import PillButton from '../../../shared/ui/PillButton.vue'
import type { VersionChoice } from '../domain/history'

/** The Prompt set filter above a Prompt test chart: every version, or one, and what the choice asks. */
defineProps<{ options: readonly { value: VersionChoice; label: string }[]; version: VersionChoice; note: string }>()
defineEmits<{ choose: [v: VersionChoice] }>()
</script>

<template>
  <div class="flex flex-col gap-2">
    <div class="flex flex-wrap items-center gap-x-3 gap-y-2">
      <span class="text-sm text-muted">Prompt set</span>
      <div role="group" aria-label="Prompt set" class="flex flex-wrap gap-1.5">
        <PillButton v-for="o in options" :key="String(o.value)" size="sm" :pressed="o.value === version" @click="$emit('choose', o.value)">{{ o.label }}</PillButton>
      </div>
    </div>
    <p v-if="note" data-part="version-note" class="m-0 max-w-[72ch] text-sm text-ink-2">{{ note }}</p>
  </div>
</template>
