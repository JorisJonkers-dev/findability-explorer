<script setup lang="ts">
import { ref, watch } from 'vue'
import BigFigure from '../../manage/components/widgets/BigFigure.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { ACTIONS, FIELD, LABEL, PRIMARY } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { usePlan } from '../composables/usePlan'
import { checkGoals, type GoalField, goalFields } from '../domain/marking'

/** The five values Tribelt is measured against, with Save. */
const { plan, isLoading, error, isSaving, saveError, setTargets } = usePlan()
const fields = ref<GoalField[]>([])
const problem = ref<string | null>(null)
const notice = ref<string | null>(null)

// The form starts from the stored goals once they load; a save answers with them again.
watch(
  plan,
  (p) => {
    if (p) fields.value = goalFields(p.watched, p.readinessChecks)
  },
  { immediate: true },
)

function save(): void {
  notice.value = null
  const check = checkGoals(fields.value)
  if (check.kind === 'invalid') {
    problem.value = check.message
    return
  }
  problem.value = null
  setTargets(check.values)
  notice.value = 'The goals are saved.'
}
</script>

<template>
  <ManageLayout
    title="Goals"
    purpose="The five values Tribelt is measured against."
    :back="{ to: '/manage/recommendations', label: 'Back to Recommendations' }"
    :results="{ to: '/recommendations', label: 'See the recommendations' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the goals…</p>
    <WidgetGrid v-else>
      <ManageWidget title="The five goals" :span="8">
        <form novalidate aria-label="The five goals" class="flex flex-col gap-3" @submit.prevent="save">
          <label v-for="f in fields" :key="f.target" :class="LABEL">
            {{ f.label }}
            <input v-model="f.value" type="text" inputmode="decimal" :class="FIELD" />
          </label>
          <p v-if="problem" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ problem }}</p>
          <p v-if="saveError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ saveError }}</p>
          <div :class="ACTIONS">
            <button type="submit" :class="PRIMARY" :disabled="isSaving">Save the goals</button>
            <span class="font-medium" aria-live="polite">{{ isSaving || saveError ? '' : notice }}</span>
          </div>
        </form>
      </ManageWidget>
      <ManageWidget title="How they are used" :span="4">
        <BigFigure :value="String(fields.length)" label="goals" />
        <p class="m-0 text-ink-2">The Overview and the goal pages compare Tribelt with these values. Type shares as percentages.</p>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
