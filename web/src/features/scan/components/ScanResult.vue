<script setup lang="ts">
import { vScrollHint } from '../../../shared/scrollHint'
import { computed, ref } from 'vue'
import type { Analysis } from '../../../infrastructure/api/types.gen'
import { percent } from '../../../shared/percent'
import { tierFill } from '../../../shared/tiers'
import { tierLabel } from '../../../shared/vocabulary'
import LanguageFilter from '../../../shared/ui/LanguageFilter.vue'
import PillButton from '../../../shared/ui/PillButton.vue'
import { languageChoices, ofLanguage } from '../../terminology/domain/languages'
import {
  equivalentRows,
  matchedBy,
  missingTerms,
  newTermCount,
  PHRASE_KINDS,
  type PhraseKind,
  phrases,
  resultLine,
  scannedAt,
  siteHost,
} from '../domain/results'
import { matchingText } from '../domain/status'

const props = defineProps<{ scan: Analysis }>()

/** How many of the terms Tribelt is missing are shown as chips. */
const MISSING_SHOWN = 8

const kind = ref<PhraseKind>('all')
// The language the terms are shown in; a page around the result may follow it too.
const language = defineModel<string | null>('language', { default: null })
const languages = computed(() => languageChoices([...props.scan.keywords, ...props.scan.knownTerms].map((t) => t.language)))
const summary = computed(() => props.scan.summary)
const missing = computed(() => missingTerms(ofLanguage(props.scan.knownTerms, language.value)).slice(0, MISSING_SHOWN))
const ranked = computed(() => phrases(ofLanguage(props.scan.keywords, language.value), { language: '', singleWords: true, kind: kind.value }))
const equivalents = computed(() => equivalentRows(props.scan.equivalents))
const matching = computed(() => matchingText(props.scan.matching))
const model = computed(() => matchedBy(props.scan.matchingModel))
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
      <h3 class="m-0 font-display text-2xl leading-none font-semibold">{{ siteHost(summary.url) }}</h3>
      <span class="text-sm text-muted">{{ resultLine(scan.pages.length, newTermCount(scan.keywords)) }}</span>
    </div>
    <p class="m-0 -mt-2 text-sm text-muted">Scanned {{ scannedAt(summary.finishedAt ?? summary.createdAt) }}</p>
    <LanguageFilter :choices="languages" :selected="language" @select="(l) => (language = l)" />

    <div class="flex flex-wrap items-center gap-2">
      <span class="text-sm text-muted">{{ missing.length > 0 ? 'Tribelt is missing:' : 'Tribelt uses every map term this site uses.' }}</span>
      <span
        v-for="t in missing"
        :key="`${t.term}-${t.language}`"
        class="inline-flex items-center gap-2 rounded-full bg-white px-3 py-1 text-sm"
      >
        <span aria-hidden="true" class="size-3 flex-none rounded-full" :class="tierFill(t.tier)"></span>
        {{ t.term }}<span class="sr-only">, {{ tierLabel(t.tier) }}</span>
      </span>
    </div>

    <p v-if="matching !== null" class="m-0 text-sm text-ink-2">{{ matching }}</p>
    <details v-else-if="equivalents.length > 0" class="rounded-2xl bg-white px-4 py-3 text-sm">
      <summary class="font-medium">
        {{ equivalents.length }} {{ equivalents.length === 1 ? 'product' : 'products' }} matched to Tribelt's belts
      </summary>
      <ul class="m-0 mt-2 flex list-none flex-col gap-2 p-0">
        <li v-for="row in equivalents" :key="row.theirs">
          <strong class="font-semibold">{{ row.theirs }}</strong> is Tribelt's {{ row.ours }}.
          <span v-if="row.words !== ''" class="text-muted">Its words: {{ row.words }}.</span>
        </li>
      </ul>
      <p v-if="model !== null" class="m-0 mt-2 text-muted">{{ model }}</p>
    </details>

    <div role="group" aria-label="Which phrases" class="flex flex-wrap gap-2">
      <PillButton v-for="k in PHRASE_KINDS" :key="k.value" size="sm" :pressed="kind === k.value" @click="kind = k.value">
        {{ k.label }}
      </PillButton>
    </div>

    <ol
      v-if="ranked.length > 0"
      v-scroll-hint
      aria-label="Ranked phrases"
      tabindex="0"
      class="scroll-hint m-0 max-h-[28rem] list-none overflow-y-auto p-0"
    >
      <li
        v-for="(p, index) in ranked"
        :key="`${p.phrase}-${p.language}`"
        class="grid grid-cols-[2rem_minmax(0,1fr)_auto] items-center gap-3 border-b border-grey py-2.5"
      >
        <span class="text-sm text-muted">{{ index + 1 }}</span>
        <span class="flex min-w-0 flex-col gap-1.5">
          <span class="truncate font-medium">{{ p.phrase }}<span class="sr-only">, {{ tierLabel(p.tier) }}</span></span>
          <span aria-hidden="true" class="h-1.5 bg-grey">
            <span class="block h-full" :class="tierFill(p.tier)" :style="{ width: percent(p.share) }"></span>
          </span>
        </span>
        <span
          class="rounded-full px-2.5 py-0.5 text-xs font-medium text-black"
          :class="p.inMap ? 'bg-grey' : 'bg-orange'"
        >
          {{ p.inMap ? 'Already on our list' : 'New term' }}
        </span>
      </li>
    </ol>
    <p v-else class="m-0 text-sm text-muted">No phrases of this kind.</p>
  </div>
</template>
