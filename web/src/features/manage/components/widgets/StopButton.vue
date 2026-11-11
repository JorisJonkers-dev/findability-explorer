<script setup lang="ts">
import { ref } from 'vue'

/** Stop, which first says what stopping keeps, and stops on the second click. */
defineProps<{ label: string; keeps: string; disabled?: boolean }>()
const emit = defineEmits<{ stop: [] }>()
const asking = ref(false)

function stop(): void {
  asking.value = false
  emit('stop')
}
</script>

<template>
  <div v-if="asking" class="flex flex-wrap items-center gap-3" data-part="confirm-stop">
    <span class="text-ink-2">{{ keeps }}</span>
    <button type="button" class="min-h-11 rounded-full border-2 border-orange-ink bg-orange-ink px-4 font-semibold text-white hover:bg-black" :disabled="disabled" @click="stop">
      {{ label }}
    </button>
    <button type="button" class="min-h-11 rounded-full border-2 border-silver bg-white px-4 font-semibold hover:border-black" @click="asking = false">Keep running</button>
  </div>
  <button
    v-else
    type="button"
    class="min-h-11 self-start rounded-full border-2 border-orange-ink bg-white px-4 font-semibold text-orange-ink hover:bg-orange-pale"
    :disabled="disabled"
    @click="asking = true"
  >
    {{ label }}
  </button>
</template>
