<script setup lang="ts">
import { computed, toRef } from 'vue'
import { RouterLink } from 'vue-router'
import SectionView from '../../shell/components/SectionView.vue'
import { useCompany } from '../composables/useCompany'
import { useCompanyProfile } from '../composables/useCompanyHistory'
import ScanCompare from './ScanCompare.vue'
import ScanTimeline from './ScanTimeline.vue'
import TermProfile from './TermProfile.vue'

const props = defineProps<{ companyId: string }>()

const id = toRef(() => props.companyId)
const { detail, notFound, error } = useCompany(id)
const { profile, error: profileError } = useCompanyProfile(id)
const title = computed(() => (detail.value ? `${detail.value.competitor.name}: its search terms, scan by scan.` : 'Search terms, scan by scan.'))
const scans = computed(() => detail.value?.scans ?? [])
</script>

<template>
  <SectionView section="competitors" :title="title" :source="['Scans: live, from this app', 'Grouped by the terminology map']">
    <div class="flex flex-col gap-10 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.6fr)_minmax(0,0.7fr)_minmax(0,1.1fr)] stage:grid-rows-[auto_minmax(0,1fr)] stage:gap-x-24 stage:gap-y-10">
      <RouterLink :to="{ path: '/competitors', query: { company: companyId } }" class="self-start font-medium stage:col-span-3 stage:text-[30px]">
        Back to the company
      </RouterLink>
      <p v-if="notFound" role="alert" class="m-0 text-orange-ink stage:col-span-3">No company has this id.</p>
      <p v-else-if="error" role="alert" class="m-0 text-orange-ink stage:col-span-3">{{ error }}</p>
      <template v-else>
        <p v-if="profileError" role="alert" class="m-0 text-orange-ink">{{ profileError }}</p>
        <TermProfile v-else-if="profile" :profile="profile" />
        <p v-else class="m-0 text-muted">Loading the search terms…</p>
        <ScanTimeline :scans="scans" :company-id="companyId" />
        <ScanCompare :company-id="companyId" :scans="scans" />
      </template>
    </div>
  </SectionView>
</template>
