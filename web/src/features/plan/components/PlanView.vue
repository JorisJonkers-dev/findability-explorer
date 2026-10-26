<script setup lang="ts">
import { vScrollHint } from '../../../shared/scrollHint'
import { computed, ref } from 'vue'
import bakery from '../../../assets/bakery.webp'
import { percent } from '../../../shared/percent'
import RoundedPhoto from '../../../shared/ui/RoundedPhoto.vue'
import SectionHeading from '../../../shared/ui/SectionHeading.vue'
import SectionView from '../../shell/components/SectionView.vue'
import { GOALS, MONTHS, MOVES, type Phase, type PhaseTone, PHASES, SOURCE, THROUGHOUT, TITLE } from '../domain/content'
import { gridColumn, toggleOpen } from '../domain/roadmap'

const picked = ref(0)
const phase = computed(() => PHASES[picked.value] as Phase)
const open = ref<number | null>(0)

const TONE: Record<PhaseTone, string> = {
  orange: 'bg-orange text-black',
  black: 'bg-black text-white',
  'blue-grey': 'bg-blue-grey text-black',
}
</script>

<template>
  <SectionView section="plan" :title="TITLE" :source="SOURCE">
    <div class="flex flex-col gap-14 stage:grid stage:min-h-0 stage:grid-cols-[minmax(0,1.2fr)_minmax(0,1fr)_minmax(0,0.85fr)] stage:gap-24">
      <section aria-labelledby="rm-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-8">
        <SectionHeading id="rm-title">Roadmap</SectionHeading>
        <div class="flex flex-col gap-3">
          <div aria-hidden="true" class="hidden grid-cols-12 text-xs text-muted stage:grid stage:text-[24px]">
            <span v-for="month in MONTHS" :key="month" class="border-l-2 border-grey pb-2 pl-2.5">{{ month }}</span>
          </div>
          <div role="group" aria-label="Phases" class="flex flex-col gap-2 stage:grid stage:grid-cols-12">
            <button
              v-for="(p, index) in PHASES"
              :key="p.name"
              type="button"
              :aria-pressed="picked === index"
              class="flex min-h-20 cursor-pointer flex-col justify-between gap-2 rounded-xl border-0 px-4 py-3 text-left stage:min-h-[150px] stage:rounded-[20px] stage:px-7 stage:py-[22px]"
              :class="[TONE[p.tone], picked === index ? 'outline-3 outline-offset-4 outline-black outline-solid stage:outline-[5px] stage:outline-offset-[6px]' : '']"
              :style="{ gridColumn: gridColumn(p) }"
              @click="picked = index"
            >
              <span class="font-display text-xl leading-tight font-semibold stage:text-[44px]">{{ p.name }}</span>
              <span class="text-sm stage:text-[26px]">{{ p.when }} · {{ p.mode }}</span>
            </button>
            <p class="col-span-12 m-0 rounded-xl bg-mist px-4 py-3 text-sm text-ink-2 stage:rounded-[20px] stage:px-7 stage:py-3.5 stage:text-[26px]">
              {{ THROUGHOUT }}
            </p>
          </div>
        </div>
        <div aria-live="polite" class="flex flex-col gap-5 rounded-2xl bg-mist p-5 stage:min-h-0 stage:flex-1 stage:gap-7 stage:overflow-hidden stage:rounded-[28px] stage:p-12">
          <span class="text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:text-[26px]">{{ phase.name }} · {{ phase.when }}</span>
          <ul class="m-0 flex list-disc flex-col gap-2 pl-5 stage:gap-2.5 stage:pl-10 stage:text-[32px]">
            <li v-for="action in phase.actions" :key="action">{{ action }}</li>
          </ul>
          <dl class="m-0 grid grid-cols-1 gap-4 sm:grid-cols-3 stage:mt-auto stage:gap-8 stage:text-[28px]">
            <div class="flex flex-col gap-1">
              <dt class="text-sm text-muted stage:text-[24px]">Owner</dt>
              <dd class="m-0">{{ phase.owner }}</dd>
            </div>
            <div class="flex flex-col gap-1">
              <dt class="text-sm text-muted stage:text-[24px]">Evidence and KPI</dt>
              <dd class="m-0">{{ phase.kpi }}</dd>
            </div>
            <div class="flex flex-col gap-1">
              <dt class="text-sm font-semibold text-orange-ink stage:text-[24px]">Decision</dt>
              <dd class="m-0 font-semibold">{{ phase.decision }}</dd>
            </div>
          </dl>
        </div>
      </section>

      <section aria-labelledby="mv-title" class="flex flex-col gap-4 stage:min-h-0 stage:gap-6">
        <SectionHeading id="mv-title">Five moves</SectionHeading>
        <ol v-scroll-hint class="scroll-hint m-0 flex list-none flex-col p-0 stage:min-h-0 stage:overflow-y-auto">
          <li v-for="(move, index) in MOVES" :key="move.number" class="border-t-2 border-grey">
            <button
              type="button"
              :aria-expanded="open === index"
              :aria-controls="`move-${move.number}`"
              class="grid min-h-16 w-full cursor-pointer grid-cols-[2.5rem_minmax(0,1fr)_auto] items-center gap-3 border-0 bg-transparent py-3 text-left text-black stage:min-h-[120px] stage:grid-cols-[96px_minmax(0,1fr)_auto] stage:gap-6 stage:py-4"
              @click="open = toggleOpen(open, index)"
            >
              <span
                aria-hidden="true"
                class="font-display text-4xl leading-none font-semibold stage:text-[80px]"
                :class="open === index ? 'text-orange-ink' : 'text-muted'"
              >
                {{ move.number }}
              </span>
              <span class="flex flex-col gap-1">
                <span class="text-lg leading-tight font-semibold stage:text-[40px] stage:leading-[1.15]">{{ move.title }}</span>
                <span class="text-sm text-muted stage:text-[26px]">{{ move.phase }} · goals {{ move.goals }}</span>
              </span>
              <svg
                aria-hidden="true"
                viewBox="0 0 24 24"
                class="size-6 motion-safe:transition-transform motion-safe:duration-200 stage:size-10"
                :class="{ 'rotate-180': open === index }"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="M6 9l6 6 6-6" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </button>
            <div v-show="open === index" :id="`move-${move.number}`" class="flex flex-col gap-3 pb-6 pl-13 stage:gap-4 stage:pb-8 stage:pl-30">
              <ul class="m-0 flex list-disc flex-col gap-1.5 pl-5 stage:gap-2 stage:pl-9 stage:text-[30px]">
                <li v-for="step in move.steps" :key="step">{{ step }}</li>
              </ul>
              <p class="m-0 text-ink-2 stage:text-[28px]">{{ move.why }}</p>
            </div>
          </li>
        </ol>
      </section>

      <aside aria-labelledby="gl-title" class="flex flex-col gap-5 stage:min-h-0 stage:gap-8">
        <RoundedPhoto :src="bakery" alt="Bread on a metal oven belt in an industrial bakery" class="stage:h-[480px]" />
        <SectionHeading id="gl-title">Goals for twelve months</SectionHeading>
        <ul class="m-0 flex list-none flex-col gap-4 p-0 stage:gap-[22px]">
          <li v-for="goal in GOALS" :key="goal.id" class="flex flex-col gap-2 border-t-2 border-grey pt-3 stage:gap-2.5 stage:pt-[18px]">
            <span class="flex items-baseline justify-between gap-4 stage:gap-5">
              <span class="font-semibold stage:text-[32px]"><span class="text-orange-ink">{{ goal.id }}</span> {{ goal.goal }}</span>
              <span class="font-semibold whitespace-nowrap stage:text-[28px]">{{ goal.target }}</span>
            </span>
            <span class="text-sm text-muted stage:text-[24px]">{{ goal.measure }}</span>
            <span v-if="goal.bar" aria-hidden="true" class="relative h-2.5 bg-mist stage:h-4">
              <span class="absolute top-0 left-0 h-full bg-orange" :style="{ width: percent(goal.bar.target) }"></span>
              <span class="absolute top-0 left-0 h-full bg-black" :style="{ width: percent(goal.bar.today) }"></span>
            </span>
          </li>
        </ul>
        <span class="flex flex-wrap gap-4 text-xs text-muted stage:mt-auto stage:gap-7 stage:text-[22px]">
          <span class="inline-flex items-center gap-2 stage:gap-2.5"><span aria-hidden="true" class="size-3 bg-black stage:size-[22px]"></span>Today</span>
          <span class="inline-flex items-center gap-2 stage:gap-2.5"><span aria-hidden="true" class="size-3 bg-orange stage:size-[22px]"></span>Target</span>
          <span>Baselines are set in phase 1.</span>
        </span>
      </aside>
    </div>
  </SectionView>
</template>
