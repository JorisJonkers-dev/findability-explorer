import { createRouter, createWebHistory, type Router, type RouterHistory } from 'vue-router'
import { trackPreviousPath } from '../features/shell/composables/navigation'

const competitors = () => import('../features/competitors/components/CompetitorsView.vue')

/** The app's routes. Each view loads on demand, so the first paint carries only the shell. */
export function createAppRouter(history: RouterHistory = createWebHistory()): Router {
  const router = createRouter({
    history,
    routes: [
      { path: '/', name: 'home', component: () => import('../features/home/components/HomePoster.vue'), meta: { title: 'Overview' } },
      { path: '/buyers', name: 'buyers', component: () => import('../features/buyers/components/BuyersView.vue'), meta: { title: 'Buyers' } },
      { path: '/ai-search', name: 'ai-search', component: () => import('../features/ai-search/components/AiSearchView.vue'), meta: { title: 'AI search' } },
      // One view for the companies alone and for a scan beside them, so a started scan keeps the form.
      { path: '/competitors', name: 'competitors', component: competitors, meta: { title: 'Competitors' } },
      { path: '/scan/:id', name: 'scan', component: competitors, props: (to) => ({ scanId: String(to.params.id) }), meta: { title: 'Scan' } },
      { path: '/terms', name: 'terms', component: () => import('../features/terminology/components/TermsView.vue'), meta: { title: 'Search terms' } },
      { path: '/plan', name: 'plan', component: () => import('../features/plan/components/PlanView.vue'), meta: { title: 'The plan' } },
      { path: '/settings', name: 'settings', component: () => import('../features/settings/components/SettingsPage.vue'), meta: { title: 'Settings' } },
      // Earlier versions' addresses, which links and bookmarks may still carry.
      { path: '/terminology', redirect: (to) => ({ path: '/terms', query: to.query }) },
      { path: '/keywords', redirect: (to) => ({ path: '/terms', query: to.query }) },
      { path: '/competitors/:id', redirect: (to) => ({ path: '/competitors', query: { company: String(to.params.id) } }) },
      { path: '/scan', redirect: (to) => ({ path: '/competitors', query: to.query }) },
      { path: '/analyses', redirect: (to) => ({ path: '/competitors', query: to.query }) },
      { path: '/analyses/:id', redirect: (to) => ({ path: `/scan/${String(to.params.id)}`, query: to.query }) },
      { path: '/:rest(.*)*', redirect: '/' },
    ],
    scrollBehavior: (to, from, saved) => {
      if (saved) return saved
      // Filters, a picked company and polling change the query, not the view; only a new view starts at the top.
      return to.matched[0]?.components?.default === from.matched[0]?.components?.default ? false : { top: 0 }
    },
  })
  trackPreviousPath(router)
  return router
}
