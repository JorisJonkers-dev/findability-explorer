<script setup lang="ts">
import { ref } from 'vue'
import { useTermCandidates } from '../../review/composables/useTermReview'
import { reviewLine } from '../../review/domain/candidates'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import BigFigure from './widgets/BigFigure.vue'
import IndexCard from './widgets/IndexCard.vue'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/** The Search term list: how many terms it holds, how many wait, the review, and a card per subpage. */
const { total } = useTermMap()
const { counts, review } = useTermCandidates(ref('pending'))
</script>

<template>
  <ManageLayout title="Search term list" purpose="The search terms buyers use for metal conveyor belts." :results="{ to: '/terms', label: 'See the search terms' }">
    <WidgetGrid>
      <ManageWidget title="On the list" :span="4">
        <BigFigure :value="String(total)" label="search terms" />
      </ManageWidget>
      <ManageWidget title="Waiting terms" :span="4">
        <BigFigure :value="String(counts?.pending ?? 0)" label="Found by scans, not decided yet." />
      </ManageWidget>
      <ManageWidget title="Term review" :span="4">
        <p class="m-0 text-ink-2" aria-live="polite">{{ review ? reviewLine(review) : 'Loading…' }}</p>
      </ManageWidget>
      <IndexCard
        title="Term review"
        text="Claude Haiku decides which waiting search terms join the list, by itself or when you ask."
        to="/manage/terms/review"
        link="Open the term review"
        :span="6"
      />
      <IndexCard
        title="Waiting terms"
        text="New search terms that scans found. Accept the ones buyers use, and reject the rest."
        to="/manage/terms/waiting"
        link="Review the terms"
        :span="6"
      />
      <IndexCard
        title="Add or remove terms"
        text="Add a search term by hand, or take one off the list."
        to="/manage/terms/edit"
        link="Change the list"
        :span="6"
      />
    </WidgetGrid>
  </ManageLayout>
</template>
