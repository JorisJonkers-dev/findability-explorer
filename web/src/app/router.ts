import { createRouter, createWebHistory, type Router, type RouterHistory } from 'vue-router'
import { trackPreviousPath } from '../features/shell/composables/navigation'

const competitors = () => import('../features/competitors/components/CompetitorsView.vue')

/** The app's routes. Each view loads on demand, so the first paint carries only the shell. */
export function createAppRouter(history: RouterHistory = createWebHistory()): Router {
  const router = createRouter({
    history,
    routes: [
      { path: '/', name: 'home', component: () => import('../features/home/components/HomePoster.vue'), meta: { title: 'Overview' } },
      {
        path: '/catching-up',
        name: 'catching-up',
        component: () => import('../features/catching-up/components/CatchingUpView.vue'),
        meta: { title: 'Catching up' },
      },
      { path: '/buyers', name: 'buyers', component: () => import('../features/buyers/components/BuyersView.vue'), meta: { title: 'Buyers' } },
      { path: '/ai-search', name: 'ai-search', component: () => import('../features/ai-search/components/AiSearchView.vue'), meta: { title: 'AI search' } },
      {
        path: '/ai-search/readiness',
        name: 'ai-readiness',
        component: () => import('../features/readiness/components/ReadinessView.vue'),
        meta: { title: 'AI readiness' },
      },
      {
        path: '/ai-search/prompt-test',
        name: 'prompt-test',
        component: () => import('../features/prompt-test/components/PromptTestView.vue'),
        meta: { title: 'Prompt test' },
      },
      {
        path: '/settings/prompt-test',
        name: 'prompt-test-settings',
        component: () => import('../features/prompt-test/components/PromptSettingsPage.vue'),
        meta: { title: 'Prompt test settings' },
      },
      // One view for the companies alone and for a scan beside them, so a started scan keeps the form.
      { path: '/competitors', name: 'competitors', component: competitors, meta: { title: 'Competitors' } },
      {
        path: '/competitors/all',
        name: 'all-competitors',
        component: () => import('../features/competitors/components/AllCompetitorsView.vue'),
        meta: { title: 'All competitors' },
      },
      { path: '/scan/:id', name: 'scan', component: competitors, props: (to) => ({ scanId: String(to.params.id) }), meta: { title: 'Scan' } },
      {
        path: '/competitors/:id',
        name: 'company',
        component: () => import('../features/competitors/components/CompanyScanView.vue'),
        props: (to) => ({ companyId: String(to.params.id) }),
        meta: { title: 'Latest scan' },
      },
      // A company's search terms and scan history, a section view of its own: the stage has no room beside the list.
      {
        path: '/competitors/:id/history',
        name: 'company-history',
        component: () => import('../features/competitors/components/CompanyHistoryView.vue'),
        props: (to) => ({ companyId: String(to.params.id) }),
        meta: { title: 'Over time' },
      },
      { path: '/products', name: 'products', component: () => import('../features/products/components/ProductsView.vue'), meta: { title: 'Products' } },
      {
        path: '/products/:family',
        name: 'product',
        component: () => import('../features/products/components/ProductView.vue'),
        props: (to) => ({ slug: String(to.params.family) }),
        meta: { title: 'Products' },
      },
      { path: '/terms', name: 'terms', component: () => import('../features/terminology/components/TermsView.vue'), meta: { title: 'Search terms' } },
      { path: '/plan', name: 'plan', component: () => import('../features/plan/components/PlanView.vue'), meta: { title: 'The plan' } },
      { path: '/settings', name: 'settings', component: () => import('../features/settings/components/SettingsPage.vue'), meta: { title: 'Settings' } },
      { path: '/settings/terms', name: 'term-review', component: () => import('../features/review/components/TermReviewPage.vue'), meta: { title: 'Term review' } },
      // Earlier versions' addresses, which links and bookmarks may still carry.
      // The competitor candidates are suggestions on the list of all competitors now.
      { path: '/settings/candidates', redirect: '/competitors/all' },
      { path: '/terminology', redirect: (to) => ({ path: '/terms', query: to.query }) },
      { path: '/keywords', redirect: (to) => ({ path: '/terms', query: to.query }) },
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
