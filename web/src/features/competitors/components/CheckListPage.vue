<script setup lang="ts">
import StepCard from '../../claude-steps/components/StepCard.vue'
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { PRIMARY, ROW_BUTTON } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useCompetitorChecks, useListDecision } from '../composables/useCompetitorChecks'
import { useCompetitors } from '../composables/useCompetitors'
import { haikuArchived, isFlagged, listCheckLine, verdictLabel, verdictReason } from '../domain/checks'

/**
 * Claude Haiku's check of the competitor list: it archives only the companies it is sure do not
 * compete, each with its reason and Restore, and leaves the ones it is unsure of to you.
 */
const { competitors } = useCompetitors()
const { checks, unchecked, review, isStarting, error: checkError, start } = useCompetitorChecks(ref(true))
const { isSaving, error: decisionError, notice, keep, archive, restore } = useListDecision()
const flagged = computed(() => competitors.value.filter((c) => !c.archived && c.role === 'competitor' && isFlagged(c, checks.value.get(c.id))))
const archived = computed(() => haikuArchived(competitors.value))
</script>

<template>
  <ManageLayout
    title="Check the list"
    purpose="Let Claude Haiku archive the companies that clearly do not compete."
    :back="{ to: '/manage/competitors', label: 'Back to Competitor list' }"
    :results="{ to: '/competitors', label: 'See the competitors' }"
  >
    <WidgetGrid>
      <ManageWidget title="Archived by Haiku" :span="8">
        <p class="m-0 text-ink-2">Each company keeps the reason it was archived, and you can restore any of them.</p>
        <p v-if="archived.length === 0" class="m-0 text-ink-2">Haiku has not archived any company.</p>
        <ul v-else aria-label="Archived by Haiku" class="m-0 flex list-none flex-col p-0">
          <li v-for="c in archived" :key="c.id" class="flex flex-wrap items-center justify-between gap-3 border-b border-grey py-3">
            <span class="flex min-w-0 flex-col">
              <RouterLink :to="`/manage/competitors/companies/${c.id}`" class="font-semibold">{{ c.name }}</RouterLink>
              <span class="text-sm text-ink-2">{{ verdictReason(checks.get(c.id)) }}</span>
            </span>
            <button type="button" :class="ROW_BUTTON" :disabled="isSaving" @click="restore(c)">Restore<span class="sr-only">{{ ` ${c.name}` }}</span></button>
          </li>
        </ul>
      </ManageWidget>

      <ManageWidget title="Check the list" :span="4">
        <p class="m-0 text-ink-2">
          Compares each site with what a competitor is. It archives only the companies it is sure do not compete, each with its reason.
        </p>
        <p v-if="review" class="m-0 font-medium" aria-live="polite">{{ listCheckLine(review, unchecked) }}</p>
        <p v-if="checkError" role="alert" class="m-0 font-medium text-orange-ink">{{ checkError }}</p>
        <template #foot>
          <RouterLink v-if="review?.problem === 'noKey'" to="/manage/keys/anthropic">Add an Anthropic key</RouterLink>
          <button v-else type="button" :class="PRIMARY" :disabled="review?.running || isStarting" @click="start">Check the list</button>
        </template>
      </ManageWidget>

      <ManageWidget v-if="flagged.length > 0" title="Haiku is unsure of these" :span="12">
        <p class="m-0 text-ink-2">Keep one on the list, and Haiku never archives it; or archive it yourself.</p>
        <ul aria-label="Companies Haiku is unsure of" class="m-0 flex list-none flex-col p-0">
          <li v-for="c in flagged" :key="c.id" class="flex flex-wrap items-center justify-between gap-3 border-b border-grey py-3">
            <span class="flex min-w-0 flex-col">
              <RouterLink :to="`/manage/competitors/companies/${c.id}`" class="font-semibold">{{ c.name }}</RouterLink>
              <span class="text-sm text-ink-2">{{ verdictLabel(checks.get(c.id)) }}. {{ verdictReason(checks.get(c.id)) }}</span>
            </span>
            <span role="group" :aria-label="`Decide on ${c.name}`" class="flex flex-wrap gap-2">
              <button type="button" :class="ROW_BUTTON" :disabled="isSaving" @click="keep(c)">Keep<span class="sr-only">{{ ` ${c.name}` }}</span></button>
              <button type="button" :class="ROW_BUTTON" :disabled="isSaving" @click="archive(c)">Archive<span class="sr-only">{{ ` ${c.name}` }}</span></button>
            </span>
          </li>
        </ul>
      </ManageWidget>

      <p class="m-0 w-full font-medium" aria-live="polite">{{ notice }}</p>
      <p v-if="decisionError" role="alert" class="m-0 w-full font-medium text-orange-ink">{{ decisionError }}</p>
      <StepCard step-id="candidateCheck" variant="widget" />
    </WidgetGrid>
  </ManageLayout>
</template>
