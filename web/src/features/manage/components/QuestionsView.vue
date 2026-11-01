<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { PromptFamily, PublishedPromptSet } from '../../../infrastructure/api/types.gen'
import { usePromptSettings } from '../../prompt-test/composables/usePromptTest'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { usePromptHistory, usePromptSets, usePublishPromptSet } from '../composables/useManage'
import {
  addQuestion,
  costSentence,
  diffOf,
  type Draft,
  draftOf,
  editQuestion,
  familyRows,
  fillScope,
  firstGap,
  isUnchanged,
  makeScope,
  publishLine,
  removeQuestion,
  requestOf,
  restoreOf,
  scopeNameOf,
  scopesOf,
} from '../domain/draft'
import { questionGroups, versionLine } from '../domain/questions'
import AddQuestionRow from './widgets/AddQuestionRow.vue'
import { AREA, DANGER, PRIMARY, SECONDARY } from './widgets/look'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/**
 * Questions: one editor grouped by belt family, a draft that changes nothing until it is published,
 * Publish kept in view with what it changes and costs, and every version, each restorable into the
 * draft.
 */
const { sets, isLoading, error } = usePromptSets()
const { history } = usePromptHistory()
const { settings } = usePromptSettings()
const publisher = usePublishPromptSet()

const NUMBER = new Intl.NumberFormat('en-GB')
const published = computed(() => sets.value?.versions[0])
const draft = ref<Draft>({ questions: [], scopes: [] })
const offer = ref<string | null>(null)
const notice = ref<string | null>(null)

// The draft starts as the published set, and again after each publication.
watch(
  published,
  (p) => {
    if (p) draft.value = draftOf(p)
  },
  { immediate: true },
)

const scopes = computed(() => scopesOf(sets.value?.scopes ?? [], draft.value))
const rows = computed(() => familyRows(draft.value, scopes.value))
const diff = computed(() => (published.value ? diffOf(published.value, draft.value) : null))
const unchanged = computed(() => diff.value === null || isUnchanged(diff.value))
const nextVersion = computed(() => (published.value?.version ?? 0) + 1)
const publishedGroups = computed(() => (published.value ? questionGroups(published.value.questions) : []))
const versions = computed(() =>
  [...(history.value?.versions ?? [])].reverse().map((v) => ({ ...v, set: sets.value?.versions.find((s) => s.version === v.version) })),
)
const cost = computed(() =>
  settings.value && published.value
    ? costSentence(settings.value.prices.runUsd, published.value.questions.length, draft.value.questions.length, settings.value.cadence)
    : '',
)
const plural = (n: number) => (n === 1 ? '1 question' : `${NUMBER.format(n)} questions`)

function add(family: PromptFamily, choice: { scope: string } | { language: string; region: string }, text: string): void {
  let next = draft.value
  let scope: string
  let made = false
  if ('scope' in choice) {
    scope = choice.scope
  } else {
    const r = makeScope(next, sets.value?.scopes ?? [], choice.language, choice.region)
    next = r.draft
    scope = r.scope.id
    made = r.made
  }
  draft.value = addQuestion(next, family, scope, text)
  offer.value = made ? scope : null
  notice.value = made ? `${scopeNameOf(scopes.value, scope)} is a new scope, and the question is added to the draft.` : 'The question is added to the draft.'
}

function fill(scope: string): void {
  const r = fillScope(draft.value, scope)
  draft.value = r.draft
  offer.value = null
  notice.value = `${plural(r.added)} added to the draft for ${scopeNameOf(scopes.value, scope)}. Translate each one before you publish.`
}

function discard(): void {
  if (published.value) draft.value = draftOf(published.value)
  offer.value = null
  notice.value = 'The draft is reset to the published set.'
}

function restore(set: PublishedPromptSet): void {
  draft.value = restoreOf(set)
  offer.value = null
  notice.value = `Version ${String(set.version)} is copied into the draft. Publish the draft to make a new version.`
}

function publish(): void {
  notice.value = null
  offer.value = null
  publisher.publish(requestOf(draft.value))
}
</script>

<template>
  <ManageLayout
    title="Questions"
    help-topic="questions"
    purpose="The buyer questions that the AI assistants are asked."
    :back="{ to: '/manage/prompt-test', label: 'Back to Prompt test' }"
    :results="{ to: '/ai-responses', label: 'See the AI responses' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the questions…</p>
    <WidgetGrid v-else-if="published">
      <ManageWidget title="The questions" :span="8">
        <p class="m-0 text-ink-2">
          Edit the draft here, grouped by belt family. Each question belongs to a scope, a language with or without a region. Nothing changes until
          you publish. {{ plural(draft.questions.length) }} in the draft.
        </p>
        <details class="rounded-2xl bg-white p-4">
          <summary class="cursor-pointer font-semibold text-orange-ink">The published set, version {{ published.version }}, {{ plural(published.questions.length) }}</summary>
          <div class="mt-3 flex flex-col gap-2 text-sm">
            <template v-for="g in publishedGroups" :key="g.family">
              <h3 class="m-0 text-base font-semibold">{{ g.label }}</h3>
              <ul class="m-0 flex flex-col gap-1 pl-5">
                <li v-for="q in g.questions" :key="q.id"><strong>{{ q.scope }}.</strong> {{ q.text }}</li>
              </ul>
            </template>
          </div>
        </details>
        <p aria-live="polite" class="m-0 font-medium" data-part="draft-notice">{{ notice }}</p>
        <div v-if="offer" role="status" data-part="scope-offer" class="flex flex-wrap items-center gap-3 rounded-2xl bg-orange-pale p-4">
          <span>{{ scopeNameOf(scopes, offer) }} is new, and only some belt families have a question in it.</span>
          <button type="button" :class="SECONDARY" @click="fill(offer)">Add one for every other belt family</button>
        </div>

        <section v-for="r in rows" :key="r.family" :aria-labelledby="`family-${r.family}`" class="flex min-w-0 flex-col gap-3">
          <h3 :id="`family-${r.family}`" class="m-0 flex flex-wrap items-baseline gap-x-3 border-b-2 border-black pt-2 pb-1 text-lg font-semibold">
            {{ r.label }} <span class="text-sm font-normal text-ink-2">{{ plural(r.questions.length) }}</span>
          </h3>
          <p v-if="r.questions.length === 0" class="m-0 text-sm text-ink-2">No questions for this belt family yet.</p>
          <div v-for="q in r.questions" :key="q.key" class="flex min-w-0 flex-col gap-2 border-b border-grey pb-3" data-part="question">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <span class="font-semibold">
                {{ scopeNameOf(scopes, q.scope) }}<span v-if="!q.id" class="ml-2 rounded-full bg-orange px-2 py-0.5 text-xs text-black">New</span>
              </span>
              <button type="button" :class="[DANGER, 'min-h-10 px-4 text-sm']" @click="draft = removeQuestion(draft, q.key)">
                Remove<span class="sr-only">{{ ` the ${scopeNameOf(scopes, q.scope)} question of ${r.label}` }}</span>
              </button>
            </div>
            <textarea
              :value="q.text"
              :aria-label="`Question, ${r.label}, ${scopeNameOf(scopes, q.scope)}`"
              rows="3"
              maxlength="1000"
              :class="[AREA, 'min-h-24']"
              @input="draft = editQuestion(draft, q.key, ($event.target as HTMLTextAreaElement).value)"
            ></textarea>
          </div>
          <AddQuestionRow
            :family="r.family"
            :family-label="r.label"
            :scopes="scopes"
            :start-scope="firstGap(draft, scopes, r.family)"
            @add="(choice, text) => add(r.family, choice, text)"
          />
        </section>
      </ManageWidget>

      <ManageWidget :title="`Publish as version ${nextVersion}`" :span="4" :accent="!unchanged" class="self-start min-[1100px]:sticky min-[1100px]:top-4">
        <p v-if="!diff || unchanged" class="m-0 text-ink-2" data-part="publish-summary">
          No changes to publish yet. Edit, add, or remove a question in the draft, and this card says what would change.
        </p>
        <p v-else class="m-0" data-part="publish-summary">{{ publishLine(diff, draft, scopes) }} {{ cost }} The charts then mark a break where the questions changed.</p>
        <p aria-live="polite" class="m-0 font-medium" data-part="publish-notice">{{ publisher.notice.value }}</p>
        <p v-if="publisher.error.value" role="alert" class="m-0 font-medium text-orange-ink">{{ publisher.error.value }}</p>
        <template #foot>
          <button type="button" :class="PRIMARY" :disabled="unchanged || publisher.isPublishing.value" @click="publish">Publish as version {{ nextVersion }}</button>
          <button type="button" :class="SECONDARY" :disabled="unchanged && !draft.restoredFrom" @click="discard">Discard the draft</button>
        </template>
      </ManageWidget>

      <ManageWidget title="Version history" :span="12">
        <p class="m-0 text-ink-2">Old versions are never deleted. Restoring one copies it into the draft, and publishing makes a new version.</p>
        <div class="flex flex-col gap-2">
          <details v-for="v in versions" :key="v.version" class="rounded-2xl bg-white p-4">
            <summary class="flex cursor-pointer flex-wrap items-baseline gap-x-3 font-semibold">
              Version {{ v.version }}
              <span v-if="v.version === published.version" class="rounded-full bg-black px-2.5 py-0.5 text-sm text-white">Published</span>
              <span class="text-sm font-normal text-ink-2">{{ versionLine(v) }}</span>
            </summary>
            <div class="mt-3 flex flex-col gap-2 text-sm">
              <p v-if="v.note" class="m-0 text-ink-2">{{ v.note }}</p>
              <template v-for="g in questionGroups(v.set?.questions ?? [])" :key="g.family">
                <h4 class="m-0 font-semibold">{{ g.label }}</h4>
                <ul class="m-0 flex flex-col gap-1 pl-5">
                  <li v-for="q in g.questions" :key="q.id">{{ q.scope }}. {{ q.text }}</li>
                </ul>
              </template>
              <button v-if="v.set" type="button" :class="[SECONDARY, 'self-start']" @click="restore(v.set)">Restore version {{ v.version }} into the draft</button>
            </div>
          </details>
        </div>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
