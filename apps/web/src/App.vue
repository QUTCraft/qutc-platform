<script setup lang="ts">
import { computed } from 'vue'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'
import AdminLayout from '@/layouts/AdminLayout.vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const isAdmin = computed(() => route.meta.layout === 'admin')
const isAuth = computed(() => route.meta.layout === 'auth')
const isFull = computed(() => route.meta.layout === 'full')
</script>

<template>
  <el-config-provider :locale="zhCn">
    <AdminLayout v-if="isAdmin"><RouterView /></AdminLayout>
    <main v-else-if="isAuth" class="auth-shell"><RouterView /></main>
    <RouterView v-else-if="isFull" />
    <template v-else>
      <AppHeader />
      <main class="page-shell"><RouterView /></main>
      <AppFooter />
    </template>
  </el-config-provider>
</template>
