<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { ACTIONS, AREA, FIELD, LABEL, PRIMARY } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { usePlan } from '../composables/usePlan'
import { type Draft, draftOf, editable, editedLine, toEdit } from '../domain/editing'

/** One recommendation a Recommendation run wrote: its wording and scores, with Save. */
const props = defineProps<{ id: string }>()
const { plan, isLoading, error, isSaving, saveError, edit } = usePlan()
const it = computed(() => editable(plan.value?.improvements ?? []).find((i) => i.id === props.id))
const draft = ref<Draft | null>(null)
const notice = ref<string | null>(null)
watch(
  it,
  (now) => {
    if (now && !draft.value) draft.value = draftOf(now)
  },
  { immediate: true },
)
const SCORES = [1, 2, 3, 4, 5]

function save(): void {
  if (!draft.value) return
  notice.value = null
  edit(props.id, toEdit(draft.value), () => {
    notice.value = 'Saved. A new Recommendation run keeps this recommendation.'
  })
}
</script>

<template>
  <ManageLayout
    :title="it?.title ?? 'Edit a recommendation'"
    help-topic="edit-recommendation"
    purpose="Change the wording and scores. An edited recommendation is kept when the next run replaces the others."
    :back="{ to: '/manage/recommendations/edit', label: 'Back to Edit a recommendation' }"
    :results="{ to: `/recommendations/${id}`, label: 'See the recommendation' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the recommendation…</p>
    <p v-else-if="!it || !draft" class="m-0 text-ink-2">No recommendation a Recommendation run wrote has this address.</p>
    <WidgetGrid v-else>
      <ManageWidget title="Wording and scores" :span="12">
        <p class="m-0 text-ink-2">{{ editedLine(it) }}</p>
        <label :class="LABEL">Title<input v-model="draft.title" name="title" :class="FIELD" maxlength="200" /></label>
        <label :class="LABEL">Page on tribelt.nl, or blank for the web agency to decide<input v-model="draft.where" name="where" :class="FIELD" maxlength="500" /></label>
        <label :class="LABEL">Why<textarea v-model="draft.why" name="why" :class="AREA" maxlength="500" /></label>
        <label :class="LABEL">Steps, one to a line<textarea v-model="draft.steps" name="steps" :class="AREA" /></label>
        <label :class="LABEL">When it counts as done<input v-model="draft.doneWhen" name="doneWhen" :class="FIELD" maxlength="300" /></label>
        <div class="flex flex-wrap gap-4">
          <label :class="LABEL">Benefit
            <select v-model.number="draft.benefit" name="benefit" :class="FIELD">
              <option v-for="n in SCORES" :key="n" :value="n">{{ n }} of 5</option>
            </select>
          </label>
          <label :class="LABEL">Effort
            <select v-model.number="draft.effort" name="effort" :class="FIELD">
              <option v-for="n in SCORES" :key="n" :value="n">{{ n }} of 5</option>
            </select>
          </label>
        </div>
        <div :class="ACTIONS">
          <button type="button" :class="PRIMARY" :disabled="isSaving" @click="save">Save</button>
        </div>
        <p class="m-0 font-medium" aria-live="polite">{{ notice }}</p>
        <p v-if="saveError" role="alert" class="m-0 font-medium text-orange-ink">{{ saveError }}</p>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
