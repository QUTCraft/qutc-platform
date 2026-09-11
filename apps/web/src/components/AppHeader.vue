<script setup lang="ts">
import { Menu, UserFilled } from '@element-plus/icons-vue'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { resolveApiUrl } from '@/api/client'
import { organizationSlug } from '@/api/portal'
import { useApplyTransition } from '@/composables/useApplyTransition'
import { usePortalIdentity } from '@/composables/usePortalIdentity'
import { session, signOut } from '@/stores/session'

const route = useRoute()
const router = useRouter()
const { navigateToApply } = useApplyTransition()
const signedIn = computed(() => Boolean(session.user))
const mobileOpen = ref(false)
const { organization, loadPortalOrganization } = usePortalIdentity()
const isQutcraftPortal = organizationSlug === 'qutcraft'
const organizationName = computed(() => organization.value?.name ?? (organizationSlug === 'qutcraft' ? 'QUTCraft Commons' : organizationSlug))
const organizationSubtitle = computed(() => organization.value?.short_name ? `${organization.value.short_name} · 公共门户` : '公共门户')
const organizationLogo = computed(() => organization.value?.logo_url ? resolveApiUrl(organization.value.logo_url) : '')
const logoLoadFailed = ref(false)

watch(organizationLogo, () => {
  logoLoadFailed.value = false
})

const goToLogin = () => router.push({ name: 'login', query: { redirect: route.fullPath } })
const goToRegister = () => router.push({ name: 'register' })
const goToWorkspace = () => router.push({ name: 'admin-dashboard' })
async function logout() {
  await signOut()
  if (route.path.startsWith('/admin')) await router.replace('/')
}
const skinSiteUrl = 'https://skin.qutcraft.cn/'
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
      <span class="brand-mark" :class="{ 'has-image': organizationLogo && !logoLoadFailed }">
        <img v-if="organizationLogo && !logoLoadFailed" :src="organizationLogo" :alt="`${organizationName} Logo`" @error="logoLoadFailed = true" />
        <span v-else>Q</span>
      </span>
      <span>
        <strong>{{ organizationName }}</strong>
        <small>{{ organizationSubtitle }}</small>
      </span>
    </RouterLink>

    <nav class="desktop-nav" aria-label="公开门户导航">
      <RouterLink v-for="link in links" :key="link.to" :to="link.to">{{ link.label }}</RouterLink>
    </nav>

    <div class="header-actions">
      <a
        v-if="isQutcraftPortal"
        class="header-skin-link"
        :href="skinSiteUrl"
        target="_blank"
        rel="noopener noreferrer"
        aria-label="打开 QUTC Skin 皮肤站"
      >
        <span>QUTC Skin</span>
        <span class="header-external-mark" aria-hidden="true">↗</span>
      </a>
      <template v-if="signedIn">
        <span class="header-session-name">{{ session.user?.display_name }}</span>
        <el-button class="header-workspace-btn" text @click="goToWorkspace">工作台</el-button>
        <el-button class="header-login-btn" text @click="logout">退出</el-button>
      </template>
      <template v-else>
        <el-button class="header-login-btn" text :icon="UserFilled" aria-label="成员登录" @click="goToLogin">成员登录</el-button>
        <el-button class="header-register-btn" text aria-label="注册成员账户" @click="goToRegister">注册</el-button>
        <el-button class="header-join-btn" type="primary" round @click="(event: MouseEvent) => runPrimaryAction(event)">{{ isQutcraftPortal ? '加入我们' : '公开项目' }}</el-button>
      </template>
      <el-button class="menu-button" text circle :icon="Menu" aria-label="打开导航" @click="mobileOpen = true" />
    </div>
  </header>

  <el-drawer v-model="mobileOpen" append-to-body direction="rtl" size="min(86vw, 360px)" title="导航">
    <nav class="mobile-nav" aria-label="移动端公开门户导航">
      <RouterLink v-for="link in links" :key="link.to" :to="link.to" @click="mobileOpen = false">{{ link.label }}</RouterLink>
      <a
        v-if="isQutcraftPortal"
        class="mobile-external-link"
        :href="skinSiteUrl"
        target="_blank"
        rel="noopener noreferrer"
        aria-label="打开 QUTC Skin 皮肤站"
      >
        <span>QUTC Skin 皮肤站</span>
        <span aria-hidden="true">↗</span>
      </a>
      <template v-if="signedIn">
        <RouterLink to="/admin" @click="mobileOpen = false">工作台</RouterLink>
        <button type="button" class="mobile-logout" @click="mobileOpen = false; logout()">退出登录</button>
      </template>
      <template v-else>
        <RouterLink :to="{ name: 'login', query: { redirect: route.fullPath } }" @click="mobileOpen = false">成员登录</RouterLink>
        <RouterLink to="/register" @click="mobileOpen = false">注册成员账户</RouterLink>
      </template>
    </nav>
  </el-drawer>
</template>
