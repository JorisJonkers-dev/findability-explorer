<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { usePlan } from '../composables/usePlan'
import { editable } from '../domain/editing'
import { KIND_LABEL, kindOf, scoreLine } from '../domain/recommendations'

/** The recommendations a Recommendation run wrote, at full width; one opens its wording and scores. */
const { plan, isLoading, error } = usePlan()
const items = computed(() => editable(plan.value?.improvements ?? []))
</script>

<template>
  <ManageLayout
    title="Edit a recommendation"
    purpose="Change the wording and scores of a recommendation a Recommendation run wrote. An edited one is kept when the next run replaces the others."
    :back="{ to: '/manage/recommendations', label: 'Back to Recommendations' }"
    :results="{ to: '/recommendations', label: 'See the recommendations' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the recommendations…</p>
    <WidgetGrid v-else>
      <ManageWidget title="Written by a Recommendation run" :span="12">
        <p v-if="items.length === 0" class="m-0 text-ink-2">No Recommendation run has written any yet.</p>
        <ul v-else aria-label="Written by a Recommendation run" class="m-0 flex list-none flex-col p-0">
          <li v-for="it in items" :key="it.id" class="flex flex-col gap-1 border-b border-grey py-3">
            <RouterLink :to="`/manage/recommendations/edit/${it.id}`" class="font-semibold">{{ it.title }}</RouterLink>
            <span class="flex flex-wrap gap-x-4 gap-y-1 text-sm text-ink-2">
              <span>{{ KIND_LABEL[kindOf(it)] }}</span>
              <span>{{ scoreLine(it) }}</span>
              <span v-if="it.editedAt">Edited</span>
              <span v-if="it.done">Done</span>
            </span>
          </li>
        </ul>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
