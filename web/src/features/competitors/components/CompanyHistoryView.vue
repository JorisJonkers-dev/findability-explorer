<script setup lang="ts">
import { computed, toRef } from 'vue'
import PageView from '../../shell/components/PageView.vue'
import { companyPath } from '../../shell/domain/sections'
import ReadinessFiles from '../../readiness/components/ReadinessFiles.vue'
import { useCompany } from '../composables/useCompany'
import { useCompanyProfile } from '../composables/useCompanyHistory'
import ScanCompare from './ScanCompare.vue'
import ScanTimeline from './ScanTimeline.vue'
import TermProfile from './TermProfile.vue'

const props = defineProps<{ companyId: string }>()

const id = toRef(() => props.companyId)
const { detail, notFound, error } = useCompany(id)
const { profile, error: profileError } = useCompanyProfile(id)
const title = computed(() => (detail.value ? `How ${detail.value.competitor.name} changed, scan by scan.` : 'Search terms, scan by scan.'))
const scans = computed(() => detail.value?.scans ?? [])
</script>

<template>
  <PageView section="competitors" :title="title" :back="{ to: companyPath(companyId), label: 'Back to the latest scan' }" :source="['Scans: live, from this app', 'Grouped by the search term list']">
    <div class="flex flex-col gap-10">
      <p v-if="notFound" role="alert" class="m-0 text-orange-ink">No company has this id.</p>
      <p v-else-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
      <template v-else>
        <div class="flex min-w-0 flex-col">
          <p v-if="profileError" role="alert" class="m-0 text-orange-ink">{{ profileError }}</p>
          <TermProfile v-else-if="profile" :profile="profile" />
          <p v-else class="m-0 text-muted">Loading the search terms…</p>
        </div>
        <div class="flex min-w-0 flex-col">
          <ScanTimeline :scans="scans" :company-id="companyId" />
        </div>
        <div class="flex min-w-0 flex-col">
          <!-- The raw files the newest scan read, under the comparison. -->
          <div class="flex flex-col gap-10">
            <ScanCompare :company-id="companyId" :scans="scans" />
            <ReadinessFiles :company-id="companyId" />
          </div>
        </div>
      </template>
    </div>
  </PageView>
</template>
