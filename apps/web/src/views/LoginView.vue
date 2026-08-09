<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ArrowLeft, Key, User } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { isMockApiMode } from '@/api/client'
import { organizationSlug } from '@/api/portal'
import { usePortalIdentity } from '@/composables/usePortalIdentity'
import { signIn } from '@/stores/session'

const route = useRoute()
const router = useRouter()
const mockMode = isMockApiMode
const isQutcraftPortal = organizationSlug === 'qutcraft'
const { organization, loadPortalOrganization } = usePortalIdentity()
const organizationName = computed(() => organization.value?.name ?? (isQutcraftPortal ? 'QUTCraft Commons' : 'Campus Commons'))
const formRef = ref<FormInstance>()
const submitting = ref(false)

const form = reactive({
  email: mockMode ? 'admin@qutcraft.local' : '',
  password: mockMode ? 'demo-admin-pass' : '',
})

const rules: FormRules = {
  email: [{ required: true, type: 'email', message: '请输入有效邮箱账号', trigger: 'blur' }],
  password: [{ required: true, min: 6, message: '请输入正确的管理密码', trigger: 'blur' }],
}

async function submit() {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  submitting.value = true
  try {
    await signIn(form.email, form.password)
    ElMessage.success('登录成功，欢迎回到管理工作台！')
    await router.replace(typeof route.query.redirect === 'string' ? route.query.redirect : '/admin')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '登录失败，请检查凭证后再试。')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  void loadPortalOrganization().catch(() => undefined)
})
</script>

<template>
  <div class="login-wrapper">
    <div class="login-container">
      <!-- Left Visual Hero Banner -->
      <div class="login-hero">
        <div class="hero-brand-mark">Q</div>
        <div class="hero-badge">
          <i class="status-dot-online" /> {{ isQutcraftPortal ? 'QUTCraft Network' : 'Organization Commons' }}
        </div>
        <h1 class="hero-title">{{ organizationName }}</h1>
        <p class="hero-desc">
          {{ isQutcraftPortal ? '青岛理工大学 Minecraft 官方合作社团与学术交流平台管理工作台。' : '面向校园社团与民间组织的内容、项目、知识与活动运营工作台。' }}
        </p>

        <div class="hero-features">
          <span>✦ {{ isQutcraftPortal ? '白名单审批' : '活动策划' }}</span>
          <span>✦ 知识库沉淀</span>
          <span>✦ {{ isQutcraftPortal ? '资源协作' : '项目协作' }}</span>
        </div>
      </div>

      <!-- Right Login Form Card -->
      <div class="login-form-panel">
        <RouterLink to="/" class="login-back-link">
          <el-icon><ArrowLeft /></el-icon> 返回公开门户
        </RouterLink>

        <div class="login-header">
          <h2>登录管理工作台</h2>
          <p>欢迎回到 {{ organizationName }} 协作与管理平台，请验证您的成员身份。</p>
        </div>

        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          label-position="top"
          size="large"
          class="login-form"
          @submit.prevent="submit"
        >
          <el-form-item label="管理邮箱" prop="email">
            <el-input
              v-model="form.email"
              :prefix-icon="User"
              autocomplete="email"
              :placeholder="isQutcraftPortal ? 'name@qutcraft.local' : 'name@example.com'"
            />
          </el-form-item>

          <el-form-item label="密码" prop="password">
            <el-input
              v-model="form.password"
              :prefix-icon="Key"
              type="password"
              show-password
              autocomplete="current-password"
              placeholder="输入您的管理密码"
            />
          </el-form-item>

          <el-button
            type="primary"
            native-type="submit"
            :loading="submitting"
            size="large"
            round
            class="submit-login-btn"
          >
            登录工作台 →
          </el-button>
        </el-form>
      </div>
    </div>
  </div>
</template>
