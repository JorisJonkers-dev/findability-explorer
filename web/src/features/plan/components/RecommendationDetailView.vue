<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import SectionView from '../../shell/components/SectionView.vue'
import { useMe } from '../../shell/composables/useMe'
import { companyPath } from '../../shell/domain/sections'
import { usePlan } from '../composables/usePlan'
import { alsoImproving, benefitText, comparison, effortNote, effortText, MEASURE_NAME, ordinal, placeNote, placeOf, scoreDots } from '../domain/recommendations'
import { foundLine } from '../domain/routine'

const props = defineProps<{ id: string }>()
const { isAdmin } = useMe()
const { plan, isLoading, error, isSaving, saveError, tick } = usePlan()

const items = computed(() => plan.value?.improvements ?? [])
const it = computed(() => items.value.find((i) => i.id === props.id))
const place = computed(() => placeOf(items.value, props.id))
const rows = computed(() => (it.value && plan.value ? comparison(it.value, plan.value.watched, plan.value.readinessChecks) : []))
const also = computed(() => (it.value ? alsoImproving(items.value, it.value) : []))
const found = computed(() => (it.value ? foundLine(it.value.parts) : ''))

// The steps a reader has worked through are theirs alone; nothing is stored.
const ticked = ref<Record<number, boolean>>({})
const TONE = { tribelt: 'bg-orange', leader: 'bg-blue-grey', target: 'bg-black' } as const
const back = { to: '/recommendations', label: 'Back to Recommendations' }
</script>

<template>
  <SectionView section="recommendations" :title="it?.title ?? 'Recommendation'" :back="back">
    <template v-if="it" #lede>
      <p class="m-0 max-w-[66ch] text-lg text-ink-2">{{ it.why }}</p>
    </template>
    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the recommendation…</p>
    <p v-else-if="!it" class="m-0 text-ink-2">No recommendation has this address. <RouterLink to="/recommendations">See all the recommendations.</RouterLink></p>
    <div v-else class="flex flex-col gap-8">
      <p v-if="it.done" role="status" class="m-0 flex items-center gap-3 rounded-2xl bg-orange-pale px-5 py-4 text-[17px] font-bold text-orange-ink">
        <svg aria-hidden="true" viewBox="0 0 16 16" class="size-5" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 8.5l3.2 3L13 4.5" />
        </svg>
        Marked as done. The next recommendation moves up.
      </p>

      <section aria-label="Key figures" class="grid gap-5 md:grid-cols-3">
        <div class="flex flex-col gap-2.5 rounded-3xl bg-orange-pale p-6">
          <span aria-hidden="true" class="inline-flex gap-[5px] py-2">
            <span v-for="(on, i) in scoreDots(it.benefit)" :key="i" class="size-4 rounded" :class="on ? 'bg-orange' : 'bg-grey'"></span>
          </span>
          <span class="text-[17px] font-semibold">{{ benefitText(it.benefit) }}</span>
          <span class="text-[15px] text-muted">{{ ['', 'One', 'Two', 'Three', 'Four', 'Five'][it.benefit] }} out of five.</span>
        </div>
        <div class="flex flex-col gap-2.5 rounded-3xl bg-mist p-6">
          <span class="font-display text-4xl leading-none font-extrabold">{{ effortText(it.effort) }}</span>
          <span class="text-[17px] font-semibold">of work</span>
          <span class="text-[15px] text-muted">{{ effortNote(items, it) }}</span>
        </div>
        <div v-if="place" class="flex flex-col gap-2.5 rounded-3xl bg-mist p-6">
          <span class="font-display text-4xl leading-none font-extrabold">{{ ordinal(place.rank) }}</span>
          <span class="text-[17px] font-semibold">Ranked by best value</span>
          <span class="text-[15px] text-muted">{{ placeNote(place.rank, place.of) }}</span>
        </div>
      </section>

      <section class="grid items-start gap-7 lg:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)]">
        <div class="flex flex-col gap-4">
          <h2 class="m-0 font-display text-2xl font-extrabold">The steps</h2>
          <p class="m-0 text-[15px] text-ink-2">Tick off each step as you go.</p>
          <ol class="m-0 flex list-none flex-col gap-2 p-0">
            <li v-for="(step, i) in it.steps" :key="step">
              <button
                type="button"
                :aria-pressed="!!ticked[i]"
                class="grid w-full cursor-pointer grid-cols-[2.25rem_minmax(0,1fr)] items-start gap-3.5 rounded-2xl border-2 border-grey bg-white px-3 py-3.5 text-left text-black"
                @click="ticked = { ...ticked, [i]: !ticked[i] }"
              >
                <span aria-hidden="true" class="inline-flex size-8 items-center justify-center rounded-full text-[15px] font-extrabold" :class="ticked[i] ? 'bg-orange text-black' : 'bg-black text-white'">
                  {{ ticked[i] ? '✓' : i + 1 }}
                </span>
                <span :class="ticked[i] ? 'text-muted line-through' : ''">{{ step }}</span>
              </button>
            </li>
          </ol>
          <p class="m-0"><strong>Done when</strong> {{ it.doneWhen }}</p>
          <p v-if="found" class="m-0 font-medium">{{ found }}</p>
          <div class="flex flex-wrap items-center gap-3">
            <button
              v-if="isAdmin && !it.auto"
              type="button"
              :aria-pressed="it.done"
              :disabled="isSaving"
              class="min-h-12 cursor-pointer rounded-full border-0 bg-black px-5 font-bold text-white hover:bg-ink-2"
              @click="tick(it.id, !it.done)"
            >
              {{ it.done ? 'Undo, it is not done' : 'Mark as done' }}
            </button>
            <span class="text-[15px] text-muted">
              {{
                it.auto
                  ? 'The latest scan of tribelt.nl checks this one.'
                  : it.done && it.tickedBy
                    ? `Marked as done by ${it.tickedBy}.`
                    : "An administrator marks this one as done, because a scan of tribelt.nl can't see whether it was done."
              }}
            </span>
          </div>
          <p v-if="saveError" role="alert" class="m-0 font-medium text-orange-ink">{{ saveError }}</p>
        </div>

        <div class="flex flex-col gap-6">
          <div class="flex flex-col gap-3.5 rounded-3xl border-2 border-mist p-6">
            <h2 class="m-0 font-display text-xl font-extrabold">The measure it improves</h2>
            <p class="m-0 text-[17px] font-bold">{{ MEASURE_NAME[it.measure] }}</p>
            <p class="m-0 text-ink-2">{{ it.moves }}</p>
            <div v-for="row in rows" :key="row.who" class="grid grid-cols-[6.5rem_minmax(0,1fr)_4.5rem] items-center gap-3 text-[15px]">
              <span :class="row.tone === 'tribelt' ? 'font-bold' : 'text-ink-2'">{{ row.who }}</span>
              <span aria-hidden="true" class="relative block h-3 overflow-hidden rounded-md bg-mist">
                <span class="absolute inset-y-0 left-0 rounded-md" :class="TONE[row.tone]" :style="{ width: `${String(Math.round(row.length * 100))}%` }"></span>
              </span>
              <span class="text-right" :class="row.tone === 'tribelt' ? 'font-bold' : 'text-ink-2'">{{ row.text }}</span>
            </div>
          </div>

          <div v-if="it.leader" class="flex flex-col gap-2.5">
            <h2 class="m-0 font-display text-xl font-extrabold">The leader to look at</h2>
            <RouterLink :to="companyPath(it.leader.competitorId)" class="flex min-h-[60px] items-center justify-between gap-3 rounded-2xl border-2 border-grey px-3 py-2 text-black no-underline hover:bg-mist">
              <span class="flex flex-col">
                <span class="text-[17px] font-bold">{{ it.leader.name }}</span>
                <span class="text-sm text-muted">Leads on this measure.</span>
              </span>
              <svg aria-hidden="true" viewBox="0 0 24 24" class="size-[18px]" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 5l7 7-7 7" /></svg>
            </RouterLink>
          </div>

          <div v-if="also.length > 0" class="flex flex-col gap-2.5">
            <h2 class="m-0 font-display text-xl font-extrabold">Also improves this measure</h2>
            <RouterLink
              v-for="o in also"
              :key="o.id"
              :to="`/recommendations/${o.id}`"
              class="flex min-h-[60px] items-center justify-between gap-3 rounded-2xl border-2 border-grey px-3 py-2 text-black no-underline hover:bg-mist"
            >
              <span class="flex flex-col">
                <span class="text-[17px] font-bold">{{ o.title }}</span>
                <span class="text-sm text-muted">{{ effortText(o.effort) }} of work. {{ benefitText(o.benefit) }}.</span>
              </span>
              <svg aria-hidden="true" viewBox="0 0 24 24" class="size-[18px]" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 5l7 7-7 7" /></svg>
            </RouterLink>
          </div>
        </div>
      </section>
    </div>
  </SectionView>
</template>
