<script setup lang="ts">
import tribeltLogo from '../../../assets/tribelt-logo-on-dark.svg'
import utLogo from '../../../assets/ut-logo-white.png'
import { COURSE_LINE, TEAM_LINE } from '../../home/domain/content'
import AccountBar from './AccountBar.vue'
</script>

<template>
  <footer
    class="flex flex-col gap-6 bg-black px-4 py-8 text-white sm:px-8 stage:-mx-28 stage:grid stage:grid-cols-[auto_minmax(0,1fr)_auto] stage:items-center stage:gap-24 stage:px-28 stage:py-0"
  >
    <img :src="tribeltLogo" alt="Tribelt, part of Lesjöfors Group" class="block h-14 w-auto self-start stage:h-[104px] stage:self-center" />
    <div class="flex items-start gap-4 border-ink-2 text-sm stage:items-center stage:gap-12 stage:border-l-2 stage:pl-24 stage:text-[30px]">
      <span aria-hidden="true" class="mt-2 h-1.5 w-6 flex-none bg-orange stage:mt-0 stage:h-3 stage:w-12"></span>
      <span class="flex flex-col gap-2 stage:gap-3">
        <span class="font-semibold">{{ COURSE_LINE }}</span>
        <span class="text-silver">{{ TEAM_LINE }}</span>
        <AccountBar tone="dark" />
      </span>
    </div>
    <img :src="utLogo" alt="University of Twente" class="block h-14 w-auto self-start stage:h-[116px] stage:self-center" />
  </footer>
</template>
