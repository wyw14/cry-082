import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

const monitoringRoutes: RouteRecordRaw[] = [
  { path: '/', name: 'live-monitoring', component: () => import('../pages/DashboardPage.vue'), meta: { section: 'monitoring' } },
  { path: '/alerts', name: 'alert-response', component: () => import('../pages/AlertsPage.vue'), meta: { section: 'response' } },
  { path: '/reports', name: 'regulatory-reports', component: () => import('../pages/ReportsPage.vue'), meta: { section: 'reporting' } },
  { path: '/devices', name: 'device-governance', component: () => import('../pages/DevicesPage.vue'), meta: { section: 'devices' } },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

export const monitoringRouter = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: monitoringRoutes,
  scrollBehavior: () => ({ top: 0 }),
})
