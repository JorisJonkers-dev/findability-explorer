<script setup lang="ts">
import { computed } from 'vue'
import { highlight } from '../domain/fold'

const props = defineProps<{ text: string; query: string }>()
const segments = computed(() => highlight(props.text, props.query))
</script>

<template>
  <span>
    <template v-for="(segment, index) in segments" :key="index">
      <mark v-if="segment.match">{{ segment.text }}</mark>
      <template v-else>{{ segment.text }}</template>
    </template>
  </span>
</template>
