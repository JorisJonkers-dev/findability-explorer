<script setup lang="ts">
import { computed } from 'vue'
import { usePlan } from '../../plan/composables/usePlan'
import { toTickOff } from '../../plan/domain/marking'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import BigFigure from './widgets/BigFigure.vue'
import IndexCard from './widgets/IndexCard.vue'
import ManageWidget from './widgets/ManageWidget.vue'
import WidgetGrid from './widgets/WidgetGrid.vue'

/** Recommendations in Manage: how many are open, and the cards for ticking off and the goals. */
const { plan } = usePlan()
const items = computed(() => plan.value?.improvements ?? [])
const open = computed(() => items.value.filter((i) => !i.done).length)
const waiting = computed(() => toTickOff(items.value).length)
</script>

<template>
  <ManageLayout title="Recommendations" purpose="Mark recommendations as done, and set the goals." :results="{ to: '/recommendations', label: 'See the recommendations' }">
    <WidgetGrid>
      <ManageWidget title="Open recommendations" :span="12">
        <BigFigure :value="String(open)" :label="`${String(items.length - open)} done`" />
      </ManageWidget>
      <IndexCard
        title="Tick off yourself"
        text="Recommendations no scan can check, so you say when they are done."
        to="/manage/recommendations/tick-off"
        link="Tick them off"
        :figures="[{ value: String(waiting), label: 'waiting' }]"
        :span="6"
      />
      <IndexCard title="Goals" text="The five goals Tribelt is measured against." to="/manage/recommendations/goals" link="Change the goals" :span="6" />
    </WidgetGrid>
  </ManageLayout>
</template>
