<script setup lang="ts">
import { computed } from 'vue'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { usePromptHistory, usePromptSets } from '../composables/useManage'
import { questionGroups, versionLine } from '../domain/questions'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/**
 * The published Prompt set, grouped by belt family, read-only in this release, and every version
 * the Prompt set has had, each with its questions.
 */
const { sets, isLoading, error } = usePromptSets()
const { history } = usePromptHistory()

const NUMBER = new Intl.NumberFormat('en-GB')
const published = computed(() => sets.value?.versions[0])
const groups = computed(() => (published.value ? questionGroups(published.value.questions) : []))
const versions = computed(() =>
  [...(history.value?.versions ?? [])].reverse().map((v) => ({ ...v, questionsOf: questionGroups(sets.value?.versions.find((s) => s.version === v.version)?.questions ?? []) })),
)
const plural = (n: number) => (n === 1 ? '1 question' : `${NUMBER.format(n)} questions`)
</script>

<template>
  <ManageLayout
    title="Questions"
    purpose="The buyer questions that the AI assistants are asked."
    :back="{ to: '/manage/prompt-test', label: 'Back to Prompt test' }"
    :results="{ to: '/ai-responses', label: 'See the AI responses' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the questions…</p>
    <WidgetGrid v-else-if="published">
      <ManageWidget title="The questions" :span="8">
        <p class="m-0 text-ink-2">
          The published set, version {{ published.version }}, grouped by belt family. Each question belongs to a scope, a language with or without
          a region. {{ plural(published.questions.length) }} in all.
        </p>
        <section v-for="g in groups" :key="g.family" :aria-labelledby="`family-${g.family}`" class="flex flex-col gap-2">
          <h3 :id="`family-${g.family}`" class="m-0 flex flex-wrap items-baseline gap-x-3 border-b-2 border-black pb-1 text-lg font-semibold">
            {{ g.label }} <span class="text-sm font-normal text-ink-2">{{ plural(g.questions.length) }}</span>
          </h3>
          <ul class="m-0 flex list-none flex-col p-0">
            <li v-for="q in g.questions" :key="q.id" class="flex flex-col gap-0.5 border-b border-grey py-2 last:border-b-0">
              <span class="text-sm font-semibold text-ink-2">{{ q.scope }}</span>
              <span>{{ q.text }}</span>
              <span v-if="q.english" class="text-sm text-ink-2">{{ q.english }}</span>
            </li>
          </ul>
        </section>
      </ManageWidget>
      <ManageWidget :title="`Version ${published.version} is published`" :span="4">
        <p class="m-0 text-ink-2">{{ versionLine({ publishedAt: published.publishedAt, questions: published.questions.length }) }}.</p>
        <p v-if="published.note" class="m-0">{{ published.note }}</p>
        <p class="m-0 text-ink-2">Every run asks the published set, so each run compares with the ones before it. Editing the questions arrives with the next release.</p>
      </ManageWidget>
      <ManageWidget title="Version history" :span="12">
        <p class="m-0 text-ink-2">Old versions are never deleted. The charts mark where the questions changed.</p>
        <div class="flex flex-col gap-2">
          <details v-for="v in versions" :key="v.version" class="rounded-2xl bg-white p-4">
            <summary class="flex cursor-pointer flex-wrap items-baseline gap-x-3 font-semibold">
              Version {{ v.version }}
              <span v-if="v.version === published.version" class="rounded-full bg-black px-2.5 py-0.5 text-sm text-white">Published</span>
              <span class="text-sm font-normal text-ink-2">{{ versionLine(v) }}</span>
            </summary>
            <div class="mt-3 flex flex-col gap-2 text-sm">
              <p v-if="v.note" class="m-0 text-ink-2">{{ v.note }}</p>
              <template v-for="g in v.questionsOf" :key="g.family">
                <h4 class="m-0 font-semibold">{{ g.label }}</h4>
                <ul class="m-0 flex flex-col gap-1 pl-5">
                  <li v-for="q in g.questions" :key="q.id">{{ q.scope }}. {{ q.text }}</li>
                </ul>
              </template>
            </div>
          </details>
        </div>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
