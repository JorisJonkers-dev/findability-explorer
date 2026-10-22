<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useScan } from '../composables/useScan'
import ScanForm from './ScanForm.vue'
import ScanResult from './ScanResult.vue'

const props = defineProps<{ id?: string }>()
const route = useRoute()
const router = useRouter()

const id = computed(() => props.id ?? '')
const { scan, error } = useScan(id)

// The overview's quick scan arrives as ?url=; the form starts it once, and a reload does not repeat it.
const autostart = typeof route.query.url === 'string' ? route.query.url : null
// A company page's "Scan this website" arrives as ?competitor=, which fills in its website.
const preselect = typeof route.query.competitor === 'string' ? route.query.competitor : null
if (autostart !== null || preselect !== null) void router.replace({ query: {} })

const current = computed(() =>
  scan.value ? { url: scan.value.summary.url, maxPages: scan.value.summary.maxPages } : null,
)

function started(next: string): void {
  void router.push(`/scan/${next}`)
}
</script>

<template>
  <div class="flex flex-col gap-14">
    <section aria-labelledby="scan-title" class="flex max-w-[980px] flex-col gap-5">
      <h1 id="scan-title" class="m-0 font-display text-[40px] leading-none font-semibold tracking-[-0.005em] md:text-[52px]">
        Scan a competitor
      </h1>
      <p class="m-0 max-w-[62ch] text-lg text-ink-2">
        Paste a website. The scan reads up to 30 of its pages and shows the search terms the site is built around, and
        which of them Tribelt does not use yet.
      </p>
      <ScanForm :autostart="autostart" :current="current" :preselect="preselect" @started="started" />
    </section>

    <template v-if="id !== ''">
      <p v-if="error" role="alert" class="m-0 rounded-xl border border-line bg-surface p-6 text-ink-2">{{ error }}</p>
      <ScanResult v-else-if="scan" :scan="scan" />
      <p v-else class="m-0 text-muted">Loading the scan…</p>
    </template>
  </div>
</template>
