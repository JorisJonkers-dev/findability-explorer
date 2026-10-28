<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { barLength, type Measure } from '../domain/compare'

/** A competitor against Tribelt on each measure; picking a measure says what it means. */
const props = defineProps<{ name: string; measures: readonly Measure[] }>()

const id = ref<string | null>(null)
const picked = computed(() => props.measures.find((m) => m.id === id.value) ?? props.measures[0])
const aria = (m: Measure) => `${m.label}: ${props.name} ${m.themText}, Tribelt ${m.usText}.`
const style = (value: number | undefined, scale: number) => ({ width: `${String((barLength(value, scale) ?? 0) * 100)}%` })
</script>

<template>
  <section aria-labelledby="compare-title" class="flex flex-col gap-4 rounded-3xl bg-mist p-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h2 id="compare-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em]">{{ name }} against Tribelt</h2>
      <p aria-hidden="true" class="m-0 flex gap-4 text-sm font-medium">
        <span class="inline-flex items-center gap-2"><span class="size-3 rounded-sm bg-blue-grey"></span>{{ name }}</span>
        <span class="inline-flex items-center gap-2"><span class="size-3 rounded-sm bg-orange"></span>Tribelt</span>
      </p>
    </div>
    <p class="m-0 text-ink-2">Pick a measure to read what it means.</p>
    <div class="flex flex-col gap-1">
      <button
        v-for="m in measures"
        :key="m.id"
        type="button"
        :aria-pressed="picked?.id === m.id"
        :aria-label="aria(m)"
        class="grid min-h-16 w-full cursor-pointer gap-2 rounded-2xl border-2 bg-transparent px-3 py-2 text-left text-black hover:bg-white sm:grid-cols-[12rem_minmax(0,1fr)] sm:items-center sm:gap-4"
        :class="picked?.id === m.id ? 'border-black bg-white' : 'border-transparent'"
        @click="id = m.id"
      >
        <span class="font-medium">{{ m.label }}</span>
        <span aria-hidden="true" class="flex flex-col gap-1.5">
          <span v-for="side in [{ k: 'them', tone: 'bg-blue-grey' }, { k: 'us', tone: 'bg-orange' }]" :key="side.k" class="grid grid-cols-[minmax(0,1fr)_5rem] items-center gap-3">
            <span class="relative h-3.5 rounded-[7px] bg-grey">
              <span class="absolute inset-y-0 left-0 rounded-[7px]" :class="side.tone" :style="style(side.k === 'them' ? m.them : m.us, m.scale)"></span>
            </span>
            <span class="text-right text-sm tabular-nums" :class="(side.k === 'them' ? m.them : m.us) === undefined ? 'text-muted' : 'font-bold'">{{ side.k === 'them' ? m.themText : m.usText }}</span>
          </span>
        </span>
      </button>
    </div>
    <div v-if="picked" aria-live="polite" class="flex flex-col gap-2 rounded-2xl bg-white p-5">
      <span class="text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase">{{ picked.label }}</span>
      <p class="m-0">{{ picked.text }}</p>
      <RouterLink :to="picked.path" class="font-medium">{{ picked.pathLabel }}</RouterLink>
    </div>
  </section>
</template>
