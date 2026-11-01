<script setup lang="ts">
import { clock } from '../../../../shared/dates'
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import ProgressBar from '../../../../shared/ui/ProgressBar.vue'
import { costLabel } from '../../../prompt-test/domain/results'
import type { RunningWork } from '../../domain/activity'
import ManageWidget from './ManageWidget.vue'
import StopButton from './StopButton.vue'

/** One piece of work that runs: its meter, its cost so far and to go, and its own Stop. */
const props = withDefaults(defineProps<{ work: RunningWork; span?: 3 | 4 | 6 | 8 | 12; stopping?: boolean }>(), { span: 12, stopping: false })
defineEmits<{ stop: [] }>()

const NUMBER = new Intl.NumberFormat('en-GB')
const fraction = computed(() => (props.work.total > 0 ? Math.min(props.work.done / props.work.total, 1) : 0))
const counted = computed(() =>
  props.work.total > 0 ? `${NUMBER.format(props.work.done)} of ${NUMBER.format(props.work.total)} ${props.work.unit}` : `${NUMBER.format(props.work.done)} ${props.work.unit}`,
)
const share = computed(() => `${String(Math.floor(fraction.value * 100))}%`)
</script>

<template>
  <ManageWidget :title="work.label" :span="span">
    <template #action>
      <span class="flex flex-wrap items-center gap-3 text-sm">
        <span class="rounded-full bg-orange px-2.5 py-0.5 font-semibold text-black">Running</span>
        <RouterLink :to="work.to" class="font-medium">Open<span class="sr-only">{{ ' ' + work.label }}</span></RouterLink>
      </span>
    </template>
    <div class="flex items-baseline justify-between gap-3">
      <span>{{ counted }}</span>
      <strong>{{ share }}</strong>
    </div>
    <ProgressBar :fraction="fraction" :label="work.label" :text="counted" thick />
    <p class="m-0 flex flex-wrap gap-x-5 gap-y-1 text-ink-2">
      <span><strong class="text-black">{{ costLabel(work.costSoFarUsd) }}</strong> so far</span>
      <span v-if="work.costToGoUsd !== undefined"><strong class="text-black">{{ costLabel(work.costToGoUsd) }}</strong> to go</span>
      <span>started {{ clock(work.startedAt) }}</span>
    </p>
    <template v-if="work.stop" #foot>
      <StopButton :label="work.stop.label" :keeps="work.stop.keeps" :disabled="stopping" @stop="$emit('stop')" />
    </template>
  </ManageWidget>
</template>
