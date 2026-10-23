<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { TermCandidateStatus } from '../../../infrastructure/api/types.gen'
import AccountBar from '../../shell/components/AccountBar.vue'
import SectionEyebrow from '../../shell/components/SectionEyebrow.vue'
import { useTermCandidates, useTermDecision } from '../composables/useTermReview'
import { parseStatus, STATUSES, statusTab } from '../domain/candidates'
import CandidateItem from './CandidateItem.vue'

const route = useRoute()
const router = useRouter()
// The tab lives in the address, so a reload or a link shows the same one.
const status = computed<TermCandidateStatus>(() => parseStatus(route.query.status))
const { candidates, counts, isLoading, forbidden, error } = useTermCandidates(status)
const { isDeciding, notice, error: decisionError, accept, reject, undo } = useTermDecision()

function show(next: TermCandidateStatus): void {
  void router.replace({ query: next === 'pending' ? {} : { status: next } })
}

const FORBIDDEN = 'Only administrators can review terms.'
</script>

<template>
  <div class="mx-auto flex max-w-5xl flex-col gap-10 px-4 pt-5 pb-20 sm:px-8 lg:pt-10">
    <header class="flex flex-col gap-8">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <RouterLink
          to="/settings"
          class="inline-flex min-h-11 items-center gap-2 rounded-full bg-black px-5 font-medium text-white no-underline hover:bg-ink-2 hover:text-white"
        >
          <svg aria-hidden="true" viewBox="0 0 24 24" class="size-5" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M15 5 8 12l7 7" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          Settings
        </RouterLink>
        <AccountBar tone="light" />
      </div>
      <div class="flex flex-col gap-4">
        <SectionEyebrow>Settings</SectionEyebrow>
        <h1 class="m-0 font-display text-4xl leading-[0.95] font-semibold tracking-[-0.03em] sm:text-6xl">Term review</h1>
        <p class="m-0 max-w-[62ch] text-lg text-ink-2">
          Phrases the scans found that the terminology map does not hold yet, with the companies that use them. An
          accepted term joins the map, its search and the next scans; a rejected one stays out until the decision is
          undone.
        </p>
      </div>
    </header>

    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">{{ FORBIDDEN }}</p>
      <p class="m-0 text-ink-2">Ask an administrator for the admin role in auth-api, then sign in again.</p>
    </div>
    <template v-else>
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
  </div>
</template>
