<script setup lang="ts">
import { computed, nextTick, ref, useTemplateRef } from 'vue'
import { useRemoveCompany, useUpdateCompany } from '../composables/useCompanyAdmin'
import { ARCHIVE_ACTION, canRemove, checkCompanyForm, removeAction } from '../domain/admin'
import type { Company } from '../domain/company'

const props = defineProps<{ company: Company }>()
const emit = defineEmits<{ removed: [] }>()

const { isSaving, error: saveError, update, reset } = useUpdateCompany()
const { isRemoving, error: removeError, remove } = useRemoveCompany()
// Archive is the default for every company; an added one can also be deleted with its scans.
const mode = ref<'idle' | 'edit' | 'archive' | 'remove'>('idle')
const name = ref('')
const website = ref('')
const formError = ref<string | null>(null)
const nameInput = useTemplateRef<HTMLInputElement>('nameInput')
const cancelButton = useTemplateRef<HTMLButtonElement>('cancelButton')
const action = computed(() => (mode.value === 'remove' ? removeAction(props.company) : ARCHIVE_ACTION))

const BUTTON = 'min-h-11 rounded-full border-2 px-5 font-medium'
const FIELD = 'h-11 min-w-0 rounded-xl border-2 border-grey bg-white px-3 text-base text-black'

async function edit(): Promise<void> {
  name.value = props.company.name
  website.value = props.company.websiteUrl ?? ''
  formError.value = null
  reset()
  mode.value = 'edit'
  await nextTick()
  nameInput.value?.focus()
}

async function ask(what: 'archive' | 'remove'): Promise<void> {
  mode.value = what
  await nextTick()
  // Cancel is the safe default, so it takes the focus.
  cancelButton.value?.focus()
}

function save(): void {
  const check = checkCompanyForm(name.value, website.value)
  if (check.kind === 'invalid') {
    formError.value = check.message
    return
  }
  formError.value = null
  const body = {
    ...(check.name === props.company.name ? {} : { name: check.name }),
    ...(check.url === props.company.websiteUrl ? {} : { url: check.url }),
  }
  if (Object.keys(body).length === 0) {
    mode.value = 'idle'
    return
  }
  update(props.company.id, body, () => {
    mode.value = 'idle'
  })
}

function restore(): void {
  update(props.company.id, { archived: false }, () => {
    mode.value = 'idle'
  })
}

function confirm(): void {
  if (mode.value === 'archive') {
    update(props.company.id, { archived: true }, () => {
      mode.value = 'idle'
    })
    return
  }
  remove(props.company.id, () => {
    mode.value = 'idle'
    emit('removed')
  })
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <p v-if="saveError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ saveError }}</p>
    <p v-if="removeError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ removeError }}</p>
    <form
      v-if="mode === 'edit'"
      novalidate
      :aria-label="`Edit ${company.name}`"
      class="flex flex-col gap-3 rounded-2xl bg-mist p-4"
      @submit.prevent="save"
    >
      <label class="flex flex-col gap-1.5 text-sm text-ink-2">
        Name
        <input ref="nameInput" v-model="name" type="text" maxlength="200" :class="FIELD" />
      </label>
      <label class="flex flex-col gap-1.5 text-sm text-ink-2">
        Website
        <input v-model="website" type="url" inputmode="url" spellcheck="false" :class="FIELD" />
      </label>
      <p v-if="formError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ formError }}</p>
      <div class="flex flex-wrap gap-3">
        <button type="submit" :disabled="isSaving" :class="BUTTON" class="border-black bg-black text-white hover:bg-ink-2 disabled:opacity-70">
          {{ isSaving ? 'Saving…' : 'Save' }}
        </button>
        <button type="button" :class="BUTTON" class="border-grey bg-white text-black hover:border-black" @click="mode = 'idle'">Cancel</button>
      </div>
    </form>
    <div
      v-else-if="mode === 'archive' || mode === 'remove'"
      role="group"
      :aria-label="`${mode === 'archive' ? 'Archive' : 'Remove'} ${company.name}?`"
      class="flex flex-wrap items-center gap-3"
    >
      <span class="font-medium">{{ action.question }}</span>
      <button
        type="button"
        :disabled="isRemoving || isSaving"
        :class="BUTTON"
        class="border-black bg-black text-white hover:bg-ink-2 disabled:opacity-70"
        @click="confirm"
      >
        {{ isRemoving ? 'Removing…' : isSaving ? 'Archiving…' : action.button }}
      </button>
      <button ref="cancelButton" type="button" :class="BUTTON" class="border-grey bg-white text-black hover:border-black" @click="mode = 'idle'">Cancel</button>
    </div>
    <div v-else role="group" aria-label="Manage this company" class="flex flex-wrap gap-3">
      <button v-if="company.archived" type="button" :disabled="isSaving" :class="BUTTON" class="border-black bg-white text-black hover:bg-black hover:text-white" @click="restore">
        Restore
      </button>
      <template v-else>
        <button type="button" :class="BUTTON" class="border-grey bg-white text-black hover:border-black" @click="edit">Edit</button>
        <button type="button" :class="BUTTON" class="border-grey bg-white text-black hover:border-black" @click="ask('archive')">Archive</button>
        <button v-if="canRemove(company)" type="button" :class="BUTTON" class="border-grey bg-white text-black hover:border-black" @click="ask('remove')">
          Remove
        </button>
      </template>
    </div>
  </div>
</template>
