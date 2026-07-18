import { createRouter, createWebHistory } from 'vue-router'
import { restoreSession, session } from '@/stores/session'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    { path: '/', name: 'home', component: () => import('@/views/HomeView.vue') },
    { path: '/projects', name: 'projects', component: () => import('@/views/ProjectsView.vue') },
    { path: '/resources', name: 'resources', component: () => import('@/views/ResourcesView.vue') },
    { path: '/knowledge', name: 'knowledge', component: () => import('@/views/KnowledgeView.vue') },
    { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { layout: 'auth', guestOnly: true } },
    { path: '/admin', name: 'admin-dashboard', component: () => import('@/views/admin/AdminDashboardView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/content', name: 'admin-content', component: () => import('@/views/admin/AdminContentView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/users', name: 'admin-users', component: () => import('@/views/admin/AdminUsersView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/reviews', name: 'admin-reviews', component: () => import('@/views/admin/AdminReviewsView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/settings', name: 'admin-settings', component: () => import('@/views/admin/AdminSettingsView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/NotFoundView.vue') },
  ],
})

router.beforeEach(async (to) => {
  await restoreSession()
  if (to.meta.requiresAuth && !session.user) return { name: 'login', query: { redirect: to.fullPath } }
  if (to.meta.guestOnly && session.user) return { path: typeof to.query.redirect === 'string' ? to.query.redirect : '/admin' }
})

export default router
