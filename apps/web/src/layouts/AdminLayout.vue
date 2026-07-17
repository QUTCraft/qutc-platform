<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft, DataAnalysis, Document, Monitor, Setting, UserFilled } from '@element-plus/icons-vue'

const route = useRoute()
const menuOpen = ref(false)
const navigation = [
  { label: '概览', to: '/admin', icon: DataAnalysis },
  { label: '内容', to: '/admin/content', icon: Document },
  { label: '成员', to: '/admin/users', icon: UserFilled },
  { label: '审核与服务器', to: '/admin/reviews', icon: Monitor },
  { label: '设置', to: '/admin/settings', icon: Setting },
]
const title = computed(() => navigation.find((item) => item.to === route.path)?.label ?? '后台管理')
</script>

<template>
  <!-- MD3 admin tokens reuse --md-primary/surface; navigation rail is deliberately independent from the public portal header. -->
  <div class="admin-app">
    <aside class="admin-rail" :class="{ 'is-open': menuOpen }">
      <RouterLink to="/admin" class="admin-brand" @click="menuOpen = false"><span>Q</span><strong>QUTCraft<br><small>管理工作台</small></strong></RouterLink>
      <nav aria-label="后台导航">
        <RouterLink v-for="item in navigation" :key="item.to" :to="item.to" class="rail-link" @click="menuOpen = false"><el-icon><component :is="item.icon" /></el-icon><span>{{ item.label }}</span></RouterLink>
      </nav>
      <div class="rail-footer"><RouterLink to="/" class="portal-link"><el-icon><ArrowLeft /></el-icon>返回门户</RouterLink><div class="account-chip"><span>BK</span><div><strong>BBKarasu</strong><small>管理员</small></div></div></div>
    </aside>
    <div class="admin-workspace">
      <header class="admin-topbar"><el-button class="admin-menu" circle aria-label="打开后台导航" @click="menuOpen = !menuOpen"><el-icon><DataAnalysis /></el-icon></el-button><div><p class="eyebrow">QUTCRAFT / ADMIN</p><h1>{{ title }}</h1></div><div class="admin-topbar-actions"><span class="sync-state"><i />模拟环境</span><RouterLink to="/"><el-button plain>查看门户</el-button></RouterLink></div></header>
      <main class="admin-content"><slot /></main>
    </div>
  </div>
</template>
