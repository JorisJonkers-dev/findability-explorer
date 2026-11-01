<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { PromptAnswer } from '../../../infrastructure/api/types.gen'
import { saveText } from '../../../shared/download'
import PageFrame from '../../shell/components/PageFrame.vue'
import { useMe } from '../../shell/composables/useMe'
import { useEveryResponse } from '../composables/useResponses'
import { FAMILY_OPTIONS, MODES, placeSentence, type ResponseState, responsePath, responsesCsv, responsesFile, runDataLine, runLabel, stateOf } from '../domain/responses'

const route = useRoute()
const router = useRouter()
const { isAdmin } = useMe()
const page = useEveryResponse(computed(() => route.query))
const f = page.filters

const MODE_OPTIONS = [
  { value: 'all', label: 'Both' },
  { value: 'knowledge', label: 'Own knowledge' },
  { value: 'web', label: 'Web search' },
] as const
const NAMES = [
  { value: 'all', label: 'All' },
  { value: 'yes', label: 'Yes' },
  { value: 'no', label: 'No' },
] as const

/** How each square looks. */
const SQUARES: Record<ResponseState, string> = {
  first: 'border-orange bg-orange',
  named: 'border-orange-soft bg-orange-soft',
  not: 'border-silver bg-white',
  failed: 'border-silver bg-[repeating-linear-gradient(45deg,var(--color-silver)_0_3px,white_3px_6px)]',
  unread: 'border-dashed border-muted bg-white',
}
const SQUARE = 'block size-7 rounded-md border-2'
const SELECT = 'min-h-11 max-w-full rounded-lg border-2 border-solid border-silver bg-white px-3'

const labels = computed(() => new Map(page.grid.value.groups.map((g) => [g.provider, g.label])))
function squareLabel(a: PromptAnswer): string {
  return `${labels.value.get(a.provider) ?? a.provider}, ${MODES[a.mode].long}. ${placeSentence(a)}.`
}

/** Changes the address; a filter set to all leaves it. */
function setQuery(change: Record<string, string>): void {
  const query: Record<string, string> = {}
  for (const [k, v] of Object.entries({ ...route.query, ...change })) if (typeof v === 'string' && v !== 'all') query[k] = v
  void router.replace({ query })
}
const picked = (e: Event) => (e.target as HTMLSelectElement).value

function download(): void {
  if (page.run.value) saveText(responsesFile(page.run.value), responsesCsv(page.list.value))
}
</script>

<template>
  <PageFrame
    section="ai-responses"
    title="Every response"
    purpose="Each square is one model’s response to one question. Its colour shows whether the response names Tribelt, and where."
    :data-line="page.run.value ? runDataLine(page.run.value) : undefined"
    :back="{ to: '/ai-responses', label: 'Back to AI responses' }"
    :manage="{ to: '/manage/prompt-test', label: 'Manage the Prompt runs' }"
    help-topic="prompt-test"
  >
    <p v-if="page.error.value" role="alert" class="m-0 text-orange-ink">{{ page.error.value }}</p>
    <p v-else-if="page.isLoading.value" class="m-0 text-muted">Loading the runs…</p>
    <p v-else-if="!page.run.value" class="m-0 text-ink-2">
      No Prompt run has finished yet. The first one starts on the schedule.
      <template v-if="isAdmin">
        An administrator can start one in <RouterLink to="/manage/prompt-test" class="underline underline-offset-4">the Prompt test</RouterLink>.
      </template>
    </p>
    <template v-else>
      <div class="flex flex-wrap items-end gap-x-4 gap-y-3">
        <label class="flex max-w-full min-w-0 flex-col gap-1 text-sm font-medium">
          Prompt run
          <select :value="page.run.value.id" :class="SELECT" @change="setQuery({ run: picked($event) })">
            <option v-for="r in page.runs.value" :key="r.id" :value="r.id">{{ runLabel(r) }}</option>
          </select>
        </label>
        <label class="flex flex-col gap-1 text-sm font-medium">
          Search mode
          <select :value="f.mode" :class="SELECT" @change="setQuery({ mode: picked($event) })">
            <option v-for="o in MODE_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>
        </label>
        <label class="flex flex-col gap-1 text-sm font-medium">
          Scope
          <select :value="f.scope" :class="SELECT" @change="setQuery({ scope: picked($event) })">
            <option v-for="o in page.scopes.value" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>
        </label>
        <label class="flex flex-col gap-1 text-sm font-medium">
          Belt family
          <select :value="f.family" :class="SELECT" @change="setQuery({ family: picked($event) })">
            <option v-for="o in FAMILY_OPTIONS" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>
        </label>
        <label class="flex max-w-full min-w-0 flex-col gap-1 text-sm font-medium">
          Model
          <select :value="f.model" :class="SELECT" @change="setQuery({ model: picked($event) })">
            <option v-for="o in page.models.value" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>
        </label>
        <div class="flex flex-col gap-1 text-sm font-medium">
          <span id="names-tribelt">Names Tribelt</span>
          <div role="group" aria-labelledby="names-tribelt" class="flex gap-1.5">
            <button
              v-for="o in NAMES"
              :key="o.value"
              type="button"
              :aria-pressed="f.names === o.value"
              class="inline-flex min-h-11 items-center rounded-full border-2 border-solid px-4 font-medium"
              :class="f.names === o.value ? 'border-black bg-black text-white' : 'border-silver bg-white text-ink-2 hover:border-black'"
              @click="setQuery({ names: o.value })"
            >
              {{ o.label }}
            </button>
          </div>
        </div>
        <button
          type="button"
          class="inline-flex min-h-11 items-center rounded-full border-2 border-solid border-silver bg-white px-4 font-medium hover:border-black disabled:opacity-60"
          :disabled="page.list.value.length === 0"
          @click="download"
        >
          Download as CSV
        </button>
      </div>

      <p v-if="page.answersError.value" role="alert" class="m-0 text-orange-ink">{{ page.answersError.value }}</p>
      <p v-else-if="page.answersLoading.value" class="m-0 text-muted">Loading the responses…</p>
      <p v-else-if="page.list.value.length === 0" class="m-0 rounded-3xl bg-mist p-6 text-ink-2">No response matches these filters. Widen a filter to see more.</p>
      <section v-else aria-label="Every response" class="flex min-w-0 flex-col gap-4 rounded-3xl bg-mist p-5 sm:p-6">
        <div class="flex flex-col gap-2 border-b border-grey pb-4">
          <ul aria-label="What the squares mean" class="m-0 flex list-none flex-wrap gap-x-6 gap-y-2 p-0 text-sm">
            <li v-for="k in page.key.value" :key="k.state" class="flex items-center gap-2">
              <span aria-hidden="true" class="size-[18px] flex-none rounded border-2" :class="SQUARES[k.state]"></span>{{ k.label }}
            </li>
          </ul>
          <p data-part="shown" class="m-0 text-sm text-muted">
            {{ page.list.value.length === 1 ? '1 response' : `${page.list.value.length} responses` }} shown. A square opens the response. Point at a row label to see
            what its buyer intent means.
          </p>
        </div>
        <div class="overflow-x-auto">
          <table class="border-separate border-spacing-x-2 border-spacing-y-1 text-sm">
            <caption class="sr-only">Every response, by question, model, and search mode</caption>
            <thead>
              <tr>
                <th scope="col" rowspan="2" class="pr-3 text-left align-bottom font-medium text-muted">Question</th>
                <th v-for="g in page.grid.value.groups" :key="g.provider" scope="colgroup" :colspan="g.modes.length" class="max-w-40 px-1 text-left align-bottom font-semibold">
                  {{ g.label }}
                </th>
              </tr>
              <tr>
                <th v-for="c in page.grid.value.columns" :key="`${c.provider}-${c.mode}`" scope="col" class="w-16 px-1 text-left align-bottom text-xs font-normal text-muted">
                  {{ MODES[c.mode].short }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in page.grid.value.rows" :key="r.promptId">
                <th scope="row" class="pr-3 text-left font-normal whitespace-nowrap">
                  {{ r.place }}, <span :title="r.intent.help" class="cursor-help underline decoration-dotted underline-offset-4">{{ r.intent.name }}</span>
                  <span class="sr-only">. {{ r.intent.help }}</span>
                </th>
                <td v-for="(a, i) in r.cells" :key="i" class="px-1">
                  <RouterLink
                    v-if="a"
                    :to="responsePath(a, page.run.value.id)"
                    :title="squareLabel(a)"
                    :aria-label="squareLabel(a)"
                    :class="[SQUARE, SQUARES[stateOf(a)]]"
                    class="hover:outline-2 hover:outline-offset-1 hover:outline-black"
                  ></RouterLink>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </PageFrame>
</template>
