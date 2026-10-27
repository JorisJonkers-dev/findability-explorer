<script setup lang="ts">
import StageColumn from '../../../shared/ui/StageColumn.vue'
import { useColumnLayout } from '../../../shared/useColumnLayout'
import { computed, toRef } from 'vue'
import { RouterLink } from 'vue-router'
import SectionView from '../../shell/components/SectionView.vue'
import { companyTabs } from '../../shell/domain/sections'
import ReadinessFiles from '../../readiness/components/ReadinessFiles.vue'
import { useMe } from '../../shell/composables/useMe'
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
const { isAdmin } = useMe()

// Each column can be widened or minimised on the stage; the browser remembers the layout.
const COLUMNS = ['Search terms by belt family', 'Scan history', 'Compare two scans'] as const
const columns = useColumnLayout('history', [1.6, 0.7, 1.1])
function columnProps(i: number) {
  return { title: COLUMNS[i] ?? '', state: columns.states.value[i] ?? 'normal', canMinimise: columns.canMinimise(i) }
}
</script>

<template>
  <SectionView section="competitors" :title="title" :tabs="companyTabs(companyId)" :source="['Scans: live, from this app', 'Grouped by the terminology map']">
    <div
      class="flex flex-col gap-10 stage:grid stage:min-h-0 stage:grid-rows-[auto_minmax(0,1fr)] stage:gap-x-24 stage:gap-y-10"
      :style="{ gridTemplateColumns: columns.template.value }"
    >
      <RouterLink :to="{ path: '/competitors', query: { company: companyId } }" class="self-start font-medium stage:col-span-3 stage:text-[30px]">
        Back to the company
      </RouterLink>
      <p v-if="notFound" role="alert" class="m-0 text-orange-ink stage:col-span-3">No company has this id.</p>
      <p v-else-if="error" role="alert" class="m-0 text-orange-ink stage:col-span-3">{{ error }}</p>
      <template v-else>
        <StageColumn v-bind="columnProps(0)" @widen="columns.widen(0)" @minimise="columns.minimise(0)" @restore="columns.restore(0)">
          <p v-if="profileError" role="alert" class="m-0 text-orange-ink">{{ profileError }}</p>
          <TermProfile v-else-if="profile" :profile="profile" />
          <p v-else class="m-0 text-muted">Loading the search terms…</p>
        </StageColumn>
        <StageColumn v-bind="columnProps(1)" @widen="columns.widen(1)" @minimise="columns.minimise(1)" @restore="columns.restore(1)">
          <ScanTimeline :scans="scans" :company-id="companyId" />
        </StageColumn>
        <StageColumn v-bind="columnProps(2)" @widen="columns.widen(2)" @minimise="columns.minimise(2)" @restore="columns.restore(2)">
          <!-- An administrator also sees the raw files the newest scan read, under the comparison. -->
          <div class="flex flex-col gap-10 stage:grid stage:min-h-0 stage:gap-14" :class="isAdmin ? 'stage:grid-rows-[minmax(0,1fr)_minmax(0,1fr)]' : 'stage:grid-rows-[minmax(0,1fr)]'">
            <ScanCompare :company-id="companyId" :scans="scans" />
            <ReadinessFiles v-if="isAdmin" :company-id="companyId" />
          </div>
        </StageColumn>
      </template>
    </div>
  </SectionView>
</template>
