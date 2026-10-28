<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import ScanForm from '../../scan/components/ScanForm.vue'
import SectionView from '../../shell/components/SectionView.vue'
import { useMe } from '../../shell/composables/useMe'
import { useCompetitorBoard } from '../composables/useCompetitorBoard'
import AddCompetitors from './AddCompetitors.vue'
import AdminTools from './AdminTools.vue'
import CompetitorList from './CompetitorList.vue'
import FamilyChart from './FamilyChart.vue'
import FigureRow from './FigureRow.vue'
import ListUpkeep from './ListUpkeep.vue'
import { useCompanies } from '../composables/useCompanies'

const router = useRouter()
const { isAdmin } = useMe()
const { companies } = useCompanies()
const { rows, outOf, tribelt, tribeltShare, tribeltReady, group, title, figures, bars, isLoading, error } = useCompetitorBoard()
const tribeltRow = computed(() => (tribelt.value ? { name: tribelt.value.shortName, share: tribeltShare.value, ready: tribeltReady.value } : undefined))
// The scanning tools work on the companies on the list, not on the archived ones.
const active = computed(() => companies.value.filter((c) => !c.archived && c.role !== 'mirror'))

/** A started scan opens on its own page. */
function started(id: string): void {
  void router.push(`/scan/${id}`)
}
</script>

<template>
  <SectionView section="competitors" :title="title" :source="['Desk research, 22 and 26 October 2026', 'Scans and Prompt runs: live, from this app']">
    <template #lede>
      <p class="m-0 max-w-3xl text-lg text-ink-2 stage:max-w-none stage:text-[34px]">
        Each competitor has its own page, with the belts it sells, what it calls them, and how often AI answers name it.
      </p>
    </template>
    <div class="flex flex-col gap-12 stage:min-h-0 stage:gap-16 stage:overflow-y-auto">
      <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
      <p v-else-if="isLoading" class="m-0 text-muted">Loading the competitors…</p>
      <template v-else>
        <FigureRow :figures="figures" />
        <div class="grid gap-12 stage:gap-16 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)] lg:items-start">
          <FamilyChart v-if="bars.length > 0" :bars="bars" />
          <CompetitorList :rows="rows" :tribelt="tribeltRow" :group="group.map((c) => c.shortName)" :out-of="outOf" />
        </div>
        <section v-if="isAdmin" aria-labelledby="admin-tools-title" class="flex flex-col gap-8 rounded-3xl bg-mist p-6 stage:gap-12 stage:rounded-[36px] stage:p-12">
          <h2 id="admin-tools-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Manage the list</h2>
          <ScanForm @started="started" />
          <AdminTools :companies="active" @added="(id: string) => router.push(`/competitors/${id}`)" />
          <AddCompetitors :companies="active" />
          <ListUpkeep />
        </section>
      </template>
    </div>
  </SectionView>
</template>
