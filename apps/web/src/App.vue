<script setup lang="ts">
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'
</script>

<template>
  <el-config-provider :locale="zhCn">
    <AppHeader />
    <main class="page-shell"><RouterView /></main>
    <AppFooter />
  </el-config-provider>
</template>
