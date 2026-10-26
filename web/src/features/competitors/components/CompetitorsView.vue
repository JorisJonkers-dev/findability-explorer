<script setup lang="ts">
import { computed, useTemplateRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SectionView from '../../shell/components/SectionView.vue'
import ScanPanel from '../../scan/components/ScanPanel.vue'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import { useMe } from '../../shell/composables/useMe'
import { useCompanies } from '../composables/useCompanies'
import { splitArchived } from '../domain/admin'
import { researchedCompetitors, selectedCompany } from '../domain/list'
import AdminTools from './AdminTools.vue'
import CompanyCard from './CompanyCard.vue'
import CompanyList from './CompanyList.vue'

defineProps<{ scanId?: string }>()
const route = useRoute()
const router = useRouter()
const { companies: all, isLoading, error } = useCompanies()
const { isAdmin } = useMe()
// Archived researched companies are hidden; administrators see them apart, to restore one.
const split = computed(() => splitArchived(all.value))
const companies = computed(() => split.value.active)
const archived = computed(() => (isAdmin.value ? split.value.archived : []))
const map = useTermMap()
const panel = useTemplateRef<InstanceType<typeof ScanPanel>>('panel')

const selected = computed(() =>
  selectedCompany([...companies.value, ...archived.value], typeof route.query.company === 'string' ? route.query.company : undefined),
)
const title = computed(() => {
  const count = researchedCompetitors(companies.value).length
  return count > 0 ? `${String(count)} makers, the same belts, other names.` : 'The same belts, other names.'
})

/** Shows a company: the address keeps it, so a link or a reload shows the same one. */
function pick(id: string): void {
  void router.replace({ query: { ...route.query, company: id } })
}

/** A removed company is gone from the list, so the view shows the first one again. */
function removed(): void {
  const query = { ...route.query }
  delete query.company
  void router.replace({ query })
}

function scan(url: string): void {
  panel.value?.scan(url)
}
</script>

<template>
  <SectionView section="competitors" :title="title" :source="['Desk research, 22 and 26 October 2026', 'Scans: live, from this app']">
    <div
      class="flex flex-col gap-12 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.3fr)_minmax(0,0.95fr)_minmax(0,1fr)] stage:gap-24"
    >
      <CompanyList :companies="companies" :archived="archived" :selected-id="selected?.id" :loading="isLoading" @pick="pick" />
      <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
      <CompanyCard v-else-if="selected" :company="selected" :terms="map.terms.value" :is-admin="isAdmin" @scan="scan" @removed="removed" />
      <p v-else class="m-0 text-muted">Loading the companies…</p>
      <ScanPanel ref="panel" :scan-id="scanId" :is-admin="isAdmin">
        <AdminTools v-if="isAdmin" :companies="companies" @added="pick" />
      </ScanPanel>
    </div>
  </SectionView>
</template>
