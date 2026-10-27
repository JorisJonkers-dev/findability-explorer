<script setup lang="ts">
import PillButton from './PillButton.vue'

/**
 * Filters a list of terms by language: every language, or one of those the list holds. It shows
 * only when there is a choice to make, or a language is chosen that the list no longer holds.
 */
defineProps<{ choices: readonly { value: string; label: string }[]; selected: string | null }>()
const emit = defineEmits<{ select: [language: string | null] }>()
</script>

<template>
  <div v-if="choices.length > 1 || selected !== null" role="group" aria-label="Language" class="flex flex-wrap gap-2 stage:gap-2.5">
    <PillButton size="sm" :pressed="selected === null" @click="emit('select', null)">All languages</PillButton>
    <PillButton v-for="c in choices" :key="c.value" size="sm" :pressed="selected === c.value" @click="emit('select', c.value)">
      {{ c.label }}
    </PillButton>
  </div>
</template>
