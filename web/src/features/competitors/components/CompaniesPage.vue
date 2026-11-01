<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import type { Competitor } from '../../../infrastructure/api/types.gen'
import DataTable from '../../manage/components/widgets/DataTable.vue'
import type { Column } from '../../manage/domain/table'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { ACTIONS, FIELD, LABEL, PRIMARY, ROW_BUTTON } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useAddCompany, useUpdateCompany } from '../composables/useCompanyAdmin'
import { useCompetitors } from '../composables/useCompetitors'
import { checkCompanyForm } from '../domain/admin'
import { listCompanies, listLine, statusOf } from '../domain/companies'
import AddCompetitors from './AddCompetitors.vue'

/** The companies that count as competitors at full width, and below them the ways to add more. */
const { competitors, isLoading, error } = useCompetitors()
const rows = computed(() => listCompanies(competitors.value))
const active = computed(() => rows.value.filter((c) => !c.archived))
const { isSaving, error: saveError, update } = useUpdateCompany()
const notice = ref<string | null>(null)

function setArchived(c: Competitor, archived: boolean): void {
  notice.value = null
  update(c.id, { archived }, () => {
    notice.value = archived ? `${c.name} is archived. It no longer counts as a competitor.` : `${c.name} is a competitor again.`
  })
}

const COLUMNS: readonly Column<Competitor>[] = [
  { key: 'name', label: 'Name', value: (c) => c.name },
  { key: 'website', label: 'Website', value: (c) => c.host ?? c.website, wide: true },
  { key: 'country', label: 'Country', value: (c) => c.country || 'Not known', wide: true },
  { key: 'status', label: 'Status', value: (c) => statusOf(c) },
  { key: 'actions', label: 'Change' },
]

// Adding one company by its name and website.
const { isAdding, error: addError, add, reset } = useAddCompany()
const name = ref('')
const website = ref('')
const formError = ref<{ field: 'name' | 'url'; message: string } | null>(null)
const added = ref<string | null>(null)

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
    added.value = `${company.name} is added to the list.`
    name.value = ''
    website.value = ''
  })
}
</script>

<template>
  <ManageLayout
    title="Companies"
    purpose="The companies that count as competitors, and those that were archived."
    :back="{ to: '/manage/competitors', label: 'Back to Competitor list' }"
    :results="{ to: '/competitors', label: 'See the competitors' }"
  >
    <WidgetGrid>
      <ManageWidget title="The list" :span="12">
        <p class="m-0 text-ink-2">{{ listLine(rows) }} Archiving takes a company out of the competitor figures.</p>
        <p class="m-0 font-medium" aria-live="polite">{{ notice }}</p>
        <p v-if="saveError" role="alert" class="m-0 font-medium text-orange-ink">{{ saveError }}</p>
        <p v-if="error" role="alert" class="m-0 font-medium text-orange-ink">{{ error }}</p>
        <p v-else-if="isLoading" class="m-0 text-muted">Loading the companies…</p>
        <DataTable
          v-else
          :rows="rows"
          :columns="COLUMNS"
          :row-key="(c) => c.id"
          label="Companies"
          :search-text="(c) => `${c.name} ${c.host ?? ''} ${c.country}`"
          empty="No company is on the list yet. Add one below."
        >
          <template #cell-name="{ row }">
            <RouterLink :to="`/manage/competitors/companies/${row.id}`" class="font-semibold">{{ row.name }}</RouterLink>
          </template>
          <template #cell-actions="{ row }">
            <span class="flex flex-wrap gap-2">
              <button v-if="row.archived" type="button" :class="ROW_BUTTON" :disabled="isSaving" @click="setArchived(row, false)">
                Restore<span class="sr-only">{{ ` ${row.name}` }}</span>
              </button>
              <button v-else type="button" :class="ROW_BUTTON" :disabled="isSaving" @click="setArchived(row, true)">
                Archive<span class="sr-only">{{ ` ${row.name}` }}</span>
              </button>
              <RouterLink :to="`/manage/competitors/companies/${row.id}`" :class="ROW_BUTTON">
                Open<span class="sr-only">{{ ` ${row.name}` }}</span>
              </RouterLink>
            </span>
          </template>
        </DataTable>
      </ManageWidget>

      <ManageWidget title="Add a company" :span="6">
        <form novalidate aria-label="Add a company" class="flex flex-col gap-3" @submit.prevent="submit">
          <label :class="LABEL">
            Name
            <input v-model="name" type="text" maxlength="200" autocomplete="organization" :aria-invalid="formError?.field === 'name'" :class="FIELD" />
          </label>
          <label :class="LABEL">
            Website
            <input
              v-model="website"
              type="text"
              inputmode="url"
              spellcheck="false"
              placeholder="www.example.com"
              :aria-invalid="formError?.field === 'url'"
              :class="FIELD"
            />
          </label>
          <p v-if="formError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ formError.message }}</p>
          <p v-if="addError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ addError }}</p>
          <div :class="ACTIONS">
            <button type="submit" :class="PRIMARY" :disabled="isAdding">{{ isAdding ? 'Adding…' : 'Add the company' }}</button>
            <span class="text-sm font-medium" aria-live="polite">{{ added }}</span>
          </div>
        </form>
      </ManageWidget>

      <ManageWidget title="Add several companies" :span="6">
        <AddCompetitors :companies="active" :heading="false" />
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
