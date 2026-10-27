<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId, useTemplateRef, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { CompetitorCandidate, CompetitorCandidateStatus } from '../../../infrastructure/api/types.gen'
import { useCandidateDecision, useCompetitorCandidates } from '../../prompt-test/composables/usePromptTest'
import { CANDIDATE_STATUSES, candidateTab, parseCandidateStatus } from '../../prompt-test/domain/settings'
import SettingsShell from '../../shell/components/SettingsShell.vue'
import { CHECK_FILTERS, type CheckFilter, idsOf, keptBy, matchesSearch, reviewLine } from '../domain/candidates'
import CandidateRow from './CandidateRow.vue'

/**
 * The companies AI answers name that are not on the competitor list, one per company: names on one
 * website, and names Haiku merged, are one. Haiku checks each; a person adds or dismisses, one at
 * a time, by keyboard, or several at once.
 */
const id = useId()
const route = useRoute()
const router = useRouter()
// The tab lives in the address, so a reload or a link shows the same one.
const status = computed<CompetitorCandidateStatus>(() => parseCandidateStatus(route.query.status))
const { candidates, counts, review, isLoading, forbidden, error } = useCompetitorCandidates(status)
const { isDeciding, notice, results, error: decisionError, add, dismiss, undo, decideMany, split, startReview } = useCandidateDecision()

const search = ref('')
const filter = ref<CheckFilter>('all')
const shown = computed(() => candidates.value.filter((c) => matchesSearch(c, search.value) && keptBy(c, filter.value)))
const selected = ref(new Set<string>())
const chosen = computed(() => shown.value.filter((c) => selected.value.has(c.id)))
const allShownSelected = computed(() => shown.value.length > 0 && chosen.value.length === shown.value.length)

function show(next: CompetitorCandidateStatus): void {
  selected.value = new Set()
  void router.replace({ query: next === 'pending' ? {} : { status: next } })
}

function select(c: CompetitorCandidate, on: boolean): void {
  const next = new Set(selected.value)
  if (on) next.add(c.id)
  else next.delete(c.id)
  selected.value = next
}

function selectAllShown(on: boolean): void {
  selected.value = on ? new Set(idsOf(shown.value)) : new Set()
}

function many(action: 'add' | 'dismiss'): void {
  decideMany(action, idsOf(chosen.value))
  selected.value = new Set()
}

// The row the keys act on; after a decision the next one moves up into its place.
const active = ref(0)
const rows = useTemplateRef<{ add: () => Promise<void>; focus: () => void }[]>('rows')
const listHasFocus = ref(false)

function focusRow(i: number): void {
  active.value = Math.max(0, Math.min(i, shown.value.length - 1))
  void nextTick(() => rows.value?.[active.value]?.focus())
}

watch(
  () => shown.value.length,
  () => {
    if (active.value >= shown.value.length) active.value = Math.max(0, shown.value.length - 1)
    if (listHasFocus.value) focusRow(active.value)
  },
)

// The keys act on a focused row; they are heard on the window, as long as the page is open.
onMounted(() => { window.addEventListener('keydown', onKey) })
onBeforeUnmount(() => { window.removeEventListener('keydown', onKey) })

function onKey(event: KeyboardEvent): void {
  if (!(event.target instanceof HTMLElement) || !('row' in event.target.dataset)) return
  const c = shown.value[active.value]
  if (!c) return
  const act: Record<string, () => void> = {
    ArrowDown: () => { focusRow(active.value + 1) },
    j: () => { focusRow(active.value + 1) },
    ArrowUp: () => { focusRow(active.value - 1) },
    k: () => { focusRow(active.value - 1) },
    x: () => { select(c, !selected.value.has(c.id)) },
    ' ': () => { select(c, !selected.value.has(c.id)) },
    a: () => { if (c.status === 'pending' && !isDeciding.value) void rows.value?.[active.value]?.add() },
    d: () => { if (c.status === 'pending' && !isDeciding.value) dismiss(c.id, c.name) },
    u: () => { if (c.status !== 'pending' && !isDeciding.value) undo(c.id, c.name) },
  }
  const run = act[event.key]
  if (run) {
    event.preventDefault()
    run()
  }
}
</script>

<template>
  <SettingsShell title="Competitor review">
    <template #lead>
      <p class="m-0 max-w-[62ch] text-lg text-ink-2">
        Companies the AI answers name that are not on the competitor list yet, the most named first. Names on one website
        are one company, and Haiku merges the other duplicates and checks which are competitors. You decide what joins the
        list.
      </p>
    </template>

    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">Only administrators can review competitors.</p>
      <p class="m-0 text-ink-2">Ask an administrator for the admin role in auth-api, then sign in again.</p>
    </div>
    <template v-else>
      <section v-if="review && counts" aria-label="Haiku's review" class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-2xl bg-mist p-4">
        <p class="m-0 font-medium">{{ reviewLine(review, counts.unchecked) }}</p>
        <RouterLink v-if="review.problem === 'noKey'" to="/settings" class="font-medium">Add one in Language models</RouterLink>
        <button
          v-else
          type="button"
          :disabled="review.running || isDeciding"
          class="min-h-11 rounded-full border-0 bg-black px-5 font-semibold text-white hover:bg-ink-2 disabled:opacity-60"
          @click="startReview"
        >
          Review with Haiku
        </button>
      </section>

      <div class="flex flex-wrap items-end gap-3">
        <label :for="`${id}-search`" class="flex min-w-60 flex-1 flex-col gap-1.5 text-sm font-medium">
          Search names and websites
          <input :id="`${id}-search`" v-model="search" type="search" class="h-11 rounded-xl border-2 border-silver bg-white px-3 text-base font-normal" />
        </label>
        <div role="group" aria-label="Show" class="flex flex-wrap gap-2">
          <button
            v-for="s in CANDIDATE_STATUSES"
            :key="s"
            type="button"
            :aria-pressed="status === s"
            class="inline-flex min-h-11 items-center rounded-full border-2 px-4 text-sm font-medium"
            :class="status === s ? 'border-black bg-black text-white' : 'border-grey bg-white text-black hover:border-black'"
            @click="show(s)"
          >
            {{ candidateTab(s, counts) }}
          </button>
        </div>
      </div>
      <div role="group" aria-label="Haiku's check" class="flex flex-wrap gap-2">
        <button
          v-for="f in CHECK_FILTERS"
          :key="f.value"
          type="button"
          :aria-pressed="filter === f.value"
          class="inline-flex min-h-9 items-center rounded-full border px-3 text-sm"
          :class="filter === f.value ? 'border-orange-ink bg-orange-pale font-semibold' : 'border-grey bg-white hover:border-black'"
          @click="filter = f.value"
        >
          {{ f.label }}
        </button>
      </div>

      <p class="m-0 font-medium" aria-live="polite">{{ notice }}</p>
      <ul v-if="results.length > 0" aria-label="Skipped" class="m-0 flex list-none flex-col gap-1 p-0 text-sm text-ink-2">
        <li v-for="r in results" :key="r.id">{{ r.name ?? 'One company' }}. {{ r.reason }}</li>
      </ul>
      <p v-if="decisionError" role="alert" class="m-0 font-medium text-orange-ink">{{ decisionError }}</p>
      <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
      <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the companies…</p>
      <p v-else-if="shown.length === 0" class="m-0 text-lg text-ink-2">Nothing here.</p>
      <template v-else>
        <div class="sticky top-0 z-10 flex flex-wrap items-center gap-3 border-b-2 border-black bg-white py-2">
          <label class="inline-flex items-center gap-2 text-sm font-medium">
            <input type="checkbox" class="size-4 accent-orange-ink" :checked="allShownSelected" @change="selectAllShown(($event.target as HTMLInputElement).checked)" />
            Select all {{ shown.length }} shown
          </label>
          <template v-if="chosen.length > 0">
            <span class="text-sm font-semibold">{{ chosen.length }} selected</span>
            <button
              v-if="status === 'pending'"
              type="button"
              :disabled="isDeciding"
              class="min-h-10 rounded-full border-0 bg-orange px-4 text-sm font-semibold text-black hover:bg-orange-soft disabled:opacity-70"
              @click="many('add')"
            >
              Add with their websites
            </button>
            <button
              v-if="status === 'pending'"
              type="button"
              :disabled="isDeciding"
              class="min-h-10 rounded-full border-2 border-grey bg-white px-4 text-sm font-medium hover:border-black disabled:opacity-70"
              @click="many('dismiss')"
            >
              Dismiss
            </button>
            <button type="button" class="border-0 bg-transparent p-0 text-sm font-medium underline underline-offset-4" @click="selected = new Set()">Clear</button>
          </template>
          <span class="ml-auto hidden text-xs text-muted sm:inline">Keys: ↑ and ↓ move, A adds, D dismisses, X selects, U undoes.</span>
        </div>
        <ul
          aria-label="Companies"
          class="m-0 flex list-none flex-col p-0"
          @focusin="listHasFocus = true"
          @focusout="listHasFocus = false"
        >
          <CandidateRow
            v-for="(c, i) in shown"
            :key="c.id"
            ref="rows"
            :candidate="c"
            :busy="isDeciding"
            :selected="selected.has(c.id)"
            :active="i === active"
            @focus="active = i"
            @select="(on) => select(c, on)"
            @add="(name, website) => add(c.id, name, website)"
            @dismiss="dismiss(c.id, c.name)"
            @undo="undo(c.id, c.name)"
            @split="(memberId, name) => split(memberId, name)"
          />
        </ul>
      </template>
    </template>
  </SettingsShell>
</template>
