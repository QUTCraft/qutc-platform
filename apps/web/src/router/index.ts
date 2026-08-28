import { createRouter, createWebHistory } from 'vue-router'
import { restoreSession, session } from '@/stores/session'

// 视图采用按路由懒加载，减少公共门户首次打开时下载管理端代码的开销。
const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    { path: '/', name: 'home', component: () => import('@/views/HomeView.vue') },
    { path: '/posts', name: 'posts', component: () => import('@/views/PostsView.vue') },
    { path: '/posts/:id', name: 'post-detail', component: () => import('@/views/ContentDetailView.vue'), props: { contentType: 'news' } },
    { path: '/projects', name: 'projects', component: () => import('@/views/ProjectsView.vue') },
    { path: '/resources', name: 'resources', component: () => import('@/views/ResourcesView.vue') },
    { path: '/resources/:id', name: 'resource-detail', component: () => import('@/views/ContentDetailView.vue'), props: { contentType: 'resource' } },
    { path: '/knowledge', name: 'knowledge', component: () => import('@/views/KnowledgeView.vue') },
    { path: '/knowledge/:id', name: 'knowledge-detail', component: () => import('@/views/ContentDetailView.vue'), props: { contentType: 'knowledge' } },
    { path: '/apply', name: 'apply', component: () => import('@/views/ApplyView.vue'), meta: { layout: 'full' } },
    { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue'), meta: { layout: 'auth', guestOnly: true } },
    { path: '/register', name: 'register', component: () => import('@/views/RegisterView.vue'), meta: { layout: 'auth', guestOnly: true } },
    { path: '/invite/:token', name: 'invite', component: () => import('@/views/InviteView.vue') },
    { path: '/admin', name: 'admin-dashboard', component: () => import('@/views/admin/AdminDashboardView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/content', name: 'admin-content', component: () => import('@/views/admin/AdminContentView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/content/new', name: 'admin-content-new', component: () => import('@/views/admin/AdminContentEditorView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/content/:id/edit', name: 'admin-content-edit', component: () => import('@/views/admin/AdminContentEditorView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/assets', name: 'admin-assets', component: () => import('@/views/admin/AdminAssetsView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/knowledge', name: 'admin-knowledge', component: () => import('@/views/admin/AdminKnowledgeDirectoriesView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/projects', name: 'admin-projects', component: () => import('@/views/admin/AdminProjectsView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/users', name: 'admin-users', component: () => import('@/views/admin/AdminUsersView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/reviews', name: 'admin-reviews', component: () => import('@/views/admin/AdminReviewsView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/audit', name: 'admin-audit', component: () => import('@/views/admin/AdminAuditView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/ai', name: 'admin-ai', component: () => import('@/views/admin/AdminAISettingsView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/activity-planner', name: 'admin-activity-planner', component: () => import('@/views/admin/AdminActivityPlannerView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/admin/settings', name: 'admin-settings', component: () => import('@/views/admin/AdminSettingsView.vue'), meta: { layout: 'admin', requiresAuth: true } },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/NotFoundView.vue') },
  ],
})

// 路由前置守卫：每次导航先恢复一次服务端会话，再依据声明式路由元信息决定跳转；
// 权限细粒度校验仍由后端执行，前端守卫仅负责用户体验与入口保护。
router.beforeEach(async (to) => {
  await restoreSession()
  if (to.meta.requiresAuth && !session.user) return { name: 'login', query: { redirect: to.fullPath } }
  if (to.meta.guestOnly && session.user) return { path: typeof to.query.redirect === 'string' ? to.query.redirect : '/admin' }
})

export default router
