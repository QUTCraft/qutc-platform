<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Collection, DataAnalysis, Document, Folder, MagicStick, Monitor, Setting, SwitchButton, Tickets, UserFilled } from '@element-plus/icons-vue'
import { session, signOut } from '@/stores/session'

const route = useRoute()
const router = useRouter()
const menuOpen = ref(false)

const navigation = [
  { label: '概览', to: '/admin', icon: DataAnalysis },
  { label: '内容管理', to: '/admin/content', icon: Document },
  { label: '知识目录', to: '/admin/knowledge', icon: Collection },
  { label: '项目管理', to: '/admin/projects', icon: Folder },
  { label: '成员管理', to: '/admin/users', icon: UserFilled },
  { label: '审核与服务器', to: '/admin/reviews', icon: Monitor },
  { label: '审计记录', to: '/admin/audit', icon: Tickets },
  { label: '智能体配置', to: '/admin/ai', icon: MagicStick },
  { label: '系统设置', to: '/admin/settings', icon: Setting },
]

const title = computed(() => route.path.startsWith('/admin/content/') ? '内容编辑器' : navigation.find((item) => item.to === route.path)?.label ?? '后台管理')
const roleLabel = computed(() => session.user?.roles.includes('owner') ? '所有者' : session.user?.roles.includes('administrator') ? '管理员' : '成员')

async function logout() {
  await signOut()
  await router.replace('/login')
}
</script>

<template>
  <div class="admin-app">
    <!-- Sidebar Navigation Rail -->
    <aside class="admin-rail" :class="{ 'is-open': menuOpen }">
      <RouterLink to="/admin" class="admin-brand" @click="menuOpen = false">
        <span class="brand-badge">Q</span>
        <div class="brand-info">
          <strong>QUTCraft</strong>
          <small>管理工作台</small>
        </div>
      </RouterLink>

      <nav class="admin-nav" aria-label="后台导航">
        <RouterLink
          v-for="item in navigation"
          :key="item.to"
          :to="item.to"
          class="rail-link"
          @click="menuOpen = false"
        >
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="rail-footer">
        <RouterLink to="/" class="portal-link">
          <el-icon><ArrowLeft /></el-icon>
          <span>返回公开门户</span>
        </RouterLink>

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
          <RouterLink to="/" class="portal-btn-link">
            <el-button plain round>
              查看门户 <el-icon class="el-icon--right"><ArrowLeft style="transform: rotate(180deg);" /></el-icon>
            </el-button>
          </RouterLink>
        </div>
      </header>

      <main class="admin-content">
        <slot />
      </main>
    </div>
  </div>
</template>
