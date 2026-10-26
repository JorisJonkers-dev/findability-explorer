<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { CompetitorCandidate, CompetitorCandidateStatus } from '../../../infrastructure/api/types.gen'
import { atLabel } from '../../settings/domain/time'
import { useCandidateDecision, useCompetitorCandidates } from '../composables/usePromptTest'
import { providerName } from '../domain/results'
import { CANDIDATE_STATUSES, candidateTab, candidateWebsite, parseCandidateStatus } from '../domain/settings'
import SettingsShell from './SettingsShell.vue'

const route = useRoute()
const router = useRouter()
// The tab lives in the address, so a reload or a link shows the same one.
const status = computed<CompetitorCandidateStatus>(() => parseCandidateStatus(route.query.status))
const { candidates, counts, isLoading, forbidden, error } = useCompetitorCandidates(status)
const { isDeciding, notice, error: decisionError, add, dismiss, undo } = useCandidateDecision()

/** The candidate being added, with the name and website it is added under. */
const adding = ref<{ id: string; name: string; url: string } | null>(null)

function show(next: CompetitorCandidateStatus): void {
  void router.replace({ query: next === 'pending' ? {} : { status: next } })
}

function openAdd(c: CompetitorCandidate): void {
  adding.value = { id: c.id, name: c.name, url: candidateWebsite(c) }
}

function submitAdd(): void {
  if (!adding.value) return
  add(adding.value.id, adding.value.name.trim(), adding.value.url.trim())
  adding.value = null
}
</script>

<template>
  <SettingsShell title="Competitor candidates">
    <template #lead>
      <p class="m-0 max-w-[62ch] text-lg text-ink-2">
        Companies the Prompt runs' answers name that are not on the company list. Add one to scan it like any competitor,
        or dismiss it, as a distributor or no competitor; a dismissed name stays dismissed in later runs. Both can be undone.
      </p>
    </template>

    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">Only administrators can review competitor candidates.</p>
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
      <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the candidates…</p>
      <p v-else-if="candidates.length === 0" class="m-0 text-lg text-ink-2">Nothing here.</p>
      <ul v-else aria-label="Candidates" class="m-0 flex list-none flex-col gap-4 p-0">
        <li v-for="c in candidates" :key="c.id" class="flex flex-col gap-3 rounded-2xl bg-mist p-5 sm:p-6" :aria-labelledby="`cc-${c.id}`">
          <div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
            <h3 :id="`cc-${c.id}`" class="m-0 text-xl font-semibold">{{ c.name }}</h3>
            <span class="text-sm text-ink-2">Named in {{ c.answers }} {{ c.answers === 1 ? 'answer' : 'answers' }}</span>
          </div>
          <p class="m-0 text-sm text-ink-2">
            By {{ c.providers.map(providerName).join(', ') }} · last on {{ atLabel(c.lastSeenAt) }}
          </p>
          <p v-if="c.models.length > 0" class="m-0 text-sm"><span class="text-muted">Models: </span>{{ c.models.join(', ') }}</p>
          <p v-if="c.addresses.length > 0" class="m-0 text-sm break-all"><span class="text-muted">Addresses given: </span>{{ c.addresses.join(', ') }}</p>
          <p v-if="c.decision" class="m-0 text-sm font-medium">
            {{ c.decision.status === 'added' ? 'Added' : 'Dismissed' }} by {{ c.decision.by }} on {{ atLabel(c.decision.at) }}.
          </p>

          <form
            v-if="adding?.id === c.id"
            novalidate
            :aria-label="`Add ${c.name}`"
            class="flex flex-col gap-3 rounded-xl bg-white p-4"
            @submit.prevent="submitAdd"
          >
            <div class="grid gap-3 sm:grid-cols-2">
              <label class="flex flex-col gap-1.5 text-sm font-medium">
                Name
                <input v-model="adding.name" type="text" required class="h-11 rounded-xl border-2 border-grey px-3 text-base" />
              </label>
              <label class="flex flex-col gap-1.5 text-sm font-medium">
                Website
                <input v-model="adding.url" type="url" required placeholder="https://" class="h-11 rounded-xl border-2 border-grey px-3 text-base" />
              </label>
            </div>
            <div class="flex flex-wrap gap-3">
              <button type="submit" :disabled="isDeciding" class="min-h-11 rounded-full border-0 bg-black px-5 font-semibold text-white">Add to the competitors</button>
              <button type="button" class="min-h-11 rounded-full border-2 border-grey bg-white px-5 font-medium" @click="adding = null">Cancel</button>
            </div>
          </form>
          <div v-else class="flex flex-wrap gap-3">
            <template v-if="!c.decision">
              <button type="button" :disabled="isDeciding" class="min-h-11 rounded-full border-0 bg-orange px-5 font-semibold text-black" @click="openAdd(c)">
                Add
              </button>
              <button type="button" :disabled="isDeciding" class="min-h-11 rounded-full border-2 border-black bg-white px-5 font-medium" @click="dismiss(c.id, c.name)">
                Dismiss
              </button>
            </template>
            <button v-else type="button" :disabled="isDeciding" class="min-h-11 rounded-full border-2 border-black bg-white px-5 font-medium" @click="undo(c.id, c.name)">
              Undo
            </button>
          </div>
        </li>
      </ul>
    </template>
  </SettingsShell>
</template>
