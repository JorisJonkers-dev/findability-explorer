import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent, h } from 'vue'
import { trackPreviousPath, useBackToOverview, usePreviousPath } from './navigation'

const page = { render: () => null }

describe('navigation', () => {
  it('remembers where a view came from, and goes back there', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: page },
        { path: '/ai-search', component: page },
      ],
    })
    trackPreviousPath(router)
    await router.push('/')
    expect(usePreviousPath().value).toBeNull()
    await router.push('/ai-search')
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
})
