<script setup lang="ts">
import StepCard from '../../claude-steps/components/StepCard.vue'
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import BigFigure from '../../manage/components/widgets/BigFigure.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { CHOICE, CHOICE_OFF, CHOICE_ON, DANGER, PRIMARY } from '../../manage/components/widgets/look'
import { dollars } from '../../settings/domain/usage'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useStartReview, useTermCandidates, useTermReviewControl } from '../composables/useTermReview'
import { reviewLine } from '../domain/candidates'

/** The term review: Claude Haiku decides which waiting search terms join the list, when you ask. */
const { counts, review, forbidden, error } = useTermCandidates(ref('pending'))
const { isStarting, error: startError, start } = useStartReview()
const control = useTermReviewControl()
</script>

<template>
  <ManageLayout
    title="Term review"
    purpose="Let Claude Haiku decide which new search terms join the list."
    :back="{ to: '/manage/terms', label: 'Back to Search term list' }"
    :results="{ to: '/terms', label: 'See the search terms' }"
  >
    <p v-if="forbidden" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">Only administrators can review terms.</p>
    <p v-else-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <WidgetGrid v-else>
      <ManageWidget title="Review now" :span="6">
        <p class="m-0 text-ink-2">Decides now for every waiting search term, without waiting for the next scan.</p>
        <BigFigure :value="String(review?.waiting ?? 0)" label="search terms Claude Haiku has not reviewed yet" />
        <p v-if="review" class="m-0 font-medium" aria-live="polite">{{ reviewLine(review) }}</p>
        <p v-if="startError" role="alert" class="m-0 font-medium text-orange-ink">{{ startError }}</p>
        <template #foot>
          <RouterLink v-if="review?.problem === 'noModel'" to="/manage/keys/anthropic">Add an Anthropic key</RouterLink>
          <button v-else-if="review?.running" type="button" :class="DANGER" :disabled="control.isBusy.value" @click="control.stop()">Stop the review</button>
          <button v-else type="button" :class="PRIMARY" :disabled="!review || review.waiting === 0 || isStarting" @click="start">Review now</button>
        </template>
      </ManageWidget>
      <ManageWidget title="Automatic review" :span="6">
        <template #action>
          <span data-part="switch-state" class="rounded-full bg-white px-3 py-0.5 text-sm font-semibold">{{ review?.on ? 'On' : 'Off' }}</span>
        </template>
        <p class="m-0 text-ink-2">When on, Claude Haiku checks the waiting search terms every ten minutes and accepts or rejects them on its own.</p>
        <div role="group" aria-label="Automatic review" class="flex flex-wrap gap-2">
          <button type="button" :aria-pressed="review?.on === true" :class="[CHOICE, review?.on ? CHOICE_ON : CHOICE_OFF]" :disabled="!review || control.isBusy.value" @click="control.setOn(true)">On</button>
          <button type="button" :aria-pressed="review?.on === false" :class="[CHOICE, review?.on === false ? CHOICE_ON : CHOICE_OFF]" :disabled="!review || control.isBusy.value" @click="control.setOn(false)">Off</button>
        </div>
        <p v-if="control.error.value" role="alert" class="m-0 font-medium text-orange-ink">{{ control.error.value }}</p>
      </ManageWidget>
      <ManageWidget title="Spent this month" :span="6">
        <BigFigure :value="dollars(review?.monthCostUsd ?? 0)" label="What the term review cost so far, paid from the Anthropic key." />
      </ManageWidget>
      <ManageWidget title="Waiting for you" :span="6">
        <BigFigure :value="String(counts?.pending ?? 0)" label="search terms wait for a decision" />
        <p class="m-0 text-ink-2">The ones Claude Haiku is unsure about wait for you, and any of its decisions can be undone.</p>
        <template #foot>
          <RouterLink to="/manage/terms/waiting" class="font-medium">Review the terms</RouterLink>
        </template>
      </ManageWidget>
      <StepCard step-id="termReview" variant="widget" />
    </WidgetGrid>
  </ManageLayout>
</template>
