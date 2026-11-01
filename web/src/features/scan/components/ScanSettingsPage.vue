<script setup lang="ts">
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { FIELD, LABEL } from '../../manage/components/widgets/look'
import ModelChooser from '../../settings/components/ModelChooser.vue'
import { useLlmSettings } from '../../settings/composables/useLlmSettings'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { usePageBudget } from '../composables/usePageBudget'
import { MAX_SCAN_PAGES } from '../domain/newScan'

/** How a scan reads a website: the default number of pages, and the model that matches products. */
const { pages } = usePageBudget()
const { settings, error } = useLlmSettings()

function setPages(event: Event): void {
  pages.value = Number((event.target as HTMLInputElement).value)
}
</script>

<template>
  <ManageLayout
    title="Scan settings"
    purpose="How a scan reads a website."
    :back="{ to: '/manage/scans', label: 'Back to Website scans' }"
    :results="{ to: '/ai-readiness', label: 'See AI readiness' }"
  >
    <WidgetGrid>
      <ManageWidget title="Default number of pages" :span="6">
        <p class="m-0 text-ink-2">
          A scan reads this many pages unless you set another number. More pages find more search terms and belts, and cost more at the
          matching step. The number holds for the rest of this visit.
        </p>
        <label :class="LABEL">
          Pages (1 to {{ MAX_SCAN_PAGES }})
          <input id="set-pages" type="number" min="1" :max="MAX_SCAN_PAGES" :value="pages" :class="FIELD" @change="setPages" />
        </label>
      </ManageWidget>
      <ManageWidget title="Product matching" :span="12">
        <p class="m-0 text-ink-2">
          Matching compares each page's belts with the five belt families. Turned off, scans use no tokens, but the belt family tables
          are not updated.
        </p>
        <p v-if="error" role="alert" class="m-0 font-medium text-orange-ink">{{ error }}</p>
        <ModelChooser v-else-if="settings" :settings="settings" :heading="false" />
        <p v-else class="m-0 text-muted">Loading the settings…</p>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
