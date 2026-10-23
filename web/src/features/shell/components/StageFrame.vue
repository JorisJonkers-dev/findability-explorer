<script setup lang="ts">
import { useStage } from '../composables/useStage'

const { transform } = useStage()
</script>

<template>
  <!--
    On a large window the content is a 3840 by 2160 slide, scaled to fit and centred on white. The
    frame clips rather than hides its overflow, so moving focus can never scroll the slide away.
    On a small one the same content is a page that scrolls.
  -->
  <div class="bg-white stage:fixed stage:inset-0 stage:overflow-clip">
    <div
      class="stage-focus stage:absolute stage:top-0 stage:left-0 stage:h-[2160px] stage:w-[3840px] stage:origin-top-left"
      :style="transform === undefined ? undefined : { transform }"
    >
      <slot />
    </div>
  </div>
</template>
