<script setup lang="ts">
import { Menu, UserFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { ref } from 'vue'

const mobileOpen = ref(false)
const showLoginHint = () => ElMessage.info('认证接口将在 /api/v1/auth 下接入。')
const showJoinHint = () => ElMessage.info('加入申请将通过公开申请接口提交。')
const links = [
  { to: '/', label: '首页' },
  { to: '/projects', label: '项目' },
  { to: '/resources', label: '资源' },
  { to: '/knowledge', label: '知识库' },
]
</script>

<template>
  <header class="app-header">
    <RouterLink class="brand" to="/" aria-label="QUTCraft Commons 首页">
      <span class="brand-mark">Q</span>
      <span><strong>QUTCraft Commons</strong><small>Qingdao University of Technology</small></span>
    </RouterLink>

    <nav class="desktop-nav" aria-label="公开门户导航">
      <RouterLink v-for="link in links" :key="link.to" :to="link.to">{{ link.label }}</RouterLink>
    </nav>

    <div class="header-actions">
      <el-button text :icon="UserFilled" aria-label="成员登录" @click="showLoginHint">成员登录</el-button>
      <el-button type="primary" round @click="showJoinHint">加入我们</el-button>
      <el-button class="menu-button" text circle :icon="Menu" aria-label="打开导航" @click="mobileOpen = true" />
    </div>

  </header>

  <el-drawer v-model="mobileOpen" append-to-body direction="rtl" size="min(86vw, 360px)" title="导航">
    <nav class="mobile-nav" aria-label="移动端公开门户导航">
      <RouterLink v-for="link in links" :key="link.to" :to="link.to" @click="mobileOpen = false">{{ link.label }}</RouterLink>
    </nav>
  </el-drawer>
</template>
