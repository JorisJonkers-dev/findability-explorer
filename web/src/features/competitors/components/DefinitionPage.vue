<script setup lang="ts">
import { computed, ref } from 'vue'
import BigFigure from '../../manage/components/widgets/BigFigure.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { AREA, LABEL, PRIMARY, ROW_BUTTON } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useCompetitorDefinition } from '../composables/useCompetitorDefinition'
import { definitionLine, outdatedLine, recheckLine, versionLine } from '../domain/definition'
import { longDate } from '../../../shared/dates'

/**
 * The competitor definition: the text Claude Haiku checks every company and candidate against,
 * edited and saved as a new version, every version to restore, and Check again with its estimate.
 * Saving starts no check.
 */
const { definition, draft, dirty, isSaving, isChecking, notice, error, restore, save, checkAgain } = useCompetitorDefinition(ref(true))
const outdated = computed(() => (definition.value ? outdatedLine(definition.value) : null))
</script>

<template>
  <ManageLayout
    title="Competitor definition"
    purpose="The text Claude Haiku uses to decide whether a company competes."
    :back="{ to: '/manage/competitors', label: 'Back to Competitor list' }"
    :results="{ to: '/competitors', label: 'See the competitors' }"
  >
    <WidgetGrid>
      <ManageWidget v-if="definition && outdated" title="Check again" :span="12" accent>
        <p class="m-0 text-ink-2">{{ outdated }}</p>
        <template #foot>
          <button type="button" :class="PRIMARY" :disabled="isChecking" @click="checkAgain">Check again</button>
          <span class="text-sm text-ink-2">{{ recheckLine(definition) }}</span>
        </template>
      </ManageWidget>

      <ManageWidget title="The definition" :span="8" :accent="dirty">
        <label :class="LABEL">
          Definition
          <textarea v-model="draft" :class="AREA" rows="12" maxlength="8000" />
        </label>
        <template #foot>
          <button type="button" :class="PRIMARY" :disabled="isSaving || !definition" @click="save">Save as a new version</button>
          <span v-if="definition" class="text-sm text-ink-2">{{ definitionLine(definition.current) }}{{ dirty ? ' You have unsaved edits.' : '' }}</span>
        </template>
      </ManageWidget>

      <ManageWidget title="About the definition" :span="4">
        <template v-if="definition">
          <BigFigure :value="`Version ${String(definition.current.version)}`" :label="`Saved ${longDate(definition.current.savedAt)}`" />
          <p class="m-0 text-ink-2">Claude Haiku reads this text for every company and every candidate. Write what makes a company a competitor, and what does not.</p>
          <span class="self-start rounded-full bg-white px-3 py-0.5 text-sm font-semibold">{{
            outdated ? 'Some checks used an older version' : 'Every check used this version'
          }}</span>
        </template>
      </ManageWidget>

      <p class="m-0 w-full font-medium" aria-live="polite">{{ notice }}</p>
      <p v-if="error" role="alert" class="m-0 w-full font-medium text-orange-ink">{{ error }}</p>

      <ManageWidget v-if="definition" title="Versions" :span="12">
        <ul aria-label="Versions of the definition" class="m-0 flex list-none flex-col p-0">
          <li v-for="v in definition.versions" :key="v.version" class="flex flex-col gap-2 border-b border-grey py-3">
            <span class="flex flex-wrap items-center justify-between gap-3">
              <span class="flex min-w-0 flex-col">
                <span class="font-semibold">Version {{ v.version }}</span>
                <span class="text-sm text-ink-2">{{ versionLine(v) }}</span>
              </span>
              <span v-if="v.version === definition.current.version" class="rounded-full bg-white px-3 py-0.5 text-sm font-semibold">In use</span>
              <button v-else type="button" :class="ROW_BUTTON" @click="restore(v.version)">
                Restore<span class="sr-only">{{ ` version ${String(v.version)}` }}</span>
              </button>
            </span>
            <p class="m-0 text-sm whitespace-pre-line text-ink-2">{{ v.text }}</p>
          </li>
        </ul>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
