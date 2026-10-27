<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { CompetitorCandidateStatus } from '../../../infrastructure/api/types.gen'
import { useCandidateDecision, useCompetitorCandidates } from '../../prompt-test/composables/usePromptTest'
import { CANDIDATE_STATUSES, candidateTab, parseCandidateStatus } from '../../prompt-test/domain/settings'
import SettingsShell from '../../shell/components/SettingsShell.vue'
import CandidateCard from './CandidateCard.vue'

/** The companies AI answers name that are not on the competitor list, for a person to check. */
const route = useRoute()
const router = useRouter()
// The tab lives in the address, so a reload or a link shows the same one.
const status = computed<CompetitorCandidateStatus>(() => parseCandidateStatus(route.query.status))
const { candidates, counts, isLoading, forbidden, error } = useCompetitorCandidates(status)
const { isDeciding, notice, error: decisionError, add, dismiss, undo } = useCandidateDecision()

function show(next: CompetitorCandidateStatus): void {
  void router.replace({ query: next === 'pending' ? {} : { status: next } })
}
</script>

<template>
  <SettingsShell title="Competitor review">
    <template #lead>
      <p class="m-0 max-w-[62ch] text-lg text-ink-2">
        Companies the AI answers name that are not on the competitor list yet, the most named first. An added company joins
        the list, where it can be scanned. A dismissed one stays out until the decision is undone.
      </p>
    </template>

    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">Only administrators can review competitors.</p>
      <p class="m-0 text-ink-2">Ask an administrator for the admin role in auth-api, then sign in again.</p>
    </div>
    <template v-else>
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
      <p class="m-0 font-medium" aria-live="polite">{{ notice }}</p>
      <p v-if="decisionError" role="alert" class="m-0 font-medium text-orange-ink">{{ decisionError }}</p>
      <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
      <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the companies…</p>
      <p v-else-if="candidates.length === 0" class="m-0 text-lg text-ink-2">Nothing here.</p>
      <ul v-else aria-label="Companies" class="m-0 flex list-none flex-col gap-4 p-0">
        <CandidateCard
          v-for="c in candidates"
          :key="c.id"
          :candidate="c"
          :busy="isDeciding"
          @add="(name, website) => add(c.id, name, website)"
          @dismiss="dismiss(c.id, c.name)"
          @undo="undo(c.id, c.name)"
        />
      </ul>
    </template>
  </SettingsShell>
</template>
