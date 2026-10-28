<script setup lang="ts">
import tribeltLogo from '../../../assets/tribelt-logo-on-dark.svg'
import utLogo from '../../../assets/ut-logo-white.png'
import { COURSE_LINE, TEAM_LINE } from '../../home/domain/content'
import AccountBar from './AccountBar.vue'
</script>

<template>
  <footer class="bg-black text-white">
    <div class="mx-auto flex w-full max-w-[1360px] flex-wrap items-center gap-x-10 gap-y-6 px-4 py-8 sm:px-10">
      <img :src="tribeltLogo" alt="Tribelt, part of Lesjöfors Group" class="block h-12 w-auto" />
      <div class="flex min-w-0 flex-1 basis-[28rem] items-start gap-4 border-ink-2 text-sm lg:border-l-2 lg:pl-10">
        <span aria-hidden="true" class="mt-2 h-1.5 w-6 flex-none bg-orange"></span>
        <span class="flex flex-col gap-1.5">
          <span class="font-semibold">{{ COURSE_LINE }}</span>
          <span class="text-silver">{{ TEAM_LINE }}</span>
          <AccountBar tone="dark" />
        </span>
      </div>
      <img :src="utLogo" alt="University of Twente" class="block h-12 w-auto" />
    </div>
  </footer>
</template>
