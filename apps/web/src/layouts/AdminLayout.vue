<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Collection, DataAnalysis, Document, Files, Folder, MagicStick, Monitor, Promotion, Setting, SwitchButton, Tickets, UserFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { authApi } from '@/api/auth'
import type { OrganizationMembership } from '@/api/types'
import { session, signOut, switchSessionOrganization } from '@/stores/session'

const route = useRoute()
const router = useRouter()
const menuOpen = ref(false)
const organizations = ref<OrganizationMembership[]>([])
const selectedOrganizationId = ref(session.user?.organization_id ?? '')
const organizationLoading = ref(false)
const organizationSwitching = ref(false)

const navigation = [
  { label: '概览', to: '/admin', icon: DataAnalysis },
  { label: '内容管理', to: '/admin/content', icon: Document },
  { label: '资源文件', to: '/admin/assets', icon: Files },
  { label: '知识目录', to: '/admin/knowledge', icon: Collection },
  { label: '项目管理', to: '/admin/projects', icon: Folder },
  { label: '成员管理', to: '/admin/users', icon: UserFilled },
  { label: '审核与服务器', to: '/admin/reviews', icon: Monitor },
  { label: '审计记录', to: '/admin/audit', icon: Tickets },
  { label: '活动策划', to: '/admin/activity-planner', icon: Promotion },
  { label: '智能体配置', to: '/admin/ai', icon: MagicStick },
  { label: '系统设置', to: '/admin/settings', icon: Setting },
]

const currentOrganization = computed(() => organizations.value.find((item) => item.id === session.user?.organization_id || item.current))
const isQutcraftOrganization = computed(() => !currentOrganization.value || currentOrganization.value.slug === 'qutcraft')
const adminBrandName = computed(() => currentOrganization.value?.short_name || currentOrganization.value?.name || 'Commons')
const navigationLabel = (item: typeof navigation[number]) => item.to === '/admin/reviews' && !isQutcraftOrganization.value ? '申请审核' : item.label
const title = computed(() => route.path.startsWith('/admin/content/') ? '内容编辑器' : navigation.find((item) => item.to === route.path) ? navigationLabel(navigation.find((item) => item.to === route.path)!) : '后台管理')
const roleLabel = computed(() => session.user?.roles.includes('owner') ? '所有者' : session.user?.roles.includes('administrator') ? '管理员' : '成员')
const portalPreviewHref = computed(() => `/?organization=${encodeURIComponent(currentOrganization.value?.slug ?? 'qutcraft')}`)

function organizationRoleLabel(organization: OrganizationMembership) {
  if (organization.roles.includes('owner')) return '所有者'
  if (organization.roles.includes('administrator')) return '管理员'
  if (organization.roles.includes('editor')) return '编辑'
  return '成员'
}

async function loadOrganizations() {
  organizationLoading.value = true
  try {
    organizations.value = await authApi.getOrganizations()
    selectedOrganizationId.value = organizations.value.find((item) => item.current)?.id ?? session.user?.organization_id ?? ''
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '可用组织加载失败。')
  } finally {
    organizationLoading.value = false
  }
}

async function switchToOrganization(organizationId: string) {
  if (!organizationId || organizationId === session.user?.organization_id) return
  organizationSwitching.value = true
  try {
    await switchSessionOrganization(organizationId)
    const selected = organizations.value.find((item) => item.id === organizationId)
    ElMessage.success(`已切换到${selected?.short_name || selected?.name || '目标组织'}。`)
    window.location.assign(router.resolve('/admin').href)
  } catch (error) {
    selectedOrganizationId.value = session.user?.organization_id ?? ''
    ElMessage.error(error instanceof Error ? error.message : '组织切换失败。')
  } finally {
    organizationSwitching.value = false
  }
}

function isNavigationActive(path: string) {
  if (path === '/admin') return route.path === path
  return route.path === path || (!path.startsWith('/admin/activity-planner') && route.path.startsWith(`${path}/`))
}

async function logout() {
  await signOut()
  await router.replace('/login')
}

onMounted(loadOrganizations)
</script>

<template>
  <div class="admin-app">
    <!-- Sidebar Navigation Rail -->
    <aside class="admin-rail" :class="{ 'is-open': menuOpen }">
      <RouterLink to="/admin" class="admin-brand" @click="menuOpen = false">
        <span class="brand-badge">Q</span>
        <div class="brand-info">
          <strong>{{ adminBrandName }}</strong>
          <small>管理工作台</small>
        </div>
      </RouterLink>

      <nav class="admin-nav" aria-label="后台导航">
        <RouterLink
          v-for="item in navigation"
          :key="item.to"
          :to="item.to"
          class="rail-link"
          :class="{ 'is-active': isNavigationActive(item.to) }"
          :aria-current="isNavigationActive(item.to) ? 'page' : undefined"
          :data-testid="`admin-nav-${item.to.replaceAll('/', '-').replace(/^-/, '')}`"
          @click="menuOpen = false"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ navigationLabel(item) }}</span>
        </RouterLink>
      </nav>

      <div class="rail-footer">
        <a :href="portalPreviewHref" class="portal-link">
          <el-icon><ArrowLeft /></el-icon>
          <span>返回公开门户</span>
        </a>

        <div class="account-chip">
          <div class="account-main">
            <span class="avatar-bubble">{{ session.user?.display_name.slice(0, 1) ?? 'Q' }}</span>
            <div class="account-meta">
              <strong>{{ session.user?.display_name ?? '未登录' }}</strong>
              <small>{{ roleLabel }}</small>
            </div>
          </div>
          <button class="logout-btn" title="退出登录" aria-label="退出登录" @click="logout">
            <el-icon><SwitchButton /></el-icon>
          </button>
        </div>
      </div>
    </aside>

    <!-- Main Workspace Container -->
    <div class="admin-workspace">
      <header class="admin-topbar">
        <div class="topbar-left">
          <el-button class="admin-menu" circle aria-label="打开后台导航" @click="menuOpen = !menuOpen">
            <el-icon><DataAnalysis /></el-icon>
          </el-button>
          <div>
            <h1>{{ title }}</h1>
          </div>
        </div>

        <div class="admin-topbar-actions">
          <el-select
            v-model="selectedOrganizationId"
            class="organization-switcher"
            aria-label="切换当前组织"
            :loading="organizationLoading || organizationSwitching"
            :disabled="organizationLoading || organizationSwitching || organizations.length < 2"
            @change="switchToOrganization"
          >
            <el-option
              v-for="organization in organizations"
              :key="organization.id"
              :label="`${organization.short_name || organization.name} · ${organizationRoleLabel(organization)}`"
              :value="organization.id"
            />
          </el-select>
          <a :href="portalPreviewHref" class="portal-btn-link">
            <el-button plain round>
              查看门户 <el-icon class="el-icon--right"><ArrowLeft style="transform: rotate(180deg);" /></el-icon>
            </el-button>
          </a>
        </div>
      </header>

      <main class="admin-content">
        <slot />
      </main>
    </div>
  </div>
</template>
