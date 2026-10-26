<script setup lang="ts">
import { computed, onMounted, useTemplateRef } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import worker from '../../../assets/worker.webp'
import tribeltLogo from '../../../assets/tribelt-logo-on-dark.svg'
import utLogo from '../../../assets/ut-logo-white.png'
import { CRAWLER_PAGES } from '../../ai-search/domain/content'
import { useCompetitors } from '../../competitors/composables/useCompetitors'
import AccountBar from '../../shell/components/AccountBar.vue'
import StageFrame from '../../shell/components/StageFrame.vue'
import { usePreviousPath } from '../../shell/composables/navigation'
import { SECTIONS, sectionOfPath } from '../../shell/domain/sections'
import { useTermMap } from '../../terminology/composables/useTermSearch'
import { useZoom } from '../composables/useZoom'
import { COURSE_LINE, EYEBROW, LEAD, TEAM_LINE, TILE_TEXT } from '../domain/content'
import { crawlerBars, familyBars, languageBars, makerCount, originOf, termsText } from '../domain/tiles'
import TileBars from './TileBars.vue'
import TilePairs from './TilePairs.vue'
import TilePlan from './TilePlan.vue'

const router = useRouter()
const root = useTemplateRef<HTMLElement>('root')
const tiles = useTemplateRef<HTMLElement[]>('tiles')
const { competitors } = useCompetitors()
const map = useTermMap()

const { zoomClass, origin, zoomIn, zoomOut, finished } = useZoom((index) => {
  void router.push(SECTIONS[index]?.path ?? '/')
})

/** Each tile's big figure and line; the competitors and terms tiles count the live data. */
const figures = computed(() => {
  const makers = makerCount(competitors.value)
  const terms = map.terms.value
  return [
    TILE_TEXT.buyers,
    TILE_TEXT.aiSearch,
    { big: makers > 0 ? String(makers) : '…', text: TILE_TEXT.competitors },
    { big: terms.length > 0 ? String(map.total.value) : '…', text: terms.length > 0 ? termsText(terms) : 'search terms in the terminology map.' },
    TILE_TEXT.plan,
  ]
})

/** Where a tile's centre is in the poster, measured in the poster's own (unscaled) pixels. */
function centreOf(index: number): string {
  const tile = tiles.value?.[index]
  return originOf({ left: tile?.offsetLeft ?? 0, top: tile?.offsetTop ?? 0, width: tile?.offsetWidth ?? 0, height: tile?.offsetHeight ?? 0 })
}

/** A plain click zooms into the tile, then opens its section; a click to open a new tab is left alone. */
function open(index: number, event: MouseEvent): void {
  if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
  event.preventDefault()
  zoomIn(index, centreOf(index))
}

// Back from a section, the poster shrinks back out of that section's tile.
const previous = usePreviousPath().value
onMounted(() => {
  const from = previous === null ? undefined : sectionOfPath(previous)
  if (from) {
    const index = SECTIONS.indexOf(from)
    zoomOut(index, centreOf(index))
  }
})
</script>

<template>
  <StageFrame>
    <div
      ref="root"
      class="relative flex flex-col gap-10 overflow-x-clip pt-5 stage:grid stage:h-full stage:grid-rows-[620px_minmax(0,1fr)_240px] stage:gap-16 stage:overflow-visible stage:px-28 stage:pt-28 stage:text-[34px] stage:leading-[1.4]"
      :class="zoomClass"
      :style="origin === undefined ? undefined : { transformOrigin: origin }"
      @animationend.self="finished"
    >
      <header class="flex flex-col gap-6 px-4 sm:px-8 stage:grid stage:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)] stage:items-stretch stage:gap-24 stage:px-0">
        <div class="flex flex-col gap-5 stage:justify-between stage:gap-0 stage:py-2">
          <span class="flex items-center gap-3 text-sm font-medium tracking-[0.08em] text-ink-2 uppercase stage:gap-6 stage:text-[30px]">
            <span aria-hidden="true" class="h-1.5 w-9 flex-none bg-orange stage:h-3 stage:w-18"></span>{{ EYEBROW }}
          </span>
          <h1 class="m-0 font-display text-6xl leading-[0.95] font-semibold tracking-[-0.03em] sm:text-7xl stage:text-[196px]">One belt,<br />five names.</h1>
          <p class="m-0 max-w-[60ch] text-lg leading-snug font-light text-ink-2 stage:max-w-[1800px] stage:text-[48px] stage:leading-[1.3]">{{ LEAD }}</p>
        </div>
        <img
          :src="worker"
          alt="A Tribelt employee at a belt production line"
          class="block h-64 w-full rounded-2xl object-cover object-[60%_30%] stage:h-[620px] stage:rounded-[28px]"
        />
      </header>

      <nav aria-label="Sections" class="grid grid-cols-1 gap-4 px-4 sm:grid-cols-2 sm:px-8 stage:min-h-0 stage:grid-cols-5 stage:gap-12 stage:px-0">
        <RouterLink v-for="(section, index) in SECTIONS" :key="section.id" v-slot="{ href }" :to="section.path" custom>
          <a
            ref="tiles"
            :href="href"
            :aria-label="`${section.label}: ${figures[index]?.big} ${figures[index]?.text} Zoom in.`"
            class="flex cursor-zoom-in flex-col gap-4 overflow-hidden rounded-2xl bg-mist p-5 text-black no-underline transition-colors duration-200 hover:bg-grey hover:text-black stage:min-h-0 stage:gap-6 stage:rounded-[28px] stage:px-[52px] stage:pt-[52px] stage:pb-11"
            @click="open(index, $event)"
          >
            <span class="flex items-baseline gap-3 text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:gap-[18px] stage:text-[28px]">
              <span>{{ section.number }}</span><span class="text-ink-2">{{ section.label }}</span>
            </span>
            <span class="font-display text-6xl leading-[0.9] font-semibold tracking-[-0.03em] stage:text-[168px]">{{ figures[index]?.big }}</span>
            <span class="leading-snug text-ink-2 stage:min-h-[132px] stage:text-[34px] stage:leading-[1.3]">{{ figures[index]?.text }}</span>
            <span aria-hidden="true" class="flex flex-col justify-end gap-3 stage:min-h-0 stage:flex-1 stage:gap-[18px]">
              <TilePairs v-if="section.id === 'buyers'" />
              <TileBars
                v-else-if="section.id === 'ai-search'"
                :bars="crawlerBars(CRAWLER_PAGES)"
                fill="blue-grey"
                strong="Googlebot"
                mark-zeros
                :legend="[{ label: 'Pages fetched, of 248', fill: 'blue-grey' }]"
              />
              <TileBars
                v-else-if="section.id === 'competitors'"
                :bars="familyBars(competitors)"
                fill="orange"
                :legend="[{ label: `Competitors offering each Tribelt family, of ${figures[2]?.big}`, fill: 'orange' }]"
              />
              <TileBars
                v-else-if="section.id === 'terms'"
                :bars="languageBars(map.terms.value)"
                fill="orange"
                :legend="[
                  { label: 'Tribelt uses', fill: 'orange' },
                  { label: 'Only others use', fill: 'blue-grey' },
                ]"
              />
              <TilePlan v-else />
            </span>
            <span aria-hidden="true" class="inline-flex items-center gap-2 text-sm font-medium text-muted stage:gap-3.5 stage:text-[24px]">
              <svg viewBox="0 0 24 24" class="size-5 stage:size-[30px]" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="10.5" cy="10.5" r="6.5" />
                <path d="M15.5 15.5 21 21M10.5 7.5v6M7.5 10.5h6" stroke-linecap="round" />
              </svg>
              Zoom in
            </span>
          </a>
        </RouterLink>
      </nav>

      <footer
        class="flex flex-col gap-6 bg-black px-4 py-8 text-white sm:px-8 stage:-mx-28 stage:grid stage:grid-cols-[auto_minmax(0,1fr)_auto] stage:items-center stage:gap-24 stage:px-28 stage:py-0"
      >
        <img :src="tribeltLogo" alt="Tribelt, part of Lesjöfors Group" class="block h-14 w-auto self-start stage:h-[104px] stage:self-center" />
        <div class="flex items-start gap-4 border-ink-2 text-sm stage:items-center stage:gap-12 stage:border-l-2 stage:pl-24 stage:text-[30px]">
          <span aria-hidden="true" class="mt-2 h-1.5 w-6 flex-none bg-orange stage:mt-0 stage:h-3 stage:w-12"></span>
          <span class="flex flex-col gap-2 stage:gap-3">
            <span class="font-semibold">{{ COURSE_LINE }}</span>
            <span class="text-silver">{{ TEAM_LINE }}</span>
            <AccountBar tone="dark" />
          </span>
        </div>
        <img :src="utLogo" alt="University of Twente" class="block h-14 w-auto self-start stage:h-[116px] stage:self-center" />
      </footer>
    </div>
  </StageFrame>
</template>
