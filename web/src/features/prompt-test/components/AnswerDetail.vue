<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import type { Competitor, PromptAnswer } from '../../../infrastructure/api/types.gen'
import { companyPath } from '../../shell/domain/sections'
import { answerCost, counts, excerpt, hostOf, isTribeltSite, TRIBELT_ID } from '../domain/grid'
import { CORRECTNESS, familyName, modeName, providerName, scopeName } from '../domain/results'

const props = defineProps<{ answer: PromptAnswer | undefined; competitors: readonly Competitor[] }>()

const whole = ref(false)
watch(
  () => props.answer?.id,
  () => {
    whole.value = false
  },
)

const read = computed(() => (counts(props.answer) ? props.answer : undefined))
const names = computed(() =>
  (read.value?.extraction.companies ?? []).map((c) => {
    const known = c.competitorId ? props.competitors.find((k) => k.id === c.competitorId) : undefined
    return {
      position: c.position,
      name: known?.name ?? c.name,
      id: c.competitorId,
      tribelt: c.competitorId === TRIBELT_ID,
      href: known?.websiteUrl ?? c.url,
    }
  }),
)
const NUMBER = new Intl.NumberFormat('en-GB')
</script>

<template>
  <div aria-live="polite" class="flex flex-col gap-3 rounded-2xl bg-mist p-5 stage:gap-5 stage:rounded-[28px] stage:p-10">
    <p v-if="!answer" class="m-0 text-ink-2 stage:text-[28px]">Pick a square to read its answer.</p>
    <template v-else>
      <span class="text-sm font-semibold tracking-[0.04em] text-orange-ink uppercase stage:text-[22px]">
        {{ providerName(answer.provider) }} {{ modeName(answer.mode).toLowerCase() }}. {{ familyName(answer.family) }}, {{ scopeName(answer.scope) }}
      </span>
      <p class="m-0 font-medium stage:text-[28px]">{{ answer.prompt }}</p>
      <p v-if="!read" class="m-0 font-semibold text-orange-ink stage:text-[28px]">
        No answer to read. {{ answer.failure ?? 'The answer could not be read.' }}
      </p>
      <template v-else>
        <p class="m-0 text-ink-2 stage:text-[26px]">“{{ whole ? read.text : excerpt(read.text) }}”</p>
        <button
          v-if="excerpt(read.text) !== read.text.replace(/\s+/g, ' ').trim()"
          type="button"
          class="self-start border-0 bg-transparent p-0 font-medium text-orange-ink underline underline-offset-4 stage:text-[24px]"
          @click="whole = !whole"
        >
          {{ whole ? 'Show less' : 'Read the whole answer' }}
        </button>
        <div v-if="names.length > 0" class="flex flex-col gap-1.5">
          <span class="text-sm text-muted stage:text-[22px]">Companies it names, in order</span>
          <ol aria-label="Companies named" class="m-0 flex list-none flex-wrap gap-2 p-0">
            <li
              v-for="n in names"
              :key="`${String(n.position)}-${n.name}`"
              class="inline-flex min-h-9 items-center gap-2 rounded-full border-2 bg-white px-3 text-sm stage:min-h-14 stage:px-5 stage:text-[24px]"
              :class="n.tribelt ? 'border-orange font-bold text-orange-ink' : 'border-grey'"
            >
              <span class="text-muted">{{ n.position }}</span>
              <a v-if="n.href" :href="n.href" target="_blank" rel="noopener" :title="`Open ${hostOf(n.href)} in a new tab`">{{ n.name }}</a>
              <span v-else>{{ n.name }}</span>
              <RouterLink v-if="n.id && !n.tribelt" :to="companyPath(n.id)" class="text-xs text-ink-2 stage:text-[20px]">its page</RouterLink>
              <span v-if="!n.id" class="text-xs text-muted stage:text-[20px]">not a competitor yet</span>
            </li>
          </ol>
        </div>
        <p v-if="read.extraction.correctness" class="m-0 text-sm stage:text-[24px]">
          {{ CORRECTNESS[read.extraction.correctness] }}.
          <template v-for="claim in read.extraction.wrongClaims" :key="claim"> “{{ claim }}”</template>
        </p>
        <div v-if="read.extraction.addresses.length > 0" class="flex flex-col gap-1.5">
          <span class="text-sm text-muted stage:text-[22px]">Sources it cites</span>
          <ul aria-label="Sources cited" class="m-0 flex list-none flex-wrap gap-x-4 gap-y-1 p-0 text-sm stage:text-[22px]">
            <li v-for="a in read.extraction.addresses" :key="a" :class="isTribeltSite(a) ? 'font-bold' : ''">
              <a :href="a" target="_blank" rel="noopener">{{ hostOf(a) }}</a>
            </li>
          </ul>
        </div>
      </template>
      <p class="m-0 text-xs text-muted stage:text-[22px]">
        {{ answer.model }}. {{ NUMBER.format(answer.inputTokens + answer.outputTokens) }} tokens. {{ answerCost(answer.costUsd) }}.
      </p>
      <slot />
    </template>
  </div>
</template>
