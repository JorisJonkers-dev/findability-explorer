// First, before any module builds a zod schema.
import './infrastructure/jitless'
// Self-hosted fonts: the Content-Security-Policy allows no third-party origin. Only the weights in use.
import '@fontsource/barlow/400.css'
import '@fontsource/barlow/500.css'
import '@fontsource/barlow/600.css'
import '@fontsource/barlow-condensed/600.css'
import '@fontsource/barlow-condensed/700.css'
import './app/styles.css'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { createApp } from 'vue'
import App from './app/App.vue'
import { createAppRouter } from './app/router'
import { configureApi } from './infrastructure/http'

configureApi()
createApp(App).use(createAppRouter()).use(VueQueryPlugin).mount('#app')
