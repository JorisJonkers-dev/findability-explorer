<script setup lang="ts">
import { RouterLink } from 'vue-router'
import ManageLayout from '../../shell/components/ManageLayout.vue'

/** Recommendations in Manage, until its own pages are built: where marking and the goals are done today. */
</script>

<template>
  <ManageLayout
    title="Recommendations"
    purpose="Marking recommendations as done, and the goals."
    :results="{ to: '/recommendations', label: 'See the recommendations' }"
  >
    <section aria-labelledby="tick-off-title" class="flex max-w-[64rem] flex-col gap-3 rounded-3xl bg-mist p-6 sm:p-8">
      <h2 id="tick-off-title" class="m-0 font-display text-2xl font-semibold tracking-[-0.01em]">On the Recommendations page for now</h2>
      <p class="m-0 text-ink-2">
        An administrator marks a recommendation no scan can check as done on the Recommendations page, and changes the goals below the
        list there.
      </p>
      <RouterLink to="/recommendations" class="self-start font-medium">Open Recommendations</RouterLink>
    </section>
  </ManageLayout>
</template>
