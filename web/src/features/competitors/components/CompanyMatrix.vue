<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { familyLabel, PRODUCT_FAMILIES } from '../../../shared/vocabulary'
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
            <RouterLink :to="{ path: '/terminology', query: { user: row.name } }" class="text-ink no-underline hover:text-straw-hover">
              {{ row.name }}
            </RouterLink>
            <span class="block text-sm font-normal text-muted">{{ row.where }}</span>
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
