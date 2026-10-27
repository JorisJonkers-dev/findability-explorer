<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import type { PlanTarget } from '../../../infrastructure/api/types.gen'
import { percent } from '../../../shared/percent'
import SectionView from '../../shell/components/SectionView.vue'
import { useMe } from '../../shell/composables/useMe'
import { companyPath } from '../../shell/domain/sections'
import { usePlan } from '../composables/usePlan'
import { eyeOnText, foundLine, MONTHS, ROUTINE, standingLine, TARGETS, watchedRow, YEAR } from '../domain/routine'

const { isAdmin } = useMe()
const { plan, isLoading, error, isSaving, saveError, tick, setTargets } = usePlan()
const picked = ref(0)
const step = computed(() => ROUTINE[picked.value] ?? ROUTINE[0])
const done = computed(() => plan.value?.improvements.filter((i) => i.done).length ?? 0)

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

const TONE = { orange: 'bg-orange text-black', black: 'bg-black text-white', 'blue-grey': 'bg-blue-grey text-black' } as const
</script>

<template>
  <SectionView section="plan" title="Check every Monday. Fix one thing a week. Watch the numbers move.">
    <div class="flex flex-col gap-14 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1fr)_minmax(0,1.15fr)_minmax(0,0.9fr)] stage:gap-24">
      <section aria-labelledby="week-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-7">
        <h2 id="week-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Every week, one hour</h2>
        <ol class="m-0 flex list-none flex-col gap-1 p-0">
          <li v-for="(s, i) in ROUTINE" :key="s.number">
            <button
              type="button"
              :aria-pressed="picked === i"
              class="grid w-full grid-cols-[2.5rem_minmax(0,1fr)] items-center gap-3 rounded-xl border-0 p-3 text-left stage:grid-cols-[96px_minmax(0,1fr)] stage:gap-6 stage:rounded-[24px] stage:p-6"
              :class="picked === i ? 'bg-mist' : 'bg-transparent hover:bg-mist'"
              @click="picked = i"
            >
              <span aria-hidden="true" class="font-display text-3xl font-semibold stage:text-[72px]" :class="picked === i ? 'text-orange-ink' : 'text-silver'">{{ s.number }}</span>
              <span class="flex flex-col">
                <span class="font-semibold stage:text-[34px]">{{ s.title }}</span>
                <span class="text-sm text-ink-2 stage:text-[24px]">{{ s.when }}</span>
              </span>
            </button>
          </li>
        </ol>
        <div v-if="step" aria-live="polite" class="flex flex-col gap-3 border-l-4 border-orange pl-4 stage:gap-4 stage:border-l-[6px] stage:pl-8">
          <p class="m-0 stage:text-[30px]">{{ step.what }}</p>
          <p class="m-0 text-ink-2 stage:text-[28px]">{{ step.look }}</p>
          <RouterLink :to="step.link.path" class="self-start font-semibold stage:text-[28px]">{{ step.link.label }}</RouterLink>
        </div>
        <div class="flex flex-col gap-2 stage:mt-auto stage:gap-3">
          <span class="text-sm font-semibold tracking-[0.04em] text-orange-ink uppercase stage:text-[22px]">The year</span>
          <div class="grid grid-cols-12 gap-1">
            <span
              v-for="bar in YEAR"
              :key="bar.label"
              class="truncate px-1.5 py-1 text-xs font-semibold stage:px-3 stage:py-2 stage:text-[22px]"
              :class="TONE[bar.tone]"
              :style="{ gridColumn: `${String(bar.start)} / span ${String(bar.span)}` }"
            >{{ bar.label }}</span>
          </div>
          <div aria-hidden="true" class="grid grid-cols-12 gap-1 text-[10px] text-muted stage:text-[20px]">
            <span v-for="m in MONTHS" :key="m">{{ m }}</span>
          </div>
        </div>
      </section>

      <section aria-labelledby="fix-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-7">
        <div class="flex flex-wrap items-baseline justify-between gap-3">
          <h2 id="fix-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Improvement checklist</h2>
          <span v-if="plan" class="stage:text-[28px]"><strong class="text-orange-ink">{{ done }}</strong> of {{ plan.improvements.length }} done</span>
        </div>
        <p class="m-0 text-ink-2 stage:text-[28px]">Work through them from the top. Each one is a change to tribelt.nl.</p>
        <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
        <p v-else-if="isLoading" class="m-0 text-muted">Loading the plan…</p>
        <ol v-else-if="plan" aria-label="Improvement checklist" class="m-0 flex list-none flex-col p-0 stage:min-h-0 stage:overflow-y-auto">
          <li
            v-for="(it, i) in plan.improvements"
            :key="it.id"
            class="grid grid-cols-[1.75rem_minmax(0,1fr)_auto] items-start gap-3 border-t-2 border-grey py-3 stage:grid-cols-[56px_minmax(0,1fr)_260px] stage:gap-6 stage:py-6"
          >
            <input
              :id="`imp-${it.id}`"
              type="checkbox"
              :checked="it.done"
              :disabled="it.auto || !isAdmin || isSaving"
              class="mt-1 size-5 accent-orange-ink stage:size-10"
              @change="tick(it.id, ($event.target as HTMLInputElement).checked)"
            />
            <div class="flex min-w-0 flex-col gap-1.5 stage:gap-3">
              <label :for="`imp-${it.id}`" class="font-semibold stage:text-[32px]" :class="it.done ? 'text-muted line-through' : ''">
                <span class="text-orange-ink">{{ i + 1 }}.</span> {{ it.title }}
              </label>
              <p class="m-0 text-sm stage:text-[26px]">{{ it.why }}</p>
              <p class="m-0 text-sm text-ink-2 stage:text-[24px]">
                {{ [`Moves ${it.moves}.`, standingLine(it)].filter(Boolean).join(' ') }}
              </p>
              <p v-if="foundLine(it.parts)" class="m-0 text-sm font-medium stage:text-[24px]">{{ foundLine(it.parts) }}</p>
              <details class="group text-sm stage:text-[24px]">
                <summary class="w-fit cursor-pointer font-medium text-orange-ink underline underline-offset-4">How to do it</summary>
                <div class="mt-2 flex flex-col gap-3 rounded-xl bg-mist p-3 stage:mt-4 stage:gap-4 stage:rounded-[20px] stage:p-6">
                  <ol class="m-0 flex flex-col gap-1.5 pl-5 stage:gap-3 stage:pl-8">
                    <li v-for="s in it.steps" :key="s">{{ s }}</li>
                  </ol>
                  <p class="m-0 flex flex-col">
                    <span class="text-xs font-semibold tracking-[0.04em] text-orange-ink uppercase stage:text-[20px]">Done when</span>{{ it.doneWhen }}
                  </p>
                </div>
              </details>
              <span class="text-xs text-muted stage:text-[20px]">
                {{ it.auto ? 'The latest scan of tribelt.nl checks this.' : it.tickedBy ? `Ticked off by ${it.tickedBy}.` : 'An administrator ticks this off.' }}
              </span>
            </div>
            <RouterLink v-if="it.leader" :to="companyPath(it.leader.competitorId)" class="flex flex-col text-right no-underline">
              <span class="text-xs text-muted stage:text-[20px]">Look at</span>
              <span class="font-semibold stage:text-[28px]">{{ it.leader.name }}</span>
            </RouterLink>
          </li>
        </ol>
        <p v-if="saveError" role="alert" class="m-0 font-medium text-orange-ink">{{ saveError }}</p>
      </section>

      <aside aria-labelledby="watch-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-8">
        <div class="flex flex-wrap items-baseline justify-between gap-3">
          <h2 id="watch-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em] stage:text-[56px]">Numbers to watch</h2>
          <button
            v-if="isAdmin && plan && !editing"
            type="button"
            class="border-0 bg-transparent p-0 font-medium text-orange-ink underline underline-offset-4 stage:text-[24px]"
            @click="edit"
          >
            Change the targets
          </button>
        </div>
        <ul aria-label="Legend" class="m-0 flex list-none gap-5 p-0 text-sm text-ink-2 stage:text-[22px]">
          <li class="inline-flex items-center gap-2"><span aria-hidden="true" class="size-3 bg-orange stage:size-5"></span>Now</li>
          <li class="inline-flex items-center gap-2"><span aria-hidden="true" class="size-3 border-2 border-black stage:size-5"></span>Target for the year</li>
        </ul>
        <form v-if="editing && plan" aria-label="Targets" class="flex flex-col gap-3" @submit.prevent="save">
          <label v-for="w in plan.watched" :key="w.target" class="flex items-center justify-between gap-3 text-sm stage:text-[24px]">
            {{ TARGETS[w.target].label }}
            <input
              v-model="draft[w.target]"
              type="number"
              min="0"
              :max="TARGETS[w.target].max"
              :step="TARGETS[w.target].step"
              class="h-10 w-28 rounded-lg border-2 border-silver px-2 stage:h-14 stage:w-40"
            />
          </label>
          <span class="text-xs text-muted stage:text-[20px]">Shares are fractions: 0.4 is 40%.</span>
          <div class="flex gap-3">
            <button type="submit" class="min-h-11 rounded-full border-0 bg-orange px-5 font-semibold text-black hover:bg-orange-soft stage:text-[24px]">Save the targets</button>
            <button type="button" class="min-h-11 rounded-full border-2 border-silver bg-white px-5 font-medium stage:text-[24px]" @click="editing = false">Cancel</button>
          </div>
        </form>
        <div v-else-if="plan" class="flex flex-col gap-4">
          <div v-for="w in plan.watched" :key="w.target" class="flex flex-col gap-1.5">
            <span class="flex flex-wrap justify-between gap-2 text-sm stage:text-[26px]">
              <span>{{ watchedRow(w).label }}</span>
              <span><strong class="text-orange-ink">{{ watchedRow(w).now }}</strong> to <strong>{{ watchedRow(w).goal }}</strong></span>
            </span>
            <span aria-hidden="true" class="relative block h-3 bg-mist stage:h-5">
              <span class="absolute inset-y-0 left-0 bg-orange" :style="{ width: percent(watchedRow(w).nowLength) }"></span>
              <span class="absolute inset-y-0 left-0 border-r-4 border-black" :style="{ width: percent(watchedRow(w).goalLength) }"></span>
            </span>
          </div>
        </div>
        <div v-if="plan && plan.eyeOn.length > 0" class="flex flex-col gap-2 rounded-2xl bg-mist p-5 stage:mt-auto stage:gap-4 stage:rounded-[28px] stage:p-10">
          <span class="text-sm font-semibold tracking-[0.04em] text-orange-ink uppercase stage:text-[22px]">Who to keep an eye on</span>
          <p v-for="e in plan.eyeOn" :key="e.why" class="m-0 stage:text-[28px]">
            <RouterLink :to="companyPath(e.leader.competitorId)" class="font-semibold text-black">{{ e.leader.name }}</RouterLink> {{ eyeOnText(e) }}
          </p>
        </div>
      </aside>
    </div>
  </SectionView>
</template>
