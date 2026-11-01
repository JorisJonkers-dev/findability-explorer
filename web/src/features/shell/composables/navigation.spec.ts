import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent, h } from 'vue'
import { backLink, setPageTitle, trackPreviousPath, useBackToOverview, usePreviousPage, usePreviousPath } from './navigation'

const page = { render: () => null }

describe('navigation', () => {
  it('remembers where a view came from, and goes back there', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: page },
        { path: '/ai-responses', component: page },
      ],
    })
    trackPreviousPath(router)
    await router.push('/')
    expect(usePreviousPath().value).toBeNull()
    await router.push('/ai-responses')
    expect(usePreviousPath().value).toBe('/')
    // A navigation that goes nowhere, such as to the page already shown, leaves it as it was.
    await router.push('/ai-responses')
    expect(usePreviousPath().value).toBe('/')

    let back: (() => void) | undefined
    const wrapper = mount(
      defineComponent({
        setup() {
          back = useBackToOverview()
          return () => h('div')
        },
      }),
      { global: { plugins: [router] } },
    )
    back?.()
    await new Promise((resolve) => setTimeout(resolve, 20))
    expect(router.currentRoute.value.path).toBe('/')
    wrapper.unmount()
  })

  it('keeps a trail of the pages seen, so a page goes back to the one before it, and going back takes it off', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: ['/', '/a', '/b', '/c'].map((path) => ({ path, component: page })),
    })
    trackPreviousPath(router)
    // Before the first page there is nothing to name.
    setPageTitle('Nothing')
    await router.push('/')
    expect(usePreviousPage().value).toBeNull()
    setPageTitle('Overview')
    setPageTitle('Overview')
    await router.push('/a?x=1')
    setPageTitle('Page A')
    // A new query on the same page is the same page, at its new address.
    await router.push('/a?x=2')
    await router.push('/b')
    expect(usePreviousPage().value).toEqual({ path: '/a', fullPath: '/a?x=2', title: 'Page A' })
    setPageTitle('Page B')
    await router.push('/c')
    expect(usePreviousPage().value?.title).toBe('Page B')
    // Going back to the page before takes the current one off the trail, as the browser's back does.
    await router.push('/b')
    expect(usePreviousPage().value?.title).toBe('Page A')
    await router.push('/a?x=2')
    expect(usePreviousPage().value?.title).toBe('Overview')
  })

  it('goes back to the page before when it has a title, and to the page’s own way back otherwise', () => {
    const fallback = { to: '/ai-responses/all', label: 'Back to Every response' }
    expect(backLink(fallback, { path: '/goals/x', fullPath: '/goals/x?version=1', title: 'A goal' })).toEqual({ to: '/goals/x?version=1', label: 'Back to A goal' })
    expect(backLink(fallback, { path: '/manage', fullPath: '/manage', title: '' })).toBe(fallback)
    expect(backLink(fallback, null)).toBe(fallback)
    expect(backLink(undefined, { path: '/goals/x', fullPath: '/goals/x', title: 'A goal' })).toBeUndefined()
  })
})
