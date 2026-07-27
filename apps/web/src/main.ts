import 'element-plus/dist/index.css'
import '@/styles/main.css'
import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import App from '@/App.vue'
import router from '@/router'
import { expireSession } from '@/stores/session'

window.addEventListener('qutc:session-expired', () => {
  const redirect = router.currentRoute.value.fullPath
  expireSession()
  if (router.currentRoute.value.name !== 'login') {
    void router.replace({ name: 'login', query: { redirect } })
  }
})

createApp(App).use(router).use(ElementPlus).mount('#app')
