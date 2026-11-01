<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { TermCandidateStatus } from '../../../infrastructure/api/types.gen'
import LanguageFilter from '../../../shared/ui/LanguageFilter.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { CHOICE, CHOICE_OFF, CHOICE_ON, SECONDARY } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { languageChoices, parseLanguage, withLanguage } from '../../terminology/domain/languages'
import { useTermCandidates, useTermDecision } from '../composables/useTermReview'
import { candidatesIn, parseStatus, reviewLine, STATUSES, statusTab } from '../domain/candidates'
import CandidateItem from './CandidateItem.vue'

const route = useRoute()
const router = useRouter()
// The tab and the language live in the address, so a reload or a link shows the same list.
const status = computed<TermCandidateStatus>(() => parseStatus(route.query.status))
const language = computed(() => parseLanguage(route.query.language))
const { candidates, counts, review, isLoading, forbidden, error } = useTermCandidates(status)
const { isDeciding, notice, error: decisionError, accept, reject, undo } = useTermDecision()
const languages = computed(() => languageChoices(candidates.value.flatMap((c) => c.languages)))
const shown = computed(() => candidatesIn(candidates.value, language.value))

function go(next: TermCandidateStatus, lang: string | null): void {
  void router.replace({ query: withLanguage(next === 'pending' ? {} : { status: next }, lang) })
}

function show(next: TermCandidateStatus): void {
  go(next, language.value)
}

const FORBIDDEN = 'Only administrators can review terms.'
</script>

<template>
  <ManageLayout
    title="Waiting terms"
    purpose="New search terms that scans found, waiting for a decision."
    :back="{ to: '/manage/terms', label: 'Back to Search term list' }"
    :results="{ to: '/terms', label: 'See the search terms' }"
  >
    <div v-if="forbidden" role="alert" class="flex flex-col gap-2 rounded-2xl bg-mist p-6">
      <p class="m-0 text-xl font-semibold">{{ FORBIDDEN }}</p>
      <p class="m-0 text-ink-2">Ask an administrator for the admin role in auth-api, then sign in again.</p>
    </div>
    <WidgetGrid v-else>
      <ManageWidget title="Search terms" :span="12">
        <template #action>
          <RouterLink to="/manage/terms/review" :class="SECONDARY">Let Claude Haiku review them</RouterLink>
        </template>
        <p class="m-0 max-w-[70ch] text-ink-2">
          Claude Haiku adds the ones buyers search for to the list and leaves the rest out. The ones it is unsure about wait here for you.
          Accept adds a search term to the list, Reject leaves it off, and Undo puts it back among the waiting terms.
        </p>
        <p v-if="review" class="m-0 font-medium" aria-live="polite" data-part="review-line">{{ reviewLine(review) }}</p>
        <div role="group" aria-label="Show" class="flex flex-wrap gap-2">
          <button
            v-for="s in STATUSES"
            :key="s"
            type="button"
            :aria-pressed="status === s"
            :class="[CHOICE, status === s ? CHOICE_ON : CHOICE_OFF]"
            @click="show(s)"
          >
            {{ statusTab(s, counts) }}
          </button>
        </div>
        <LanguageFilter :choices="languages" :selected="language" @select="(l) => go(status, l)" />
        <p class="m-0 font-medium" aria-live="polite">{{ notice }}</p>
        <p v-if="decisionError" role="alert" class="m-0 font-medium text-orange-ink">{{ decisionError }}</p>
        <p v-if="error" role="alert" class="m-0 rounded-2xl bg-white p-6 text-lg">{{ error }}</p>
        <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the search terms…</p>
        <p v-else-if="shown.length === 0" class="m-0 text-lg text-ink-2">Nothing here.</p>
        <ul v-else aria-label="Candidates" class="m-0 flex list-none flex-col gap-4 p-0">
          <CandidateItem
            v-for="c in shown"
            :key="c.id"
            :candidate="c"
            :busy="isDeciding"
            @accept="(body) => accept(c.id, body)"
            @reject="reject(c.id, c.phrase)"
            @undo="undo(c.id, c.phrase)"
          />
        </ul>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
