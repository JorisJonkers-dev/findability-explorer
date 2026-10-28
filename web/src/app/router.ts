import { createRouter, createWebHistory, type Router, type RouterHistory } from 'vue-router'
import { trackPreviousPath } from '../features/shell/composables/navigation'

/** The app's routes. Each view loads on demand, so the first paint carries only the shell. */
export function createAppRouter(history: RouterHistory = createWebHistory()): Router {
  const router = createRouter({
    history,
    routes: [
      { path: '/', name: 'home', component: () => import('../features/home/components/HomeView.vue'), meta: { title: 'Overview' } },
      {
        path: '/numbers/:measure',
        name: 'number',
        component: () => import('../features/numbers/components/NumberDetailView.vue'),
        props: (to) => ({ measure: String(to.params.measure) }),
        meta: { title: 'Number' },
      },
      { path: '/buyers', name: 'buyers', component: () => import('../features/buyers/components/BuyersView.vue'), meta: { title: 'Buyers' } },
      { path: '/ai-search', name: 'ai-search', component: () => import('../features/ai-search/components/AiSearchView.vue'), meta: { title: 'AI search' } },
      {
        path: '/ai-search/answers',
        name: 'ai-answers',
        component: () => import('../features/prompt-test/components/PromptTestView.vue'),
        meta: { title: 'Every answer' },
      },
      {
        path: '/ai-search/readiness',
        name: 'ai-readiness',
        component: () => import('../features/readiness/components/ReadinessView.vue'),
        meta: { title: 'AI readiness' },
      },
      {
        path: '/settings/prompt-test',
        name: 'prompt-test-settings',
        component: () => import('../features/prompt-test/components/PromptSettingsPage.vue'),
        meta: { title: 'Prompt test settings' },
      },
      { path: '/competitors', name: 'competitors', component: () => import('../features/competitors/components/CompetitorsView.vue'), meta: { title: 'Competitors' } },
      {
        path: '/scan/:id',
        name: 'scan',
        component: () => import('../features/scan/components/ScanView.vue'),
        props: (to) => ({ scanId: String(to.params.id) }),
        meta: { title: 'Scan' },
      },
      {
        path: '/competitors/:id',
        name: 'company',
        component: () => import('../features/competitors/components/CompetitorView.vue'),
        props: (to) => ({ companyId: String(to.params.id) }),
        meta: { title: 'Competitor' },
      },
      // A company's search terms and scan history, a view of its own, with room for both.
      {
        path: '/competitors/:id/history',
        name: 'company-history',
        component: () => import('../features/competitors/components/CompanyHistoryView.vue'),
        props: (to) => ({ companyId: String(to.params.id) }),
        meta: { title: 'Over time' },
      },
      { path: '/terms', name: 'terms', component: () => import('../features/terminology/components/TermsView.vue'), meta: { title: 'Search terms' } },
      // Every term in the map, ranked, with its filters in the query: a page of its own, linked from Search terms.
      { path: '/terms/all', name: 'all-terms', component: () => import('../features/terminology/components/AllTermsView.vue'), meta: { title: 'Every term' } },
      {
        path: '/terms/families/:family',
        name: 'family',
        component: () => import('../features/products/components/FamilyView.vue'),
        props: (to) => ({ slug: String(to.params.family) }),
        meta: { title: 'Belt family' },
      },
      {
        path: '/recommendations',
        name: 'recommendations',
        component: () => import('../features/plan/components/RecommendationsView.vue'),
        meta: { title: 'Recommendations' },
      },
      {
        path: '/recommendations/:id',
        name: 'recommendation',
        component: () => import('../features/plan/components/RecommendationDetailView.vue'),
        props: (to) => ({ id: String(to.params.id) }),
        meta: { title: 'Recommendation' },
      },
      { path: '/settings', name: 'settings', component: () => import('../features/settings/components/SettingsPage.vue'), meta: { title: 'Settings' } },
      { path: '/settings/terms', name: 'term-review', component: () => import('../features/review/components/TermReviewPage.vue'), meta: { title: 'Term review' } },
      // Earlier versions' addresses, which links and bookmarks may still carry.
      {
        path: '/settings/competitors',
        name: 'competitor-review',
        component: () => import('../features/competitors/components/CompetitorReviewPage.vue'),
        meta: { title: 'Competitor review' },
      },
      // The first version's addresses, which links and bookmarks may still carry.
      { path: '/catching-up', redirect: '/' },
      { path: '/plan', redirect: '/recommendations' },
      { path: '/products', redirect: '/terms' },
      { path: '/products/:family', redirect: (to) => `/terms/families/${String(to.params.family)}` },
      { path: '/ai-search/prompt-test', redirect: '/ai-search/answers' },
      { path: '/competitors/all', redirect: '/competitors' },
      { path: '/settings/candidates', redirect: '/settings/competitors' },
      { path: '/terminology', redirect: (to) => ({ path: '/terms/all', query: to.query }) },
      { path: '/keywords', redirect: (to) => ({ path: '/terms/all', query: to.query }) },
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
