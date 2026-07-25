<script setup lang="ts">
import { Menu, UserFilled } from '@element-plus/icons-vue'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { usePageTransition } from '@/composables/usePageTransition'

const { navigateToApply } = usePageTransition()
const router = useRouter()
const mobileOpen = ref(false)

const goToLogin = () => router.push({ name: 'login' })

const links = [
  { to: '/', label: '首页' },
  { to: '/posts', label: '动态' },
  { to: '/projects', label: '项目' },
  { to: '/resources', label: '资源' },
  { to: '/knowledge', label: '知识库' },
]
</script>

<template>
  <header class="app-header">
    <RouterLink class="brand" to="/" aria-label="QUTCraft Commons 首页">
      <span class="brand-mark">Q</span>
      <span>
        <strong>QUTCraft Commons</strong>
        <small>Qingdao University of Technology</small>
      </span>
    </RouterLink>

    <nav class="desktop-nav" aria-label="公开门户导航">
      <RouterLink v-for="link in links" :key="link.to" :to="link.to">{{ link.label }}</RouterLink>
    </nav>

    <div class="header-actions">
      <el-button class="header-login-btn" text :icon="UserFilled" aria-label="成员登录" @click="goToLogin">成员登录</el-button>
      <el-button class="header-join-btn" type="primary" round @click="(e: MouseEvent) => navigateToApply(e)">加入我们</el-button>
      <el-button class="menu-button" text circle :icon="Menu" aria-label="打开导航" @click="mobileOpen = true" />
    </div>
  </header>

  <el-drawer v-model="mobileOpen" append-to-body direction="rtl" size="min(86vw, 360px)" title="导航">
    <nav class="mobile-nav" aria-label="移动端公开门户导航">
      <RouterLink v-for="link in links" :key="link.to" :to="link.to" @click="mobileOpen = false">{{ link.label }}</RouterLink>
    </nav>
  </el-drawer>
</template>
