<script setup lang="ts">
import { computed, toRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageFrame from '../../shell/components/PageFrame.vue'
import { useReadinessFiles, useReadinessTable } from '../composables/useReadiness'
import { siteName } from '../domain/checks'
import { type FileKey, fileKeyOf, filesFoundLine, fileTabs } from '../domain/files'
import { scanDataLine, siteHost } from '../domain/site'
import FileViewer from './FileViewer.vue'

/** The files the site's newest scan fetched, a tab each, to read, search, copy, or download. */
const props = defineProps<{ companyId: string }>()

const route = useRoute()
const router = useRouter()
const { rowOf, isLoading: tableLoading, error: tableError } = useReadinessTable()
const row = computed(() => rowOf(props.companyId))
const { files, isLoading: filesLoading, error: filesError } = useReadinessFiles(
  computed(() => row.value?.scan?.id),
  toRef(() => true),
)
const tabs = computed(() => fileTabs(files.value, row.value?.readiness))
// The open tab lives in the address, so a link can open the sitemap or llms.txt.
const key = computed(() => fileKeyOf(route.query.file))
const tab = computed(() => tabs.value.find((t) => t.key === key.value))
const host = computed(() => (row.value ? siteHost(row.value) : ''))

function pick(k: FileKey): void {
  void router.replace({ query: { ...route.query, file: k } })
}

const back = { to: '/ai-readiness', label: 'AI readiness' }
</script>

<template>
  <PageFrame
    v-if="row?.scan"
    section="ai-readiness"
    :title="`Files found on ${host}`"
    purpose="The files the site's newest scan fetched. Pick a file to read, search, copy, or download it."
    :data-line="scanDataLine(row)"
    :back="{ to: `/ai-readiness/${companyId}`, label: `${siteName(row)}’s AI readiness` }"
    :manage="{ to: '/manage/scans', label: 'Manage the scans' }"
    help-topic="websites"
  >
    <section aria-label="The files" class="flex flex-col gap-4">
      <div role="tablist" aria-label="Files" class="flex flex-wrap gap-2">
        <button
          v-for="t in tabs"
          :id="`file-tab-${t.key}`"
          :key="t.key"
          type="button"
          role="tab"
          :aria-selected="t.key === key"
          aria-controls="file-panel"
          class="inline-flex min-h-11 cursor-pointer items-center rounded-full border-2 px-4 font-mono text-sm"
          :class="t.key === key ? 'border-black bg-black text-white' : 'border-grey bg-white text-ink-2 hover:border-black'"
          @click="pick(t.key)"
        >
          {{ t.name }}
        </button>
      </div>
      <p v-if="filesError" role="alert" class="m-0 text-orange-ink">{{ filesError }}</p>
      <p v-else-if="filesLoading" class="m-0 text-muted">Loading the files…</p>
      <template v-else>
        <p class="m-0 text-sm text-ink-2">{{ filesFoundLine(tabs) }}</p>
        <div id="file-panel" role="tabpanel" :aria-labelledby="`file-tab-${key}`" class="min-w-0 rounded-2xl bg-mist p-4 sm:p-5">
          <FileViewer v-if="tab" :key="tab.key" :tab="tab" :host="host" />
        </div>
      </template>
    </section>
  </PageFrame>

  <PageFrame v-else-if="tableLoading" section="ai-readiness" title="Files found" purpose="Loading the site…" :back="back" />
  <PageFrame v-else-if="tableError" section="ai-readiness" title="Files found" purpose="The site could not be loaded." :back="back">
    <p role="alert" class="m-0 text-orange-ink">{{ tableError }}</p>
  </PageFrame>
  <PageFrame v-else section="ai-readiness" title="No files" purpose="No scan has this address." :back="back">
    <p class="m-0 rounded-2xl bg-mist p-5 text-ink-2">No scan of this site was read.</p>
  </PageFrame>
</template>
