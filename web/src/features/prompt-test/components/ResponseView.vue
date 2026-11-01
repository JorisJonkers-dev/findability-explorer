<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { renderMarkdown } from '../../../shared/markdown'
import FrameSection from '../../../shared/ui/FrameSection.vue'
import PageFrame from '../../shell/components/PageFrame.vue'
import { companyPath } from '../../shell/domain/sections'
import { useOneResponse } from '../composables/useResponses'
import { checkedBy, companiesNamed, leftOutOf, longDate, namesTribelt, placeSentence, responseTitle, sourcesOf, stateOf, wrongClaimsOf } from '../domain/responses'

const props = defineProps<{ id: string }>()
const route = useRoute()
const runId = computed(() => (typeof route.query.run === 'string' ? route.query.run : undefined))
const { run, answer, isLoading, error } = useOneResponse(
  computed(() => props.id),
  runId,
)

const state = computed(() => (answer.value ? stateOf(answer.value) : undefined))
const back = computed(() => ({ to: { path: '/ai-responses/all', query: runId.value ? { run: runId.value } : {} }, label: 'Back to Every response' }))
const html = computed(() => (answer.value ? renderMarkdown(answer.value.text) : ''))
const companies = computed(() => (answer.value ? companiesNamed(answer.value) : []))
const sources = computed(() => (answer.value ? sourcesOf(answer.value) : []))
const wrong = computed(() => (answer.value ? wrongClaimsOf(answer.value) : []))
const leftOut = computed(() => (answer.value ? leftOutOf(answer.value) : []))
const isLink = (address: string) => /^https?:\/\//i.test(address)
const MANAGE = { to: '/manage/prompt-test', label: 'Manage the Prompt runs' }
const EMPTY = 'm-0 rounded-2xl bg-mist p-5 text-ink-2'
</script>

<template>
  <PageFrame
    v-if="answer && run && state"
    section="ai-responses"
    :title="responseTitle(answer)"
    :purpose="`The response to “${answer.prompt}” in the Prompt run of ${longDate(run.startedAt)}.`"
    :data-line="`${placeSentence(answer)}. Received on ${longDate(answer.at)}.`"
    :back="back"
    :manage="MANAGE"
    help-topic="prompt-test"
  >
    <div class="grid gap-10 min-[1100px]:grid-cols-2">
      <FrameSection title="The response">
        <p v-if="state === 'failed'" :class="EMPTY">This response failed, so the model gave no text. {{ answer.failure }}</p>
        <!-- eslint-disable vue/no-v-html -- renderMarkdown escapes the whole response before it adds any markup. -->
        <div
          v-else
          data-part="response"
          :lang="answer.language"
          class="min-w-0 rounded-3xl bg-mist p-5 leading-relaxed break-words sm:p-6 [&_a]:text-orange-ink [&_a]:underline [&_h4]:mt-4 [&_h4]:mb-2 [&_h4]:text-base [&_h4]:font-bold [&_h4:first-child]:mt-0 [&_li]:my-1 [&_ol]:my-2 [&_ol]:list-decimal [&_ol]:pl-6 [&_p]:my-2 [&_p:first-child]:mt-0 [&_ul]:my-2 [&_ul]:list-disc [&_ul]:pl-6"
          v-html="html"
        ></div>
        <!-- eslint-enable vue/no-v-html -->
      </FrameSection>

      <div class="flex min-w-0 flex-col gap-10">
        <FrameSection title="Companies named" :intro="namesTribelt(answer) ? `${placeSentence(answer)}.` : undefined">
          <p v-if="state === 'failed'" :class="EMPTY">This response failed, so no companies were read from it.</p>
          <p v-else-if="state === 'unread'" :class="EMPTY">No model has read this response yet, so the companies it names are not known.</p>
          <p v-else-if="companies.length === 0" :class="EMPTY">This response names no company.</p>
          <ol v-else aria-label="Companies named, in order" class="m-0 flex list-none flex-col gap-1.5 p-0">
            <li
              v-for="c in companies"
              :key="`${c.position}-${c.name}`"
              :data-tribelt="c.tribelt"
              class="flex items-center gap-3 rounded-xl px-3 py-1.5"
              :class="c.tribelt ? 'bg-orange-pale font-bold' : ''"
            >
              <span class="w-6 text-sm text-muted tabular-nums">{{ c.position }}</span>
              <RouterLink v-if="c.competitorId" :to="companyPath(c.competitorId)">{{ c.name }}</RouterLink>
              <span v-else>{{ c.name }}</span>
              <span v-if="c.tribelt" class="rounded-full bg-white px-2 text-xs font-semibold text-orange-ink">Tribelt</span>
            </li>
          </ol>
        </FrameSection>

        <FrameSection title="Sources cited">
          <p v-if="sources.length === 0 && answer.mode === 'knowledge'" :class="EMPTY">This response came from the model’s own knowledge, so it cites no sources.</p>
          <p v-else-if="sources.length === 0" :class="EMPTY">This response cites no sources.</p>
          <ul v-else aria-label="Sources cited" class="m-0 flex flex-col gap-1.5 pl-5">
            <li v-for="s in sources" :key="s.address" class="break-all">
              <a v-if="isLink(s.address)" :href="s.address" target="_blank" rel="noopener noreferrer">{{ s.address }}</a>
              <span v-else>{{ s.address }}</span>
              <span v-if="s.tribelt" class="ml-2 rounded-full bg-orange-pale px-2 text-xs font-semibold text-orange-ink">tribelt.nl</span>
            </li>
          </ul>
        </FrameSection>

        <FrameSection title="Checked against the fact sheet">
          <p v-if="state === 'failed'" :class="EMPTY">A failed response is not checked against the fact sheet.</p>
          <p v-else-if="state === 'unread'" :class="EMPTY">No model has checked this response yet.</p>
          <div v-else class="grid gap-4 sm:grid-cols-2">
            <div class="rounded-3xl bg-mist p-5">
              <h3 class="m-0 text-base font-bold">Wrong about Tribelt</h3>
              <ul v-if="wrong.length > 0" class="m-0 mt-2 flex flex-col gap-1 pl-5">
                <li v-for="w in wrong" :key="w">{{ w }}</li>
              </ul>
              <p v-else class="m-0 mt-2 text-muted">Nothing in this response contradicts the fact sheet.</p>
            </div>
            <div class="rounded-3xl bg-mist p-5">
              <h3 class="m-0 text-base font-bold">Left out</h3>
              <ul v-if="leftOut.length > 0" class="m-0 mt-2 flex flex-col gap-1 pl-5">
                <li v-for="l in leftOut" :key="l">{{ l }}</li>
              </ul>
              <p v-else-if="namesTribelt(answer)" class="m-0 mt-2 text-muted">The response left out none of the facts on the fact sheet.</p>
              <p v-else class="m-0 mt-2 text-muted">The response does not name Tribelt, so no facts count as left out.</p>
            </div>
          </div>
        </FrameSection>
      </div>
    </div>

    <FrameSection title="Checked by">
      <p class="m-0">{{ checkedBy(answer) }}</p>
    </FrameSection>
  </PageFrame>

  <PageFrame v-else-if="isLoading" section="ai-responses" title="Response" purpose="Loading the response…" :back="back" :manage="MANAGE" />

  <PageFrame v-else section="ai-responses" title="Response not found" purpose="No response has this address." :back="back" :manage="MANAGE">
    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-else :class="EMPTY">No response has this address. Go back to Every response and pick one.</p>
  </PageFrame>
</template>
