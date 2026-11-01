<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { usePromptSettings, useUpdatePromptSettings } from '../../prompt-test/composables/usePromptTest'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { whenLabel } from '../domain/activity'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/** The fact sheet the responses about Tribelt are checked against; each save is a new version. */
const { settings, isLoading, error } = usePromptSettings()
const { isSaving, notice, error: saveError, save } = useUpdatePromptSettings()

const text = ref('')
watch(
  settings,
  (s) => {
    if (s) text.value = s.factSheet.text
  },
  { immediate: true },
)
const sheet = computed(() => settings.value?.factSheet)
const dirty = computed(() => sheet.value !== undefined && text.value !== sheet.value.text)

function submit(): void {
  if (!sheet.value) return
  save({ factSheet: text.value }, `The fact sheet is saved as version ${String(sheet.value.version + 1)}.`)
}
</script>

<template>
  <ManageLayout
    title="Fact sheet"
    purpose="The text about Tribelt that the AI responses are checked against."
    :back="{ to: '/manage/prompt-test', label: 'Back to Prompt test' }"
    :results="{ to: '/ai-responses', label: 'See the AI responses' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the fact sheet…</p>
    <WidgetGrid v-else-if="sheet">
      <ManageWidget title="The text" :span="8" :accent="dirty">
        <form id="fact-sheet" class="flex flex-col gap-2" novalidate @submit.prevent="submit">
          <label class="flex flex-col gap-1.5 font-medium">
            Fact sheet, version {{ sheet.version }}
            <textarea v-model="text" rows="14" maxlength="4000" class="rounded-xl border-2 border-grey bg-white p-3 text-base"></textarea>
          </label>
        </form>
        <template #foot>
          <button type="submit" form="fact-sheet" :disabled="isSaving || !dirty" class="min-h-11 rounded-full border-0 bg-black px-6 font-semibold text-white hover:bg-ink-2 disabled:opacity-60">
            Save as a new version
          </button>
          <button
            type="button"
            :disabled="!dirty"
            class="min-h-11 rounded-full border-2 border-silver bg-white px-5 font-semibold hover:border-black disabled:opacity-60"
            @click="text = sheet.text"
          >
            Undo my edits
          </button>
          <span class="text-ink-2" data-part="version-note">
            Version {{ sheet.version }} is in use. Saving makes version {{ sheet.version + 1 }}.{{ dirty ? ' You have unsaved edits.' : '' }}
          </span>
          <p class="m-0 w-full font-medium" aria-live="polite">{{ notice }}</p>
          <p v-if="saveError" role="alert" class="m-0 w-full font-medium text-orange-ink">{{ saveError }}</p>
        </template>
      </ManageWidget>
      <ManageWidget title="How it is used" :span="4">
        <p class="m-0">
          The fact check compares each response that names Tribelt with this text. A claim that contradicts it is marked wrong, and a claim it does
          not mention counts as left out.
        </p>
        <p class="m-0">Saving makes a new version. Runs that finish after that are checked against it.</p>
        <template #foot>
          <RouterLink to="/manage/prompt-test/runs" class="inline-flex min-h-11 items-center rounded-full border-2 border-black px-5 font-semibold text-black no-underline hover:bg-black hover:text-white">
            Open the Runs page
          </RouterLink>
        </template>
      </ManageWidget>
      <ManageWidget title="Versions">
        <p class="m-0 text-ink-2" data-part="versions">
          Version {{ sheet.version }} is in use<template v-if="sheet.updatedBy && sheet.updatedAt">, saved by {{ sheet.updatedBy }} on {{ whenLabel(sheet.updatedAt) }}</template>.
          Every run records the version its responses were checked against.
        </p>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
