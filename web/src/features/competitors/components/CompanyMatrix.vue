<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { familyLabel, PRODUCT_FAMILIES } from '../../../shared/vocabulary'
import { scanCountLabel } from '../domain/company'
import { type MatrixRow, moreLabel } from '../domain/matrix'

defineProps<{ rows: MatrixRow[]; label: string; caption: string }>()
</script>

<template>
  <div role="region" :aria-label="label" tabindex="0" class="relative overflow-x-auto rounded-xl border border-line">
    <table class="w-full min-w-[1080px] border-collapse text-[15px]">
      <caption class="sr-only">{{ caption }}</caption>
      <thead>
        <tr class="bg-surface text-left align-bottom">
          <th scope="col" class="w-[200px] px-5 py-4 text-sm font-medium text-muted">Company</th>
          <th
            v-for="family in PRODUCT_FAMILIES"
            :key="family"
            scope="col"
            class="px-5 py-4 font-display text-xl leading-[1.1] font-semibold"
          >
            {{ familyLabel(family) }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in rows" :key="row.key" class="border-t border-line align-top">
          <th scope="row" class="px-5 py-4 text-left font-semibold">
            <span class="flex items-start justify-between gap-2">
              <RouterLink :to="`/competitors/${row.id}`" class="text-ink no-underline hover:text-straw-hover">
                {{ row.name }}
              </RouterLink>
              <a
                v-if="row.websiteUrl"
                :href="row.websiteUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="-m-1 flex-none rounded p-1 text-muted hover:text-straw-hover"
              >
                <svg aria-hidden="true" viewBox="0 0 16 16" class="size-4" fill="none" stroke="currentColor" stroke-width="1.75">
                  <path d="M9 3h4v4M13 3 7 9M11 9.5V13H3V5h3.5" stroke-linecap="round" stroke-linejoin="round" />
                </svg>
                <span class="sr-only">Visit {{ row.name }}'s website (opens in a new tab)</span>
              </a>
            </span>
            <span class="block text-sm font-normal text-muted">{{ row.where }}</span>
            <span class="mt-1.5 flex flex-wrap gap-1.5 text-[13px] font-normal">
              <span v-if="row.added" class="rounded-full border border-straw/60 px-2 py-px text-straw">Added</span>
              <span v-if="row.scans > 0" class="text-muted">{{ scanCountLabel(row.scans) }}</span>
            </span>
          </th>
          <td v-for="cell in row.cells" :key="cell.family" class="px-5 py-4" :class="cell.first === null ? 'hatched' : 'text-ink'">
            <template v-if="cell.first !== null">
              {{ cell.first }}
              <span v-if="cell.more > 0" class="block text-[13px] text-muted">{{ moreLabel(cell.more) }}</span>
            </template>
            <span v-else class="sr-only">None</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
