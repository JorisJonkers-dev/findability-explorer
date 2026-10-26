<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { RouterLink } from 'vue-router'
import type { PromptFamily, PromptMode, PromptRun, PromptScope } from '../../../infrastructure/api/types.gen'
import { usePromptAnswers } from '../composables/usePromptTest'
import { FAMILIES, MODES, modeName, providerName, SCOPES, scopeName } from '../domain/results'

/** A run's answers in full, each beside how the extractor read it, one provider, scope, family and mode at a time. */
const props = defineProps<{ run: PromptRun }>()

const provider = ref(props.run.models[0]?.provider ?? '')
const scope = ref<PromptScope>('twente')
const family = ref<PromptFamily | ''>('')
const mode = ref<PromptMode | ''>('')
const filter = computed(() => ({
  provider: provider.value,
  scope: scope.value,
  ...(family.value === '' ? {} : { family: family.value }),
  ...(mode.value === '' ? {} : { mode: mode.value }),
}))
const { answers, isLoading, error } = usePromptAnswers(toRef(() => props.run.id), filter)
const CORRECTNESS = { correct: 'Correct about Tribelt', partly: 'Partly correct about Tribelt', wrong: 'Wrong about Tribelt' } as const
</script>

<template>
  <div class="flex flex-col gap-4 stage:gap-6">
    <div class="flex flex-wrap gap-3 stage:gap-6">
      <label class="flex flex-col gap-1 text-sm text-ink-2 stage:text-[24px]">
        Model
        <select v-model="provider" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[72px] stage:text-[26px]">
          <option v-for="m in run.models" :key="m.provider" :value="m.provider">{{ providerName(m.provider) }}</option>
        </select>
      </label>
      <label class="flex flex-col gap-1 text-sm text-ink-2 stage:text-[24px]">
        Scope
        <select v-model="scope" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[72px] stage:text-[26px]">
          <option v-for="s in SCOPES" :key="s.value" :value="s.value">{{ s.label }}</option>
        </select>
      </label>
      <label class="flex flex-col gap-1 text-sm text-ink-2 stage:text-[24px]">
        Belt family
        <select v-model="family" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[72px] stage:text-[26px]">
          <option value="">Every family</option>
          <option v-for="f in FAMILIES" :key="f.value" :value="f.value">{{ f.label }}</option>
        </select>
      </label>
      <label class="flex flex-col gap-1 text-sm text-ink-2 stage:text-[24px]">
        Mode
        <select v-model="mode" class="h-11 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[72px] stage:text-[26px]">
          <option value="">Both</option>
          <option v-for="m in MODES" :key="m.value" :value="m.value">{{ m.label }}</option>
        </select>
      </label>
    </div>
    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the answers…</p>
    <p v-else-if="answers && answers.items.length === 0" class="m-0 text-ink-2">No answers here yet.</p>
    <ul v-else-if="answers" aria-label="Answers" class="m-0 flex list-none flex-col gap-4 p-0 stage:gap-6">
      <li v-for="a in answers.items" :key="a.id" class="grid gap-3 rounded-2xl bg-mist p-4 lg:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)] stage:gap-6 stage:p-8">
        <div class="flex flex-col gap-2 stage:gap-3">
          <span class="text-xs font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[22px]">
            {{ scopeName(a.scope) }} · {{ a.kind }} · {{ modeName(a.mode) }}
          </span>
          <p class="m-0 font-medium">{{ a.prompt }}</p>
          <p class="m-0 text-xs text-muted stage:text-[22px]">{{ a.model }} · prompt set {{ a.promptSetVersion }}</p>
          <p v-if="a.failure" class="m-0 font-medium text-orange-ink">No answer: {{ a.failure }}</p>
          <details v-else>
            <summary class="cursor-pointer text-sm font-medium stage:text-[24px]">The answer</summary>
            <p class="m-0 mt-2 text-sm whitespace-pre-wrap stage:text-[24px]">{{ a.text }}</p>
          </details>
        </div>
        <div v-if="a.extraction" class="flex flex-col gap-2 text-sm stage:gap-3 stage:text-[24px]" data-testid="extraction">
          <p v-if="a.extraction.failed" class="m-0 font-medium text-orange-ink">The extractor could not read this answer.</p>
          <template v-else>
            <p class="m-0 font-semibold">
              {{ a.extraction.namesTribelt ? 'Names Tribelt' : 'Does not name Tribelt' }} ·
              {{ a.extraction.citesTribelt ? 'cites it' : 'does not cite it' }}
            </p>
            <p v-if="a.extraction.correctness" class="m-0" :class="a.extraction.correctness === 'correct' ? '' : 'font-semibold text-orange-ink'">
              {{ CORRECTNESS[a.extraction.correctness] }}<template v-for="claim in a.extraction.wrongClaims" :key="claim">: “{{ claim }}”</template>
            </p>
            <ol v-if="a.extraction.companies.length > 0" aria-label="Companies named" class="m-0 flex flex-col gap-0.5 pl-5">
              <li v-for="c in a.extraction.companies" :key="c.position">
                <RouterLink v-if="c.competitorId" :to="{ path: '/competitors', query: { company: c.competitorId } }">{{ c.name }}</RouterLink>
                <span v-else>{{ c.name }} <span class="text-muted">(not on the list)</span></span>
              </li>
            </ol>
            <p v-else class="m-0 text-muted">Names no company.</p>
            <p v-if="a.extraction.addresses.length > 0" class="m-0 break-all text-xs text-muted stage:text-[20px]">Cites {{ a.extraction.addresses.join(', ') }}</p>
          </template>
        </div>
      </li>
    </ul>
  </div>
</template>
