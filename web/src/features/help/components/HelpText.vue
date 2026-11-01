<script setup lang="ts">
/** Text with the parts that match the search marked. */
defineProps<{ parts: readonly { text: string; hit: boolean }[] }>()
</script>

<template>
  <template v-for="(p, i) in parts" :key="i">
    <mark v-if="p.hit" class="rounded-sm bg-orange-pale px-0.5 text-inherit">{{ p.text }}</mark>
    <template v-else>{{ p.text }}</template>
  </template>
</template>
