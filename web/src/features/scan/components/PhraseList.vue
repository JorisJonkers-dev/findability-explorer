<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Keyword } from '../../../infrastructure/api/types.gen'
import { tierFill } from '../../../shared/tiers'
import { languageLabel, tierLabel } from '../../../shared/vocabulary'
import { keywordLanguages, PHRASE_KINDS, type PhraseKind, phrases, scoreLabel } from '../domain/results'

/** How many phrases show before the list is opened in full. */
const FIRST = 10

const props = defineProps<{ keywords: Keyword[] }>()
const language = ref('')
const singleWords = ref(false)
const kind = ref<PhraseKind>('all')
const expanded = ref(false)
const languages = computed(() => keywordLanguages(props.keywords))
const all = computed(() => phrases(props.keywords, { language: language.value, singleWords: singleWords.value, kind: kind.value }))
const shown = computed(() => (expanded.value ? all.value : all.value.slice(0, FIRST)))
</script>

<template>
  <section aria-labelledby="phrase-title" class="flex flex-col gap-[18px] py-1 md:px-1 md:py-7">
    <div class="flex flex-col gap-1.5">
      <h3 id="phrase-title" class="m-0 font-display text-[28px] leading-[1.1] font-semibold">Phrases the site is built around</h3>
      <p class="m-0 text-[15px] text-muted">
        Scored by where they appear: a title counts five times body text.
        {{ singleWords ? 'Single words are shown.' : 'Single words are hidden.' }}
      </p>
      <p class="m-0 text-[15px] text-muted">
        New terms are search phrases this site uses that are not in the terminology map yet.
      </p>
    </div>
    <div class="flex flex-wrap items-center gap-3">
      <div role="group" aria-label="Which phrases" class="inline-flex rounded-[18px] border border-field p-0.5">
        <button
          v-for="option in PHRASE_KINDS"
          :key="option.value"
          type="button"
          :aria-pressed="kind === option.value"
          class="h-8 rounded-2xl px-3 text-sm font-medium"
          :class="kind === option.value ? 'bg-pressed text-ink outline-1 outline-straw' : 'text-ink-2'"
          @click="kind = option.value"
        >
          {{ option.label }}
        </button>
      </div>
      <button
        type="button"
        :aria-pressed="singleWords"
        class="h-9 rounded-[18px] border px-3 text-sm font-medium"
        :class="singleWords ? 'border-straw bg-pressed text-ink' : 'border-field text-ink-2'"
        @click="singleWords = !singleWords"
      >
        Show single words
      </button>
      <label v-if="languages.length > 1" class="inline-flex items-center gap-2 text-sm text-ink-2">
        Language
        <select v-model="language" class="h-9 rounded-md border border-field bg-surface px-2 text-sm text-ink">
          <option value="">All languages</option>
          <option v-for="code in languages" :key="code" :value="code">{{ languageLabel(code) }}</option>
        </select>
      </label>
    </div>
    <p v-if="all.length === 0" class="m-0 text-[15px] text-muted">No phrases to show.</p>
    <ol v-else class="m-0 flex list-none flex-col gap-3.5 p-0">
      <li v-for="phrase in shown" :key="`${phrase.phrase}|${phrase.language}`" class="flex flex-col gap-1.5">
        <div class="flex items-baseline justify-between gap-3">
          <span class="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
            <span aria-hidden="true" class="size-2.5 flex-none rounded-full" :class="tierFill(phrase.tier)"></span>
            <span class="sr-only">{{ tierLabel(phrase.tier) }}:</span>
            <span class="text-[17px] font-medium break-words">{{ phrase.phrase }}</span>
            <span
              class="rounded-full border px-2 py-px text-[12px]"
              :class="phrase.inMap ? 'border-field text-muted' : 'border-straw/60 text-straw'"
              :title="phrase.inMap || !phrase.closestTerm ? undefined : `Closest term in the map: ${phrase.closestTerm}`"
            >
              {{ phrase.inMap ? 'In the map' : 'New term' }}
            </span>
          </span>
          <span class="flex-none text-sm text-muted">{{ scoreLabel(phrase.score) }}</span>
        </div>
        <div aria-hidden="true" class="h-1.5 rounded-[3px] bg-raised">
          <div class="h-1.5 rounded-[3px] bg-straw" :style="{ width: `${String(phrase.share * 100)}%` }"></div>
        </div>
      </li>
    </ol>
    <button
      v-if="all.length > FIRST"
      type="button"
      class="self-start rounded-md border border-field px-3 py-1.5 text-sm font-medium text-ink-2"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      {{ expanded ? 'Show fewer' : `Show all ${String(all.length)} phrases` }}
    </button>
  </section>
</template>
