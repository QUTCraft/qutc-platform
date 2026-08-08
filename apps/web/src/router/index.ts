import { defineAsyncComponent, type Component } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import RouteLoadingFallback from '@/components/RouteLoadingFallback.vue'
import { restoreSession, session } from '@/stores/session'

function lazyView(loader: () => Promise<Component | { default: Component }>) {
  return defineAsyncComponent({
    loader,
    delay: 0,
    loadingComponent: RouteLoadingFallback,
    // RouterView's async component is wrapped by a transition and a Suspense
    // boundary. Keep the explicit loading component visible in that chain.
    suspensible: false,
  }) as Component
}

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    { path: '/', name: 'home', component: lazyView(() => import('@/views/HomeView.vue')) },
    { path: '/posts', name: 'posts', component: lazyView(() => import('@/views/PostsView.vue')) },
    { path: '/posts/:id', name: 'post-detail', component: lazyView(() => import('@/views/ContentDetailView.vue')), props: { contentType: 'news' } },
    { path: '/projects', name: 'projects', component: lazyView(() => import('@/views/ProjectsView.vue')) },
    { path: '/resources', name: 'resources', component: lazyView(() => import('@/views/ResourcesView.vue')) },
    { path: '/resources/:id', name: 'resource-detail', component: lazyView(() => import('@/views/ContentDetailView.vue')), props: { contentType: 'resource' } },
    { path: '/knowledge', name: 'knowledge', component: lazyView(() => import('@/views/KnowledgeView.vue')) },
    { path: '/knowledge/:id', name: 'knowledge-detail', component: lazyView(() => import('@/views/ContentDetailView.vue')), props: { contentType: 'knowledge' } },
    { path: '/apply', name: 'apply', component: lazyView(() => import('@/views/ApplyView.vue')), meta: { layout: 'full' } },
    { path: '/login', name: 'login', component: lazyView(() => import('@/views/LoginView.vue')), meta: { layout: 'auth', guestOnly: true } },
    { path: '/register', name: 'register', component: lazyView(() => import('@/views/RegisterView.vue')), meta: { layout: 'auth', guestOnly: true } },
    { path: '/invite/:token', name: 'invite', component: lazyView(() => import('@/views/InviteView.vue')) },
    { path: '/admin', name: 'admin-dashboard', component: lazyView(() => import('@/views/admin/AdminDashboardView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/content', name: 'admin-content', component: lazyView(() => import('@/views/admin/AdminContentView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/content/new', name: 'admin-content-new', component: lazyView(() => import('@/views/admin/AdminContentEditorView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/content/:id/edit', name: 'admin-content-edit', component: lazyView(() => import('@/views/admin/AdminContentEditorView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/knowledge', name: 'admin-knowledge', component: lazyView(() => import('@/views/admin/AdminKnowledgeDirectoriesView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/projects', name: 'admin-projects', component: lazyView(() => import('@/views/admin/AdminProjectsView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/users', name: 'admin-users', component: lazyView(() => import('@/views/admin/AdminUsersView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/reviews', name: 'admin-reviews', component: lazyView(() => import('@/views/admin/AdminReviewsView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/audit', name: 'admin-audit', component: lazyView(() => import('@/views/admin/AdminAuditView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/ai', name: 'admin-ai', component: lazyView(() => import('@/views/admin/AdminAISettingsView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/activity-planner', name: 'admin-activity-planner', component: lazyView(() => import('@/views/admin/AdminActivityPlannerView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/settings', name: 'admin-settings', component: lazyView(() => import('@/views/admin/AdminSettingsView.vue')), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: lazyView(() => import('@/views/NotFoundView.vue')) },
  ],
})

router.beforeEach(async (to) => {
  await restoreSession()
  if (to.meta.requiresAuth && !session.user) return { name: 'login', query: { redirect: to.fullPath } }
  if (to.meta.guestOnly && session.user) return { path: typeof to.query.redirect === 'string' ? to.query.redirect : '/admin' }
})

export default router
