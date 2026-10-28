<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import { RouterLink } from 'vue-router'
import type { Competitor } from '../../../infrastructure/api/types.gen'
import { useCandidateDecision, useCompetitorCandidates } from '../../prompt-test/composables/usePromptTest'
import { candidateWebsite } from '../../prompt-test/domain/settings'
import { useAddCompanies } from '../composables/useCompanyAdmin'
import { hostOf } from '../domain/company'
import { addToTray, batchSummary, pasteIntoTray, type Suggestion, type TrayItem, traySuggestions } from '../domain/tray'

/** Adding several competitors at once: picks from a type-ahead or a pasted list collect in a tray. */
const props = defineProps<{ companies: readonly Competitor[] }>()

const id = useId()
const mode = ref<'type' | 'paste'>('type')
const text = ref('')
const pasted = ref('')
const skipped = ref<string[]>([])
const tray = ref<TrayItem[]>([])
const scan = ref(true)
const open = ref(false)
const active = ref(0)

const { candidates } = useCompetitorCandidates(ref('pending'))
const { isDeciding, error: decisionError, dismiss } = useCandidateDecision()
const { isAdding, batch, error, add } = useAddCompanies()

const known = computed(() => props.companies.map((c) => ({ name: c.name, host: c.host })))
const suggestions = computed(() => traySuggestions(text.value, known.value, candidates.value))
const named = computed(() => [...candidates.value].sort((a, b) => b.answers - a.answers).slice(0, 3))
const problems = computed(() => batch.value?.items.filter((i) => i.outcome !== 'added') ?? [])

function pick(s: Suggestion | undefined): void {
  if (!s?.item) return
  tray.value = addToTray(tray.value, s.item)
  text.value = ''
  open.value = false
}

function onKey(event: KeyboardEvent): void {
  const count = suggestions.value.length
  if (event.key === 'ArrowDown' && count > 0) {
    event.preventDefault()
    open.value = true
    active.value = (active.value + 1) % count
  } else if (event.key === 'ArrowUp' && count > 0) {
    event.preventDefault()
    active.value = (active.value - 1 + count) % count
  } else if (event.key === 'Enter') {
    event.preventDefault()
    pick(suggestions.value[active.value])
  } else if (event.key === 'Escape') {
    open.value = false
  }
}

function typed(): void {
  open.value = true
  active.value = 0
}

function putPasted(): void {
  const result = pasteIntoTray(tray.value, pasted.value, known.value)
  tray.value = result.tray
  skipped.value = result.skipped
  pasted.value = ''
  mode.value = 'type'
}

function send(): void {
  add(
    tray.value.map((t) => ({ name: t.name, ...(t.website ? { url: t.website } : {}), ...(t.candidateId ? { candidateId: t.candidateId } : {}) })),
    scan.value,
    (b) => {
      // What was refused stays in the tray to fix; the rest is done.
      const refused = new Set(b.items.filter((i) => i.outcome === 'rejected').map((i) => i.name))
      tray.value = tray.value.filter((t) => refused.has(t.name.trim()))
    },
  )
}
</script>

<template>
  <section aria-labelledby="add-title" class="flex flex-col gap-4">
    <div class="flex flex-wrap items-baseline justify-between gap-3">
      <h2 id="add-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em]">Add competitors</h2>
      <button
        type="button"
        class="border-0 bg-transparent p-0 font-medium text-orange-ink underline underline-offset-4"
        @click="mode = mode === 'type' ? 'paste' : 'type'"
      >
        {{ mode === 'type' ? 'Paste a list instead' : 'Search one at a time instead' }}
      </button>
    </div>

    <div v-if="mode === 'type'" class="relative flex flex-col gap-1.5">
      <label :for="`${id}-q`" class="text-sm font-medium">Name or website</label>
      <input
        :id="`${id}-q`"
        v-model="text"
        type="text"
        role="combobox"
        autocomplete="off"
        :aria-expanded="open && suggestions.length > 0"
        :aria-controls="`${id}-list`"
        :aria-activedescendant="open && suggestions.length > 0 ? `${id}-s${String(active)}` : undefined"
        aria-autocomplete="list"
        class="h-11 rounded-xl border-2 border-silver bg-white px-3 text-base"
        placeholder="wire belt, or https://www.example.com"
        @input="typed"
        @keydown="onKey"
        @blur="open = false"
      />
      <ul
        v-show="open && suggestions.length > 0"
        :id="`${id}-list`"
        role="listbox"
        aria-label="Suggestions"
        class="absolute top-full right-0 left-0 z-20 m-0 mt-1 flex list-none flex-col rounded-xl border-2 border-silver bg-white p-1 shadow-lg"
      >
        <li
          v-for="(s, i) in suggestions"
          :id="`${id}-s${String(i)}`"
          :key="`${s.kind}-${s.name}`"
          role="option"
          tabindex="-1"
          :aria-selected="i === active"
          :aria-disabled="s.kind === 'known'"
          class="flex cursor-pointer items-center justify-between gap-3 rounded-lg px-3 py-2"
          :class="[i === active ? 'bg-mist' : '', s.kind === 'known' ? 'cursor-default text-muted' : '']"
          @mousedown.prevent="pick(s)"
        >
          <span class="flex flex-col">
            <span class="font-medium">{{ s.name }}</span><span class="text-xs text-muted">{{ s.detail }}</span>
          </span>
          <span class="text-sm font-semibold" :class="s.kind === 'known' ? 'text-muted' : 'text-orange-ink'">{{ s.kind === 'known' ? 'Added' : 'Add' }}</span>
        </li>
      </ul>
    </div>

    <div v-else class="flex flex-col gap-2">
      <label :for="`${id}-paste`" class="text-sm font-medium">One company per line, with its website</label>
      <textarea
        :id="`${id}-paste`"
        v-model="pasted"
        rows="5"
        class="rounded-xl border-2 border-silver bg-white p-3 font-sans text-base"
        placeholder="Berndorf Band, berndorfband-group.com"
      ></textarea>
      <button
        type="button"
        class="inline-flex min-h-11 items-center self-start rounded-full border-2 border-black bg-white px-4 font-semibold hover:bg-mist disabled:opacity-60"
        :disabled="pasted.trim() === ''"
        @click="putPasted"
      >
        Put them in the tray
      </button>
    </div>
    <p v-if="skipped.length > 0" class="m-0 text-sm text-ink-2">Left out as competitors already: {{ skipped.join(', ') }}.</p>

    <div class="flex flex-col gap-3 rounded-2xl bg-mist p-4">
      <span class="text-sm font-semibold tracking-[0.04em] text-orange-ink uppercase">To add</span>
      <p v-if="tray.length === 0" class="m-0 text-ink-2">Nothing picked yet.</p>
      <ul v-else aria-label="To add" class="m-0 flex list-none flex-col gap-2 p-0">
        <li v-for="(t, i) in tray" :key="`${t.name}-${String(i)}`" class="flex flex-wrap items-center gap-2 rounded-xl bg-white p-2">
          <span class="min-w-32 flex-1 font-medium">{{ t.name }}</span>
          <label class="sr-only" :for="`${id}-w${String(i)}`">Website of {{ t.name }}</label>
          <input
            :id="`${id}-w${String(i)}`"
            v-model="t.website"
            type="url"
            class="h-10 min-w-0 flex-1 rounded-lg border-2 px-2 text-sm"
            :class="t.website === '' ? 'border-orange-ink' : 'border-grey'"
            placeholder="Add its website"
          />
          <button
            type="button"
            class="min-h-10 rounded-full border-0 bg-transparent px-3 text-sm font-medium text-ink-2 underline underline-offset-4"
            @click="tray = tray.filter((_, j) => j !== i)"
          >
            Remove<span class="sr-only">{{ ' ' + t.name }}</span>
          </button>
        </li>
      </ul>
      <label class="inline-flex items-center gap-2 text-sm">
        <input v-model="scan" type="checkbox" class="size-5 accent-orange-ink" />
        Scan them straight away
      </label>
      <button
        type="button"
        class="inline-flex min-h-11 items-center self-start rounded-full border-0 bg-orange px-5 font-semibold text-black hover:bg-orange-soft disabled:bg-grey disabled:text-muted"
        :disabled="tray.length === 0 || isAdding"
        @click="send"
      >
        {{ tray.length === 1 ? 'Add 1 competitor' : `Add ${String(tray.length)} competitors` }}
      </button>
      <p v-if="batch" aria-live="polite" class="m-0 font-medium">{{ batchSummary(batch) }}</p>
      <ul v-if="problems.length > 0" aria-label="Not added" class="m-0 flex list-none flex-col gap-1 p-0 text-sm text-ink-2">
        <li v-for="p in problems" :key="p.name">{{ p.name }}. {{ p.reason }}</li>
      </ul>
      <p v-if="error" role="alert" class="m-0 font-medium text-orange-ink">{{ error }}</p>
    </div>

    <div v-if="named.length > 0" class="flex flex-col gap-3">
      <h3 class="m-0 text-sm font-semibold tracking-[0.04em] text-orange-ink uppercase">Named in AI answers</h3>
      <ul aria-label="Named in AI answers" class="m-0 flex list-none flex-col gap-2 p-0">
        <li v-for="c in named" :key="c.id" class="flex flex-wrap items-center gap-3 border-b border-grey py-2">
          <span class="flex min-w-40 flex-1 flex-col">
            <a v-if="candidateWebsite(c)" :href="candidateWebsite(c)" target="_blank" rel="noopener" class="font-medium">{{ c.name }}</a>
            <span v-else class="font-medium">{{ c.name }}</span>
            <span class="text-xs text-muted">Named in {{ c.answers }} AI {{ c.answers === 1 ? 'answer' : 'answers' }}{{ candidateWebsite(c) ? `. ${hostOf(candidateWebsite(c)) ?? ''}` : '' }}</span>
          </span>
          <button
            type="button"
            class="min-h-10 rounded-full border-0 bg-orange px-4 text-sm font-semibold text-black hover:bg-orange-soft"
            @click="tray = addToTray(tray, { name: c.name, website: candidateWebsite(c), candidateId: c.id })"
          >
            Add<span class="sr-only">{{ ' ' + c.name }}</span>
          </button>
          <button
            type="button"
            class="min-h-10 rounded-full border-2 border-silver bg-white px-4 text-sm font-medium hover:border-black disabled:opacity-60"
            :disabled="isDeciding"
            @click="dismiss(c.id, c.name)"
          >
            Not a competitor<span class="sr-only">: {{ c.name }}</span>
          </button>
        </li>
      </ul>
      <p v-if="decisionError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ decisionError }}</p>
      <RouterLink to="/settings/competitors" class="self-start text-sm font-medium">Review every company named in AI answers</RouterLink>
    </div>
  </section>
</template>
