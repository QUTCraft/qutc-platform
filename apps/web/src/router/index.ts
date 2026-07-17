import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    { path: '/', name: 'home', component: () => import('@/views/HomeView.vue') },
    { path: '/projects', name: 'projects', component: () => import('@/views/ProjectsView.vue') },
    { path: '/resources', name: 'resources', component: () => import('@/views/ResourcesView.vue') },
    { path: '/knowledge', name: 'knowledge', component: () => import('@/views/KnowledgeView.vue') },
    { path: '/admin', name: 'admin-dashboard', component: () => import('@/views/admin/AdminDashboardView.vue'), meta: { layout: 'admin' } },
    { path: '/admin/content', name: 'admin-content', component: () => import('@/views/admin/AdminContentView.vue'), meta: { layout: 'admin' } },
    { path: '/admin/users', name: 'admin-users', component: () => import('@/views/admin/AdminUsersView.vue'), meta: { layout: 'admin' } },
    { path: '/admin/reviews', name: 'admin-reviews', component: () => import('@/views/admin/AdminReviewsView.vue'), meta: { layout: 'admin' } },
    { path: '/admin/settings', name: 'admin-settings', component: () => import('@/views/admin/AdminSettingsView.vue'), meta: { layout: 'admin' } },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/NotFoundView.vue') },
  ],
})

export default router
