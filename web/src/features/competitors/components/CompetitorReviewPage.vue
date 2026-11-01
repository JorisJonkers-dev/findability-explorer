<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId, useTemplateRef, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { CompetitorCandidate, CompetitorCandidateStatus } from '../../../infrastructure/api/types.gen'
import { useCandidateDecision, useCompetitorCandidates } from '../../prompt-test/composables/usePromptTest'
import { CANDIDATE_STATUSES, candidateTab, parseCandidateStatus } from '../../prompt-test/domain/settings'
import BigFigure from '../../manage/components/widgets/BigFigure.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { CHOICE, CHOICE_OFF, CHOICE_ON, FIELD, LABEL, PRIMARY } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { CHECK_FILTERS, type CheckFilter, idsOf, keptBy, matchesSearch, reviewLine } from '../domain/candidates'
import CandidateRow from './CandidateRow.vue'

/**
 * The companies AI answers name that are not on the competitor list, one per company: names on one
 * website, and names Haiku merged, are one. Haiku checks each against Tribelt's market and adds or
 * dismisses the ones it is sure about; a person decides the rest, one at a time, by keyboard, or
 * several at once, and can undo any decision.
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
  <ManageLayout
    title="Candidates"
    purpose="Review every company the AI responses named that is not on the list yet."
    :back="{ to: '/manage/competitors', label: 'Back to Competitor list' }"
    :results="{ to: '/competitors', label: 'See the competitors' }"
  >
    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">Only administrators can review competitors.</p>
      <p class="m-0 text-ink-2">Ask an administrator for the admin role in auth-api, then sign in again.</p>
    </div>
    <WidgetGrid v-else>
      <ManageWidget title="Waiting" :span="4">
        <BigFigure :value="String(counts?.pending ?? 0)" label="Not decided yet." />
      </ManageWidget>
      <ManageWidget title="Added" :span="4">
        <BigFigure :value="String(counts?.added ?? 0)" label="On the competitor list." />
      </ManageWidget>
      <ManageWidget title="Dismissed" :span="4">
        <BigFigure :value="String(counts?.dismissed ?? 0)" label="Not competitors." />
      </ManageWidget>

      <ManageWidget title="Classify automatically" :span="12">
        <p class="m-0 max-w-[70ch] text-ink-2">
          Names on one website are one company, and Haiku merges the other duplicates. Haiku adds the companies it is sure make metal
          process belts like Tribelt's, and dismisses the ones it is sure do not. You decide the rest, and you can undo any of its decisions.
        </p>
        <p v-if="review && counts" class="m-0 font-medium" aria-live="polite" data-part="review-line">{{ reviewLine(review, counts.unchecked) }}</p>
        <template v-if="review" #foot>
          <RouterLink v-if="review.problem === 'noKey'" to="/manage/keys/anthropic" class="font-medium">Add an Anthropic key in Keys and spending</RouterLink>
          <button v-else type="button" :class="PRIMARY" :disabled="review.running || isDeciding" @click="startReview">Classify automatically</button>
        </template>
      </ManageWidget>

      <ManageWidget title="All candidates" :span="12">
        <p class="m-0 text-ink-2">
          Change any decision here. Add, Dismiss, and Back to waiting work on every row, and Split divides one entry that holds two companies.
        </p>
        <div class="flex flex-wrap items-end gap-3">
          <label :for="`${id}-search`" :class="LABEL" class="min-w-60 flex-1">
            Search names and websites
            <input :id="`${id}-search`" v-model="search" type="search" :class="FIELD" />
          </label>
          <div role="group" aria-label="Show" class="flex flex-wrap gap-2">
            <button
              v-for="s in CANDIDATE_STATUSES"
              :key="s"
              type="button"
              :aria-pressed="status === s"
              :class="[CHOICE, status === s ? CHOICE_ON : CHOICE_OFF]"
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
        <p v-if="error" role="alert" class="m-0 rounded-2xl bg-white p-6 text-lg">{{ error }}</p>
        <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the companies…</p>
        <p v-else-if="shown.length === 0" class="m-0 text-lg text-ink-2">Nothing here.</p>
        <div v-else class="flex min-w-0 flex-col rounded-2xl bg-white px-3 sm:px-4">
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
        </div>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
