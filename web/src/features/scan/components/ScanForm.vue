<script setup lang="ts">
import { computed, onMounted, ref, useTemplateRef, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useCompanies } from '../../competitors/composables/useCompanies'
import { type Company, pickerState, scanCountLabel, suggestions } from '../../competitors/domain/company'
import { useStartScan } from '../composables/useStartScan'
import { checkScanUrl, DEFAULT_PAGES, describeScanUrlCheck, PAGE_OPTIONS } from '../domain/newScan'

const props = defineProps<{
  /** An address to scan as soon as the form is shown: the overview's quick scan hands it over. */
  autostart?: string | null
  /** The scan on screen, whose address and budget the form shows until something is typed. */
  current?: { url: string; maxPages: number } | null
  /** A company to fill in, by id: a company page links to the form with it. */
  preselect?: string | null
}>()
const emit = defineEmits<{ started: [id: string] }>()

const LISTBOX = 'scan-url-options'
const optionId = (index: number) => `scan-url-option-${String(index)}`

const url = ref('')
const maxPages = ref<number>(DEFAULT_PAGES)
const touched = ref(false)
const checkError = ref<string | null>(null)
const open = ref(false)
const active = ref(-1)
const newName = ref('')
const newNameEdited = ref(false)
const input = useTemplateRef<HTMLInputElement>('input')
const { companies } = useCompanies()
const { isStarting, error: startError, start, reset } = useStartScan()

const options = computed(() => (open.value ? suggestions(companies.value, url.value) : []))
const expanded = computed(() => options.value.length > 0)
const state = computed(() => pickerState(url.value, companies.value))
const urlError = computed(() => checkError.value ?? (startError.value?.field === 'url' ? startError.value.message : null))
const formError = computed(() => (startError.value?.field === 'form' ? startError.value.message : null))

// A new company is named after its host until its name is typed.
watch(state, (next) => {
  if (next.kind === 'new' && !newNameEdited.value) newName.value = next.name
})

function submit(): void {
  touched.value = true
  open.value = false
  const check = checkScanUrl(url.value)
  checkError.value = describeScanUrlCheck(check)
  if (check.kind !== 'ok') {
    input.value?.focus()
    return
  }
  const name = state.value.kind === 'new' ? newName.value : undefined
  start(check.url, maxPages.value, (id) => {
    emit('started', id)
  }, name)
}

function edited(): void {
  touched.value = true
  checkError.value = null
  reset()
}

function typed(): void {
  edited()
  open.value = true
  active.value = -1
}

function choose(company: Company): void {
  url.value = String(company.websiteUrl)
  open.value = false
  active.value = -1
  edited()
}

/** The WAI-ARIA combobox keys: arrows move through the suggestions, Enter picks, Escape closes. */
function onKeydown(event: KeyboardEvent): void {
  const count = suggestions(companies.value, url.value).length
  switch (event.key) {
    case 'ArrowDown':
      if (count === 0) return
      event.preventDefault()
      if (!open.value) open.value = true
      active.value = (active.value + 1) % count
      break
    case 'ArrowUp':
      if (count === 0) return
      event.preventDefault()
      if (!open.value) open.value = true
      active.value = active.value <= 0 ? count - 1 : active.value - 1
      break
    case 'Enter': {
      const picked = options.value[active.value]
      if (picked) {
        event.preventDefault()
        choose(picked)
      }
      break
    }
    case 'Escape':
      if (expanded.value) {
        event.preventDefault()
        open.value = false
        active.value = -1
      }
      break
  }
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

watch(
  [() => props.preselect, companies],
  ([id]) => {
    const company = companies.value.find((c) => c.id === id)
    if (company?.websiteUrl && !touched.value) {
      url.value = company.websiteUrl
      touched.value = true
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
      <div class="relative min-w-0 flex-[1_1_420px]">
        <label class="flex flex-col gap-2 text-sm font-medium text-ink-2">
          Website address
          <input
            ref="input"
            v-model="url"
            type="url"
            role="combobox"
            inputmode="url"
            autocomplete="off"
            spellcheck="false"
            placeholder="A company or https://www.competitor.com"
            aria-autocomplete="list"
            :aria-expanded="expanded"
            :aria-controls="LISTBOX"
            :aria-activedescendant="expanded && active >= 0 ? optionId(active) : undefined"
            :aria-invalid="urlError !== null"
            :aria-describedby="urlError !== null ? 'scan-url-error' : 'scan-url-company'"
            class="h-14 w-full rounded-lg border bg-surface px-[18px] text-lg font-medium text-ink"
            :class="urlError !== null ? 'border-danger' : 'border-field'"
            @input="typed"
            @keydown="onKeydown"
            @blur="open = false"
          />
        </label>
        <ul
          v-show="expanded"
          :id="LISTBOX"
          role="listbox"
          aria-label="Known companies"
          class="absolute top-full right-0 left-0 z-20 m-0 mt-1 max-h-80 list-none overflow-y-auto rounded-lg border border-field bg-raised p-1 shadow-lg shadow-black/40"
        >
          <li
            v-for="(company, index) in options"
            :id="optionId(index)"
            :key="company.id"
            role="option"
            tabindex="-1"
            :aria-selected="index === active"
            class="flex items-center justify-between gap-3 rounded-md px-3 py-2.5"
            :class="index === active ? 'bg-pressed outline-2 outline-straw' : ''"
            @mousedown.prevent="choose(company)"
          >
            <span class="flex min-w-0 flex-col">
              <span class="truncate font-semibold text-ink">{{ company.shortName }}</span>
              <span class="truncate text-sm text-muted">{{ company.host }}</span>
            </span>
            <span
              class="flex-none rounded-full border px-2.5 py-0.5 text-[13px]"
              :class="company.scans > 0 ? 'border-straw/60 text-straw' : 'border-field text-muted'"
            >
              {{ scanCountLabel(company.scans) }}
            </span>
          </li>
        </ul>
      </div>
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
    <div id="scan-url-company" aria-live="polite">
      <p v-if="state.kind === 'known'" class="m-0 flex flex-wrap items-center gap-x-2 gap-y-1 text-[15px] text-ink-2">
        <svg aria-hidden="true" viewBox="0 0 16 16" class="size-4 flex-none text-straw" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 8.5 6.5 12 13 4.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        <span>
          <RouterLink :to="`/competitors/${state.company.id}`" class="font-semibold">{{ state.company.shortName }}</RouterLink>, known
          {{ state.company.groupCompany ? "company in Tribelt's own group" : 'competitor' }}.
        </span>
        <span class="text-muted">{{ scanCountLabel(state.company.scans) }}</span>
      </p>
      <div
        v-else-if="state.kind === 'new'"
        class="flex max-w-[560px] flex-col gap-2 rounded-lg border border-line bg-surface p-4"
      >
        <label class="flex flex-col gap-2 text-sm font-medium text-ink-2">
          New company
          <input
            v-model="newName"
            type="text"
            maxlength="200"
            autocomplete="organization"
            class="h-11 w-full rounded-md border border-field bg-ground px-3 text-base font-medium text-ink"
            @input="newNameEdited = true"
          />
        </label>
        <p class="m-0 text-sm text-muted">Not in the list yet. The scan adds it as a new competitor.</p>
      </div>
    </div>
    <p v-if="formError !== null" role="alert" class="m-0 text-[15px] text-danger">{{ formError }}</p>
  </form>
</template>
