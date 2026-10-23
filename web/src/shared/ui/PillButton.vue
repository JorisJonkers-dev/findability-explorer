<script setup lang="ts">
withDefaults(defineProps<{ pressed: boolean; size?: 'md' | 'sm' }>(), { size: 'md' })
</script>

<template>
  <!-- A toggle: black when pressed, outlined when not. aria-pressed says which to a screen reader. -->
  <button
    type="button"
    :aria-pressed="pressed"
    class="inline-flex min-h-11 items-center rounded-full border-2 px-4 text-sm font-medium stage:border-[3px]"
    :class="[
      pressed ? 'border-black bg-black text-white' : 'border-grey bg-white text-black hover:border-black',
      size === 'md' ? 'stage:min-h-[88px] stage:px-9 stage:text-[30px]' : 'stage:min-h-[72px] stage:px-[26px] stage:text-[26px]',
    ]"
  >
    <slot />
  </button>
</template>
