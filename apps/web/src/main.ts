import 'element-plus/dist/index.css'
import '@/styles/main.css'
import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import App from '@/App.vue'
import router from '@/router'
import { expireSession } from '@/stores/session'
import { bootstrapPortalRuntime } from '@/portal/runtime'

const staleChunkRecoveryKey = 'qutc:stale-chunk-recovery'
const staleChunkRecoveryWindowMs = 30_000

function recoverFromStaleChunk(): void {
  const now = Date.now()
  const lastRecovery = Number(window.sessionStorage.getItem(staleChunkRecoveryKey) ?? 0)
  if (now - lastRecovery < staleChunkRecoveryWindowMs) return

  window.sessionStorage.setItem(staleChunkRecoveryKey, String(now))
  window.location.reload()
}

window.addEventListener('vite:preloadError', (event) => {
  event.preventDefault()
  recoverFromStaleChunk()
})

router.onError((error) => {
  if (/dynamically imported module|importing a module script failed|failed to fetch module/i.test(String(error))) {
    recoverFromStaleChunk()
  }
})

window.addEventListener('qutc:session-expired', () => {
  const redirect = router.currentRoute.value.fullPath
  expireSession()
  if (router.currentRoute.value.name !== 'login') {
    void router.replace({ name: 'login', query: { redirect } })
  }
})

void bootstrapPortalRuntime().finally(() => {
  const app = createApp(App).use(router).use(ElementPlus)
  // Wait for the initial async route and its metadata before rendering. This
  // prevents a login/admin refresh from briefly showing the public layout.
  void router.isReady().then(() => app.mount('#app'))
})
