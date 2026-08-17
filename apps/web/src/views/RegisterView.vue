<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { invitationApi } from '@/api/invitations'
import { signUp } from '@/stores/session'
import AsyncState from '@/components/AsyncState.vue'
import { useAsyncData } from '@/composables/useAsyncData'

const route = useRoute()
const router = useRouter()
const invitationToken = typeof route.query.invitation === 'string' ? route.query.invitation : ''
const { data: invitation, error, loading, refresh } = useAsyncData(() => invitationToken ? invitationApi.get(invitationToken) : Promise.resolve(null))
const formRef = ref<FormInstance>()
const submitting = ref(false)
const form = reactive({ email: '', display_name: '', password: '', confirm_password: '' })
watch(invitation, (value) => {
  if (value?.email) form.email = value.email
}, { immediate: true })
const rules: FormRules = {
  email: [{ required: true, type: 'email', message: '请输入有效邮箱账号', trigger: 'blur' }],
  display_name: [{ required: true, min: 1, max: 80, message: '请输入 1-80 个字符的显示名', trigger: 'blur' }],
  password: [{ required: true, min: 12, message: '密码至少需要 12 个字符', trigger: 'blur' }],
  confirm_password: [{ required: true, validator: (_rule, value, callback) => value === form.password ? callback() : callback(new Error('两次输入的密码不一致')), trigger: 'blur' }],
}

async function submit() {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  submitting.value = true
  try {
    await signUp({ email: form.email, display_name: form.display_name, password: form.password, ...(invitationToken ? { invitation_token: invitationToken } : {}) })
    ElMessage.success('注册成功，欢迎加入 QUTCraft Commons。')
    await router.replace(invitationToken ? '/admin' : '/')
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '注册失败，请稍后重试。')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="register-page">
    <section class="register-card">
      <RouterLink to="/" class="login-back-link">← 返回公开门户</RouterLink>
      <AsyncState v-if="invitationToken" :loading="loading" :error="error" @retry="refresh">
        <template v-if="invitation"><p class="register-context">{{ invitation.organization_name }} 已为 {{ invitation.email }} 预创建待激活账户。请设置显示名和密码以接受邀请。</p></template>
      </AsyncState>
      <h1>创建成员账户</h1>
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" size="large" @submit.prevent="submit">
        <el-form-item label="邮箱" prop="email"><el-input v-model="form.email" type="email" autocomplete="email" placeholder="name@example.com" :readonly="Boolean(invitation)" /></el-form-item>
        <el-form-item label="显示名" prop="display_name"><el-input v-model="form.display_name" autocomplete="name" placeholder="你的社团昵称" /></el-form-item>
        <el-form-item label="密码" prop="password"><el-input v-model="form.password" type="password" show-password autocomplete="new-password" /></el-form-item>
        <el-form-item label="确认密码" prop="confirm_password"><el-input v-model="form.confirm_password" type="password" show-password autocomplete="new-password" /></el-form-item>
        <el-button type="primary" native-type="submit" :loading="submitting" round>{{ invitationToken ? '激活账户并接受邀请' : '注册并继续' }}</el-button>
      </el-form>
      <p class="register-footnote">已有账户？<RouterLink :to="{ name: 'login', query: { redirect: invitationToken ? `/invite/${invitationToken}` : '/admin' } }">返回登录</RouterLink></p>
    </section>
  </div>
</template>

<style scoped>
.register-page { min-height: 100vh; display: grid; place-items: center; padding: 32px 20px; background: var(--md-sys-color-surface); }
.register-card { width: min(520px, 100%); padding: 36px; background: var(--md-sys-color-surface-container); border: 1px solid var(--md-sys-color-outline-variant); border-radius: 28px; box-shadow: var(--md-elevation-2); }
.register-card h1 { margin: 26px 0 22px; font-size: 2rem; }
.register-context, .register-footnote { color: var(--md-sys-color-on-surface-variant); line-height: 1.7; }
.register-footnote { margin: 20px 0 0; text-align: center; }
.register-footnote a { color: var(--md-sys-color-primary); font-weight: 700; }
</style>
