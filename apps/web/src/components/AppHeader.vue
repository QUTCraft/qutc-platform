<script setup lang="ts">
import { Menu, UserFilled } from '@element-plus/icons-vue'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { organizationSlug } from '@/api/portal'
import { useApplyTransition } from '@/composables/useApplyTransition'
import { usePortalIdentity } from '@/composables/usePortalIdentity'

const router = useRouter()
const { navigateToApply } = useApplyTransition()
const mobileOpen = ref(false)
const { organization, loadPortalOrganization } = usePortalIdentity()
const isQutcraftPortal = organizationSlug === 'qutcraft'
const organizationName = computed(() => organization.value?.name ?? (organizationSlug === 'qutcraft' ? 'QUTCraft Commons' : organizationSlug))
const organizationSubtitle = computed(() => organization.value?.short_name ? `${organization.value.short_name} · 公共门户` : '公共门户')

const goToLogin = () => router.push({ name: 'login' })
const runPrimaryAction = (event: MouseEvent) => {
  if (isQutcraftPortal) {
    navigateToApply(event)
    return
  }
  void router.push({ name: 'projects' })
}

const links = [
  { to: '/', label: '首页' },
  { to: '/posts', label: '动态' },
  { to: '/projects', label: '项目' },
  { to: '/resources', label: '资源' },
  { to: '/knowledge', label: '知识库' },
]

onMounted(() => {
  void loadPortalOrganization().catch(() => undefined)
})
</script>

<template>
  <header class="app-header">
    <RouterLink class="brand" to="/" :aria-label="`${organizationName} 首页`">
      <span class="brand-mark">Q</span>
      <span>
        <strong>{{ organizationName }}</strong>
        <small>{{ organizationSubtitle }}</small>
      </span>
    </RouterLink>

    <nav class="desktop-nav" aria-label="公开门户导航">
      <RouterLink v-for="link in links" :key="link.to" :to="link.to">{{ link.label }}</RouterLink>
    </nav>

    <div class="header-actions">
      <el-button class="header-login-btn" text :icon="UserFilled" aria-label="成员登录" @click="goToLogin">成员登录</el-button>
      <el-button class="header-join-btn" type="primary" round @click="(event: MouseEvent) => runPrimaryAction(event)">{{ isQutcraftPortal ? '加入我们' : '公开项目' }}</el-button>
      <el-button class="menu-button" text circle :icon="Menu" aria-label="打开导航" @click="mobileOpen = true" />
    </div>
  </header>

  <el-drawer v-model="mobileOpen" append-to-body direction="rtl" size="min(86vw, 360px)" title="导航">
    <nav class="mobile-nav" aria-label="移动端公开门户导航">
      <RouterLink v-for="link in links" :key="link.to" :to="link.to" @click="mobileOpen = false">{{ link.label }}</RouterLink>
      <RouterLink to="/login" @click="mobileOpen = false">成员登录</RouterLink>
    </nav>
  </el-drawer>
</template>
