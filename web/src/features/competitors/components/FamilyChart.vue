<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { type FamilyBar, familyChartLabel, familyLine } from '../domain/compare'

/** How many competitors sell each of Tribelt's belt families; picking a bar says how Tribelt sells that family. */
const props = defineProps<{ bars: readonly FamilyBar[] }>()

const family = ref<string | null>(null)
const picked = computed(() => props.bars.find((b) => b.family === family.value) ?? props.bars[0])
const most = computed(() => Math.max(1, ...props.bars.map((b) => b.makers)))
</script>

<template>
  <section aria-labelledby="family-title" class="flex flex-col gap-4 rounded-3xl bg-mist p-6 stage:gap-8 stage:rounded-[36px] stage:p-12">
    <div class="flex flex-col gap-1">
      <h2 id="family-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Competitors that sell each belt family</h2>
      <p class="m-0 text-ink-2 stage:text-[28px]">Pick a family to see it.</p>
    </div>
    <div role="group" :aria-label="familyChartLabel(bars)" class="flex flex-col gap-1">
      <button
        v-for="b in bars"
        :key="b.family"
        type="button"
        :aria-pressed="picked?.family === b.family"
        class="grid min-h-12 w-full cursor-pointer grid-cols-[minmax(0,8rem)_minmax(0,1fr)_3rem] items-center gap-3 rounded-xl border-2 bg-transparent px-2.5 py-1 text-left text-black hover:bg-white stage:min-h-20 stage:grid-cols-[320px_minmax(0,1fr)_120px]"
        :class="picked?.family === b.family ? 'border-black bg-white' : 'border-transparent'"
        @click="family = b.family"
      >
        <span class="font-medium">{{ b.label }}</span>
        <span aria-hidden="true" class="relative h-3 rounded-[5px] bg-grey stage:h-5">
          <span class="absolute inset-y-0 left-0 rounded-[5px]" :class="picked?.family === b.family ? 'bg-orange' : 'bg-blue-grey'" :style="{ width: `${String((b.makers / most) * 100)}%` }"></span>
        </span>
        <span class="text-right font-bold tabular-nums">{{ b.makers }}</span>
      </button>
    </div>
    <div v-if="picked" aria-live="polite" class="flex flex-col gap-2 rounded-2xl bg-white p-5 stage:gap-4 stage:p-9">
      <span class="text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[24px]">{{ picked.label }}</span>
      <span class="stage:text-[28px]">{{ familyLine(picked) }}</span>
      <RouterLink :to="picked.path" class="font-medium">Open the {{ picked.label.toLowerCase() }} page</RouterLink>
    </div>
  </section>
</template>
