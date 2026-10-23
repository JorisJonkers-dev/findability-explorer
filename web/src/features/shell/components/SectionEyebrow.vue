<template>
  <!-- An orange dash, then the label in small capitals: the poster's section marker. -->
  <span class="flex items-center gap-3 text-sm font-semibold tracking-[0.06em] text-orange-ink uppercase stage:gap-6 stage:text-[30px]">
    <span aria-hidden="true" class="h-1.5 w-9 flex-none bg-orange stage:h-3 stage:w-18"></span>
    <slot />
  </span>
</template>
