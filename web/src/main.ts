// First, before any module builds a zod schema.
import './infrastructure/jitless'
// Self-hosted fonts: the Content-Security-Policy allows no third-party origin. Only the weights in use.
import '@fontsource/geist/500.css'
import '@fontsource/geist/600.css'
import '@fontsource/outfit/300.css'
import '@fontsource/outfit/400.css'
import '@fontsource/outfit/500.css'
import '@fontsource/outfit/600.css'
import './app/styles.css'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { createApp } from 'vue'
import App from './app/App.vue'
import { createAppRouter } from './app/router'
import { loginUrl } from './features/shell/domain/auth'
import { configureApi } from './infrastructure/http'

configureApi({
  // Any API call the session no longer covers sends the browser to sign in, and back here after.
  onUnauthorized: () => {
    window.location.assign(loginUrl(window.location.pathname + window.location.search))
  },
})
createApp(App).use(createAppRouter()).use(VueQueryPlugin).mount('#app')
