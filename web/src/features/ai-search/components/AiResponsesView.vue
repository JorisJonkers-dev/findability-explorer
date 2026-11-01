<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import FrameSection from '../../../shared/ui/FrameSection.vue'
import HistorySection from '../../prompt-test/components/HistorySection.vue'
import { LANGUAGE_MEASURES } from '../../prompt-test/domain/history'
import PageFrame from '../../shell/components/PageFrame.vue'
import { useMe } from '../../shell/composables/useMe'
import { useAiResponsesPage } from '../composables/useAiResponsesPage'

const page = useAiResponsesPage()
const { isAdmin } = useMe()
const widest = computed(() => Math.max(0.0001, ...page.named.value.map((n) => n.share)))
const width = (share: number) => `${String(Math.min(100, (share / widest.value) * 100))}%`
const PANEL = 'rounded-3xl bg-mist p-5 sm:p-6'
const OVER_TIME_INTRO = 'How often AI responses name Tribelt in each Prompt run. The lines show each language the questions were asked in. Twente counts as Dutch.'
</script>

<template>
  <PageFrame
    section="ai-responses"
    title="AI responses"
    purpose="How often AI assistants name and cite Tribelt when buyers ask which metal conveyor belt to use and who makes it."
    :data-line="page.dataLine.value"
    :manage="{ to: '/manage/prompt-test', label: 'Manage the Prompt test' }"
    help-topic="prompt-test"
  >
    <p v-if="page.error.value" role="alert" class="m-0 text-orange-ink">{{ page.error.value }}</p>
    <p v-else-if="page.isLoading.value" class="m-0 text-muted">Loading the responses…</p>
    <p v-else-if="!page.run.value" class="m-0 text-ink-2">
      No Prompt run has finished yet. The first one starts on the schedule.
      <template v-if="isAdmin">
        An administrator can start one in <RouterLink to="/manage/prompt-test" class="underline underline-offset-4">the Prompt test</RouterLink>.
      </template>
    </p>
    <template v-else>
      <ul aria-label="Key figures" class="m-0 grid list-none gap-4 p-0 sm:grid-cols-3">
        <li v-for="(f, i) in page.figures.value" :key="f.label" class="flex flex-col gap-1.5 rounded-3xl bg-mist px-6 py-5">
          <span class="font-display text-5xl leading-none font-semibold tracking-[-0.02em]" :class="i === 0 ? 'text-orange-ink' : 'text-black'">{{ f.value }}</span>
          <span class="font-semibold">{{ f.label }}</span>
          <span v-if="f.goal" class="text-sm text-muted">{{ f.goal }}</span>
        </li>
      </ul>
      <p class="m-0">
        <RouterLink
          to="/ai-responses/all"
          class="inline-flex min-h-11 items-center rounded-full bg-black px-6 font-bold text-white no-underline hover:bg-orange-ink hover:text-white"
        >
          See every response
        </RouterLink>
      </p>

      <HistorySection id="over-time" title="Over time" :intro="OVER_TIME_INTRO" :measures="LANGUAGE_MEASURES" :goal="page.goal.value" />

      <p v-if="page.answersError.value" role="alert" class="m-0 text-orange-ink">{{ page.answersError.value }}</p>

      <div class="grid gap-10 min-[1100px]:grid-cols-2">
        <FrameSection title="Who the responses name" :intro="`The share of the ${page.read.value} responses that did not fail name each company. Tribelt is highlighted.`">
          <div :class="PANEL">
            <ol aria-label="Share of responses that name each company" class="m-0 flex list-none flex-col gap-2 p-0">
              <li
                v-for="n in page.named.value"
                :key="n.id"
                :data-tribelt="n.tribelt"
                class="grid grid-cols-[minmax(0,11rem)_minmax(0,1fr)_3rem] items-center gap-3 text-sm"
              >
                <span class="flex min-w-0 flex-wrap items-center gap-x-2" :class="n.tribelt ? 'font-bold' : ''">
                  <RouterLink v-if="n.to" :to="n.to" class="truncate">{{ n.name }}</RouterLink>
                  <span v-else class="truncate">{{ n.name }}</span>
                  <span v-if="n.tag" class="text-xs text-muted">{{ n.tag }}</span>
                </span>
                <span aria-hidden="true" class="relative h-2.5 rounded-[5px] bg-grey">
                  <span
                    class="absolute inset-y-0 left-0 rounded-[5px]"
                    :class="n.tribelt ? 'bg-orange' : n.tag ? 'bg-silver' : 'bg-blue-grey'"
                    :style="{ width: width(n.share) }"
                  ></span>
                </span>
                <span class="text-right tabular-nums" :class="n.tribelt ? 'font-bold' : ''">{{ n.value }}</span>
              </li>
            </ol>
          </div>
        </FrameSection>

        <div class="flex min-w-0 flex-col gap-10">
          <FrameSection title="By model" intro="How often each model names Tribelt from its own knowledge and with web search, and how often its web responses cite tribelt.nl.">
            <div class="overflow-x-auto" :class="PANEL">
              <table class="w-full border-collapse text-sm">
                <caption class="sr-only">How often each model names and cites Tribelt</caption>
                <thead>
                  <tr class="text-left text-muted">
                    <th scope="col" class="pb-2 font-medium">Model</th>
                    <th scope="col" class="pb-2 text-right font-medium">Own knowledge</th>
                    <th scope="col" class="pb-2 text-right font-medium">Web search</th>
                    <th scope="col" class="pb-2 text-right font-medium">Cites tribelt.nl</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="m in page.models.value" :key="m.provider" :class="m.total ? 'border-t-2 border-black font-bold' : 'border-t border-grey'">
                    <th scope="row" class="py-2.5 pr-3 text-left" :class="m.total ? 'font-bold' : 'font-normal'">{{ m.label }}</th>
                    <td class="py-2.5 text-right tabular-nums">{{ m.knowledge }}</td>
                    <td class="py-2.5 text-right tabular-nums">{{ m.web }}</td>
                    <td class="py-2.5 text-right tabular-nums">{{ m.cited }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </FrameSection>

          <FrameSection title="The questions asked" intro="The questions in their own language, with an English translation under each. Open a belt family to read its questions.">
            <p v-if="page.answersLoading.value" class="m-0 text-muted">Loading the questions…</p>
            <div v-else class="flex flex-col gap-2">
              <details v-for="g in page.questions.value" :key="g.family" class="rounded-2xl bg-mist px-5 py-3">
                <summary class="flex min-h-9 cursor-pointer items-center justify-between gap-3 font-semibold">
                  <span>{{ g.label }}</span>
                  <span class="text-sm font-normal text-muted">{{ g.questions.length === 1 ? '1 question' : `${g.questions.length} questions` }}</span>
                </summary>
                <ul class="m-0 mt-2 flex list-none flex-col gap-3 p-0">
                  <li v-for="q in g.questions" :key="q.id" class="flex flex-col gap-0.5 border-t border-grey pt-3">
                    <span :lang="q.language">{{ q.text }}</span>
                    <span class="text-sm text-muted">{{ q.where }}</span>
                    <span v-if="q.english" lang="en" class="text-sm text-ink-2">{{ q.english }}</span>
                  </li>
                </ul>
              </details>
            </div>
          </FrameSection>
        </div>
      </div>

      <FrameSection title="By belt family and scope" intro="The share of responses that name Tribelt. Click a figure to see those responses in Every response.">
        <p v-if="page.answersLoading.value" class="m-0 text-muted">Loading the responses…</p>
        <div v-else class="overflow-x-auto" :class="PANEL">
          <table class="w-full min-w-[30rem] border-collapse text-sm">
            <caption class="sr-only">The share of responses that name Tribelt, per belt family, scope, and buyer intent</caption>
            <thead>
              <tr class="text-left text-muted">
                <th scope="col" class="pb-2 font-medium">Belt family, scope, and buyer intent</th>
                <th scope="col" class="pb-2 text-right font-medium">Own knowledge</th>
                <th scope="col" class="pb-2 text-right font-medium">Web search</th>
                <th scope="col" class="pb-2 text-right font-medium">Both</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in page.families.value" :key="r.key" class="border-t border-grey">
                <th scope="row" class="py-2 pr-3 text-left font-normal">
                  <RouterLink :to="r.family.to">{{ r.family.name }}</RouterLink>, {{ r.scope }}, <span :title="r.intent.help" class="cursor-help underline decoration-dotted underline-offset-4">{{ r.intent.name }}</span>
                  <span class="sr-only">. {{ r.intent.help }}</span>
                </th>
                <td v-for="(c, i) in [r.knowledge, r.web, r.all]" :key="i" class="py-2 text-right tabular-nums">
                  <RouterLink :to="c.to" title="Show these responses">{{ c.value }}</RouterLink>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </FrameSection>
    </template>
  </PageFrame>
</template>
