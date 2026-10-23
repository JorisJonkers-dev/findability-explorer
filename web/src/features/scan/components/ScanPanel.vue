<script setup lang="ts">
import { computed, useTemplateRef } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import { useScan } from '../composables/useScan'
import { siteHost } from '../domain/results'
import { isInProgress } from '../domain/status'
import RecentScans from './RecentScans.vue'
import ScanForm from './ScanForm.vue'
import ScanProgress from './ScanProgress.vue'
import ScanResult from './ScanResult.vue'

const props = withDefaults(defineProps<{ scanId?: string; isAdmin?: boolean }>(), { scanId: undefined, isAdmin: false })
const route = useRoute()
const router = useRouter()
const form = useTemplateRef<InstanceType<typeof ScanForm>>('form')

const id = computed(() => props.scanId ?? '')
const { scan, error } = useScan(id)

/** A started scan opens in the panel, beside the same company. */
function started(next: string): void {
  void router.push({ path: `/scan/${next}`, query: route.query })
}

defineExpose({
  scan(address: string): void {
    form.value?.scan(address)
  },
})
</script>

<template>
  <section
    aria-labelledby="scan-title"
    class="flex flex-col gap-5 rounded-3xl bg-mist p-5 sm:p-8 stage:min-h-0 stage:gap-7 stage:rounded-[36px] stage:p-14"
  >
    <div class="flex flex-col gap-2 stage:gap-2.5">
      <SectionHeading id="scan-title">{{ isAdmin ? 'Scan a competitor' : 'Scans' }}</SectionHeading>
      <p class="m-0 text-ink-2 stage:text-[28px]">
        {{
          isAdmin
            ? 'Type a company or a website. The scan reads its pages, product pages first, and ranks the search phrases it is built around.'
            : 'Each scan read a company’s pages and ranked the search phrases it is built around. Administrators start them.'
        }}
      </p>
    </div>
    <ScanForm v-if="isAdmin" ref="form" @started="started" />

    <template v-if="id !== ''">
      <p v-if="error" role="alert" class="m-0 text-orange-ink stage:text-[28px]">{{ error }}</p>
      <ScanProgress v-else-if="scan && isInProgress(scan.summary.status)" :summary="scan.summary" />
      <div
        v-else-if="scan && scan.summary.status === 'failed'"
        class="flex flex-col gap-3 rounded-2xl bg-white p-5 stage:gap-4 stage:rounded-[28px] stage:p-9"
      >
        <p class="m-0 text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[24px]">Scan failed</p>
        <p class="m-0 font-display text-2xl font-semibold stage:text-[56px]">{{ siteHost(scan.summary.url) }}</p>
        <p role="alert" class="m-0 text-ink-2 stage:text-[28px]">{{ scan.summary.failure ?? 'The scan stopped before it read a page.' }}</p>
      </div>
      <ScanResult v-else-if="scan" :key="scan.summary.id" :scan="scan" />
      <p v-else class="m-0 text-muted stage:text-[28px]">Loading the scan…</p>
      <RouterLink :to="{ path: '/competitors', query: route.query }" class="self-start text-sm font-medium stage:text-[26px]">
        Show the recent scans
      </RouterLink>
    </template>
    <RecentScans v-else />
    <slot />
  </section>
</template>
