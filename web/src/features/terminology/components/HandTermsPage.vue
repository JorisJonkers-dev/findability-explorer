<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import type { BeltFamily, HandTerm, IntentTier, Language, Term } from '../../../infrastructure/api/types.gen'
import { BUYER_INTENT } from '../../../shared/tiers'
import { familyLabel, languageLabel } from '../../../shared/vocabulary'
import type { Column } from '../../manage/domain/table'
import DataTable from '../../manage/components/widgets/DataTable.vue'
import { ACTIONS, FIELD, LABEL, PRIMARY, ROW_BUTTON } from '../../manage/components/widgets/look'
import ManageWidget from '../../manage/components/widgets/ManageWidget.vue'
import WidgetGrid from '../../manage/components/widgets/WidgetGrid.vue'
import ManageLayout from '../../shell/components/ManageLayout.vue'
import { useHandTerms } from '../composables/useHandTerms'
import { useTermMap } from '../composables/useTermSearch'
import { draftError, FAMILY_CHOICES, handLine, LANGUAGE_CHOICES, TIER_CHOICES } from '../domain/handTerms'

/** Add or remove terms: the terms added by hand, a search to remove any term, the removed ones to put back, and a form to add one. */
const { terms, total } = useTermMap()
const { added, removed, error, isChanging, notice, failure, add, remove, putBack } = useHandTerms()
const id = useId()

// The form to add a term.
const text = ref('')
const language = ref<Language>('nl')
const family = ref<BeltFamily>('eyelink')
const tier = ref<IntentTier>('product')
const formError = ref('')

async function submit(): Promise<void> {
  formError.value = draftError(text.value)
  if (formError.value !== '') return
  if (await add({ term: text.value.trim(), language: language.value, family: family.value, tier: tier.value })) text.value = ''
}

// The search to remove a term: two letters at least, the first eight matches.
const find = ref('')
const hits = computed(() => {
  const q = find.value.trim().toLowerCase()
  return q.length < 2 ? [] : terms.value.filter((t) => t.term.toLowerCase().includes(q)).slice(0, 8)
})
const describe = (t: Pick<Term, 'language' | 'family' | 'tier'>) => `${languageLabel(t.language)}, ${familyLabel(t.family)}, ${BUYER_INTENT[t.tier].label}`

const COLUMNS: Column<HandTerm>[] = [
  { key: 'term', label: 'Search term', value: (h) => h.term.term },
  { key: 'language', label: 'Language', value: (h) => languageLabel(h.term.language) },
  { key: 'family', label: 'Belt family', value: (h) => familyLabel(h.term.family), wide: true },
  { key: 'tier', label: 'Buyer intent', value: (h) => BUYER_INTENT[h.term.tier].label, wide: true },
  { key: 'remove', label: '' },
]
</script>

<template>
  <ManageLayout
    title="Add or remove terms"
    help-topic="add-term"
    purpose="Add a search term by hand, or take one off the list."
    :back="{ to: '/manage/terms', label: 'Back to Search term list' }"
    :results="{ to: '/terms/all', label: 'See every term' }"
  >
    <p v-if="error" role="alert" class="m-0 text-orange-ink">{{ error }}</p>
    <p v-if="notice" aria-live="polite" class="m-0 font-medium">{{ notice }}</p>
    <p v-if="failure" role="alert" class="m-0 font-medium text-orange-ink">{{ failure }}</p>
    <WidgetGrid>
      <ManageWidget title="Added by hand" :span="8">
        <template #action>
          <span class="text-sm text-muted">{{ total }} search terms are on the list</span>
        </template>
        <DataTable :rows="added" :columns="COLUMNS" :row-key="(h) => `${h.term.term}-${h.term.language}`" label="Search terms added by hand" empty="No search term was added by hand yet.">
          <template #cell-term="{ row }">
            {{ row.term.term }}
            <span class="ml-1 rounded-full bg-orange-pale px-2 text-xs font-semibold whitespace-nowrap text-orange-ink">Added by hand</span>
          </template>
          <template #cell-tier="{ row }">
            <span :title="BUYER_INTENT[row.term.tier].help">{{ BUYER_INTENT[row.term.tier].label }}</span>
          </template>
          <template #cell-remove="{ row }">
            <button
              type="button"
              :class="ROW_BUTTON"
              :disabled="isChanging"
              :aria-label="`Remove from the list: ${row.term.term}`"
              @click="remove({ term: row.term.term, language: row.term.language })"
            >
              Remove from the list
            </button>
          </template>
        </DataTable>

        <details class="flex flex-col gap-3" :open="removed.length > 0">
          <summary class="cursor-pointer font-semibold">Remove a search term from the list</summary>
          <p class="m-0 text-sm text-muted">
            Search the list, then remove a search term that does not belong. Removed terms are no longer counted or looked for, and you can put them back.
          </p>
          <label :for="`${id}-find`" :class="LABEL" class="max-w-md">
            Search the list
            <input :id="`${id}-find`" v-model="find" name="find" type="search" placeholder="Type at least two letters" :class="FIELD" />
          </label>
          <ul v-if="hits.length > 0" aria-label="Search terms on the list that match" class="m-0 flex list-none flex-col gap-2 p-0">
            <li v-for="t in hits" :key="`${t.term}-${t.language}`" class="flex flex-wrap items-center justify-between gap-2 rounded-xl bg-white px-3 py-2">
              <span class="min-w-0">
                <strong>{{ t.term }}</strong> <span class="text-sm text-ink-2">{{ describe(t) }}</span>
              </span>
              <button type="button" :class="ROW_BUTTON" :disabled="isChanging" @click="remove({ term: t.term, language: t.language })">Remove from the list</button>
            </li>
          </ul>
          <p v-else-if="find.trim().length >= 2" class="m-0 text-sm text-muted">No search term on the list matches.</p>
          <template v-if="removed.length > 0">
            <h3 class="m-0 text-base font-semibold">Removed search terms</h3>
            <ul aria-label="Removed search terms" class="m-0 flex list-none flex-col gap-2 p-0">
              <li v-for="r in removed" :key="`${r.term.term}-${r.term.language}`" class="flex flex-wrap items-center justify-between gap-2 rounded-xl bg-white px-3 py-2">
                <span class="min-w-0">
                  <strong>{{ r.term.term }}</strong> <span class="text-sm text-ink-2">{{ languageLabel(r.term.language) }}. {{ handLine('Removed', r) }}</span>
                </span>
                <button type="button" :class="ROW_BUTTON" :disabled="isChanging" @click="putBack({ term: r.term.term, language: r.term.language })">Put back</button>
              </li>
            </ul>
          </template>
        </details>
      </ManageWidget>

      <ManageWidget title="Add a search term" :span="4">
        <p class="m-0 text-ink-2">The search term joins the list at once and is marked as added by hand. Scans look for it from the next scan on.</p>
        <form class="flex flex-col gap-3" novalidate @submit.prevent="submit">
          <label :for="`${id}-term`" :class="LABEL">
            Search term
            <input :id="`${id}-term`" v-model="text" name="term" type="text" maxlength="200" placeholder="For example roestvrijstalen transportband" :class="FIELD" />
          </label>
          <label :for="`${id}-language`" :class="LABEL">
            Language
            <select :id="`${id}-language`" v-model="language" name="language" :class="FIELD">
              <option v-for="l in LANGUAGE_CHOICES" :key="l.value" :value="l.value">{{ l.label }}</option>
            </select>
          </label>
          <label :for="`${id}-family`" :class="LABEL">
            Belt family
            <select :id="`${id}-family`" v-model="family" name="family" :class="FIELD">
              <option v-for="f in FAMILY_CHOICES" :key="f.value" :value="f.value">{{ f.label }}</option>
            </select>
          </label>
          <label :for="`${id}-tier`" :class="LABEL">
            Buyer intent
            <select :id="`${id}-tier`" v-model="tier" name="tier" :class="FIELD">
              <option v-for="t in TIER_CHOICES" :key="t.value" :value="t.value" :title="t.help">{{ t.label }}</option>
            </select>
          </label>
          <details>
            <summary class="cursor-pointer text-sm font-medium">What buyer intent means</summary>
            <ul class="m-0 mt-2 flex flex-col gap-1 pl-5 text-sm text-ink-2">
              <li v-for="t in TIER_CHOICES" :key="t.value"><strong>{{ t.label }}</strong>. {{ t.help }}</li>
            </ul>
          </details>
          <p v-if="formError" role="alert" class="m-0 text-sm font-medium text-orange-ink">{{ formError }}</p>
          <div :class="ACTIONS">
            <button type="submit" :class="PRIMARY" :disabled="isChanging">Add the search term</button>
          </div>
        </form>
      </ManageWidget>
    </WidgetGrid>
  </ManageLayout>
</template>
