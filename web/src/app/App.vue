<script setup lang="ts">
import { watchEffect } from 'vue'
import { useRoute } from 'vue-router'
import AppSidebar from './AppSidebar.vue'

const route = useRoute()

watchEffect(() => {
  const title = typeof route.meta.title === 'string' ? route.meta.title : null
  document.title = title ? `${title} · Findability Explorer` : 'Findability Explorer'
})
</script>

<template>
  <a
    href="#main"
    class="sr-only z-10 rounded-md bg-straw px-3 py-2 font-semibold text-on-straw focus:not-sr-only focus:fixed focus:top-3 focus:left-3"
  >
    Skip to content
  </a>
  <div class="flex min-h-screen flex-col md:flex-row">
    <AppSidebar />
    <div class="flex min-w-0 flex-1 flex-col">
      <main id="main" tabindex="-1" class="flex-1 px-[clamp(20px,5vw,72px)] pt-8 pb-16 focus:outline-none md:pt-12 md:pb-24">
        <RouterView />
      </main>
      <footer class="border-t border-line px-[clamp(20px,5vw,72px)] py-6 text-[13px] text-muted md:hidden">
        Built for Tribelt B.V. in the Digital Strategy course, University of Twente.
      </footer>
    </div>
  </div>
</template>
