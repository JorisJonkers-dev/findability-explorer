<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import PillButton from '../../../shared/ui/PillButton.vue'
import PageFrame from '../../shell/components/PageFrame.vue'
import SectionEyebrow from '../../shell/components/SectionEyebrow.vue'
import { useMe } from '../../shell/composables/useMe'
import { useHelpEstimates } from '../composables/useHelpEstimates'
import { GROUPS, groupCounts, type GroupId, highlight, openAt, searchTopics, type Topic, topicBody, topicsByGroup, TOPICS } from '../domain/topics'
import HelpText from './HelpText.vue'

/**
 * Help: short topics in nine groups, one group shown at a time, with a search over all of them.
 * `/help/:topic` opens the topic's group with the topic in view. Manage links are for administrators.
 */
const route = useRoute()
const { isAdmin } = useMe()
const { estimates } = useHelpEstimates(isAdmin)

const q = ref('')
const group = ref<GroupId | 'all'>('start')
const hit = ref<string | undefined>(undefined)

const param = computed(() => (typeof route.params.topic === 'string' ? route.params.topic : undefined))
watch(
  param,
  async (p) => {
    const at = openAt(p)
    group.value = at.group
    hit.value = at.topic
    if (!at.topic) return
    await nextTick()
    document.getElementById(`help-${at.topic}`)?.scrollIntoView({ block: 'start' })
  },
  { immediate: true },
)

// Typing searches every group; a group button then narrows the matches.
watch(q, (now, before) => {
  if (now.trim() !== '' && before.trim() === '') group.value = 'all'
})

const counts = computed(() => groupCounts(q.value))
const buttons = computed(() => [{ id: 'all' as const, label: 'Everything' }, ...GROUPS])
const shown = computed(() => topicsByGroup(searchTopics(q.value, group.value)))
const dataLine = `${String(TOPICS.length)} topics in ${String(GROUPS.length)} groups. Type to search them.`

const links = (t: Topic) => t.links.filter((l) => isAdmin.value || l.adminOnly !== true)
const hidden = (t: Topic) => !isAdmin.value && t.links.some((l) => l.adminOnly === true)
const body = (t: Topic) => topicBody(t, estimates.value)
const parts = (text: string) => highlight(text, q.value)
</script>

<template>
  <PageFrame
    section="help"
    title="Help"
    purpose="Short explanations of how the Findability Explorer works, each with a link to the place where you do it."
    :data-line="dataLine"
    :manage="{ to: '/manage', label: 'Manage' }"
  >
    <div class="flex flex-col gap-4 rounded-3xl bg-mist p-5 sm:px-7 sm:py-6">
      <label for="help-q" class="sr-only">Search Help</label>
      <input
        id="help-q"
        v-model="q"
        type="search"
        placeholder="Search Help, for example “stop a run”, “buyer intent”, or “budget”"
        class="h-14 w-full min-w-0 rounded-full border-2 border-solid border-white bg-white px-5 text-lg focus:border-black"
      />
      <div role="group" aria-label="Topics" class="flex flex-wrap gap-2">
        <PillButton v-for="b in buttons" :key="b.id" size="sm" :pressed="group === b.id" @click="group = b.id">
          {{ b.label }}
          <span class="ml-1.5 text-[13px]" :class="group === b.id ? 'text-silver' : 'text-muted'">{{ counts[b.id] }}</span>
        </PillButton>
      </div>
    </div>

    <div class="grid items-start gap-6 lg:grid-cols-[260px_minmax(0,1fr)]">
      <nav aria-label="Help topics" class="sticky top-6 hidden max-h-[calc(100vh-3rem)] flex-col gap-0.5 overflow-auto rounded-3xl bg-mist p-3.5 lg:flex">
        <template v-for="g in shown" :key="g.group.id">
          <span v-if="group === 'all'" class="px-2 pt-2 pb-0.5 text-xs font-semibold tracking-[0.07em] text-muted uppercase">{{ g.group.label }}</span>
          <RouterLink
            v-if="g.group.id === 'words'"
            :to="`/help/${g.topics[0]?.id ?? 'words'}`"
            class="rounded-full px-2.5 py-1 text-sm text-ink-2 no-underline hover:bg-grey hover:text-black"
          >
            {{ g.topics.length }} words explained
          </RouterLink>
          <template v-else>
            <RouterLink
              v-for="t in g.topics"
              :key="t.id"
              :to="`/help/${t.id}`"
              class="rounded-full px-2.5 py-1 text-sm text-ink-2 no-underline hover:bg-grey hover:text-black"
            >
              {{ t.title }}
            </RouterLink>
          </template>
        </template>
      </nav>

      <div class="flex min-w-0 flex-col gap-6">
        <p v-if="shown.length === 0" data-part="empty" class="m-0 rounded-3xl bg-mist p-6 text-ink-2">
          Nothing in Help matches “{{ q.trim() }}”. Try a shorter word, or pick Everything.
        </p>
        <section v-for="g in shown" :key="g.group.id" :aria-labelledby="`help-group-${g.group.id}`" class="flex flex-col gap-3">
          <h2 :id="`help-group-${g.group.id}`" class="m-0">
            <SectionEyebrow>{{ g.group.label }}</SectionEyebrow>
          </h2>

          <dl v-if="g.group.id === 'words'" class="m-0 grid gap-x-8 rounded-3xl bg-mist px-6 py-2.5 sm:grid-cols-2">
            <div
              v-for="t in g.topics"
              :id="`help-${t.id}`"
              :key="t.id"
              :data-hit="hit === t.id"
              class="min-w-0 scroll-mt-36 border-b border-solid border-grey py-2.5"
              :class="hit === t.id ? 'rounded-md bg-orange-pale px-2.5 ring-2 ring-orange' : ''"
            >
              <dt class="font-semibold"><HelpText :parts="parts(t.title)" /></dt>
              <dd class="m-0 text-ink-2"><HelpText :parts="parts(body(t).join(' '))" /></dd>
            </div>
          </dl>

          <div v-else data-part="topics" class="grid items-start gap-3 min-[1100px]:grid-cols-2">
            <article
              v-for="t in g.topics"
              :id="`help-${t.id}`"
              :key="t.id"
              :data-hit="hit === t.id"
              class="flex h-full min-w-0 scroll-mt-36 flex-col gap-2 rounded-3xl p-6"
              :class="hit === t.id ? 'bg-orange-pale ring-2 ring-orange' : 'bg-mist'"
            >
              <h3 class="m-0 font-display text-lg font-semibold">
                <template v-for="(p, i) in parts(t.title)" :key="i">
                  <mark v-if="p.hit" class="rounded-sm bg-orange-pale px-0.5 text-inherit">{{ p.text }}</mark>
                  <template v-else>{{ p.text }}</template>
                </template>
              </h3>
              <div class="flex max-w-[82ch] flex-col gap-2 text-ink-2">
                <p v-for="(s, i) in body(t)" :key="i" class="m-0"><HelpText :parts="parts(s)" /></p>
              </div>
              <div v-if="links(t).length > 0 || hidden(t)" class="mt-2 flex flex-wrap items-center gap-2">
                <RouterLink
                  v-for="l in links(t)"
                  :key="l.label"
                  :to="l.to"
                  class="inline-flex min-h-11 items-center gap-1.5 rounded-full border-2 border-solid border-silver bg-white px-4 text-sm font-medium text-black no-underline hover:border-black hover:text-black"
                >
                  {{ l.label }} <span aria-hidden="true" class="text-muted">→</span>
                </RouterLink>
                <span v-if="hidden(t)" class="text-sm text-muted">An administrator does this in Manage.</span>
              </div>
            </article>
          </div>
        </section>
      </div>
    </div>
  </PageFrame>
</template>
