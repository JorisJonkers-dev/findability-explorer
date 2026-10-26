import type { Directive } from 'vue'

/** How close to the end a list counts as scrolled to the bottom, in pixels. */
const SLACK = 4

/** Whether a scrolling element holds more below what it shows. */
export function hasMoreBelow(el: Pick<HTMLElement, 'scrollTop' | 'clientHeight' | 'scrollHeight'>): boolean {
  return el.scrollTop + el.clientHeight < el.scrollHeight - SLACK
}

/** Whether an element holds more than it shows, so that it scrolls. */
function overflows(el: Pick<HTMLElement, 'clientHeight' | 'scrollHeight'>): boolean {
  return el.scrollHeight > el.clientHeight + SLACK
}

/**
 * Marks el with data-more while it holds more below, and lets the keyboard reach it while it
 * scrolls: a list that scrolls must be focusable (axe: scrollable-region-focusable), so arrow
 * keys scroll it. A tabindex the markup sets is left alone.
 */
function look(el: HTMLElement): void {
  el.toggleAttribute('data-more', hasMoreBelow(el))
  const ours = el.hasAttribute('data-scroll-focus')
  if (overflows(el) && !el.hasAttribute('tabindex')) {
    el.setAttribute('tabindex', '0')
    el.setAttribute('data-scroll-focus', '')
  } else if (!overflows(el) && ours) {
    el.removeAttribute('tabindex')
    el.removeAttribute('data-scroll-focus')
  }
}

const cleanups = new WeakMap<HTMLElement, () => void>()

/**
 * Marks a scrolling list with data-more while it holds more below what it shows, so the
 * scroll-hint style fades its bottom edge: a list cut off by the window then reads as a list that
 * goes on; and makes it focusable while it scrolls, so the keyboard can scroll it too. It looks
 * again on scroll, when the list resizes, and when its rows change.
 */
export const vScrollHint: Directive<HTMLElement> = {
  mounted(el) {
    const update = (): void => {
      look(el)
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
    look(el)
  },
  unmounted(el) {
    cleanups.get(el)?.()
    cleanups.delete(el)
  },
}
