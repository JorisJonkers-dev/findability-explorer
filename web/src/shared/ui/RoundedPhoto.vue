<script setup lang="ts">
defineProps<{ src: string; alt: string }>()
</script>

<template>
  <img :src="src" :alt="alt" loading="lazy" class="block h-56 w-full rounded-2xl object-cover stage:rounded-[28px]" />
</template>
