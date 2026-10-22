import { createRouter, createWebHistory, type Router, type RouterHistory } from 'vue-router'

/** The app's routes. Each feature page loads on demand, so the first paint carries only the shell. */
export function createAppRouter(history: RouterHistory = createWebHistory()): Router {
  return createRouter({
    history,
    routes: [
      {
        path: '/',
        name: 'overview',
        component: () => import('../features/overview/components/OverviewPage.vue'),
        meta: { title: 'Overview' },
      },
      {
        // One page for the form alone and for the form above a scan, so the form keeps what was typed.
        path: '/scan/:id?',
        name: 'scan',
        component: () => import('../features/scan/components/ScanPage.vue'),
        props: true,
        meta: { title: 'Scan a competitor' },
      },
      {
        path: '/terminology',
        name: 'terminology',
        component: () => import('../features/terminology/components/TerminologyPage.vue'),
        meta: { title: 'Terminology' },
      },
      {
        path: '/competitors',
        name: 'competitors',
        component: () => import('../features/competitors/components/CompetitorsPage.vue'),
        meta: { title: 'Competitors' },
      },
      {
        path: '/competitors/:id',
        name: 'company',
        component: () => import('../features/competitors/components/CompanyPage.vue'),
        props: true,
        meta: { title: 'Competitor' },
      },
      // The first version's addresses, which links and bookmarks may still carry.
      { path: '/keywords', redirect: (to) => ({ path: '/terminology', query: to.query }) },
      { path: '/analyses', redirect: (to) => ({ path: '/scan', query: to.query }) },
      { path: '/analyses/:id', redirect: (to) => ({ path: `/scan/${String(to.params.id)}`, query: to.query }) },
      { path: '/:rest(.*)*', redirect: '/' },
    ],
    scrollBehavior: (to, from, saved) => {
      if (saved) return saved
      // Filters and polling change the query, not the page; only a new page starts at the top.
      return to.path === from.path ? false : { top: 0 }
    },
  })
}
