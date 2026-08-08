<script setup lang="ts">
import { computed } from 'vue'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import AppFooter from '@/components/AppFooter.vue'
import AppHeader from '@/components/AppHeader.vue'
import RouteLoadingFallback from '@/components/RouteLoadingFallback.vue'
import AdminLayout from '@/layouts/AdminLayout.vue'
import { useRoute } from 'vue-router'
import { useTheme } from '@/composables/useTheme'
import { readPortalFallback } from '@/portal/runtime'

// Keep Monet seasonal/day-night colors active globally. The manual theme
// control was removed, so the root app owns the automatic theme lifecycle.
useTheme()

const route = useRoute()
const isAdmin = computed(() => route.meta.layout === 'admin')
const isAuth = computed(() => route.meta.layout === 'auth')
const isFull = computed(() => route.meta.layout === 'full')
const portalFallback = readPortalFallback()
</script>

<template>
  <el-config-provider :locale="zhCn">
    <AdminLayout v-if="isAdmin">
      <RouterView v-slot="{ Component, route: viewRoute }">
        <Transition name="admin-page" mode="in-out">
          <div :key="viewRoute.fullPath" class="route-transition-shell admin-route-view">
            <Suspense timeout="0">
              <component :is="Component" />
              <template #fallback><RouteLoadingFallback label="正在打开管理页面…" /></template>
            </Suspense>
          </div>
        </Transition>
      </RouterView>
    </AdminLayout>
    <main v-else-if="isAuth" class="auth-shell">
      <RouterView v-slot="{ Component, route: viewRoute }">
        <Transition name="page" mode="in-out">
          <div :key="viewRoute.fullPath" class="route-transition-shell">
            <Suspense timeout="0">
              <component :is="Component" />
              <template #fallback><RouteLoadingFallback label="正在打开页面…" /></template>
            </Suspense>
          </div>
        </Transition>
      </RouterView>
    </main>
    <RouterView v-else-if="isFull" v-slot="{ Component, route: viewRoute }">
      <Transition name="page" mode="in-out">
        <div :key="viewRoute.fullPath" class="route-transition-shell">
          <Suspense timeout="0">
            <component :is="Component" />
            <template #fallback><RouteLoadingFallback label="正在打开页面…" /></template>
          </Suspense>
        </div>
      </Transition>
    </RouterView>
    <template v-else>
      <div v-if="portalFallback" class="portal-fallback-notice" role="status">
        自定义门户暂时不可用，已安全切换至默认 MD3 门户。
      </div>
      <AppHeader />
      <main class="page-shell">
        <RouterView v-slot="{ Component, route: viewRoute }">
          <Transition name="page" mode="in-out">
            <div :key="viewRoute.fullPath" class="route-transition-shell">
              <Suspense timeout="0">
                <component :is="Component" />
                <template #fallback><RouteLoadingFallback label="正在打开页面…" /></template>
              </Suspense>
            </div>
          </Transition>
        </RouterView>
      </main>
      <AppFooter />
    </template>
  </el-config-provider>
</template>
