<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminContent } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(adminApi.getContent)
const dialogOpen = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const editingId = ref<string | null>(null)

const form = reactive({ title: '', type: 'news' as 'news' | 'resource' | 'knowledge', category: '', excerpt: '', body: '' })
const rules: FormRules = { title: [{ required: true, message: '请填写内容标题', trigger: 'blur' }] }

function resetForm() { editingId.value = null; Object.assign(form, { title: '', type: 'news', category: '', excerpt: '', body: '' }); dialogOpen.value = true }
function editContent(item: AdminContent) { editingId.value = item.id; Object.assign(form, { title: item.title, type: item.type, category: item.category ?? '', excerpt: item.excerpt ?? '', body: item.body ?? '' }); dialogOpen.value = true }
async function changeStatus(id: string, action: 'publish' | 'archive') { try { if (action === 'publish') await adminApi.publishContent(id); else await adminApi.archiveContent(id); ElMessage.success(action === 'publish' ? '内容已发布到门户。' : '内容已下线。'); refresh() } catch (error) { ElMessage.error(error instanceof Error ? error.message : '操作失败。') } }

async function submit() {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  submitting.value = true
  try {
    if (editingId.value) { await adminApi.updateContent(editingId.value, form); ElMessage.success('内容草稿已保存。') }
    else { await adminApi.createContent(form); ElMessage.success('已成功创建草稿！') }
    dialogOpen.value = false
    Object.assign(form, { title: '', type: 'news', category: '', excerpt: '', body: '' }); editingId.value = null
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
        <el-button type="primary" round @click="resetForm">+ 新建内容</el-button>
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
              <el-tag :type="scope.row.status === 'published' ? 'success' : scope.row.status === 'review' ? 'warning' : scope.row.status === 'archived' ? 'danger' : 'info'">
                {{ scope.row.status === 'published' ? '已发布' : scope.row.status === 'review' ? '待审核' : scope.row.status === 'archived' ? '已下线' : '草稿' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="最后修改" width="150">
            <template #default="scope">{{ formatDate(scope.row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="scope">
              <el-button text type="primary" @click="editContent(scope.row)">编辑</el-button>
              <el-button v-if="scope.row.status !== 'published'" text type="success" @click="changeStatus(scope.row.id, 'publish')">发布</el-button>
              <el-button v-else text type="danger" @click="changeStatus(scope.row.id, 'archive')">下线</el-button>
            </template>
          </el-table-column>
        </el-table>
      </section>

      <el-dialog v-model="dialogOpen" :title="editingId ? '编辑内容草稿' : '新建内容草稿'" width="min(92vw, 560px)">
        <el-form ref="formRef" :model="form" :rules="rules" label-position="top">
          <el-form-item label="标题" prop="title">
            <el-input v-model="form.title" placeholder="例如：暑期建筑活动报名" />
          </el-form-item>
          <el-form-item label="内容类型">
            <el-radio-group v-model="form.type" class="content-type-selector">
              <el-radio-button value="news">动态</el-radio-button>
              <el-radio-button value="resource">资源</el-radio-button>
              <el-radio-button value="knowledge">知识库</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="分类 / 目录"><el-input v-model="form.category" maxlength="64" placeholder="例如：公告、项目协作、开发规范" /></el-form-item>
          <el-form-item label="门户摘要"><el-input v-model="form.excerpt" type="textarea" :rows="3" maxlength="500" show-word-limit placeholder="发布后显示在门户卡片中" /></el-form-item>
          <el-form-item label="正文"><el-input v-model="form.body" type="textarea" :rows="6" placeholder="开发阶段支持纯文本正文" /></el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="dialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submit">{{ editingId ? '保存草稿' : '创建草稿' }}</el-button>
        </template>
      </el-dialog>
    </template>
  </AsyncState>
</template>
