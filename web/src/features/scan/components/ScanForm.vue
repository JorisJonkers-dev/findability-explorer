<script setup lang="ts">
import { computed, ref, useTemplateRef, watch } from 'vue'
import { useCompanies } from '../../competitors/composables/useCompanies'
import { type Company, isScanning, pickerState, scanCountLabel, suggestions } from '../../competitors/domain/company'
import { useStartScan } from '../composables/useStartScan'
import { usePageBudget } from '../composables/usePageBudget'
import { checkScanUrl, describeScanUrlCheck } from '../domain/newScan'

const emit = defineEmits<{ started: [id: string] }>()

const LISTBOX = 'scan-url-options'
const optionId = (index: number) => `scan-url-option-${String(index)}`

const url = ref('')
const checkError = ref<string | null>(null)
const open = ref(false)
const active = ref(-1)
const newName = ref('')
const newNameEdited = ref(false)
const input = useTemplateRef<HTMLInputElement>('input')
const { companies } = useCompanies()
const { isStarting, error: startError, start, reset } = useStartScan()
const { pages } = usePageBudget()

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
  open.value = false
  const check = checkScanUrl(url.value)
  checkError.value = describeScanUrlCheck(check)
  if (check.kind !== 'ok') {
    input.value?.focus()
    return
  }
  const name = state.value.kind === 'new' ? newName.value : undefined
  start(check.url, pages.value, (id) => {
    emit('started', id)
  }, name)
}

function edited(): void {
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
      open.value = true
      active.value = (active.value + 1) % count
      break
    case 'ArrowUp':
      if (count === 0) return
      event.preventDefault()
      open.value = true
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

defineExpose({
  /** Fills in an address and starts its scan: a company's "Scan this website". */
  scan(address: string): void {
    url.value = address
    newNameEdited.value = false
    edited()
    submit()
  },
})
</script>

<template>
  <form novalidate aria-label="Scan a website" class="flex flex-col gap-3" @submit.prevent="submit">
    <div class="relative">
      <label for="scan-url" class="mb-1.5 block text-sm text-ink-2">Website address</label>
      <input
        id="scan-url"
        ref="input"
        v-model="url"
        type="text"
        role="combobox"
        inputmode="url"
        autocomplete="off"
        spellcheck="false"
        placeholder="Twentebelt, or www.competitor.com"
        aria-autocomplete="list"
        :aria-expanded="expanded"
        :aria-controls="LISTBOX"
        :aria-activedescendant="expanded && active >= 0 ? optionId(active) : undefined"
        :aria-invalid="urlError !== null"
        :aria-describedby="urlError !== null ? 'scan-url-error' : 'scan-url-company'"
        class="h-12 w-full rounded-full border-2 bg-white px-5 text-base text-black"
        :class="urlError !== null ? 'border-orange-ink' : 'border-grey'"
        @input="typed"
        @keydown="onKeydown"
        @blur="open = false"
      />
      <ul
        v-show="expanded"
        :id="LISTBOX"
        role="listbox"
        aria-label="Known companies"
        class="absolute top-full right-0 left-0 z-10 m-0 mt-2 list-none rounded-2xl bg-white p-1.5 shadow-[0_10px_24px_rgba(0,0,0,0.18)]"
      >
        <li
          v-for="(company, index) in options"
          :id="optionId(index)"
          :key="company.id"
          role="option"
          tabindex="-1"
          :aria-selected="index === active"
          class="flex cursor-pointer items-baseline justify-between gap-4 rounded-xl px-3 py-2.5"
          :class="index === active ? 'bg-mist' : 'hover:bg-mist'"
          @mousedown.prevent="choose(company)"
        >
          <span class="font-semibold">{{ company.shortName }}</span>
          <span class="truncate text-muted">{{ company.host }}</span>
        </li>
      </ul>
    </div>

    <p v-if="urlError !== null" id="scan-url-error" role="alert" class="m-0 text-sm text-orange-ink">{{ urlError }}</p>
    <div id="scan-url-company" aria-live="polite">
      <p v-if="state.kind === 'known'" class="m-0 text-sm text-ink-2">
        <strong class="font-semibold text-black">{{ state.company.shortName }}</strong>,
        {{ state.company.groupCompany ? "in Tribelt's own group" : 'a known competitor' }}.
        {{ scanCountLabel(state.company.scans, isScanning(state.company)) }}.
      </p>
      <div v-else-if="state.kind === 'new'" class="flex flex-col gap-1.5">
        <label class="flex items-center gap-3 text-sm text-ink-2">
          New company
          <input
            v-model="newName"
            type="text"
            maxlength="200"
            autocomplete="organization"
            class="h-10 min-w-0 flex-1 rounded-full border-2 border-grey bg-white px-4 text-base text-black"
            @input="newNameEdited = true"
          />
        </label>
        <p class="m-0 text-sm text-muted">Not in the list yet. The scan adds it as a new competitor.</p>
      </div>
    </div>
    <p v-if="formError !== null" role="alert" class="m-0 text-sm text-orange-ink">{{ formError }}</p>
    <button
      type="submit"
      :disabled="isStarting"
      class="min-h-12 rounded-full border-0 bg-black text-base font-semibold text-white hover:bg-ink-2 disabled:opacity-70"
    >
      {{ isStarting ? 'Starting…' : 'Scan website' }}
    </button>
  </form>
</template>
