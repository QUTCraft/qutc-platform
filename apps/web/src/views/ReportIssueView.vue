<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ArrowLeft, CircleCheck } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { useRoute } from 'vue-router'
import { portalApi } from '@/api/portal'
import type { SiteFeedbackPayload } from '@/api/types'
import { session } from '@/stores/session'

const route = useRoute()
const formRef = ref<FormInstance>()
const submitting = ref(false)
const submitted = ref(false)

const initialPageURL = typeof route.query.from === 'string' && route.query.from.startsWith('/') && !route.query.from.startsWith('//')
  ? route.query.from
  : ''

const form = reactive<SiteFeedbackPayload>({
  kind: 'bug',
  title: '',
  description: '',
  contact_name: session.user?.display_name ?? '',
  contact_email: session.user?.email ?? '',
  page_url: initialPageURL === '/report' ? '' : initialPageURL,
})

const rules: FormRules = {
  kind: [{ required: true, message: '请选择报告类型', trigger: 'change' }],
  title: [{ required: true, message: '请填写简短标题', trigger: 'blur' }],
  description: [{ required: true, message: '请描述你遇到的问题或期望的功能', trigger: 'blur' }],
  contact_name: [{ required: true, message: '请填写称呼', trigger: 'blur' }],
  contact_email: [{ required: true, type: 'email', message: '请填写有效联系邮箱', trigger: 'blur' }],
}

const kindLabel = computed(() => form.kind === 'feature' ? '功能建议' : '网页缺陷')

async function submit() {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  submitting.value = true
  try {
    await portalApi.submitFeedback({
      ...form,
      title: form.title.trim(),
      description: form.description.trim(),
      contact_name: form.contact_name.trim(),
      contact_email: form.contact_email.trim(),
      page_url: form.page_url?.trim() || undefined,
    })
    submitted.value = true
    ElMessage.success('报告已提交，我们会尽快查看。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '报告提交失败，请稍后重试。')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="page-intro">
    <div class="eyebrow">SITE FEEDBACK</div>
    <h1>报告问题</h1>
    <p>发现网页缺陷，或希望平台增加新功能，都可以在这里告诉我们。提交后会邮件提醒管理员，和入服申请使用同一套通知通道。</p>
  </section>

  <article v-if="submitted" class="surface-panel report-success">
    <el-icon class="report-success-icon"><CircleCheck /></el-icon>
    <h2>已经收到你的{{ kindLabel }}</h2>
    <p>管理员会通过邮件收到提醒。如需补充细节，可再次提交一份新的报告。</p>
    <div class="report-success-actions">
      <RouterLink to="/"><el-button round>返回门户</el-button></RouterLink>
      <el-button type="primary" round @click="submitted = false">继续提交</el-button>
    </div>
  </article>

  <article v-else class="surface-panel report-form-card">
    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="submit">
      <el-form-item label="报告类型" prop="kind">
        <el-radio-group v-model="form.kind" aria-label="报告类型">
          <el-radio-button value="bug">网页缺陷</el-radio-button>
          <el-radio-button value="feature">功能建议</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="标题" prop="title">
        <el-input v-model="form.title" maxlength="120" show-word-limit placeholder="一句话说明问题或需求" />
      </el-form-item>
      <el-form-item label="详细描述" prop="description">
        <el-input v-model="form.description" type="textarea" :rows="6" maxlength="2000" show-word-limit placeholder="复现步骤、期望结果，或你希望增加的功能说明" />
      </el-form-item>
      <div class="report-form-grid">
        <el-form-item label="称呼" prop="contact_name">
          <el-input v-model="form.contact_name" maxlength="80" placeholder="方便我们称呼你" />
        </el-form-item>
        <el-form-item label="联系邮箱" prop="contact_email">
          <el-input v-model="form.contact_email" maxlength="254" placeholder="用于必要时回访" />
        </el-form-item>
      </div>
      <el-form-item label="相关页面（可选）" prop="page_url">
        <el-input v-model="form.page_url" maxlength="500" placeholder="/posts 或你发现问题的路径" />
      </el-form-item>
      <div class="report-form-actions">
        <RouterLink to="/" class="report-back-link">
          <el-icon><ArrowLeft /></el-icon>
          返回门户
        </RouterLink>
        <el-button type="primary" round native-type="submit" :loading="submitting">提交报告</el-button>
      </div>
    </el-form>
  </article>
</template>
