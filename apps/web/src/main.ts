import 'element-plus/dist/index.css'
import '@/styles/main.css'
import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import App from '@/App.vue'
import router from '@/router'
import { expireSession } from '@/stores/session'
import { bootstrapPortalRuntime } from '@/portal/runtime'

window.addEventListener('qutc:session-expired', () => {
  const redirect = router.currentRoute.value.fullPath
  expireSession()
  if (router.currentRoute.value.name !== 'login') {
    void router.replace({ name: 'login', query: { redirect } })
  }
})

void bootstrapPortalRuntime().finally(() => {
  createApp(App).use(router).use(ElementPlus).mount('#app')
})
