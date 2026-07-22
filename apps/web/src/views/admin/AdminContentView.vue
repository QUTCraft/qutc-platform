<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(adminApi.getContent)
const dialogOpen = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()

const form = reactive({ title: '', type: 'news' as 'news' | 'resource' | 'knowledge' })
const rules: FormRules = { title: [{ required: true, message: '请填写内容标题', trigger: 'blur' }] }

async function submit() {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  submitting.value = true
  try {
    await adminApi.createContent(form)
    ElMessage.success('已成功创建草稿！')
    dialogOpen.value = false
    form.title = ''
    refresh()
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>内容工作区</h2>
          <p>在此统一管理与编辑公开门户呈现的动态、资源及知识库条目。</p>
        </div>
        <el-button type="primary" round @click="dialogOpen = true">+ 新建内容</el-button>
      </section>

      <section class="admin-panel">
        <el-table :data="data.items" class="admin-table">
          <el-table-column prop="title" label="标题" min-width="260" />
          <el-table-column prop="type" label="类型" width="120">
            <template #default="scope">
              <el-tag effect="plain">{{ scope.row.type }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="author" label="负责人" width="130" />
          <el-table-column label="状态" width="120">
            <template #default="scope">
              <el-tag :type="scope.row.status === 'published' ? 'success' : scope.row.status === 'review' ? 'warning' : 'info'">
                {{ scope.row.status === 'published' ? '已发布' : scope.row.status === 'review' ? '待审核' : '草稿' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="最后修改" width="150">
            <template #default="scope">{{ formatDate(scope.row.updated_at) }}</template>
          </el-table-column>
          <el-table-column width="100">
            <template #default>
              <el-button text type="primary">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </section>

      <el-dialog v-model="dialogOpen" title="新建内容草稿" width="min(92vw, 480px)">
        <el-form ref="formRef" :model="form" :rules="rules" label-position="top">
          <el-form-item label="标题" prop="title">
            <el-input v-model="form.title" placeholder="例如：暑期建筑活动报名" />
          </el-form-item>
          <el-form-item label="内容类型">
            <el-radio-group v-model="form.type">
              <el-radio-button value="news">动态</el-radio-button>
              <el-radio-button value="resource">资源</el-radio-button>
              <el-radio-button value="knowledge">知识库</el-radio-button>
            </el-radio-group>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="dialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submit">创建草稿</el-button>
        </template>
      </el-dialog>
    </template>
  </AsyncState>
</template>
