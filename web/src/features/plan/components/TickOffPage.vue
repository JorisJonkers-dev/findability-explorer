<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import type { PlanImprovement } from '../../../infrastructure/api/types.gen'
import BigFigure from '../../manage/components/widgets/BigFigure.vue'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import { ROW_BUTTON } from '../../manage/components/widgets/look'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { usePlan } from '../composables/usePlan'
import { tickedOff, toTickOff } from '../domain/marking'

/** The open recommendations no scan can check, each with Mark as done, and a Done list with Undo. */
const { plan, isLoading, error, isSaving, saveError, tick } = usePlan()
const open = computed(() => toTickOff(plan.value?.improvements ?? []))
const done = computed(() => tickedOff(plan.value?.improvements ?? []))
const notice = ref<string | null>(null)

function mark(item: PlanImprovement, isDone: boolean): void {
  tick(item.id, isDone)
  notice.value = isDone ? `“${item.title}” is marked as done. Undo opens it again.` : `“${item.title}” is open again.`
}
</script>

<template>
  <ManageLayout
    title="Tick off yourself"
    purpose="Recommendations that no scan can check."
    :back="{ to: '/manage/recommendations', label: 'Back to Recommendations' }"
    :results="{ to: '/recommendations', label: 'See the recommendations' }"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-6 text-lg">{{ error }}</p>
    <p v-else-if="isLoading" class="m-0 text-muted">Loading the recommendations…</p>
    <WidgetGrid v-else>
      <ManageWidget title="Waiting" :span="8">
        <p v-if="open.length === 0" class="m-0 text-ink-2">Nothing waits to be ticked off.</p>
        <ul v-else aria-label="Waiting" class="m-0 flex list-none flex-col p-0">
          <li v-for="item in open" :key="item.id" class="flex flex-wrap items-start justify-between gap-3 border-b border-grey py-3">
            <span class="flex min-w-0 flex-1 flex-col gap-1">
              <RouterLink :to="`/recommendations/${item.id}`" class="font-semibold">{{ item.title }}</RouterLink>
              <span class="text-sm text-ink-2">{{ item.doneWhen }}</span>
            </span>
            <button type="button" :class="ROW_BUTTON" :disabled="isSaving" @click="mark(item, true)">
              Mark as done<span class="sr-only">{{ `: ${item.title}` }}</span>
            </button>
          </li>
        </ul>
      </ManageWidget>
      <ManageWidget title="How it works" :span="4">
        <BigFigure :value="String(open.length)" label="waiting" />
        <p class="m-0 text-ink-2">
          These are open recommendations that no scan can check, such as a listing on a site that blocks reading. Mark one as done when you
          have done it. It stays in the Done list, so you can undo a wrong click.
        </p>
      </ManageWidget>
      <ManageWidget title="Done" :span="12">
        <p v-if="done.length === 0" class="m-0 text-ink-2">Nothing is marked as done yet.</p>
        <ul v-else aria-label="Done" class="m-0 flex list-none flex-col p-0">
          <li v-for="item in done" :key="item.id" class="flex flex-wrap items-center justify-between gap-3 border-b border-grey py-3">
            <RouterLink :to="`/recommendations/${item.id}`" class="min-w-0 text-muted line-through">{{ item.title }}</RouterLink>
            <button type="button" :class="ROW_BUTTON" :disabled="isSaving" @click="mark(item, false)">
              Undo<span class="sr-only">{{ `: ${item.title}` }}</span>
            </button>
          </li>
        </ul>
      </ManageWidget>
      <p class="m-0 w-full font-medium" aria-live="polite">{{ notice }}</p>
      <p v-if="saveError" role="alert" class="m-0 w-full font-medium text-orange-ink">{{ saveError }}</p>
    </WidgetGrid>
  </ManageLayout>
</template>
