<script setup lang="ts">
import { computed, ref } from 'vue'
import { saveText } from '../../../shared/download'
import { type FileTab, fileSize, missingSentence, searchLines } from '../domain/files'

/**
 * One file a scan found: its name, lines, size and matches in the header with a search, Copy and
 * Download, then its text with line numbers and every match highlighted. A file the scan did not
 * find, or whose text is not kept, is a sentence instead.
 */
const props = defineProps<{ tab: FileTab; host: string }>()

const query = ref('')
const status = ref('')
const size = computed(() => fileSize(props.tab.text))
const found = computed(() => searchLines(props.tab.text, query.value))
const int = (n: number) => n.toLocaleString('en-US')

async function copy(): Promise<void> {
  try {
    await navigator.clipboard.writeText(props.tab.text)
    status.value = 'The file is copied.'
  } catch {
    status.value = 'Copying is not allowed here.'
  }
}

function download(): void {
  saveText(props.tab.fileName, props.tab.text, props.tab.key === 'sitemap' ? 'application/xml' : 'text/plain')
}

const BUTTON = 'inline-flex min-h-11 cursor-pointer items-center rounded-full border-2 px-4 font-medium'
</script>

<template>
  <p v-if="tab.state !== 'found'" class="m-0 text-ink-2">{{ missingSentence(tab, host) }}</p>
  <div v-else class="flex flex-col gap-4">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div class="flex flex-col gap-1">
        <h3 class="m-0 font-mono text-lg font-semibold">{{ tab.name }}</h3>
        <p v-if="tab.source" class="m-0 text-sm break-all text-muted">From {{ tab.source }}</p>
        <p class="m-0 flex flex-wrap gap-x-5 gap-y-1 text-sm text-ink-2">
          <span><strong>{{ int(size.lines) }}</strong> {{ size.lines === 1 ? 'line' : 'lines' }}</span>
          <span><strong>{{ int(size.bytes) }}</strong> bytes</span>
          <span v-if="tab.truncated">Cut at 100 KB</span>
          <span v-if="query.trim()" aria-live="polite"><strong>{{ int(found.hits) }}</strong> {{ found.hits === 1 ? 'match' : 'matches' }}</span>
        </p>
      </div>
      <div class="flex flex-wrap items-end gap-2">
        <label class="flex flex-col gap-1 text-sm text-ink-2">
          Search in the file
          <input
            v-model="query"
            type="search"
            placeholder="For example GPTBot or /de/"
            class="h-11 w-56 max-w-full rounded-xl border-2 border-grey bg-white px-3 text-base text-black hover:border-black"
          />
        </label>
        <button type="button" :class="[BUTTON, 'border-silver bg-white text-black hover:border-black']" @click="copy">Copy</button>
        <button type="button" :class="[BUTTON, 'border-black bg-black text-white hover:bg-orange-ink']" @click="download">Download</button>
      </div>
    </div>
    <p v-if="status" class="m-0 text-sm font-medium" aria-live="polite">{{ status }}</p>
    <!-- searchLines escapes the text before it marks the matches, so v-html sets no markup from the file. -->
    <!-- eslint-disable vue/no-v-html -->
    <pre
      tabindex="0"
      :aria-label="`The text of ${tab.name}`"
      class="m-0 max-h-[620px] overflow-auto rounded-xl bg-white py-3 font-mono text-[13px] leading-relaxed [&_mark]:rounded-sm [&_mark]:bg-orange-soft [&_mark]:text-black"
    ><span
      v-for="line in found.lines"
      :key="line.n"
      data-line
      class="grid grid-cols-[3.5rem_minmax(0,1fr)]"
      :class="line.hit ? 'bg-orange-pale' : ''"
    ><span data-n class="pr-4 text-right text-muted select-none">{{ line.n }}</span><span class="pr-4 break-words whitespace-pre-wrap" v-html="line.html || ' '"></span></span></pre>
    <!-- eslint-enable vue/no-v-html -->
  </div>
</template>
