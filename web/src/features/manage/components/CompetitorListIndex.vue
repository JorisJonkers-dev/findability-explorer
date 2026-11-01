<script setup lang="ts">
import StepCard from '../../claude-steps/components/StepCard.vue'
import { computed, ref } from 'vue'
import { useCompetitorChecks } from '../../competitors/composables/useCompetitorChecks'
import { useCompetitorDefinition } from '../../competitors/composables/useCompetitorDefinition'
import { useCompetitors } from '../../competitors/composables/useCompetitors'
import { haikuArchived } from '../../competitors/domain/checks'
import { listCompanies } from '../../competitors/domain/companies'
import { useCompetitorCandidates } from '../../prompt-test/composables/usePromptTest'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { RouterLink } from 'vue-router'
import { PRIMARY } from './widgets/look'
import BigFigure from './widgets/BigFigure.vue'
import IndexCard from './widgets/IndexCard.vue'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/** The Competitor list: who counts, the candidates waiting, the checks running, and a card per subpage. */
const { competitors } = useCompetitors()
const { counts, review } = useCompetitorCandidates(ref('pending'))
const checks = useCompetitorChecks(ref(true))
const { definition } = useCompetitorDefinition(ref(true))
const changed = computed(() => (definition.value ? definition.value.outdatedCompetitors + definition.value.outdatedCandidates > 0 : false))
const listed = computed(() => listCompanies(competitors.value))
const active = computed(() => listed.value.filter((c) => !c.archived).length)
const decided = computed(() => (counts.value ? counts.value.added + counts.value.dismissed : 0))
const checking = computed(() => {
  const running = [review.value?.running ? 'Haiku is reviewing the candidates.' : '', checks.review.value?.running ? 'Haiku is checking the list.' : ''].filter((t) => t !== '')
  return running.length > 0 ? running.join(' ') : 'No check is running.'
})
</script>

<template>
  <ManageLayout title="Competitor list" purpose="Decide which companies count as competitors." :results="{ to: '/competitors', label: 'See the competitors' }">
    <WidgetGrid>
      <ManageWidget v-if="changed" title="The definition changed" :span="12" accent>
        <p class="m-0 text-ink-2">The list and the candidates were checked against an older version of the definition.</p>
        <template #foot>
          <RouterLink to="/manage/competitors/definition" :class="PRIMARY">Check them again</RouterLink>
        </template>
      </ManageWidget>
      <ManageWidget title="Competitors" :span="4">
        <BigFigure :value="String(active)" :label="`${String(listed.length - active)} archived`" />
      </ManageWidget>
      <ManageWidget title="Waiting candidates" :span="4">
        <BigFigure :value="String(counts?.pending ?? 0)" label="Companies the AI responses named." />
      </ManageWidget>
      <ManageWidget title="Checks now" :span="4">
        <p class="m-0 text-ink-2" aria-live="polite">{{ checking }}</p>
      </ManageWidget>
      <IndexCard
        title="Companies"
        text="The companies that count as competitors. Open one to edit it, archive it, or restore it."
        to="/manage/competitors/companies"
        link="See the companies"
      />
      <IndexCard
        title="Candidates"
        text="Companies the AI responses named. Review every decision, or let Claude Haiku classify them."
        to="/manage/competitors/candidates"
        link="Review the candidates"
        :figures="[{ value: String(decided), label: 'decided' }]"
      />
      <IndexCard
        title="Check the list"
        text="Haiku reads each site and archives only the companies it is sure do not compete."
        to="/manage/competitors/check"
        link="Check the list"
        :figures="[{ value: String(haikuArchived(competitors).length), label: 'archived by Haiku' }]"
      />
      <IndexCard
        title="Competitor definition"
        text="The text Haiku reads to decide whether a company competes."
        to="/manage/competitors/definition"
        link="Change the definition"
        :figures="definition ? [{ value: `Version ${String(definition.current.version)}`, label: 'in use' }] : []"
      />
      <StepCard step-id="candidateMerge" />
      <StepCard step-id="candidateCheck" />
    </WidgetGrid>
  </ManageLayout>
</template>
