<script setup lang="ts">
import { vScrollHint } from '../../../shared/scrollHint'
import { RouterLink } from 'vue-router'
import type { AnalysisSummary } from '../../../infrastructure/api/types.gen'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import { scanDate, scanModel, scanName, scanPages, scanState } from '../domain/history'

defineProps<{ scans: readonly AnalysisSummary[]; companyId: string }>()
</script>

<template>
  <section aria-labelledby="timeline-title" class="flex flex-col gap-4">
    <SectionHeading id="timeline-title">Scan history</SectionHeading>
    <p v-if="scans.length === 0" class="m-0 text-ink-2">Not scanned yet.</p>
    <ol v-else v-scroll-hint aria-label="Scans, newest first" class="scroll-hint m-0 flex list-none flex-col p-0">
      <li v-for="scan in scans" :key="scan.id" class="relative flex gap-4 pb-5">
        <span aria-hidden="true" class="mt-1.5 size-3.5 flex-none rounded-full" :class="scan.status === 'done' ? 'bg-orange' : 'bg-silver'"></span>
        <div class="flex flex-col gap-0.5">
          <RouterLink :to="{ path: `/scan/${scan.id}`, query: { company: companyId } }" class="font-semibold">
            {{ scanName(scan) }}<span class="sr-only">, {{ scanState(scan) }}</span>
          </RouterLink>
          <span class="text-sm text-ink-2">{{ scanDate(scan) }}. {{ scanState(scan) }}. {{ scanPages(scan) }}.</span>
          <span class="text-sm text-muted">{{ scanModel(scan) }}</span>
        </div>
      </li>
    </ol>
  </section>
</template>
