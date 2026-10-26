<script setup lang="ts">
import { computed, toRef } from 'vue'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import { useReadinessFiles, useReadinessTable } from '../composables/useReadiness'
import { markedLines } from '../domain/readiness'

/**
 * The raw robots.txt, llms.txt and llms-full.txt the company's newest scan with AI readiness read,
 * for administrators to check the classification against; lines about AI crawlers are marked.
 */
const props = defineProps<{ companyId: string }>()

const { rowOf } = useReadinessTable()
const scanId = computed(() => rowOf(props.companyId)?.scan?.id)
const { files, isLoading, missing, error } = useReadinessFiles(scanId, toRef(() => true))

const shown = computed(() =>
  files.value
    ? [
        { name: 'robots.txt', text: files.value.robotsTxt, truncated: files.value.robotsTruncated },
        { name: 'llms.txt', text: files.value.llmsTxt, truncated: files.value.llmsTruncated },
        { name: 'llms-full.txt', text: files.value.llmsFullTxt, truncated: files.value.llmsFullTruncated },
      ]
    : [],
)
</script>

<template>
  <section aria-labelledby="files-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-7">
    <SectionHeading id="files-title">The files it serves to crawlers</SectionHeading>
    <p v-if="scanId === undefined || missing" class="m-0 text-ink-2 stage:text-[28px]">No scan of it recorded its AI readiness yet.</p>
    <p v-else-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the files…</p>
    <div v-else class="flex flex-col gap-5 stage:min-h-0 stage:gap-8 stage:overflow-y-auto">
      <details v-for="file in shown" :key="file.name" :open="file.name === 'robots.txt'" class="flex flex-col gap-2">
        <summary class="cursor-pointer font-semibold stage:text-[30px]">
          {{ file.name }}<span class="font-normal text-muted">{{ file.text === '' ? ': not served' : file.truncated ? ': the first 100 KB' : '' }}</span>
        </summary>
        <pre v-if="file.text !== ''" class="m-0 max-h-96 overflow-auto rounded-xl bg-mist p-3 text-xs stage:max-h-[640px] stage:p-6 stage:text-[22px]"><code><span
          v-for="(line, i) in markedLines(file.text)"
          :key="i"
          :class="line.ai ? 'bg-orange/25 font-semibold' : ''"
          class="block"
        >{{ line.text || ' ' }}</span></code></pre>
      </details>
      <p class="m-0 text-sm text-muted stage:text-[24px]"><span aria-hidden="true" class="inline-block size-3 bg-orange/25 stage:size-5"></span> Lines about a search or AI crawler</p>
    </div>
  </section>
</template>
