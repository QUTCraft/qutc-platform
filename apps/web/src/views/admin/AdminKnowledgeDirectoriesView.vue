<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminKnowledgeDirectory } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const page = ref(1)
const { data, error, loading, refresh } = useAsyncData(() => adminApi.getKnowledgeDirectories({ page: page.value }))
const dialogOpen = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const formRef = ref<FormInstance>()
const form = reactive<Omit<AdminKnowledgeDirectory, 'id' | 'updated_at'>>({ parent_id: '', name: '', slug: '', description: '', sort_order: 10, is_public: true })
const rules: FormRules = {
  name: [{ required: true, message: '请填写目录名称', trigger: 'blur' }],
  slug: [{ required: true, message: '请填写目录标识', trigger: 'blur' }],
}

async function changePage(value: number) {
  page.value = value
  await refresh()
}

function resetForm() {
  editingId.value = null
  Object.assign(form, { parent_id: '', name: '', slug: '', description: '', sort_order: ((data.value?.items.length ?? 0) + 1) * 10, is_public: true })
  dialogOpen.value = true
}

function editDirectory(item: AdminKnowledgeDirectory) {
  editingId.value = item.id
  Object.assign(form, { parent_id: item.parent_id, name: item.name, slug: item.slug, description: item.description, sort_order: item.sort_order, is_public: item.is_public })
  dialogOpen.value = true
}

async function submit() {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  submitting.value = true
  try {
    if (editingId.value) await adminApi.updateKnowledgeDirectory(editingId.value, form)
    else await adminApi.createKnowledgeDirectory(form)
    ElMessage.success(editingId.value ? '知识库目录已更新。' : '知识库目录已创建。')
    dialogOpen.value = false
    await refresh()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '目录保存失败。')
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
          <h2>知识库目录</h2>
          <p>维护公开文章的分类目录；门户只读取已公开目录与已发布文章。</p>
        </div>
        <el-button type="primary" round @click="resetForm">+ 新建目录</el-button>
      </section>

      <section class="admin-panel">
        <el-table :data="data.items" class="admin-table">
          <el-table-column prop="name" label="目录名称" min-width="180" />
          <el-table-column prop="slug" label="标识" width="160" />
          <el-table-column prop="description" label="说明" min-width="260" />
          <el-table-column label="门户状态" width="120">
            <template #default="scope">
              <el-tag :type="scope.row.is_public ? 'success' : 'info'">{{ scope.row.is_public ? '公开' : '内部' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="sort_order" label="排序" width="90" />
          <el-table-column label="更新时间" width="150">
            <template #default="scope">{{ formatDate(scope.row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="scope"><el-button text type="primary" @click="editDirectory(scope.row)">编辑</el-button></template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!data.items.length" description="暂无知识库目录" />
        <el-pagination
          v-if="data.total > data.page_size"
          class="application-pagination"
          background
          layout="total, prev, pager, next"
          :current-page="data.page"
          :page-size="data.page_size"
          :total="data.total"
          @current-change="changePage"
        />
      </section>

      <el-dialog v-model="dialogOpen" :title="editingId ? '编辑知识库目录' : '新建知识库目录'" width="min(92vw, 560px)">
        <el-form ref="formRef" :model="form" :rules="rules" label-position="top">
          <div class="form-grid">
            <el-form-item label="目录名称" prop="name"><el-input v-model="form.name" placeholder="例如：技术规范" /></el-form-item>
            <el-form-item label="目录标识" prop="slug"><el-input v-model="form.slug" placeholder="例如：technology" /></el-form-item>
          </div>
          <el-form-item label="目录说明"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="500" /></el-form-item>
          <div class="form-grid">
            <el-form-item label="父目录标识"><el-input v-model="form.parent_id" placeholder="暂留空表示一级目录" /></el-form-item>
            <el-form-item label="排序"><el-input-number v-model="form.sort_order" :min="0" :max="9999" /></el-form-item>
          </div>
          <el-form-item><el-switch v-model="form.is_public" active-text="在门户公开显示" /></el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="dialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submit">保存目录</el-button>
        </template>
      </el-dialog>
    </template>
  </AsyncState>
</template>
