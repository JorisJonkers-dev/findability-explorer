<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { ACTIONS, FIELD, LABEL, PRIMARY, SECONDARY } from '../../manage/components/widgets/look'
import { scanDay } from '../../scan/domain/queue'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { companyPath } from '../../shell/domain/sections'
import { useRemoveCompany, useUpdateCompany } from '../composables/useCompanyAdmin'
import { useCompetitors } from '../composables/useCompetitors'
import { canRemove, checkCompanyForm, removeAction } from '../domain/admin'
import { archivedLine, statusOf } from '../domain/companies'

/** One company on the competitor list: its details with Save, archive or restore, and where it shows. */
const props = defineProps<{ companyId: string }>()
const router = useRouter()
const { competitors, isLoading, error } = useCompetitors()
const company = computed(() => competitors.value.find((c) => c.id === props.companyId))
const { isSaving, error: saveError, update, reset } = useUpdateCompany()
const { isRemoving, error: removeError, remove } = useRemoveCompany()

const name = ref('')
const website = ref('')
const formError = ref<string | null>(null)
const notice = ref<string | null>(null)
const removing = ref(false)

// The form starts from the company once it has loaded, and again when another company opens.
watch(
  () => company.value?.id,
  () => {
    name.value = company.value?.name ?? ''
    website.value = company.value?.websiteUrl ?? ''
  },
  { immediate: true },
)

function save(): void {
  const c = company.value
  if (!c) return
  notice.value = null
  reset()
  const check = checkCompanyForm(name.value, website.value)
  if (check.kind === 'invalid') {
    formError.value = check.message
    return
  }
  formError.value = null
  const body = { ...(check.name === c.name ? {} : { name: check.name }), ...(check.url === c.websiteUrl ? {} : { url: check.url }) }
  if (Object.keys(body).length === 0) {
    notice.value = 'Nothing changed.'
    return
  }
  update(c.id, body, () => {
    notice.value = `${check.name} is saved.`
  })
}

function setArchived(archived: boolean): void {
  const c = company.value
  if (!c) return
  notice.value = null
  update(c.id, { archived }, () => {
    notice.value = archived ? `${c.name} is archived. It no longer counts as a competitor.` : `${c.name} is a competitor again.`
  })
}

function confirmRemove(): void {
  const c = company.value
  if (c) remove(c.id, () => void router.push('/manage/competitors/companies'))
}
</script>

<template>
  <ManageLayout
    :title="company?.name ?? 'Company'"
    purpose="One company on the competitor list."
    :back="{ to: '/manage/competitors/companies', label: 'Back to Companies' }"
    :results="{ to: '/competitors', label: 'See the competitors' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the company…</p>
    <p v-else-if="!company" class="m-0 rounded-2xl bg-mist p-6 text-lg">This company is not on the list.</p>
    <WidgetGrid v-else>
      <ManageWidget title="Details" :span="6">
        <form novalidate :aria-label="`Details of ${company.name}`" class="flex flex-col gap-3" @submit.prevent="save">
          <label :class="LABEL">
            Name
            <input v-model="name" type="text" maxlength="200" :class="FIELD" />
          </label>
          <label :class="LABEL">
            Website
            <input v-model="website" type="text" inputmode="url" spellcheck="false" :class="FIELD" />
          </label>
          <p class="m-0 text-sm text-ink-2">Country: {{ company.country || 'Not known' }}</p>
          <p v-if="formError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ formError }}</p>
          <div :class="ACTIONS">
            <button type="submit" :class="PRIMARY" :disabled="isSaving">Save the changes</button>
          </div>
        </form>
      </ManageWidget>

      <ManageWidget :title="statusOf(company)" :span="6">
        <p class="m-0 text-ink-2">
          {{ archivedLine(company, scanDay) ?? 'Archiving takes the company out of the competitor figures. Its scans stay, and you can restore it at any time.' }}
        </p>
        <template v-if="removing">
          <p class="m-0 font-medium">{{ removeAction(company).question }}</p>
        </template>
        <template #foot>
          <template v-if="removing">
            <button type="button" :class="PRIMARY" :disabled="isRemoving" @click="confirmRemove">{{ removeAction(company).button }}</button>
            <button type="button" :class="SECONDARY" @click="removing = false">Cancel</button>
          </template>
          <template v-else>
            <button v-if="company.archived" type="button" :class="PRIMARY" :disabled="isSaving" @click="setArchived(false)">Restore</button>
            <button v-else type="button" :class="SECONDARY" :disabled="isSaving" @click="setArchived(true)">Archive</button>
            <button v-if="canRemove(company)" type="button" :class="SECONDARY" @click="removing = true">Remove</button>
          </template>
        </template>
      </ManageWidget>

      <p class="m-0 w-full font-medium" aria-live="polite">{{ notice }}</p>
      <p v-if="saveError" role="alert" class="m-0 w-full font-medium text-orange-ink">{{ saveError }}</p>
      <p v-if="removeError" role="alert" class="m-0 w-full font-medium text-orange-ink">{{ removeError }}</p>

      <ManageWidget title="Where it shows" :span="12">
        <ul class="m-0 flex list-none flex-col gap-2 p-0">
          <li><RouterLink :to="companyPath(company.id)">The competitor page for {{ company.name }}</RouterLink></li>
          <li><RouterLink to="/ai-readiness">AI readiness, with every company's site</RouterLink></li>
          <li v-if="company.lastScan"><RouterLink :to="`/manage/scans/${company.lastScan.id}`">The latest website scan</RouterLink></li>
        </ul>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
