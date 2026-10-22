<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { checkScanUrl, describeScanUrlCheck } from '../../scan/domain/newScan'

const router = useRouter()
const url = ref('')
const error = ref<string | null>(null)

// The group's research found these; they are the report's numbers, not the API's.
const names = [
  { name: 'TwenteFlex', who: 'Twentebelt, NL' },
  { name: 'Omni-Grid', who: 'Ashworth, US' },
  { name: 'Cam-Grid', who: 'Cambridge, US' },
  { name: 'RA rondloopband', who: 'ESFO, NL' },
] as const

/** Checks the address here, then hands it to the scan page, which starts the scan. */
function submit(): void {
  const check = checkScanUrl(url.value)
  error.value = describeScanUrlCheck(check)
  if (check.kind === 'ok') void router.push({ path: '/scan', query: { url: check.url } })
}
</script>

<template>
  <div class="flex flex-col gap-16 md:pt-2">
    <section
      aria-labelledby="hero-title"
      class="grid grid-cols-[repeat(auto-fit,minmax(min(100%,420px),1fr))] items-center gap-10"
    >
      <div class="flex flex-col gap-5">
        <h1 id="hero-title" class="m-0 font-display text-[44px] leading-[0.98] font-semibold md:text-[60px]">
          One belt, five names. Buyers search with all of them.
        </h1>
        <p class="m-0 max-w-[54ch] text-lg text-ink-2">
          A specifier who has a TwenteFlex installed searches for TwenteFlex, not for Tri-Flex. This explorer shows which
          names competitors use, in which languages, so Tribelt can be found under every one of them.
        </p>
        <form novalidate aria-label="Quick scan" class="mt-2 flex flex-col gap-2" @submit.prevent="submit">
          <div class="flex flex-wrap gap-2.5">
            <label for="quick" class="sr-only">Competitor website</label>
            <input
              id="quick"
              v-model="url"
              type="url"
              inputmode="url"
              autocomplete="url"
              spellcheck="false"
              placeholder="https://www.competitor.com"
              :aria-invalid="error !== null"
              :aria-describedby="error !== null ? 'quick-error' : undefined"
              class="h-[52px] min-w-0 flex-[1_1_280px] rounded-lg border bg-surface px-4 text-[17px] font-medium text-ink"
              :class="error !== null ? 'border-danger' : 'border-field'"
              @input="error = null"
            />
            <button
              type="submit"
              class="h-[52px] rounded-lg bg-straw px-6 text-[17px] font-semibold text-on-straw hover:bg-straw-hover"
            >
              Scan website
            </button>
          </div>
          <p v-if="error !== null" id="quick-error" role="alert" class="m-0 text-[15px] text-danger">{{ error }}</p>
        </form>
      </div>
      <figure class="m-0 flex flex-col gap-3.5 rounded-2xl border border-line bg-panel p-6 md:p-8">
        <span class="font-display text-[56px] leading-[0.9] font-bold text-straw md:text-[72px]">Tri-Flex</span>
        <ul class="m-0 flex list-none flex-col p-0">
          <li
            v-for="entry in names"
            :key="entry.name"
            class="flex items-baseline justify-between gap-4 border-t border-line py-2.5"
          >
            <span class="font-display text-[26px] leading-[1.1] font-semibold md:text-[30px]">{{ entry.name }}</span>
            <span class="text-right text-sm text-muted">{{ entry.who }}</span>
          </li>
        </ul>
        <figcaption class="text-sm text-muted">
          Tribelt's curve-going spiral-tower belt, and what four competitors call the same belt on their own websites.
        </figcaption>
      </figure>
    </section>

    <section aria-labelledby="find-title" class="flex flex-col gap-6">
      <h2 id="find-title" class="m-0 font-display text-4xl font-semibold">What the research found</h2>
      <div
        class="grid grid-cols-[repeat(auto-fit,minmax(min(100%,340px),1fr))] gap-px overflow-hidden rounded-xl border border-line bg-line"
      >
        <article class="flex flex-col gap-4 bg-ground p-7">
          <h3 class="m-0 text-[19px] font-semibold">Paid search brings visitors, not buyers</h3>
          <dl class="m-0 flex flex-col gap-2.5 text-sm">
            <div class="grid grid-cols-[120px_1fr_64px] items-center gap-3">
              <dt class="text-ink-2">Paid search</dt>
              <dd class="m-0 h-2.5 rounded-[5px] bg-raised" aria-hidden="true">
                <div class="h-2.5 w-[2.3%] min-w-1 rounded-[5px] bg-tier-product"></div>
              </dd>
              <dd class="m-0 text-right">0.04%</dd>
            </div>
            <div class="grid grid-cols-[120px_1fr_64px] items-center gap-3">
              <dt class="text-ink-2">Organic search</dt>
              <dd class="m-0 h-2.5 rounded-[5px] bg-raised" aria-hidden="true">
                <div class="h-2.5 w-full rounded-[5px] bg-straw"></div>
              </dd>
              <dd class="m-0 text-right">1.73%</dd>
            </div>
          </dl>
          <p class="m-0 text-[15px] text-muted">Share of visitors who send a form. Paid search brings 79% of all visitors.</p>
        </article>

        <article class="flex flex-col gap-4 bg-ground p-7">
          <h3 class="m-0 text-[19px] font-semibold">Gemini never named Tribelt</h3>
          <p class="m-0 font-display text-[44px] leading-none font-semibold">
            0 of 4 <span class="text-xl font-medium text-muted">supplier and quotation searches</span>
          </p>
          <p class="m-0 text-[15px] text-muted">Twentebelt, ESFO, Ashworth, Unibelt, Wire Belt and Märtens were named instead.</p>
        </article>

        <article class="flex flex-col gap-4 bg-ground p-7">
          <h3 class="m-0 text-[19px] font-semibold">AI crawlers read everything, Google nothing</h3>
          <dl class="m-0 flex flex-col gap-2.5 text-sm">
            <div class="grid grid-cols-[120px_1fr_48px] items-center gap-3">
              <dt class="text-ink-2">ClaudeBot</dt>
              <dd class="m-0 h-2.5 rounded-[5px] bg-raised" aria-hidden="true">
                <div class="h-2.5 w-full rounded-[5px] bg-tier-application"></div>
              </dd>
              <dd class="m-0 text-right">248</dd>
            </div>
            <div class="grid grid-cols-[120px_1fr_48px] items-center gap-3">
              <dt class="text-ink-2">Googlebot</dt>
              <dd class="m-0 h-2.5 rounded-[5px] bg-raised" aria-hidden="true"></dd>
              <dd class="m-0 text-right">0</dd>
            </div>
          </dl>
          <p class="m-0 text-[15px] text-muted">
            Pages fetched from our mirror of tribelt.nl in three weeks. What Google has not indexed, Gemini cannot recommend.
          </p>
        </article>
      </div>
    </section>

    <section
      aria-labelledby="map-title"
      class="flex flex-wrap items-end justify-between gap-6 border-t border-line pt-7"
    >
      <div class="flex max-w-[60ch] flex-col gap-1.5">
        <h2 id="map-title" class="m-0 font-display text-[28px] font-semibold">The terminology map</h2>
        <p class="m-0 text-ink-2">
          146 terms in 10 languages, taken from the websites of 25 competitors and Tribelt's own group, on 22 October 2026.
        </p>
      </div>
      <div class="flex flex-wrap gap-5">
        <RouterLink to="/terminology" class="font-semibold">Search the terms</RouterLink>
        <RouterLink to="/competitors" class="font-semibold">Compare competitors</RouterLink>
      </div>
    </section>
  </div>
</template>
