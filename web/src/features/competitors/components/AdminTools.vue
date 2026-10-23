<script setup lang="ts">
import { computed, ref } from 'vue'
import { usePageBudget } from '../../scan/composables/usePageBudget'
import { MAX_SCAN_PAGES } from '../../scan/domain/newScan'
import { useAddCompany, useRescan } from '../composables/useCompanyAdmin'
import { checkCompanyForm, rescanSummary } from '../domain/admin'

const emit = defineEmits<{ added: [id: string] }>()

const { pages } = usePageBudget()
const { isQueuing, batch, error: rescanError, rescan } = useRescan()
const { isAdding, error: addError, add, reset } = useAddCompany()
const name = ref('')
const website = ref('')
const formError = ref<{ field: 'name' | 'url'; message: string } | null>(null)
const added = ref<string | null>(null)
const adding = ref(false)
const summary = computed(() => (batch.value ? rescanSummary(batch.value) : null))

const BUTTON = 'min-h-11 rounded-full border-2 px-5 font-medium stage:min-h-[76px] stage:border-[3px] stage:px-8 stage:text-[26px]'
const FIELD = 'h-11 min-w-0 rounded-xl border-2 bg-white px-3 text-base text-black stage:h-[76px] stage:border-[3px] stage:px-5 stage:text-[28px]'

function budget(event: Event): void {
  pages.value = Number((event.target as HTMLInputElement).value)
}

function submit(): void {
  added.value = null
  reset()
  const check = checkCompanyForm(name.value, website.value)
  if (check.kind === 'invalid') {
    formError.value = { field: check.field, message: check.message }
    return
  }
  formError.value = null
  add(check.name, check.url, (company) => {
    added.value = `Added ${company.name}.`
    name.value = ''
    website.value = ''
    emit('added', company.id)
  })
}
</script>

<template>
  <section aria-labelledby="admin-title" class="flex flex-col gap-4 border-t-2 border-grey pt-5 stage:gap-6 stage:border-t-[3px] stage:pt-9">
    <h3 id="admin-title" class="m-0 text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[24px]">Administrator</h3>
    <div class="flex flex-wrap items-center gap-3 stage:gap-5">
      <label class="flex items-center gap-2 text-sm text-ink-2 stage:gap-4 stage:text-[26px]">
        Pages per scan
        <input
          type="number"
          min="1"
          :max="MAX_SCAN_PAGES"
          :value="pages"
          class="h-11 w-24 rounded-xl border-2 border-grey bg-white px-3 text-base text-black stage:h-[76px] stage:w-[170px] stage:border-[3px] stage:px-5 stage:text-[28px]"
          @change="budget"
        />
      </label>
      <button
        type="button"
        :disabled="isQueuing"
        :class="BUTTON"
        class="border-black bg-white text-black hover:bg-black hover:text-white disabled:opacity-70"
        @click="rescan({ all: true, maxPages: pages })"
      >
        {{ isQueuing ? 'Queuing…' : 'Re-scan all companies with data' }}
      </button>
    </div>
    <p class="m-0 text-sm stage:text-[26px]" aria-live="polite">{{ summary }}</p>
    <p v-if="rescanError" role="alert" class="m-0 text-sm font-medium text-orange-ink stage:text-[26px]">{{ rescanError }}</p>
    <button
      type="button"
      :aria-expanded="adding"
      aria-controls="add-company"
      :class="BUTTON"
      class="self-start border-grey bg-white text-black hover:border-black"
      @click="adding = !adding"
    >
      Add a company
    </button>
    <div v-show="adding" id="add-company">
      <form novalidate aria-label="Add a company" class="mt-3 flex flex-col gap-3 stage:mt-5 stage:gap-4" @submit.prevent="submit">
        <label class="flex flex-col gap-1.5 text-sm text-ink-2 stage:text-[26px]">
          Company name
          <input
            v-model="name"
            type="text"
            maxlength="200"
            autocomplete="organization"
            :aria-invalid="formError?.field === 'name'"
            :class="[FIELD, formError?.field === 'name' ? 'border-orange-ink' : 'border-grey']"
          />
        </label>
        <label class="flex flex-col gap-1.5 text-sm text-ink-2 stage:text-[26px]">
          Company website
          <input
            v-model="website"
            type="text"
            inputmode="url"
            spellcheck="false"
            placeholder="www.example.com"
            :aria-invalid="formError?.field === 'url'"
            :class="[FIELD, formError?.field === 'url' ? 'border-orange-ink' : 'border-grey']"
          />
        </label>
        <p v-if="formError" role="alert" class="m-0 text-sm font-medium text-orange-ink stage:text-[26px]">{{ formError.message }}</p>
        <p v-if="addError" role="alert" class="m-0 text-sm font-medium text-orange-ink stage:text-[26px]">{{ addError }}</p>
        <div class="flex flex-wrap items-center gap-3">
          <button type="submit" :disabled="isAdding" :class="BUTTON" class="border-black bg-black text-white hover:bg-ink-2 disabled:opacity-70">
            {{ isAdding ? 'Adding…' : 'Add company' }}
          </button>
          <span class="text-sm font-medium stage:text-[26px]" aria-live="polite">{{ added }}</span>
        </div>
      </form>
    </div>
  </section>
</template>
