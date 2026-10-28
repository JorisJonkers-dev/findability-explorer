<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { ref } from 'vue'
import { companyPath } from '../../shell/domain/sections'
import { useCompetitorChecks, useListDecision } from '../composables/useCompetitorChecks'
import { useCompetitors } from '../composables/useCompetitors'
import { haikuArchived, isFlagged, listCheckLine, verdictLabel, verdictReason } from '../domain/checks'

/** For administrators: Claude Haiku's check of the list, the companies it is unsure of, and the ones it archived. */
const { competitors } = useCompetitors()
const { checks, unchecked, review, isStarting, error: checkError, start } = useCompetitorChecks(ref(true))
const { isSaving, error: decisionError, notice, keep, archive, restore } = useListDecision()

const flagged = computed(() => competitors.value.filter((c) => !c.archived && c.role === 'competitor' && isFlagged(c, checks.value.get(c.id))))
const archived = computed(() => haikuArchived(competitors.value))
const SMALL = 'min-h-11 cursor-pointer rounded-full border-2 px-4 text-sm font-semibold disabled:opacity-60'
</script>

<template>
  <section aria-labelledby="upkeep-title" class="flex flex-col gap-4">
    <h3 id="upkeep-title" class="m-0 text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase">Haiku’s check of the list</h3>
    <div v-if="review" class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-2xl bg-white p-4">
      <p class="m-0 font-medium">{{ listCheckLine(review, unchecked) }}</p>
      <RouterLink v-if="review.problem === 'noKey'" to="/settings" class="font-medium">Add one in Language models</RouterLink>
      <button v-else type="button" :class="SMALL" class="border-black bg-black text-white hover:bg-ink-2" :disabled="review.running || isStarting" @click="start">
        Check the list with Haiku
      </button>
    </div>
    <p v-if="checkError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ checkError }}</p>
    <p v-if="decisionError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ decisionError }}</p>
    <p v-if="notice" class="m-0 text-sm font-medium" aria-live="polite">{{ notice }}</p>
    <div v-if="flagged.length > 0" class="flex flex-col gap-2">
      <h4 class="m-0 text-lg font-semibold">Haiku is unsure of these</h4>
      <ul aria-label="Companies Haiku is unsure of" class="m-0 flex list-none flex-col gap-2 p-0">
        <li v-for="c in flagged" :key="c.id" class="flex flex-wrap items-center justify-between gap-3 border-b border-grey py-2">
          <span class="flex min-w-0 flex-col">
            <RouterLink :to="companyPath(c.id)" class="font-semibold text-black">{{ c.name }}</RouterLink>
            <span class="text-sm text-ink-2">{{ verdictLabel(checks.get(c.id)) }}. {{ verdictReason(checks.get(c.id)) }}</span>
          </span>
          <span role="group" :aria-label="`Decide on ${c.name}`" class="flex gap-2">
            <button type="button" :class="SMALL" class="border-black bg-white hover:bg-black hover:text-white" :disabled="isSaving" @click="keep(c)">Keep<span class="sr-only">{{ ' ' + c.name }}</span></button>
            <button type="button" :class="SMALL" class="border-silver bg-white hover:border-black" :disabled="isSaving" @click="archive(c)">Archive<span class="sr-only">{{ ' ' + c.name }}</span></button>
          </span>
        </li>
      </ul>
    </div>
    <div v-if="archived.length > 0" class="flex flex-col gap-2">
      <h4 class="m-0 text-lg font-semibold">Archived by Haiku</h4>
      <ul aria-label="Archived by Haiku" class="m-0 flex list-none flex-col gap-2 p-0">
        <li v-for="c in archived" :key="c.id" class="flex flex-wrap items-center justify-between gap-3 border-b border-grey py-2">
          <span class="flex min-w-0 flex-col">
            <RouterLink :to="companyPath(c.id)" class="font-semibold text-black">{{ c.name }}</RouterLink>
            <span class="text-sm text-ink-2">{{ verdictReason(checks.get(c.id)) }}</span>
          </span>
          <button type="button" :class="SMALL" class="border-black bg-white hover:bg-black hover:text-white" :disabled="isSaving" @click="restore(c)">Restore<span class="sr-only">{{ ' ' + c.name }}</span></button>
        </li>
      </ul>
    </div>
  </section>
</template>
