<script setup lang="ts">
import ManageLayout from '../../shell/components/ManageLayout.vue'
import ActivityWidgets from './ActivityWidgets.vue'

/** Activity, the start of Manage. Its widgets ask the server only once the account is an administrator. */
</script>

<template>
  <ManageLayout title="Activity" purpose="See what is running, what comes next, and what is waiting for you." :results="{ to: '/', label: 'See the Overview' }">
    <ActivityWidgets />
  </ManageLayout>
</template>
