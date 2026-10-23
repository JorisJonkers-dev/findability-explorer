import { writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig, type Plugin } from 'vitest/config'

const api = process.env.API_ORIGIN ?? 'http://localhost:8080'
const proxy = { '/api': api, '/healthz': api, '/readyz': api }

// web/embed.go embeds dist/, and go:embed needs a file there before the app is built, so the
// tracked dist/.gitkeep is put back after every build empties the directory.
const keepDist: Plugin = {
  name: 'keep-dist',
  apply: 'build',
  closeBundle() {
    writeFileSync(resolve(import.meta.dirname, 'dist/.gitkeep'), '')
  },
}

export default defineConfig({
  plugins: [vue(), tailwindcss(), keepDist],
  server: { proxy },
  build: {
    // Nothing is inlined as a data: URI. The Content-Security-Policy allows data: images, but not
    // data: fonts, and the smallest @fontsource subsets fall under Vite's default 4 kB limit.
    assetsInlineLimit: 0,
    rolldownOptions: {
      output: {
        codeSplitting: {
          // zod builds the generated schemas as their module loads, and probes for eval the
          // first time unless jitless is set; the server's CSP reports that probe as an error.
          // One chunk for zod, the jitless switch and the generated client keeps them in import
          // order, so the switch runs before the first schema is built.
          groups: [{ name: 'api', test: /node_modules[\\/]zod[\\/]|src[\\/]infrastructure[\\/]/ }],
        },
      },
    },
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.spec.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,vue}'],
      // Generated code is the contract's, and main.ts is the composition root E2E covers.
      exclude: ['src/infrastructure/api/**', 'src/main.ts', 'src/**/*.spec.ts', 'src/env.d.ts'],
      // 90% overall, and 100% on the domain and the composables (docs/blueprints/vue-ts-spa.md §16).
      thresholds: {
        lines: 90,
        branches: 90,
        functions: 90,
        statements: 90,
        'src/features/*/domain/**': { lines: 100, branches: 100, functions: 100, statements: 100 },
        'src/features/*/composables/**': { lines: 100, branches: 100, functions: 100, statements: 100 },
      },
    },
  },
})
