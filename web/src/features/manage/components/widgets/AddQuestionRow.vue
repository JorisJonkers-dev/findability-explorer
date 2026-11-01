<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import type { PromptFamily, PromptSetScope } from '../../../../infrastructure/api/types.gen'
import { languageLabel } from '../../../../shared/vocabulary'
import { SCOPE_LANGUAGES } from '../../domain/draft'
import { AREA, FIELD, LABEL, PRIMARY, SECONDARY } from './look'

/**
 * The add row under a belt family: a scope picker that starts on a scope the family lacks and ends
 * with A new scope…, which asks for a language and a region; then the question; then Add the
 * question and Cancel. It hands the scope (an existing id, or a language and region) and the text up.
 */
const props = defineProps<{ family: PromptFamily; familyLabel: string; scopes: readonly PromptSetScope[]; startScope: string }>()
const emit = defineEmits<{ add: [choice: { scope: string } | { language: string; region: string }, text: string] }>()

const NEW = '__new'
const open = ref(false)
const scope = ref('')
const language = ref('fr')
const region = ref('')
const text = ref('')
const problem = ref<string | null>(null)
const id = useId()
const isNew = computed(() => scope.value === NEW)

function start(): void {
  open.value = true
  scope.value = props.startScope || NEW
  text.value = ''
  region.value = ''
  problem.value = null
}

function add(): void {
  if (text.value.trim() === '') {
    problem.value = 'Write the question first.'
    return
  }
  emit('add', isNew.value ? { language: language.value, region: region.value } : { scope: scope.value }, text.value)
  open.value = false
}
</script>

<template>
  <div class="flex flex-col">
    <button v-if="!open" type="button" :class="[SECONDARY, 'self-start']" @click="start">
      + Add a question<span class="sr-only">{{ ` to ${familyLabel}` }}</span>
    </button>
    <form v-else :aria-label="`A new question for ${familyLabel}`" novalidate class="flex min-w-0 flex-col gap-3 rounded-2xl bg-white p-4" @submit.prevent="add">
      <div class="grid min-w-0 gap-3 min-[700px]:grid-cols-3">
        <label :class="LABEL" :for="`${id}-scope`">
          Scope
          <select :id="`${id}-scope`" v-model="scope" :class="FIELD">
            <option v-for="s in scopes" :key="s.id" :value="s.id">{{ s.name }}</option>
            <option :value="NEW">A new scope…</option>
          </select>
        </label>
        <template v-if="isNew">
          <label :class="LABEL" :for="`${id}-language`">
            Language
            <select :id="`${id}-language`" v-model="language" :class="FIELD">
              <option v-for="l in SCOPE_LANGUAGES" :key="l" :value="l">{{ languageLabel(l) }}</option>
            </select>
          </label>
          <label :class="LABEL" :for="`${id}-region`">
            Region, if any
            <input :id="`${id}-region`" v-model="region" type="text" maxlength="100" placeholder="For example Lyon" :class="FIELD" />
          </label>
        </template>
      </div>
      <label :class="LABEL" :for="`${id}-text`">
        Question
        <textarea :id="`${id}-text`" v-model="text" rows="3" maxlength="1000" placeholder="Write the question as a buyer would ask it" :class="AREA"></textarea>
      </label>
      <p v-if="problem" role="alert" class="m-0 font-medium text-orange-ink">{{ problem }}</p>
      <div class="flex flex-wrap items-center gap-3">
        <button type="submit" :class="PRIMARY">Add the question</button>
        <button type="button" :class="SECONDARY" @click="open = false">Cancel</button>
      </div>
    </form>
  </div>
</template>
