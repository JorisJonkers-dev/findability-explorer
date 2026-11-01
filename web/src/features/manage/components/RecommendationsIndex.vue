<script setup lang="ts">
import StepCard from '../../claude-steps/components/StepCard.vue'
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { usePlan } from '../../plan/composables/usePlan'
import { useRecommendationRuns } from '../../plan/composables/useRecommendationRuns'
import { toTickOff } from '../../plan/domain/marking'
import { estimateLine, readsLine, replacesLine, runRow } from '../../plan/domain/runs'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import BigFigure from './widgets/BigFigure.vue'
import IndexCard from './widgets/IndexCard.vue'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'
import { DANGER, PRIMARY, TABLE, TABLE_BOX, TD, TH } from './widgets/look'

/**
 * Recommendations in Manage: how many are open, the Recommendation run with what it reads,
 * replaces and costs, and its past runs, and the cards for ticking off and the goals.
 */
const { plan } = usePlan()
const { runs, isStarting, isStopping, notice, error, start, stop } = useRecommendationRuns()
const items = computed(() => plan.value?.improvements ?? [])
const open = computed(() => items.value.filter((i) => !i.done).length)
const waiting = computed(() => toTickOff(items.value).length)
const past = computed(() => (runs.value?.runs ?? []).map((r) => ({ id: r.id, ...runRow(r) })))
</script>

<template>
  <ManageLayout
    title="Recommendations"
    purpose="Write new recommendations from the latest data, mark them as done, and set the goals."
    :results="{ to: '/recommendations', label: 'See the recommendations' }"
  >
    <WidgetGrid>
      <ManageWidget v-if="runs?.stale" title="The page text is older than the responses" :span="12" accent>
        <p class="m-0 text-ink-2">
          tribelt.nl was last scanned before the newest Prompt run. <RouterLink to="/manage/scans" class="underline">Scan tribelt.nl first</RouterLink>, or start the run anyway.
        </p>
      </ManageWidget>

      <ManageWidget title="Start a Recommendation run" :span="8">
        <template v-if="runs">
          <p class="m-0 text-ink-2">{{ readsLine(runs) }}</p>
          <p class="m-0 font-semibold">{{ replacesLine(runs.replaces) }}</p>
          <p class="m-0 text-ink-2">{{ estimateLine(runs) }}</p>
          <p class="m-0 text-sm text-ink-2">
            Claude writes them as the step <RouterLink to="/manage/claude-steps/recommendations" class="underline">Writing recommendations</RouterLink> says.
            <template v-if="runs.model">It uses {{ runs.model }}.</template>
          </p>
        </template>
        <p class="m-0 w-full font-medium" aria-live="polite">{{ notice }}</p>
        <p v-if="error" role="alert" class="m-0 font-medium text-orange-ink">{{ error }}</p>
        <template #foot>
          <RouterLink v-if="runs?.noKey" to="/manage/keys/anthropic">Add an Anthropic key</RouterLink>
          <button v-else-if="runs?.running" type="button" :class="DANGER" :disabled="isStopping" @click="stop">Stop the Recommendation run</button>
          <button v-else type="button" :class="PRIMARY" :disabled="isStarting || !runs || runs.gaps === 0" @click="start">Start a Recommendation run</button>
        </template>
      </ManageWidget>

      <ManageWidget title="Open recommendations" :span="4">
        <BigFigure :value="String(open)" :label="`${String(items.length - open)} done`" />
      </ManageWidget>

      <ManageWidget title="Past runs" :span="12">
        <p class="m-0 text-ink-2">What each run added, replaced, and kept.</p>
        <p v-if="past.length === 0" class="m-0 text-ink-2">No Recommendation run has run yet.</p>
        <div v-else :class="TABLE_BOX">
          <table :class="TABLE" aria-label="Past runs">
            <thead>
              <tr>
                <th :class="TH" scope="col">When</th>
                <th :class="TH" scope="col">How it ended</th>
                <th :class="TH" scope="col">Model</th>
                <th :class="TH" scope="col">Added</th>
                <th :class="TH" scope="col">Replaced</th>
                <th :class="TH" scope="col">Kept</th>
                <th :class="TH" scope="col">Cost</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in past" :key="r.id">
                <td :class="TD">{{ r.when }}</td>
                <td :class="TD">{{ r.ending }}</td>
                <td :class="TD">{{ r.model }}</td>
                <td :class="TD">{{ r.added }}</td>
                <td :class="TD">{{ r.replaced }}</td>
                <td :class="TD">{{ r.kept }}</td>
                <td :class="TD">{{ r.cost }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </ManageWidget>

      <IndexCard
        title="Tick off yourself"
        text="Recommendations no scan can check, so you say when they are done."
        to="/manage/recommendations/tick-off"
        link="Tick them off"
        :figures="[{ value: String(waiting), label: 'waiting' }]"
        :span="6"
      />
      <IndexCard title="Goals" text="The five goals Tribelt is measured against." to="/manage/recommendations/goals" link="Change the goals" :span="6" />
      <StepCard step-id="recommendations" />
    </WidgetGrid>
  </ManageLayout>
</template>
