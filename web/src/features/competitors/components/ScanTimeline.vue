<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { AnalysisSummary } from '../../../infrastructure/api/types.gen'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import { scanDate, scanModel, scanName, scanPages, scanState } from '../domain/history'

defineProps<{ scans: readonly AnalysisSummary[]; companyId: string }>()
</script>

<template>
  <section aria-labelledby="timeline-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-7">
    <SectionHeading id="timeline-title">Scan history</SectionHeading>
    <p v-if="scans.length === 0" class="m-0 text-ink-2 stage:text-[30px]">Not scanned yet.</p>
    <ol v-else aria-label="Scans, newest first" class="m-0 flex list-none flex-col p-0 stage:min-h-0 stage:overflow-y-auto">
      <li v-for="scan in scans" :key="scan.id" class="relative flex gap-4 pb-5 stage:gap-7 stage:pb-9">
        <span aria-hidden="true" class="mt-1.5 size-3.5 flex-none rounded-full stage:mt-3 stage:size-[26px]" :class="scan.status === 'done' ? 'bg-orange' : 'bg-silver'"></span>
        <div class="flex flex-col gap-0.5 stage:gap-1.5">
          <RouterLink :to="{ path: `/scan/${scan.id}`, query: { company: companyId } }" class="font-semibold stage:text-[32px]">
            {{ scanName(scan) }}<span class="sr-only">, {{ scanState(scan) }}</span>
          </RouterLink>
          <span class="text-sm text-ink-2 stage:text-[26px]">{{ scanDate(scan) }} · {{ scanState(scan) }} · {{ scanPages(scan) }}</span>
          <span class="text-sm text-muted stage:text-[24px]">{{ scanModel(scan) }}</span>
        </div>
      </li>
    </ol>
  </section>
</template>
