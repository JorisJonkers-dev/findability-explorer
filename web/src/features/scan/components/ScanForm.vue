<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef, watch } from 'vue'
import { useStartScan } from '../composables/useStartScan'
import { checkScanUrl, DEFAULT_PAGES, describeScanUrlCheck, PAGE_OPTIONS } from '../domain/newScan'

const props = defineProps<{
  /** An address to scan as soon as the form is shown: the overview's quick scan hands it over. */
  autostart?: string | null
  /** The scan on screen, whose address and budget the form shows until something is typed. */
  current?: { url: string; maxPages: number } | null
}>()
const emit = defineEmits<{ started: [id: string] }>()

const url = ref('')
const maxPages = ref<number>(DEFAULT_PAGES)
const touched = ref(false)
const checkError = ref<string | null>(null)
const input = useTemplateRef<HTMLInputElement>('input')
const { isStarting, error: startError, start, reset } = useStartScan()

const urlError = computed(() => checkError.value ?? (startError.value?.field === 'url' ? startError.value.message : null))
const formError = computed(() => (startError.value?.field === 'form' ? startError.value.message : null))

function submit(): void {
  touched.value = true
  const check = checkScanUrl(url.value)
  checkError.value = describeScanUrlCheck(check)
  if (check.kind !== 'ok') {
    input.value?.focus()
    return
  }
  start(check.url, maxPages.value, (id) => {
    emit('started', id)
  })
}

function edited(): void {
  touched.value = true
  checkError.value = null
  reset()
}

watch(
  () => props.current,
  (current) => {
    if (current && !touched.value) {
      url.value = current.url
      maxPages.value = (PAGE_OPTIONS as readonly number[]).includes(current.maxPages) ? current.maxPages : DEFAULT_PAGES
    }
  },
  { immediate: true },
)

onMounted(() => {
  if (props.autostart) {
    url.value = props.autostart
    submit()
  }
})
</script>

<template>
  <form novalidate aria-label="Scan a website" class="mt-2 flex flex-col gap-3" @submit.prevent="submit">
    <div class="flex flex-wrap items-end gap-3">
      <label class="flex min-w-0 flex-[1_1_420px] flex-col gap-2 text-sm font-medium text-ink-2">
        Website address
        <input
          ref="input"
          v-model="url"
          type="url"
          inputmode="url"
          autocomplete="url"
          spellcheck="false"
          placeholder="https://www.competitor.com"
          :aria-invalid="urlError !== null"
          :aria-describedby="urlError !== null ? 'scan-url-error' : undefined"
          class="h-14 w-full rounded-lg border bg-surface px-[18px] text-lg font-medium text-ink"
          :class="urlError !== null ? 'border-danger' : 'border-field'"
          @input="edited"
        />
      </label>
      <label class="flex flex-[0_0_150px] flex-col gap-2 text-sm font-medium text-ink-2">
        Pages to read
        <select
          v-model.number="maxPages"
          class="h-14 w-full rounded-lg border border-field bg-surface px-3.5 text-lg font-medium text-ink"
          @change="edited"
        >
          <option v-for="pages in PAGE_OPTIONS" :key="pages" :value="pages">{{ pages }} pages</option>
        </select>
      </label>
      <button
        type="submit"
        :disabled="isStarting"
        class="h-14 rounded-lg bg-straw px-7 text-lg font-semibold text-on-straw hover:bg-straw-hover disabled:opacity-70"
      >
        {{ isStarting ? 'Starting…' : 'Scan website' }}
      </button>
    </div>
    <p v-if="urlError !== null" id="scan-url-error" role="alert" class="m-0 text-[15px] text-danger">{{ urlError }}</p>
    <p v-if="formError !== null" role="alert" class="m-0 text-[15px] text-danger">{{ formError }}</p>
  </form>
</template>
