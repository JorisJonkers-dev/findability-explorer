import type { Directive } from 'vue'

/** How close to the end a list counts as scrolled to the bottom, in pixels. */
const SLACK = 4

/** Whether a scrolling element holds more below what it shows. */
export function hasMoreBelow(el: Pick<HTMLElement, 'scrollTop' | 'clientHeight' | 'scrollHeight'>): boolean {
  return el.scrollTop + el.clientHeight < el.scrollHeight - SLACK
}

const cleanups = new WeakMap<HTMLElement, () => void>()

/**
 * Marks a scrolling list with data-more while it holds more below what it shows, so the
 * scroll-hint style fades its bottom edge: a list cut off by the window then reads as a list that
 * goes on. It looks again on scroll, when the list resizes, and when its rows change.
 */
export const vScrollHint: Directive<HTMLElement> = {
  mounted(el) {
    const update = (): void => {
      el.toggleAttribute('data-more', hasMoreBelow(el))
    }
    update()
    el.addEventListener('scroll', update, { passive: true })
    const resize = typeof ResizeObserver === 'undefined' ? undefined : new ResizeObserver(update)
    resize?.observe(el)
    const rows = new MutationObserver(update)
    rows.observe(el, { childList: true, subtree: true })
    cleanups.set(el, () => {
      el.removeEventListener('scroll', update)
      resize?.disconnect()
      rows.disconnect()
    })
  },
  updated(el) {
    el.toggleAttribute('data-more', hasMoreBelow(el))
  },
  unmounted(el) {
    cleanups.get(el)?.()
    cleanups.delete(el)
  },
}
