<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import type { Competitor, PromptAnswer } from '../../../infrastructure/api/types.gen'
import { companyPath } from '../../shell/domain/sections'
import { answerCost, counts, excerpt, gridFamily, gridScope, hostOf, isTribeltSite, MODE_COLUMNS, squareOf, TRIBELT_ID, verdictOf } from '../domain/grid'
import { CORRECTNESS, providerName } from '../domain/results'

const props = defineProps<{ answer: PromptAnswer | undefined; competitors: readonly Competitor[]; isAdmin?: boolean }>()
const emit = defineEmits<{ add: [company: { name: string; url: string }] }>()

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
const square = computed(() => squareOf(props.answer))
</script>

<template>
  <aside aria-live="polite" aria-labelledby="answer-title" class="flex flex-col gap-4 rounded-[28px] border-2 border-grey p-6 stage:gap-7 stage:rounded-[48px] stage:border-[3px] stage:p-12">
    <span class="text-sm font-bold tracking-[0.06em] text-orange-ink uppercase stage:text-[26px]">The answer you picked</span>
    <h2 v-if="!answer" id="answer-title" class="m-0 font-display text-xl font-semibold stage:text-[44px]">Pick a square to read the answer.</h2>
    <template v-else>
      <h2 id="answer-title" class="m-0 font-display text-2xl leading-tight font-semibold stage:text-[52px]">{{ gridFamily(answer.family) }}, {{ gridScope(answer.scope) }}</h2>
      <span class="text-ink-2 stage:text-[30px]">{{ providerName(answer.provider) }}, {{ MODE_COLUMNS[answer.mode].long }}.</span>
      <div class="flex flex-col gap-1.5 rounded-2xl bg-mist px-4 py-3.5 stage:gap-3 stage:rounded-[28px] stage:px-8 stage:py-6">
        <span class="text-[13px] font-bold text-muted stage:text-[24px]">The buyer asked</span>
        <span class="leading-snug stage:text-[30px]">{{ answer.prompt }}</span>
      </div>
      <div v-if="read" class="flex flex-col gap-1.5 stage:gap-3">
        <span class="text-[13px] font-bold text-muted stage:text-[24px]">The answer</span>
        <span class="leading-relaxed text-pretty stage:text-[30px]">{{ whole ? read.text : excerpt(read.text) }}</span>
      </div>
      <div
        class="rounded-2xl px-4 py-3 font-bold stage:rounded-[28px] stage:px-8 stage:py-6 stage:text-[32px]"
        :class="read?.extraction.namesTribelt ? 'bg-orange-pale text-orange-ink' : 'bg-grey text-ink-2'"
      >
        {{ verdictOf(square) }}
      </div>
      <p v-if="!read" class="m-0 text-ink-2 stage:text-[28px]">{{ answer.failure ?? 'The answer could not be read.' }}</p>
      <template v-else>
        <div class="flex flex-col gap-1.5 stage:gap-3">
          <span class="text-[13px] font-bold text-muted stage:text-[24px]">It names</span>
          <p v-if="names.length === 0" class="m-0 stage:text-[30px]">No company.</p>
          <ol v-else aria-label="Companies named" class="m-0 flex list-none flex-wrap gap-2 p-0">
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
              <button
                v-if="!n.id && isAdmin && n.href"
                type="button"
                class="border-0 bg-transparent p-0 text-xs font-semibold text-orange-ink underline underline-offset-4 stage:text-[20px]"
                @click="emit('add', { name: n.name, url: n.href })"
              >
                Add<span class="sr-only">{{ ' ' + n.name }} as a competitor</span>
              </button>
              <RouterLink v-else-if="!n.id && isAdmin" to="/competitors" class="text-xs font-semibold stage:text-[20px]">
                Add on Competitors<span class="sr-only">: {{ n.name }}</span>
              </RouterLink>
            </li>
          </ol>
        </div>
        <p v-if="read.extraction.correctness" class="m-0 text-sm stage:text-[26px]">
          {{ CORRECTNESS[read.extraction.correctness] }}.
          <template v-for="claim in read.extraction.wrongClaims" :key="claim"> “{{ claim }}”</template>
        </p>
        <div class="flex flex-col gap-1.5 stage:gap-3">
          <span class="text-[13px] font-bold text-muted stage:text-[24px]">It cites</span>
          <ul v-if="read.extraction.addresses.length > 0" aria-label="Sources cited" class="m-0 flex list-none flex-wrap gap-x-4 gap-y-1 p-0 stage:text-[26px]">
            <li v-for="a in read.extraction.addresses" :key="a" :class="isTribeltSite(a) ? 'font-bold' : ''">
              <a :href="a" target="_blank" rel="noopener">{{ hostOf(a) }}</a>
            </li>
          </ul>
          <p v-else-if="answer.mode === 'knowledge'" class="m-0 stage:text-[30px]">Nothing. Answers from the model’s own knowledge cite no sources.</p>
          <p v-else class="m-0 stage:text-[30px]">Nothing. This answer cites no sources.</p>
        </div>
        <button
          v-if="excerpt(read.text) !== read.text.replace(/\s+/g, ' ').trim()"
          type="button"
          class="inline-flex min-h-11 items-center self-start rounded-full border-2 border-black bg-white px-5 font-bold hover:bg-mist stage:min-h-[88px] stage:border-[3px] stage:px-9 stage:text-[30px]"
          @click="whole = !whole"
        >
          {{ whole ? 'Show less' : 'Read the whole answer' }}
        </button>
      </template>
      <p class="m-0 text-xs text-muted stage:text-[22px]">
        {{ answer.model }}. {{ NUMBER.format(answer.inputTokens + answer.outputTokens) }} tokens. {{ answerCost(answer.costUsd) }}.
      </p>
      <slot />
    </template>
  </aside>
</template>
