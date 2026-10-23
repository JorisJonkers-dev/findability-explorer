<script setup lang="ts">
import { watchEffect } from 'vue'
import { useRoute } from 'vue-router'
import { useMe } from '../features/shell/composables/useMe'

const route = useRoute()
// Everything needs a signed-in account: the views wait for it, and a 401 sends the browser to sign in.
const { me, error } = useMe()

watchEffect(() => {
  const title = typeof route.meta.title === 'string' ? route.meta.title : null
  document.title = title ? `${title} · Findability Explorer` : 'Findability Explorer'
})
</script>

<template>
  <a
    href="#main"
    class="sr-only z-50 rounded-md bg-black px-3 py-2 font-semibold text-white focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
  >
    Skip to content
  </a>
  <main id="main" tabindex="-1" class="focus:outline-none">
    <RouterView v-if="me" />
    <p v-else-if="error" role="alert" class="m-0 p-8 text-lg">{{ error }}</p>
    <p v-else class="m-0 p-8 text-lg text-muted">Signing in…</p>
  </main>
</template>
