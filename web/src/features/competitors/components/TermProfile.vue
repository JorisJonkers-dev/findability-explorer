<script setup lang="ts">
import { vScrollHint } from '../../../shared/scrollHint'
import { computed, ref, watch } from 'vue'
import type { CompetitorProfile, ProfileGroup } from '../../../infrastructure/api/types.gen'
import { tierFill } from '../../../shared/tiers'
import { tierLabel } from '../../../shared/vocabulary'
import PillButton from '../../../shared/ui/PillButton.vue'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import { byLanguage, firstGroupWithTerms, groupLabel, groupTabLabel, mapLabel, scanDate, scanName, sinceLabel } from '../domain/history'

const props = defineProps<{ profile: CompetitorProfile }>()

const picked = ref<ProfileGroup | undefined>(firstGroupWithTerms(props.profile.groups))
watch(
  () => props.profile.competitorId,
  () => {
    picked.value = firstGroupWithTerms(props.profile.groups)
  },
)
const group = computed(() => props.profile.groups.find((g) => g.group === picked.value))
const languages = computed(() => byLanguage(group.value?.terms ?? []))
const latest = computed(() => props.profile.scan?.number)
</script>

<template>
  <section aria-labelledby="profile-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-7">
    <div class="flex flex-col gap-1.5 stage:gap-2">
      <SectionHeading id="profile-title">Search terms by belt family</SectionHeading>
      <p v-if="profile.scan" class="m-0 text-sm text-muted stage:text-[26px]">
        From {{ scanName(profile.scan) }} of {{ scanDate(profile.scan) }}, the latest of {{ profile.finishedScans }} finished
        {{ profile.finishedScans === 1 ? 'scan' : 'scans' }}.
      </p>
    </div>
    <p v-if="!profile.scan" class="m-0 text-ink-2 stage:text-[30px]">No scan of this company has finished yet.</p>
    <template v-else>
      <div role="group" aria-label="Belt family" class="flex flex-wrap gap-2 stage:gap-3">
        <PillButton v-for="g in profile.groups" :key="g.group" size="sm" :pressed="picked === g.group" @click="picked = g.group">
          {{ groupTabLabel(g) }}
        </PillButton>
      </div>
      <div v-scroll-hint class="scroll-hint flex flex-col gap-5 stage:min-h-0 stage:flex-1 stage:gap-8 stage:overflow-y-auto">
        <p v-if="languages.length === 0" class="m-0 text-ink-2 stage:text-[30px]">
          The scan found no {{ group ? groupLabel(group.group).toLowerCase() : '' }} terms.
        </p>
        <table v-for="lang in languages" :key="lang.language" class="w-full border-collapse text-sm stage:text-[26px]">
          <caption class="pb-1.5 text-left font-semibold stage:pb-3 stage:text-[30px]">{{ lang.label }} ({{ lang.terms.length }})</caption>
          <thead class="sr-only">
            <tr>
              <th scope="col">Term</th>
              <th scope="col">Score</th>
              <th scope="col">Pages</th>
              <th scope="col">Tier</th>
              <th scope="col">Terminology map</th>
              <th scope="col">Found</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in lang.terms" :key="t.phrase" class="border-t border-grey align-baseline">
              <th scope="row" class="py-1.5 pr-3 text-left font-medium stage:py-3 stage:pr-6">{{ t.phrase }}</th>
              <td class="py-1.5 pr-3 text-right tabular-nums stage:py-3 stage:pr-6">{{ t.score }}</td>
              <td class="py-1.5 pr-3 text-right tabular-nums text-ink-2 stage:py-3 stage:pr-6">{{ t.pages }}&nbsp;p.</td>
              <td class="py-1.5 pr-3 stage:py-3 stage:pr-6">
                <span class="inline-flex items-center gap-1.5 stage:gap-3">
                  <span aria-hidden="true" class="size-2.5 flex-none rounded-full stage:size-[16px]" :class="tierFill(t.tier)"></span>{{ tierLabel(t.tier) }}
                </span>
              </td>
              <td class="py-1.5 pr-3 text-ink-2 stage:py-3 stage:pr-6">{{ mapLabel(t) }}</td>
              <td class="py-1.5 text-muted stage:py-3" :class="t.firstScan === latest ? 'font-semibold text-orange-ink' : ''">{{ sinceLabel(t.firstScan, latest) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </section>
</template>
