<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import BarList from '../../../shared/ui/charts/BarList.vue'
import PageView from '../../shell/components/PageView.vue'
import { companyPath } from '../../shell/domain/sections'
import { useNumbers } from '../composables/useNumbers'
import { numberDetail, targetOfSlug } from '../domain/numbers'

const props = defineProps<{ measure: string }>()
const { facts, isLoading, error } = useNumbers()

const target = computed(() => targetOfSlug(props.measure))
const detail = computed(() => (facts.value && target.value ? numberDetail(facts.value, target.value) : undefined))
const title = computed(() => detail.value?.headline ?? 'Number to watch')
const missing = computed(() => !isLoading.value && !error.value && detail.value === undefined)
</script>

<template>
  <PageView section="overview" eyebrow="Number to watch" :back="{ to: '/', label: 'Back to Overview' }" :title="title">
    <template v-if="detail" #lede>
      <p class="m-0 max-w-[66ch] text-lg leading-normal text-ink-2">{{ detail.counts }}</p>
    </template>

    <p v-if="error" role="alert" class="m-0 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-lg text-muted">Loading the number…</p>
    <p v-else-if="missing" class="m-0 text-lg">There is no number to watch at this address. Pick one from the Overview.</p>

    <template v-else-if="detail">
      <section aria-label="Key figures" class="grid grid-cols-1 gap-5 md:grid-cols-3">
        <div
          v-for="f in detail.figures"
          :key="f.title"
          class="flex flex-col gap-2.5 rounded-3xl px-7 py-6"
          :class="f.tone === 'tribelt' ? 'bg-orange-pale' : 'bg-mist'"
        >
          <span class="font-display text-5xl leading-none font-semibold tracking-[-0.02em]" :class="f.tone === 'tribelt' ? 'text-orange-ink' : ''">{{ f.big }}</span>
          <span class="font-semibold">{{ f.title }}</span>
          <span class="text-muted">{{ f.note }}</span>
        </div>
      </section>

      <section aria-labelledby="compare-title" class="flex flex-col gap-4 rounded-[28px] border-2 border-solid border-grey p-7">
        <h2 id="compare-title" class="m-0 font-display text-2xl font-semibold">Tribelt against the field</h2>
        <BarList :rows="detail.bars" :max="detail.barsMax" :label="detail.label" />
        <p class="m-0 text-sm text-muted">{{ detail.comparison }}</p>
      </section>

      <section aria-labelledby="improves-title" class="flex flex-col gap-3.5">
        <h2 id="improves-title" class="m-0 font-display text-2xl font-semibold">What improves it</h2>
        <ul v-if="detail.improvements.length > 0" class="m-0 list-none border-t-2 border-black p-0">
          <li v-for="i in detail.improvements" :key="i.improvement.id" class="border-b border-grey">
            <RouterLink
              :to="`/recommendations/${i.improvement.id}`"
              class="grid min-h-16 grid-cols-[2.5rem_minmax(0,1fr)_1.5rem] items-center gap-3 rounded-xl p-2 text-black no-underline hover:bg-mist hover:text-black"
            >
              <span class="font-display text-2xl font-semibold text-orange-ink">{{ i.rank }}</span>
              <span class="flex flex-col gap-0.5">
                <span class="text-lg font-bold">{{ i.improvement.title }}</span>
                <span class="text-sm text-muted">{{ i.improvement.done ? 'Done. ' : '' }}{{ i.improvement.moves }}</span>
              </span>
              <svg aria-hidden="true" viewBox="0 0 24 24" class="size-[18px]" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 5l7 7-7 7" /></svg>
            </RouterLink>
          </li>
        </ul>
        <p v-else class="m-0 text-ink-2">No recommendation improves this number yet.</p>
        <RouterLink v-if="detail.leader" :to="companyPath(detail.leader.competitorId)" class="inline-flex min-h-11 items-center self-start font-bold">
          See what {{ detail.leader.name }} does differently
        </RouterLink>
      </section>
    </template>
  </PageView>
</template>
