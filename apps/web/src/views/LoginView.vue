<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { signIn } from '@/stores/session'

const route = useRoute()
const router = useRouter()
const mockMode = (import.meta.env.VITE_API_MODE ?? 'mock') === 'mock'
const formRef = ref<FormInstance>()
const submitting = ref(false)
const form = reactive({
  email: mockMode ? 'admin@qutcraft.local' : '',
  password: mockMode ? 'demo-admin-pass' : '',
})
const rules: FormRules = {
  email: [{ required: true, type: 'email', message: '请输入有效邮箱', trigger: 'blur' }],
  password: [{ required: true, min: 12, message: '密码至少 12 位', trigger: 'blur' }],
}

async function submit() {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  submitting.value = true
  try {
    await signIn(form.email, form.password)
    ElMessage.success('登录成功。')
    await router.replace(typeof route.query.redirect === 'string' ? route.query.redirect : '/admin')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '登录失败，请稍后重试。')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="login-card" aria-labelledby="login-title">
    <RouterLink to="/" class="login-brand">
      <span>Q</span>
      <strong>QUTCraft Platform<br><small>管理工作台</small></strong>
    </RouterLink>

    <div class="login-copy">
      <p class="eyebrow">SECURE ADMIN ACCESS</p>
      <h1 id="login-title">登录管理工作台</h1>
      <p>管理能力受 JWT 与 RBAC 保护。公开门户不需要登录。</p>
    </div>

    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" size="large" @submit.prevent="submit">
      <el-form-item label="邮箱" prop="email">
        <el-input v-model="form.email" autocomplete="email" placeholder="admin@qutcraft.local" />
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input v-model="form.password" type="password" show-password autocomplete="current-password" placeholder="••••••••••••" />
      </el-form-item>
      <el-button type="primary" native-type="submit" :loading="submitting" size="large" round>
        登录工作台
      </el-button>
    </el-form>

    <el-alert
      v-if="mockMode"
      title="Mock 模式已预填开发演示账号；切换 remote 模式后请使用后端创建的账号。"
      type="info"
      :closable="false"
      show-icon
    />
  </section>
</template>
