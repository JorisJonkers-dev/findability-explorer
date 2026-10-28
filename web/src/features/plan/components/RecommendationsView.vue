<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import type { PlanTarget } from '../../../infrastructure/api/types.gen'
import BenefitEffortChart from '../../../shared/ui/charts/BenefitEffortChart.vue'
import SectionView from '../../shell/components/SectionView.vue'
import { useMe } from '../../shell/composables/useMe'
import { usePlan } from '../composables/usePlan'
import { benefitText, comparison, compareLine, dotTip, effortText, rankOpen, scoreDots, SORTS, type Ranked, type Sort } from '../domain/recommendations'
import { targetLabel, TARGETS } from '../domain/routine'

const { isAdmin } = useMe()
const { plan, isLoading, error, isSaving, saveError, tick, setTargets } = usePlan()

const sort = ref<Sort>('value')
const picked = ref<string | undefined>()
const opened = ref<Record<string, boolean>>({})

const ranked = computed<Ranked[]>(() => rankOpen(plan.value?.improvements ?? [], sort.value))
const top = computed(() => ranked.value.find((r) => r.item.id === picked.value) ?? ranked.value[0])
const rest = computed(() => ranked.value.filter((r) => r !== top.value))
const finished = computed(() => (plan.value?.improvements ?? []).filter((i) => i.done))
const dots = computed(() => ranked.value.map((r) => ({ id: r.item.id, rank: r.rank, benefit: r.item.benefit, effort: r.item.effort, tip: dotTip(r) })))
const topLabel = computed(() => (top.value && picked.value === top.value.item.id && top.value.rank !== 1 ? 'You picked this one' : 'Do this first'))
const checks = computed(() => plan.value?.readinessChecks ?? 0)

function setSort(next: Sort): void {
  sort.value = next
  picked.value = undefined
}
function toggle(id: string): void {
  opened.value = { ...opened.value, [id]: !opened.value[id] }
}
function markDone(id: string): void {
  if (picked.value === id) picked.value = undefined
  tick(id, true)
}

// Editing the targets: a value per target, as typed, until saved.
const editing = ref(false)
const draft = ref<Partial<Record<PlanTarget, string>>>({})
function edit(): void {
  draft.value = Object.fromEntries((plan.value?.watched ?? []).map((w) => [w.target, String(w.goal)]))
  editing.value = true
}
function save(): void {
  setTargets(
    Object.entries(draft.value)
      .filter(([, v]) => v !== '')
      .map(([target, v]) => ({ target: target as PlanTarget, value: Number(v) })),
  )
  editing.value = false
}

const TONE = { tribelt: 'bg-orange', leader: 'bg-blue-grey', target: 'bg-black' } as const
const button = 'inline-flex min-h-12 items-center rounded-full border-2 border-black bg-white px-5 font-bold text-black no-underline hover:bg-mist'
</script>

<template>
  <SectionView section="recommendations" title="Start with the changes that help Tribelt most.">
    <template #lede>
      <p class="m-0 max-w-[62ch] text-lg text-ink-2">
        Each recommendation is ranked by how much it helps against how much work it takes. Mark one as done and the rest move up.
      </p>
    </template>
    <template #aside>
      <div role="group" aria-label="Rank by" class="flex flex-wrap gap-2">
        <button
          v-for="s in SORTS"
          :key="s.id"
          type="button"
          :aria-pressed="sort === s.id"
          class="min-h-11 cursor-pointer rounded-full border-2 px-4 font-semibold"
          :class="sort === s.id ? 'border-black bg-black text-white' : 'border-silver bg-white text-black hover:border-black'"
          @click="setSort(s.id)"
        >
          {{ s.label }}
        </button>
      </div>
    </template>

    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the recommendations…</p>
    <div v-else-if="plan" class="flex flex-col gap-10">
      <section v-if="top" aria-labelledby="first-title" class="flex flex-wrap gap-7">
        <article class="flex min-w-0 flex-[999_1_38rem] flex-col gap-5 rounded-[28px] bg-orange-pale p-6 sm:p-9">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="flex items-center gap-3.5">
              <span aria-hidden="true" class="inline-flex size-14 items-center justify-center rounded-2xl bg-orange text-3xl font-extrabold">{{ top.rank }}</span>
              <span class="text-sm font-bold tracking-[0.06em] text-orange-ink uppercase">{{ topLabel }}</span>
            </div>
            <div class="flex flex-wrap gap-2">
              <span class="inline-flex items-center gap-1.5 rounded-full bg-white px-3 py-1.5 text-sm font-semibold">
                Benefit
                <span aria-hidden="true" class="inline-flex gap-[3px]">
                  <span v-for="(on, i) in scoreDots(top.item.benefit)" :key="i" class="size-2.5 rounded-[3px]" :class="on ? 'bg-orange' : 'bg-grey'"></span>
                </span>
                <span class="sr-only">{{ benefitText(top.item.benefit) }}</span>
              </span>
              <span class="rounded-full bg-white px-3 py-1.5 text-sm font-semibold">{{ effortText(top.item.effort) }}</span>
            </div>
          </div>
          <h2 id="first-title" class="m-0 font-display text-4xl leading-[1.1] font-extrabold tracking-[-0.01em]">{{ top.item.title }}</h2>
          <p class="m-0 max-w-[64ch] text-lg text-ink-2">{{ top.item.why }}</p>

          <div class="flex flex-col gap-3 rounded-[20px] bg-white p-5">
            <div class="text-sm font-bold text-muted">{{ top.item.moves }}</div>
            <div v-for="row in comparison(top.item, plan.watched, checks)" :key="row.who" class="grid grid-cols-[6.5rem_minmax(0,1fr)_4.5rem] items-center gap-3 text-[15px]">
              <span :class="row.tone === 'tribelt' ? 'font-bold' : 'text-ink-2'">{{ row.who }}</span>
              <span aria-hidden="true" class="relative block h-3 overflow-hidden rounded-md bg-mist">
                <span class="absolute inset-y-0 left-0 rounded-md" :class="TONE[row.tone]" :style="{ width: `${String(Math.round(row.length * 100))}%` }"></span>
              </span>
              <span class="text-right" :class="row.tone === 'tribelt' ? 'font-bold' : 'text-ink-2'">{{ row.text }}</span>
            </div>
          </div>

          <div class="flex flex-wrap gap-3">
            <button type="button" class="min-h-12 cursor-pointer rounded-full border-0 bg-black px-5 font-bold text-white hover:bg-ink-2" :aria-expanded="!!opened[top.item.id]" @click="toggle(top.item.id)">
              {{ opened[top.item.id] ? 'Hide the steps' : 'Show the steps' }}
            </button>
            <button v-if="isAdmin && !top.item.auto" type="button" :disabled="isSaving" :class="button" class="cursor-pointer" @click="markDone(top.item.id)">Mark as done</button>
            <RouterLink :to="`/recommendations/${top.item.id}`" :class="button">Open the recommendation</RouterLink>
          </div>
          <div v-if="opened[top.item.id]" class="flex flex-col gap-3">
            <ol class="m-0 flex list-none flex-col gap-2.5 p-0">
              <li v-for="(step, i) in top.item.steps" :key="step" class="grid grid-cols-[2rem_minmax(0,1fr)] items-start gap-3">
                <span aria-hidden="true" class="inline-flex size-7 items-center justify-center rounded-full bg-black text-sm font-bold text-white">{{ i + 1 }}</span>
                <span>{{ step }}</span>
              </li>
            </ol>
            <p class="m-0 text-[15px] text-ink-2"><strong>Done when</strong> {{ top.item.doneWhen }}</p>
          </div>
        </article>

        <aside aria-labelledby="chart-title" class="flex min-w-0 flex-[1_1_25rem] flex-col gap-4 rounded-[28px] border-2 border-mist p-7">
          <h2 id="chart-title" class="m-0 font-display text-2xl font-extrabold">Benefit against effort</h2>
          <p class="m-0 text-[15px] text-ink-2">Pick a dot to see that recommendation.</p>
          <BenefitEffortChart :dots="dots" :picked="top.item.id" @pick="picked = $event" />
        </aside>
      </section>

      <section v-else aria-labelledby="all-done" class="rounded-[28px] bg-orange-pale p-9">
        <h2 id="all-done" class="m-0 font-display text-3xl font-extrabold">Every recommendation is done.</h2>
        <p class="m-0 mt-2.5 text-lg text-ink-2">Run a new scan and Prompt test to see what to work on next.</p>
      </section>

      <section v-if="rest.length > 0" aria-labelledby="next-title" class="flex flex-col gap-3.5">
        <h2 id="next-title" class="m-0 font-display text-3xl font-extrabold">Then these</h2>
        <ul aria-label="Recommendations" class="m-0 list-none border-t-2 border-black p-0">
          <li v-for="r in rest" :key="r.item.id" class="border-b border-grey">
            <div class="grid items-center gap-4 rounded-xl px-2 py-4 hover:bg-mist lg:grid-cols-[2.75rem_minmax(0,1.6fr)_minmax(0,1.3fr)_7.5rem_9.5rem_7.5rem]">
              <span class="text-2xl font-extrabold text-muted">{{ r.rank }}</span>
              <div class="flex min-w-0 flex-col gap-1">
                <RouterLink :to="`/recommendations/${r.item.id}`" class="text-xl font-bold text-black no-underline hover:text-orange-ink">{{ r.item.title }}</RouterLink>
                <span class="text-sm text-muted">{{ r.item.moves }}</span>
              </div>
              <div class="flex min-w-0 flex-col gap-1.5">
                <span aria-hidden="true" class="relative block h-2.5 overflow-hidden rounded-[5px] bg-mist">
                  <span
                    class="absolute inset-y-0 left-0 rounded-[5px] bg-orange"
                    :style="{ width: `${String(Math.round((comparison(r.item, plan.watched, checks)[0]?.length ?? 0) * 100))}%` }"
                  ></span>
                </span>
                <span class="text-[13px] text-ink-2">{{ compareLine(r.item, plan.watched, checks) }}</span>
              </div>
              <span class="inline-flex gap-[3px]" :title="benefitText(r.item.benefit)">
                <span v-for="(on, i) in scoreDots(r.item.benefit)" :key="i" aria-hidden="true" class="size-2.5 rounded-[3px]" :class="on ? 'bg-orange' : 'bg-grey'"></span>
                <span class="sr-only">{{ benefitText(r.item.benefit) }}</span>
              </span>
              <span class="text-sm font-semibold">{{ effortText(r.item.effort) }}</span>
              <button
                type="button"
                class="min-h-11 cursor-pointer justify-self-end rounded-full border-2 border-silver bg-white px-4 font-semibold hover:bg-mist"
                :aria-expanded="!!opened[r.item.id]"
                @click="toggle(r.item.id)"
              >
                {{ opened[r.item.id] ? 'Hide the steps' : 'Show the steps' }}
              </button>
            </div>
            <div v-if="opened[r.item.id]" class="flex flex-col gap-3 px-2 pt-1 pb-6 lg:pl-[4.25rem]">
              <p class="m-0 max-w-[70ch] text-ink-2">{{ r.item.why }}</p>
              <ol class="m-0 flex flex-col gap-1.5 pl-5 text-[15px]">
                <li v-for="step in r.item.steps" :key="step">{{ step }}</li>
              </ol>
              <div class="flex flex-wrap gap-2.5">
                <button type="button" :class="button" class="min-h-11 cursor-pointer text-[15px]" @click="picked = r.item.id">Show at the top</button>
                <button v-if="isAdmin && !r.item.auto" type="button" :disabled="isSaving" :class="button" class="min-h-11 cursor-pointer text-[15px]" @click="markDone(r.item.id)">
                  Mark as done
                </button>
                <RouterLink :to="`/recommendations/${r.item.id}`" :class="button" class="min-h-11 text-[15px]">Open the recommendation</RouterLink>
              </div>
            </div>
          </li>
        </ul>
      </section>

      <section v-if="finished.length > 0" aria-labelledby="done-title" class="flex flex-col gap-2.5">
        <h2 id="done-title" class="m-0 font-display text-xl font-extrabold">
          {{ finished.length === 1 ? '1 recommendation done' : `${String(finished.length)} recommendations done` }}
        </h2>
        <ul class="m-0 flex list-none flex-wrap gap-2.5 p-0">
          <li v-for="d in finished" :key="d.id" class="inline-flex items-center gap-2.5 rounded-full bg-mist py-1.5 pr-1.5 pl-4">
            <svg aria-hidden="true" viewBox="0 0 16 16" class="size-4 text-orange-ink" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M3 8.5l3.2 3L13 4.5" />
            </svg>
            <RouterLink :to="`/recommendations/${d.id}`" class="text-muted line-through">{{ d.title }}</RouterLink>
            <button
              v-if="isAdmin && !d.auto"
              type="button"
              :disabled="isSaving"
              class="min-h-9 cursor-pointer rounded-full border-0 bg-white px-3.5 text-sm font-semibold hover:bg-grey"
              @click="tick(d.id, false)"
            >
              Undo
            </button>
          </li>
        </ul>
      </section>

      <p v-if="saveError" role="alert" class="m-0 font-medium text-orange-ink">{{ saveError }}</p>

      <section v-if="isAdmin" aria-labelledby="targets-title" class="flex flex-col gap-3 border-t-2 border-grey pt-6">
        <div class="flex flex-wrap items-baseline justify-between gap-3">
          <h2 id="targets-title" class="m-0 font-display text-xl font-extrabold">Targets</h2>
          <button v-if="!editing" type="button" class="cursor-pointer border-0 bg-transparent p-0 font-medium text-orange-ink underline underline-offset-4" @click="edit">
            Change the targets
          </button>
        </div>
        <form v-if="editing" aria-label="Targets" class="flex max-w-2xl flex-col gap-3" @submit.prevent="save">
          <label v-for="w in plan.watched" :key="w.target" class="flex items-center justify-between gap-3 text-sm">
            {{ targetLabel(w.target, plan.readinessChecks, plan.tribeltTermsFrom) }}
            <input v-model="draft[w.target]" type="number" min="0" :max="w.max" :step="TARGETS[w.target].step" class="h-10 w-28 rounded-lg border-2 border-silver px-2" />
          </label>
          <span class="text-xs text-muted">Shares are fractions: 0.4 is 40%.</span>
          <div class="flex gap-3">
            <button type="submit" class="min-h-11 cursor-pointer rounded-full border-0 bg-orange px-5 font-semibold text-black hover:bg-orange-soft">Save the targets</button>
            <button type="button" class="min-h-11 cursor-pointer rounded-full border-2 border-silver bg-white px-5 font-medium" @click="editing = false">Cancel</button>
          </div>
        </form>
        <p v-else class="m-0 text-sm text-ink-2">The targets are shown on the overview. Raise one when it has been met.</p>
      </section>
    </div>
  </SectionView>
</template>
