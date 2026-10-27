<script setup lang="ts">
import { computed, nextTick, ref, useTemplateRef } from 'vue'
import { RouterLink } from 'vue-router'
import type { CompetitorCandidate } from '../../../infrastructure/api/types.gen'
import { joinWords } from '../../../shared/words'
import { providerName } from '../../prompt-test/domain/results'
import { companyPath } from '../../shell/domain/sections'
import { type CandidateDraft, candidateDecisionLine, candidateDraft, checkCandidateDraft, checkLine, otherNames } from '../domain/candidates'

/** One company the AI answers name, in a row: its names, where it was named, Haiku's check, and what to do with it. */
const props = defineProps<{ candidate: CompetitorCandidate; busy: boolean; selected: boolean; active: boolean }>()
const emit = defineEmits<{ add: [name: string, website: string]; dismiss: []; undo: []; split: [id: string, name: string]; select: [on: boolean]; focus: [] }>()

const row = useTemplateRef<HTMLLIElement>('row')
const nameInput = useTemplateRef<HTMLInputElement>('nameInput')
const titleId = computed(() => `named-${props.candidate.id}`)
const others = computed(() => otherNames(props.candidate))
const check = computed(() => checkLine(props.candidate))
const decided = computed(() => candidateDecisionLine(props.candidate))
const evidence = computed(() => {
  const c = props.candidate
  return `Named in ${c.answers === 1 ? '1 answer' : `${String(c.answers)} answers`} by ${joinWords(c.providers.map(providerName))}.`
})
const editing = ref(false)
const draft = ref<CandidateDraft>(candidateDraft(props.candidate))
const draftError = ref<string | null>(null)

/** Adds it at once when its website is known; otherwise asks for the website first. */
async function add(): Promise<void> {
  if (props.candidate.website) {
    emit('add', props.candidate.name, props.candidate.website)
    return
  }
  draft.value = candidateDraft(props.candidate)
  draftError.value = null
  editing.value = true
  await nextTick()
  nameInput.value?.focus()
}

function submit(): void {
  const c = checkCandidateDraft(draft.value)
  if (c.kind === 'invalid') {
    draftError.value = c.message
    return
  }
  editing.value = false
  emit('add', c.name, c.website)
}

defineExpose({ add, focus: () => row.value?.focus() })
</script>

<template>
  <li
    ref="row"
    data-row
    :tabindex="active ? 0 : -1"
    :aria-labelledby="titleId"
    class="grid grid-cols-[1.5rem_minmax(0,1fr)] gap-x-3 gap-y-2 border-b border-grey px-2 py-3 outline-offset-2 focus-visible:outline-2 focus-visible:outline-black sm:grid-cols-[1.5rem_minmax(0,1fr)_auto]"
    :class="selected ? 'bg-orange-pale' : active ? 'bg-mist' : ''"
    @focus="emit('focus')"
  >
    <input
      type="checkbox"
      :checked="selected"
      :aria-label="`Select ${candidate.name}`"
      class="mt-1 size-4 accent-orange-ink"
      @change="emit('select', ($event.target as HTMLInputElement).checked)"
    />
    <div class="flex min-w-0 flex-col gap-1">
      <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h3 :id="titleId" class="m-0 text-base font-semibold">{{ candidate.name }}</h3>
        <a
          v-for="s in candidate.sites ?? []"
          :key="s"
          :href="s === candidate.sites?.[0] && candidate.website ? candidate.website : `https://${s}/`"
          target="_blank"
          rel="noopener noreferrer"
          class="text-sm"
        >{{ s }}<span class="sr-only"> (opens in a new tab)</span></a>
      </div>
      <p class="m-0 text-sm text-ink-2">{{ evidence }}</p>
      <div v-if="others.length > 0" class="flex flex-wrap items-center gap-1.5 text-sm">
        <span class="text-muted">Also named</span>
        <span v-for="n in others" :key="n.id" class="inline-flex items-center gap-1 rounded-full bg-white px-2 py-0.5 ring-1 ring-grey">
          {{ n.name }}
          <button
            v-if="n.joined === 'model' && candidate.status === 'pending'"
            type="button"
            :disabled="busy"
            class="border-0 bg-transparent p-0 text-xs font-medium text-ink-2 underline underline-offset-2 hover:text-black"
            @click="emit('split', n.id, n.name)"
          >
            Split off<span class="sr-only">{{ ' ' + n.name }}</span>
          </button>
        </span>
      </div>
      <p v-if="check" class="m-0 text-sm font-medium">{{ check }}</p>
      <p v-if="decided" class="m-0 text-sm font-medium">{{ decided }}</p>

      <form v-if="editing" novalidate :aria-label="`Add ${candidate.name}`" class="mt-1 flex flex-wrap items-end gap-2" @submit.prevent="submit">
        <label class="flex flex-col gap-1 text-sm font-medium">
          Name
          <input ref="nameInput" v-model="draft.name" type="text" maxlength="200" class="h-10 rounded-lg border-2 border-grey bg-white px-2 text-base font-normal" />
        </label>
        <label class="flex flex-col gap-1 text-sm font-medium">
          Website
          <input v-model="draft.website" type="url" maxlength="500" placeholder="https://www.example.com" class="h-10 rounded-lg border-2 border-grey bg-white px-2 text-base font-normal" />
        </label>
        <button type="submit" :disabled="busy" class="h-10 rounded-full border-0 bg-black px-4 font-semibold text-white disabled:opacity-70">Add</button>
        <button type="button" class="h-10 rounded-full border-2 border-grey bg-white px-4 font-medium" @click="editing = false">Cancel</button>
        <p v-if="draftError" role="alert" class="m-0 w-full text-sm font-medium text-orange-ink">{{ draftError }}</p>
      </form>
    </div>

    <div v-if="!editing" class="col-start-2 flex flex-wrap items-center gap-2 sm:col-start-3 sm:justify-end">
      <template v-if="candidate.status === 'pending'">
        <button
          type="button"
          :disabled="busy"
          :aria-describedby="titleId"
          class="min-h-10 rounded-full border-0 bg-orange px-4 text-sm font-semibold text-black hover:bg-orange-soft disabled:opacity-70"
          @click="add"
        >
          {{ candidate.website ? 'Add' : 'Add…' }}
        </button>
        <button
          type="button"
          :disabled="busy"
          :aria-describedby="titleId"
          class="min-h-10 rounded-full border-2 border-grey bg-white px-4 text-sm font-medium hover:border-black disabled:opacity-70"
          @click="emit('dismiss')"
        >
          Dismiss
        </button>
      </template>
      <template v-else>
        <RouterLink v-if="candidate.decision?.competitorId" :to="companyPath(candidate.decision.competitorId)" class="text-sm font-medium">
          Open its page
        </RouterLink>
        <button
          type="button"
          :disabled="busy"
          :aria-describedby="titleId"
          class="min-h-10 rounded-full border-2 border-black bg-white px-4 text-sm font-medium hover:bg-black hover:text-white disabled:opacity-70"
          @click="emit('undo')"
        >
          Undo
        </button>
      </template>
    </div>
  </li>
</template>
