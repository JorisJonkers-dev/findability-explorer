<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { ClaudeStep, ClaudeStepId, ClaudeStepVersion } from '../../../infrastructure/api/types.gen'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { AREA, FIELD, LABEL, PRIMARY, ROW_BUTTON, SECONDARY } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useClaudeStep } from '../composables/useClaudeSteps'
import { costLine, dataLine, type Draft, isDirty, modelLabel, stepInfo, versionLine } from '../domain/steps'

/**
 * One Claude step: its instructions in an editable box, saved as versions, with Undo and Back to
 * the original; the model with what one job costs; what the model gets and replies; and the
 * version history, where restoring copies a version into the box.
 */
const props = defineProps<{ stepId: string }>()
const id = computed(() => props.stepId as ClaudeStepId)
const info = computed(() => stepInfo(id.value))
const { step, isLoading, error, isSaving, saveError, save } = useClaudeStep(id)

const draft = ref<Draft>({ text: '', model: '' })
const restoredFrom = ref<number | null>(null)
const notice = ref<string | null>(null)

function reset(s: ClaudeStep): void {
  draft.value = { text: s.current.instructions, model: s.current.model }
  restoredFrom.value = null
}

// The box starts from the version in use once it loads, and again after a save; a redraw while
// someone types leaves the box alone.
watch(
  () => step.value?.current.number,
  () => {
    if (step.value) reset(step.value)
  },
  { immediate: true },
)

const dirty = computed(() => (step.value ? isDirty(step.value, draft.value) : false))
const cost = computed(() => (step.value ? costLine(step.value, draft.value.model) : ''))
const history = computed(() => [...(step.value?.versions ?? [])].reverse())

function undo(): void {
  if (step.value) reset(step.value)
  notice.value = 'The unsaved changes are gone.'
}

function original(): void {
  if (!step.value) return
  draft.value = { text: step.value.original, model: step.value.versions[0]?.model ?? draft.value.model }
  restoredFrom.value = 1
  notice.value = 'The original instructions and model are in the box. Save to use them.'
}

function restore(v: ClaudeStepVersion): void {
  draft.value = { text: v.instructions, model: v.model }
  restoredFrom.value = v.number
  notice.value = `Version ${String(v.number)} is copied into the box. Save to use it.`
}

function submit(): void {
  notice.value = null
  save({ model: draft.value.model, instructions: draft.value.text, ...(restoredFrom.value ? { restoredFrom: restoredFrom.value } : {}) }, (s) => {
    notice.value = `Version ${String(s.current.number)} is saved. ${info.value.change.split('. ')[0] ?? ''}.`
  })
}
</script>

<template>
  <ManageLayout
    :title="step?.title ?? 'Claude step'"
    :purpose="info.does"
    :data-line="step ? dataLine(step) : undefined"
    :back="{ to: info.section.to, label: `Back to ${info.section.label}` }"
    :results="info.results"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading || !step" class="m-0 text-muted">Loading the step…</p>
    <WidgetGrid v-else>
      <ManageWidget v-if="step.fixedInstructions" title="Instructions" :span="8">
        <template #action><span class="rounded-full bg-white px-3 py-0.5 text-sm font-semibold">Set in the code</span></template>
        <p class="m-0 text-ink-2">
          These instructions are part of the tool’s code, so every job follows them in the same way, and jobs can be compared. They change
          only with a new release of the tool. The model that follows them is chosen on the right.
        </p>
        <pre class="m-0 max-h-[32rem] overflow-auto rounded-2xl bg-white p-4 text-sm whitespace-pre-wrap">{{ step.current.instructions }}</pre>
      </ManageWidget>
      <ManageWidget v-else title="Instructions" :span="8" :accent="dirty">
        <p class="m-0 text-ink-2">
          What the model should do, and how to judge. The program sends the data in a separate message after these instructions, and fixes
          the reply format, so editing this text cannot break either.
        </p>
        <label :class="LABEL">
          Instructions
          <textarea v-model="draft.text" rows="18" spellcheck="true" :class="[AREA, 'font-mono text-sm']"></textarea>
        </label>
        <template #foot>
          <button type="button" :class="PRIMARY" :disabled="!dirty || isSaving" @click="submit">Save as a new version</button>
          <button type="button" :class="SECONDARY" :disabled="!dirty" @click="undo">Undo the changes</button>
          <button type="button" :class="SECONDARY" @click="original">Back to the original</button>
        </template>
      </ManageWidget>

      <ManageWidget title="Model" :span="4" :accent="draft.model !== step.current.model">
        <label :class="LABEL">
          Model
          <select v-model="draft.model" :class="FIELD">
            <option v-for="m in step.models" :key="m.id" :value="m.id">{{ modelLabel(m) }}</option>
          </select>
        </label>
        <p class="m-0" data-part="cost">{{ cost }}</p>
        <p class="m-0 text-ink-2">
          Claude Haiku is the cheapest and fastest. Claude Sonnet reads and writes more carefully, and Claude Opus most carefully, each at a
          higher price.
        </p>
        <p class="m-0 text-ink-2"><strong class="text-black">What a change does.</strong> {{ info.change }}</p>
        <template v-if="step.fixedInstructions" #foot>
          <button type="button" :class="PRIMARY" :disabled="!dirty || isSaving" @click="submit">Save the model</button>
          <button type="button" :class="SECONDARY" :disabled="!dirty" @click="undo">Undo</button>
        </template>
      </ManageWidget>

      <p class="m-0 w-full font-medium" aria-live="polite">{{ notice }}</p>
      <p v-if="saveError" role="alert" class="m-0 w-full font-medium text-orange-ink">{{ saveError }}</p>

      <ManageWidget title="What the model gets" :span="6">
        <p class="m-0 text-ink-2">With every request, after the instructions.</p>
        <dl class="m-0 flex flex-col gap-3">
          <div v-for="p in info.gets" :key="p.name" class="flex flex-col gap-0.5">
            <dt class="font-semibold">{{ p.name }}</dt>
            <dd class="m-0 text-ink-2">{{ p.what }} <span class="block text-sm text-muted">From {{ p.from.charAt(0).toLowerCase() + p.from.slice(1) }}</span></dd>
          </div>
        </dl>
      </ManageWidget>
      <ManageWidget title="What the model replies" :span="6">
        <p class="m-0 text-ink-2">A fixed format the program reads. The instructions say what each part means, but cannot change the parts.</p>
        <dl class="m-0 flex flex-col gap-3">
          <div v-for="f in info.replies" :key="f.field" class="flex flex-col gap-0.5">
            <dt><code class="font-mono text-sm font-semibold">{{ f.field }}</code></dt>
            <dd class="m-0 text-ink-2">{{ f.what }}</dd>
          </div>
        </dl>
      </ManageWidget>

      <ManageWidget :title="step.fixedInstructions ? 'Model history' : 'Version history'" :span="12">
        <p class="m-0 text-ink-2">Each save is a version. Restoring one copies it into the box above, and saving makes a new version.</p>
        <ul aria-label="Versions" class="m-0 flex list-none flex-col gap-3 p-0">
          <li v-for="v in history" :key="v.number" class="flex flex-col gap-2 rounded-2xl bg-white p-4">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <span class="flex min-w-0 flex-col">
                <span class="font-semibold">{{ versionLine(v) }}<span v-if="v.number === step.current.number" class="ml-2 rounded-full bg-mist px-2.5 py-0.5 text-sm">In use</span></span>
                <span class="text-sm text-ink-2">{{ v.modelName }}<template v-if="v.note">. {{ v.note }}</template></span>
              </span>
              <button v-if="v.number !== step.current.number" type="button" :class="ROW_BUTTON" @click="restore(v)">
                Restore<span class="sr-only">{{ ` version ${String(v.number)}` }}</span>
              </button>
            </div>
            <details v-if="!step.fixedInstructions">
              <summary class="min-h-11 cursor-pointer content-center font-medium text-orange-ink">Show the instructions</summary>
              <pre class="m-0 mt-2 max-h-96 overflow-auto rounded-xl bg-mist p-3 text-sm whitespace-pre-wrap">{{ v.instructions }}</pre>
            </details>
          </li>
        </ul>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
