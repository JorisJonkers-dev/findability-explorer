<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import FrameSection from '../../../shared/ui/FrameSection.vue'
import { languageLabel, tierLabel } from '../../../shared/vocabulary'
import { useOverview } from '../../home/composables/useOverview'
import { usePlan } from '../../plan/composables/usePlan'
import { useProducts } from '../../products/composables/useProducts'
import PageFrame from '../../shell/components/PageFrame.vue'
import { familyPath } from '../../shell/domain/sections'
import { useTermMap } from '../composables/useTermSearch'
import { termsFigures } from '../domain/overview'
import { BUYER_INTENT_LIST, familyName, familyTable, languageTable, termPath, topMissing } from '../domain/searchTerms'
import { TABLE } from '../../../shared/ui/table'

/** Search terms: the list, what tribelt.nl uses of it by language and belt family, and the terms it lacks. */
const { overview } = useOverview()
const { plan } = usePlan()
const { terms, total, error: termsError } = useTermMap()
const { products, error: productsError } = useProducts()

const goal = computed(() => plan.value?.watched.find((w) => w.target === 'termsUsed')?.goal)
const figures = computed(() => termsFigures(overview.value?.terms, goal.value))
const byLanguage = computed(() => languageTable(terms.value))
const byFamily = computed(() => familyTable(terms.value, products.value))
const missing = computed(() => topMissing(terms.value))
const dataLine = computed(() => `From the search term list (${String(total.value)} terms) and each site's newest scan.`)
const error = computed(() => termsError.value ?? productsError.value)
</script>

<template>
  <PageFrame
    section="terms"
    title="Search terms"
    purpose="The words buyers use to look for metal conveyor belts, and how many of them tribelt.nl uses. Pages with these words are easier to find."
    :data-line="dataLine"
    :manage="{ to: '/manage/terms', label: 'Manage the search term list' }"
    help-topic="terms"
  >
    <p v-if="error" role="alert" class="m-0 rounded-2xl bg-mist p-5">{{ error }}</p>

    <div aria-label="Key figures" role="group" class="grid grid-cols-1 gap-4 sm:grid-cols-3">
      <div v-for="(f, i) in figures" :key="f.text" class="flex flex-col gap-1.5 rounded-3xl bg-mist px-6 py-5">
        <span class="font-display text-5xl leading-none font-semibold tracking-[-0.02em]" :class="i === 0 ? 'text-orange-ink' : ''">{{ f.big }}</span>
        <span class="text-base">{{ f.text }}</span>
        <span v-if="f.note" class="text-sm text-muted">{{ f.note }}</span>
      </div>
    </div>

    <FrameSection
      id="adoption"
      title="Search term adoption over time"
      intro="The share of the search term list that tribelt.nl uses, at each scan, for all terms and for each language."
    >
      <p class="m-0 rounded-3xl bg-mist p-6 text-ink-2">
        The share over time follows once the app keeps which terms each scan of tribelt.nl found. Today it knows the newest scan only.
      </p>
    </FrameSection>

    <div class="grid grid-cols-1 items-start gap-10 min-[1100px]:grid-cols-2">
      <FrameSection id="by-language" title="By language">
        <div :class="TABLE.box">
          <table :class="TABLE.table">
            <thead>
              <tr>
                <th scope="col" :class="TABLE.head">Language</th>
                <th scope="col" :class="[TABLE.head, 'text-right']">In the list</th>
                <th scope="col" :class="[TABLE.head, 'text-right']">Used by tribelt.nl</th>
                <th scope="col" :class="[TABLE.head, 'text-right']">Used only by competitors</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in byLanguage.rows" :key="r.language">
                <th scope="row" :class="[TABLE.cell, 'font-normal']">
                  <RouterLink :to="{ path: '/terms/all', query: { language: r.language } }">{{ r.label }}</RouterLink>
                </th>
                <td :class="[TABLE.cell, 'text-right']">{{ r.total }}</td>
                <td :class="[TABLE.cell, 'text-right']">{{ r.used }}</td>
                <td :class="[TABLE.cell, 'text-right']">{{ r.only }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr>
                <th scope="row" :class="TABLE.foot">All three languages</th>
                <td :class="[TABLE.foot, 'text-right']">{{ byLanguage.all.total }}</td>
                <td :class="[TABLE.foot, 'text-right']">{{ byLanguage.all.used }}</td>
                <td :class="[TABLE.foot, 'text-right']">{{ byLanguage.all.only }}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </FrameSection>

      <FrameSection id="by-family" title="By belt family">
        <div :class="TABLE.box">
          <table :class="TABLE.table">
            <thead>
              <tr>
                <th scope="col" :class="TABLE.head">Belt family</th>
                <th scope="col" :class="TABLE.head">Tribelt's own names</th>
                <th scope="col" :class="[TABLE.head, 'text-right']">Terms</th>
                <th scope="col" :class="[TABLE.head, 'text-right']">Used by tribelt.nl</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in byFamily.rows" :key="r.family">
                <th scope="row" :class="[TABLE.cell, 'font-normal']">
                  <RouterLink :to="familyPath(r.family)">{{ r.label }}</RouterLink>
                </th>
                <td :class="TABLE.cell">{{ r.names }}</td>
                <td :class="[TABLE.cell, 'text-right']">{{ r.total }}</td>
                <td :class="[TABLE.cell, 'text-right']">{{ r.used }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr>
                <th scope="row" :class="TABLE.foot">All belt families</th>
                <td :class="[TABLE.foot, 'font-normal text-muted']">{{ byFamily.shared }}</td>
                <td :class="[TABLE.foot, 'text-right']">{{ byFamily.all.total }}</td>
                <td :class="[TABLE.foot, 'text-right']">{{ byFamily.all.used }}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </FrameSection>
    </div>

    <details class="rounded-3xl bg-mist px-6 py-4">
      <summary class="cursor-pointer font-semibold text-orange-ink">What buyer intent means</summary>
      <p class="mt-3 mb-2 text-ink-2">Buyer intent is how far along a buyer is when they search. Every search term falls into one of five kinds.</p>
      <ul class="m-0 flex list-none flex-col gap-1.5 p-0">
        <li v-for="k in BUYER_INTENT_LIST" :key="k.tier"><strong class="font-semibold">{{ k.label }}.</strong> {{ k.help }}</li>
      </ul>
    </details>

    <FrameSection id="missing" title="Terms competitors use and tribelt.nl does not" intro="The 15 terms that the most competitors use.">
      <template #aside><RouterLink to="/terms/all" class="font-medium">See every term</RouterLink></template>
      <p v-if="missing.length === 0" class="m-0 rounded-3xl bg-mist p-6 text-ink-2">tribelt.nl uses every term a competitor uses.</p>
      <div v-else :class="TABLE.box">
        <table :class="TABLE.table">
          <thead>
            <tr>
              <th scope="col" :class="TABLE.head">Term</th>
              <th scope="col" :class="TABLE.head">Language</th>
              <th scope="col" :class="TABLE.head">Belt family</th>
              <th scope="col" :class="TABLE.head">Buyer intent</th>
              <th scope="col" :class="[TABLE.head, 'text-right']">Competitors that use it</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="t in missing" :key="termPath(t)">
              <th scope="row" :class="[TABLE.cell, 'font-normal']"><RouterLink :to="termPath(t)">{{ t.term }}</RouterLink></th>
              <td :class="TABLE.cell">{{ languageLabel(t.language) }}</td>
              <td :class="TABLE.cell">{{ familyName(t.family) }}</td>
              <td :class="TABLE.cell">{{ tierLabel(t.tier) }}</td>
              <td :class="[TABLE.cell, 'text-right']">{{ t.usedBy.length }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </FrameSection>
  </PageFrame>
</template>
