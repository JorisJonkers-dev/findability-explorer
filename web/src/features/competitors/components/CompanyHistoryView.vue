<script setup lang="ts">
import { computed, toRef } from 'vue'
import PageFrame from '../../shell/components/PageFrame.vue'
import { companyPath } from '../../shell/domain/sections'
import ReadinessFiles from '../../readiness/components/ReadinessFiles.vue'
import { useCompany } from '../composables/useCompany'
import { useCompanyProfile } from '../composables/useCompanyHistory'
import { scanDate } from '../domain/table'
import ScanCompare from './ScanCompare.vue'
import ScanTimeline from './ScanTimeline.vue'
import TermProfile from './TermProfile.vue'

/** A company's scan history: each scan, two compared, the newest scan's terms, and the files it read. */
const props = defineProps<{ companyId: string }>()

const id = toRef(() => props.companyId)
const { detail, notFound, error } = useCompany(id)
const { profile, error: profileError } = useCompanyProfile(id)
const company = computed(() => detail.value?.competitor)
const scans = computed(() => detail.value?.scans ?? [])
const title = computed(() => (company.value ? `${company.value.name}, scan history` : 'Scan history'))
const purpose = computed(() =>
  company.value
    ? `Each scan of ${company.value.host ?? company.value.website}, with the pages read and the AI readiness score. Pick two scans to see what changed.`
    : 'Each scan of a company’s website, with the pages read and the AI readiness score.',
)
const dataLine = computed(() => {
  if (!company.value) return undefined
  const newest = scans.value[0]
  return newest === undefined ? 'No scan has run yet.' : `The newest scan is from ${scanDate(newest.createdAt)}.`
})
const back = computed(() => ({ to: companyPath(props.companyId), label: `Back to ${company.value?.name ?? 'the competitor'}` }))
</script>

<template>
  <PageFrame
    section="competitors"
    :title="title"
    :purpose="purpose"
    :data-line="dataLine"
    :back="back"
    :manage="{ to: '/manage/scans', label: 'Manage the website scans' }"
    help-topic="rescan"
  >
    <p v-if="notFound" role="alert" class="m-0 text-orange-ink">No company has this id.</p>
    <p v-else-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <template v-else>
      <ScanTimeline :scans="scans" :company-id="companyId" />
      <ScanCompare :company-id="companyId" :scans="scans" />
      <div class="flex min-w-0 flex-col">
        <p v-if="profileError" role="alert" class="m-0 text-orange-ink">{{ profileError }}</p>
        <TermProfile v-else-if="profile" :profile="profile" />
        <p v-else class="m-0 text-muted">Loading the search terms…</p>
      </div>
      <ReadinessFiles :company-id="companyId" />
    </template>
  </PageFrame>
</template>
