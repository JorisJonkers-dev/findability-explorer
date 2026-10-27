<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { TermCandidateStatus } from '../../../infrastructure/api/types.gen'
import SettingsShell from '../../shell/components/SettingsShell.vue'
import { useStartReview, useTermCandidates, useTermDecision } from '../composables/useTermReview'
import { parseStatus, reviewLine, STATUSES, statusTab } from '../domain/candidates'
import CandidateItem from './CandidateItem.vue'

const route = useRoute()
const router = useRouter()
// The tab lives in the address, so a reload or a link shows the same one.
const status = computed<TermCandidateStatus>(() => parseStatus(route.query.status))
const { candidates, counts, review, isLoading, forbidden, error } = useTermCandidates(status)
const { isStarting, error: startError, start } = useStartReview()
const { isDeciding, notice, error: decisionError, accept, reject, undo } = useTermDecision()

function show(next: TermCandidateStatus): void {
  void router.replace({ query: next === 'pending' ? {} : { status: next } })
}

const FORBIDDEN = 'Only administrators can review terms.'
</script>

<template>
  <SettingsShell title="Term review">
    <template #lead>
      <p class="m-0 max-w-[62ch] text-lg text-ink-2">
        A language model reviews every phrase the scans find that the terminology map does not hold yet. It adds the
        ones buyers search for to the map and leaves the rest out. The phrases it is unsure about wait here for you,
        and any of its decisions can be undone.
      </p>
    </template>

    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">{{ FORBIDDEN }}</p>
      <p class="m-0 text-ink-2">Ask an administrator for the admin role in auth-api, then sign in again.</p>
    </div>
    <template v-else>
      <section v-if="review" aria-label="The language model's review" class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-2xl bg-mist p-4 sm:p-5">
        <p class="m-0 font-medium" aria-live="polite">{{ reviewLine(review) }}</p>
        <RouterLink v-if="review.problem === 'noModel'" to="/settings" class="font-medium">Add one in Language models</RouterLink>
        <button
          v-else-if="!review.running && review.waiting > 0"
          type="button"
          :disabled="isStarting"
          class="min-h-11 rounded-full border-0 bg-orange px-5 font-semibold text-black hover:bg-orange-soft disabled:opacity-70"
          @click="start"
        >
          Review them now
        </button>
        <p v-if="startError" role="alert" class="m-0 w-full font-medium text-orange-ink">{{ startError }}</p>
      </section>
      <div role="group" aria-label="Show" class="flex flex-wrap gap-2">
        <button
          v-for="s in STATUSES"
          :key="s"
          type="button"
          :aria-pressed="status === s"
          class="inline-flex min-h-11 items-center rounded-full border-2 px-4 text-sm font-medium"
          :class="status === s ? 'border-black bg-black text-white' : 'border-grey bg-white text-black hover:border-black'"
          @click="show(s)"
        >
          {{ statusTab(s, counts) }}
        </button>
      </div>
      <p class="m-0 font-medium" aria-live="polite">{{ notice }}</p>
      <p v-if="decisionError" role="alert" class="m-0 font-medium text-orange-ink">{{ decisionError }}</p>
      <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
      <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the candidates…</p>
      <p v-else-if="candidates.length === 0" class="m-0 text-lg text-ink-2">Nothing here.</p>
      <ul v-else aria-label="Candidates" class="m-0 flex list-none flex-col gap-4 p-0">
        <CandidateItem
          v-for="c in candidates"
          :key="c.id"
          :candidate="c"
          :busy="isDeciding"
          @accept="(body) => accept(c.id, body)"
          @reject="reject(c.id, c.phrase)"
          @undo="undo(c.id, c.phrase)"
        />
      </ul>
    </template>
  </SettingsShell>
</template>
