import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h, nextTick, ref, withDirectives } from 'vue'
import { hasMoreBelow, vScrollHint } from './scrollHint'

describe('hasMoreBelow', () => {
  it('is true until the list is scrolled to its end', () => {
    expect(hasMoreBelow({ scrollTop: 0, clientHeight: 100, scrollHeight: 500 })).toBe(true)
    expect(hasMoreBelow({ scrollTop: 398, clientHeight: 100, scrollHeight: 500 })).toBe(false) // within the slack
    expect(hasMoreBelow({ scrollTop: 0, clientHeight: 100, scrollHeight: 100 })).toBe(false)
  })
})

/** jsdom lays nothing out, so the sizes are set by hand. */
function size(el: HTMLElement, clientHeight: number, scrollHeight: number): void {
  Object.defineProperty(el, 'clientHeight', { configurable: true, value: clientHeight })
  Object.defineProperty(el, 'scrollHeight', { configurable: true, value: scrollHeight })
}

describe('vScrollHint', () => {
  it('marks a list with more below, and unmarks it once scrolled to the end or on update', async () => {
    const rows = ref(3)
    const List = defineComponent(() => () => withDirectives(h('ul', Array.from({ length: rows.value }, (_, i) => h('li', String(i)))), [[vScrollHint]]))
    // Sized before mounting, through the prototype, so the first look already sees them.
    const proto = HTMLElement.prototype
    const saved = [Object.getOwnPropertyDescriptor(proto, 'clientHeight'), Object.getOwnPropertyDescriptor(proto, 'scrollHeight')]
    size(proto, 100, 500)
    const wrapper = mount(List, { attachTo: document.body })
    const ul = wrapper.element as HTMLElement
    expect(ul.hasAttribute('data-more')).toBe(true)
    // It scrolls, so the keyboard can reach it.
    expect(ul.getAttribute('tabindex')).toBe('0')

    ul.scrollTop = 400
    ul.dispatchEvent(new Event('scroll'))
    expect(ul.hasAttribute('data-more')).toBe(false)

    size(ul, 100, 900)
    rows.value = 4
    await nextTick()
    expect(ul.hasAttribute('data-more')).toBe(true)

    // Once it no longer scrolls, the tabindex it was given goes again.
    size(ul, 900, 900)
    ul.scrollTop = 0
    rows.value = 5
    await nextTick()
    expect(ul.hasAttribute('tabindex')).toBe(false)

    wrapper.unmount()
    for (const [name, d] of [['clientHeight', saved[0]], ['scrollHeight', saved[1]]] as const) {
      if (d) Object.defineProperty(proto, name, d)
      else Reflect.deleteProperty(proto, name)
    }
  })
})

describe('vScrollHint and a tabindex of its own', () => {
  it('leaves a tabindex the markup set alone', () => {
    const List = defineComponent(() => () => withDirectives(h('ol', { tabindex: '-1' }, [h('li', 'a')]), [[vScrollHint]]))
    const wrapper = mount(List, { attachTo: document.body })
    const ol = wrapper.element as HTMLElement
    size(ol, 10, 500)
    ol.dispatchEvent(new Event('scroll'))
    expect(ol.getAttribute('tabindex')).toBe('-1')
    wrapper.unmount()
  })
})
