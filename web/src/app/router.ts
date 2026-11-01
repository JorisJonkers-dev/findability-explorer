import { createRouter, createWebHistory, type Router, type RouterHistory } from 'vue-router'
import { trackPreviousPath } from '../features/shell/composables/navigation'

declare module 'vue-router' {
  interface RouteMeta {
    /** The page's name in the browser tab. */
    title?: string
    /** Results is for everyone signed in; Manage is for administrators. */
    view?: 'results' | 'manage'
  }
}

/** The app's routes. Each view loads on demand, so the first paint carries only the shell. */
export function createAppRouter(history: RouterHistory = createWebHistory()): Router {
  const router = createRouter({
    history,
    // The site map of the spec's section 3: Results for everyone signed in, Manage for administrators.
    // Each page says which view it is in; a page another ticket has not rebuilt yet is today's page.
    routes: [
      { path: '/', name: 'home', component: () => import('../features/home/components/HomeView.vue'), meta: { title: 'Overview', view: 'results' } },
      {
        path: '/goals/:measure',
        name: 'goal',
        component: () => import('../features/numbers/components/NumberDetailView.vue'),
        props: (to) => ({ measure: String(to.params.measure) }),
        meta: { title: 'Goal', view: 'results' },
      },
      {
        path: '/ai-responses',
        name: 'ai-responses',
        component: () => import('../features/ai-search/components/AiSearchView.vue'),
        meta: { title: 'AI responses', view: 'results' },
      },
      {
        path: '/ai-responses/all',
        name: 'every-response',
        component: () => import('../features/prompt-test/components/PromptTestView.vue'),
        meta: { title: 'Every response', view: 'results' },
      },
      {
        path: '/ai-readiness',
        name: 'ai-readiness',
        component: () => import('../features/readiness/components/ReadinessView.vue'),
        meta: { title: 'AI readiness', view: 'results' },
      },
      { path: '/terms', name: 'terms', component: () => import('../features/terminology/components/TermsView.vue'), meta: { title: 'Search terms', view: 'results' } },
      // Every term in the map, ranked, with its filters in the query: a page of its own, linked from Search terms.
      {
        path: '/terms/all',
        name: 'all-terms',
        component: () => import('../features/terminology/components/AllTermsView.vue'),
        meta: { title: 'Every term', view: 'results' },
      },
      {
        path: '/terms/families/:family',
        name: 'family',
        component: () => import('../features/products/components/FamilyView.vue'),
        props: (to) => ({ slug: String(to.params.family) }),
        meta: { title: 'Belt family', view: 'results' },
      },
      {
        path: '/competitors',
        name: 'competitors',
        component: () => import('../features/competitors/components/CompetitorsView.vue'),
        meta: { title: 'Competitors', view: 'results' },
      },
      {
        path: '/competitors/:id',
        name: 'company',
        component: () => import('../features/competitors/components/CompetitorView.vue'),
        props: (to) => ({ companyId: String(to.params.id) }),
        meta: { title: 'Competitor', view: 'results' },
      },
      // A company's search terms and scan history, a view of its own, with room for both.
      {
        path: '/competitors/:id/history',
        name: 'company-history',
        component: () => import('../features/competitors/components/CompanyHistoryView.vue'),
        props: (to) => ({ companyId: String(to.params.id) }),
        meta: { title: 'Over time', view: 'results' },
      },
      {
        path: '/recommendations',
        name: 'recommendations',
        component: () => import('../features/plan/components/RecommendationsView.vue'),
        meta: { title: 'Recommendations', view: 'results' },
      },
      {
        path: '/recommendations/:id',
        name: 'recommendation',
        component: () => import('../features/plan/components/RecommendationDetailView.vue'),
        props: (to) => ({ id: String(to.params.id) }),
        meta: { title: 'Recommendation', view: 'results' },
      },
      { path: '/help', name: 'help', component: () => import('../features/help/components/HelpView.vue'), meta: { title: 'Help', view: 'results' } },
      {
        path: '/help/:topic',
        name: 'help-topic',
        component: () => import('../features/help/components/HelpView.vue'),
        meta: { title: 'Help', view: 'results' },
      },

      // Manage. A section whose only page so far is one of its subpages opens on that subpage.
      { path: '/manage', name: 'activity', component: () => import('../features/manage/components/ActivityView.vue'), meta: { title: 'Activity', view: 'manage' } },
      {
        path: '/manage/prompt-test',
        name: 'manage-prompt-test',
        component: () => import('../features/prompt-test/components/PromptSettingsPage.vue'),
        meta: { title: 'Prompt test', view: 'manage' },
      },
      // Website scans.
      { path: '/manage/scans', name: 'manage-scans', component: () => import('../features/manage/components/ScansIndex.vue'), meta: { title: 'Website scans', view: 'manage' } },
      { path: '/manage/scans/queue', name: 'scan-queue', component: () => import('../features/scan/components/ScanQueuePage.vue'), meta: { title: 'Scan queue', view: 'manage' } },
      { path: '/manage/scans/recent', name: 'recent-scans', component: () => import('../features/scan/components/RecentScansPage.vue'), meta: { title: 'Recent scans', view: 'manage' } },
      {
        path: '/manage/scans/settings',
        name: 'scan-settings',
        component: () => import('../features/scan/components/ScanSettingsPage.vue'),
        meta: { title: 'Scan settings', view: 'manage' },
      },
      {
        path: '/manage/scans/:id',
        name: 'scan',
        component: () => import('../features/scan/components/ScanView.vue'),
        props: (to) => ({ scanId: String(to.params.id) }),
        meta: { title: 'Scan', view: 'manage' },
      },
      // Competitor list.
      {
        path: '/manage/competitors',
        name: 'competitor-list',
        component: () => import('../features/manage/components/CompetitorListIndex.vue'),
        meta: { title: 'Competitor list', view: 'manage' },
      },
      {
        path: '/manage/competitors/companies',
        name: 'companies',
        component: () => import('../features/competitors/components/CompaniesPage.vue'),
        meta: { title: 'Companies', view: 'manage' },
      },
      {
        path: '/manage/competitors/companies/:id',
        name: 'manage-company',
        component: () => import('../features/competitors/components/CompanyPage.vue'),
        props: (to) => ({ companyId: String(to.params.id) }),
        meta: { title: 'Company', view: 'manage' },
      },
      {
        path: '/manage/competitors/candidates',
        name: 'competitor-candidates',
        component: () => import('../features/competitors/components/CompetitorReviewPage.vue'),
        meta: { title: 'Candidates', view: 'manage' },
      },
      {
        path: '/manage/competitors/check',
        name: 'check-the-list',
        component: () => import('../features/competitors/components/CheckListPage.vue'),
        meta: { title: 'Check the list', view: 'manage' },
      },
      // Search term list.
      { path: '/manage/terms', name: 'term-list', component: () => import('../features/manage/components/TermListIndex.vue'), meta: { title: 'Search term list', view: 'manage' } },
      { path: '/manage/terms/review', name: 'term-review', component: () => import('../features/review/components/TermReviewRunPage.vue'), meta: { title: 'Term review', view: 'manage' } },
      {
        path: '/manage/terms/waiting',
        name: 'waiting-terms',
        component: () => import('../features/review/components/TermReviewPage.vue'),
        meta: { title: 'Waiting terms', view: 'manage' },
      },
      // Recommendations.
      {
        path: '/manage/recommendations',
        name: 'manage-recommendations',
        component: () => import('../features/manage/components/RecommendationsIndex.vue'),
        meta: { title: 'Recommendations', view: 'manage' },
      },
      {
        path: '/manage/recommendations/tick-off',
        name: 'tick-off',
        component: () => import('../features/plan/components/TickOffPage.vue'),
        meta: { title: 'Tick off yourself', view: 'manage' },
      },
      { path: '/manage/recommendations/goals', name: 'goals-edit', component: () => import('../features/plan/components/GoalsPage.vue'), meta: { title: 'Goals', view: 'manage' } },
      // Keys and spending.
      { path: '/manage/keys', name: 'keys', component: () => import('../features/manage/components/KeysIndex.vue'), meta: { title: 'Keys and spending', view: 'manage' } },
      { path: '/manage/keys/budgets', name: 'budgets', component: () => import('../features/settings/components/BudgetsPage.vue'), meta: { title: 'Budgets', view: 'manage' } },
      { path: '/manage/keys/events', name: 'events', component: () => import('../features/settings/components/EventsPage.vue'), meta: { title: 'Events', view: 'manage' } },
      {
        path: '/manage/keys/:provider(openrouter|anthropic|mistral)',
        name: 'key',
        component: () => import('../features/settings/components/KeyPage.vue'),
        props: (to) => ({ provider: String(to.params.provider) }),
        meta: { title: 'Key', view: 'manage' },
      },

      // The addresses of 1.12.0, which links and bookmarks may still carry. A redirect keeps the query.
      { path: '/numbers/:measure', redirect: (to) => `/goals/${String(to.params.measure)}` },
      { path: '/ai-search', redirect: '/ai-responses' },
      { path: '/ai-search/answers', redirect: '/ai-responses/all' },
      { path: '/ai-search/readiness', redirect: '/ai-readiness' },
      { path: '/buyers', redirect: '/' },
      { path: '/scan/:id', redirect: (to) => `/manage/scans/${String(to.params.id)}` },
      { path: '/settings', redirect: '/manage/keys' },
      { path: '/settings/prompt-test', redirect: '/manage/prompt-test' },
      { path: '/settings/terms', redirect: '/manage/terms' },
      { path: '/settings/competitors', redirect: '/manage/competitors' },
      { path: '/how-it-works', redirect: '/help' },
      // Earlier versions' addresses.
      { path: '/catching-up', redirect: '/' },
      { path: '/plan', redirect: '/recommendations' },
      { path: '/products', redirect: '/terms' },
      { path: '/products/:family', redirect: (to) => `/terms/families/${String(to.params.family)}` },
      { path: '/ai-search/prompt-test', redirect: '/ai-responses/all' },
      { path: '/competitors/all', redirect: '/competitors' },
      { path: '/settings/candidates', redirect: '/manage/competitors/candidates' },
      { path: '/terminology', redirect: '/terms/all' },
      { path: '/keywords', redirect: '/terms/all' },
      { path: '/scan', redirect: '/competitors' },
      { path: '/analyses', redirect: '/competitors' },
      { path: '/analyses/:id', redirect: (to) => `/manage/scans/${String(to.params.id)}` },
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
