<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import ProgressBar from '../../../shared/ui/ProgressBar.vue'
import { usePageBudget } from '../../scan/composables/usePageBudget'
import { MAX_SCAN_PAGES } from '../../scan/domain/newScan'
import { siteHost } from '../../scan/domain/results'
import { progress as scanProgress } from '../../scan/domain/status'
import { useAddCompany, useRescan } from '../composables/useCompanyAdmin'
import type { AnalysisSummary } from '../../../infrastructure/api/types.gen'
import type { Company } from '../domain/company'
import { checkCompanyForm, rescanCandidates, rescanSummary } from '../domain/admin'

const props = defineProps<{ companies: readonly Company[] }>()
const emit = defineEmits<{ added: [id: string] }>()

const { pages } = usePageBudget()
const { isQueuing, batch, progress, error: rescanError, rescan } = useRescan()
const candidates = computed(() => rescanCandidates(props.companies).length)
// A re-scan of all reads many sites and uses the language model for each, so it asks first.
const confirming = ref(false)
const { isAdding, error: addError, add, reset } = useAddCompany()
const name = ref('')
const website = ref('')
const formError = ref<{ field: 'name' | 'url'; message: string } | null>(null)
const added = ref<string | null>(null)
const adding = ref(false)
const summary = computed(() => (batch.value ? rescanSummary(batch.value) : null))
const route = useRoute()

/** The company a running scan reads, by its short name, or its website's host. */
function companyOf(scan: AnalysisSummary): string {
  return props.companies.find((c) => c.id === scan.competitorId)?.shortName ?? siteHost(scan.url)
}

const BUTTON = 'min-h-11 rounded-full border-2 px-5 font-medium stage:min-h-[76px] stage:border-[3px] stage:px-8 stage:text-[26px]'
const FIELD = 'h-11 min-w-0 rounded-xl border-2 bg-white px-3 text-base text-black stage:h-[76px] stage:border-[3px] stage:px-5 stage:text-[28px]'

function rescanAll(): void {
  confirming.value = false
  rescan({ all: true, maxPages: pages.value })
}

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
        v-if="!confirming"
        type="button"
        :disabled="isQueuing || (progress !== null && !progress.finished)"
        :class="BUTTON"
        class="border-black bg-white text-black hover:bg-black hover:text-white disabled:opacity-70"
        @click="confirming = true"
      >
        {{ isQueuing ? 'Queuing…' : `Re-scan all ${String(candidates)} companies with data` }}
      </button>
    </div>
    <div v-if="confirming" role="group" aria-labelledby="rescan-question" class="flex flex-col gap-3 rounded-2xl bg-white p-4 stage:gap-5 stage:p-8">
      <p id="rescan-question" class="m-0 text-sm stage:text-[26px]">
        Re-scan {{ candidates }} companies, up to {{ pages }} pages each? Each scan also uses the language model, so this spends
        part of the monthly budget.
      </p>
      <div class="flex flex-wrap gap-3">
        <button type="button" :class="BUTTON" class="border-black bg-black text-white hover:bg-ink-2" @click="rescanAll">
          Re-scan {{ candidates }} companies
        </button>
        <button type="button" :class="BUTTON" class="border-grey bg-white text-black hover:border-black" @click="confirming = false">Cancel</button>
      </div>
    </div>
    <div v-if="summary || progress" class="flex flex-col gap-3 rounded-2xl bg-white p-4 stage:gap-5 stage:p-8" data-testid="rescan-progress">
      <template v-if="progress">
        <p class="m-0 text-base font-semibold stage:text-[30px]" aria-live="polite">{{ progress.text }}</p>
        <ProgressBar thick :fraction="progress.fraction" label="Re-scan progress" :text="`${progress.settled} of ${progress.total} finished`" />
        <ul v-if="progress.running.length > 0" aria-label="Scanning now" class="m-0 flex list-none flex-col gap-3 p-0 stage:gap-5">
          <li v-for="scan in progress.running" :key="scan.id" class="flex flex-col gap-1.5 stage:gap-2.5">
            <div class="flex flex-wrap items-baseline justify-between gap-x-3 text-sm stage:text-[26px]">
              <RouterLink :to="{ path: `/scan/${scan.id}`, query: route.query }" class="font-medium">{{ companyOf(scan) }}</RouterLink>
              <span class="text-ink-2">{{ scanProgress(scan).label }}</span>
            </div>
            <ProgressBar :fraction="scanProgress(scan).fraction" :label="`Scan of ${companyOf(scan)}`" :text="scanProgress(scan).label" />
          </li>
        </ul>
        <p v-if="progress.waiting > 0" class="m-0 text-sm text-ink-2 stage:text-[26px]">
          {{ progress.waiting === 1 ? '1 company waits its turn.' : `${progress.waiting} companies wait their turn.` }}
        </p>
      </template>
      <p v-if="summary" class="m-0 text-sm text-ink-2 stage:text-[26px]">{{ summary }}</p>
    </div>
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
