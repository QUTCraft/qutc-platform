<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminProject } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(adminApi.getProjects)
const dialogOpen = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const form = reactive({ title: '', summary: '', status: 'research' as AdminProject['status'], tags: '', is_public: true })
const statusLabel: Record<AdminProject['status'], string> = { active: '进行中', research: '研究中', completed: '已完成' }

function resetForm() {
  editingId.value = null
  Object.assign(form, { title: '', summary: '', status: 'research', tags: '', is_public: true })
  dialogOpen.value = true
}

function editProject(item: AdminProject) {
  editingId.value = item.id
  Object.assign(form, { title: item.title, summary: item.summary, status: item.status, tags: item.tags.join(', '), is_public: item.is_public })
  dialogOpen.value = true
}

async function submit() {
  submitting.value = true
  try {
    const payload = { ...form, tags: form.tags.split(',').map((tag) => tag.trim()).filter(Boolean) }
    if (editingId.value) await adminApi.updateProject(editingId.value, payload)
    else await adminApi.createProject(payload)
    ElMessage.success(editingId.value ? '项目已保存。' : '项目已创建。')
    dialogOpen.value = false
    await refresh()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '项目保存失败。')
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
          <h2>项目管理</h2>
          <p>维护项目负责人、状态、里程碑入口和门户公开范围。</p>
        </div>
        <el-button type="primary" round @click="resetForm">+ 新建项目</el-button>
      </section>

      <section class="admin-panel">
        <el-table :data="data.items" class="admin-table">
          <el-table-column prop="title" label="项目" min-width="240" />
          <el-table-column label="状态" width="120">
            <template #default="scope"><el-tag>{{ statusLabel[scope.row.status as AdminProject['status']] }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="owner" label="负责人" width="140" />
          <el-table-column prop="member_count" label="成员" width="90" />
          <el-table-column prop="milestone_count" label="里程碑" width="100" />
          <el-table-column label="公开" width="100">
            <template #default="scope"><el-tag :type="scope.row.is_public ? 'success' : 'info'">{{ scope.row.is_public ? '已公开' : '私有' }}</el-tag></template>
          </el-table-column>
          <el-table-column label="更新时间" width="150">
            <template #default="scope">{{ formatDate(scope.row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="scope"><el-button text type="primary" @click="editProject(scope.row)">编辑</el-button></template>
          </el-table-column>
        </el-table>
      </section>

      <el-dialog v-model="dialogOpen" :title="editingId ? '编辑项目' : '新建项目'" width="min(92vw, 560px)">
        <el-form :model="form" label-position="top">
          <el-form-item label="项目名称"><el-input v-model="form.title" maxlength="160" /></el-form-item>
          <el-form-item label="项目简介"><el-input v-model="form.summary" type="textarea" :rows="3" maxlength="500" show-word-limit /></el-form-item>
          <div class="form-grid">
            <el-form-item label="项目状态">
              <el-select v-model="form.status" style="width: 100%">
                <el-option label="进行中" value="active" />
                <el-option label="研究中" value="research" />
                <el-option label="已完成" value="completed" />
              </el-select>
            </el-form-item>
            <el-form-item label="标签"><el-input v-model="form.tags" placeholder="用逗号分隔" /></el-form-item>
          </div>
          <el-form-item><el-switch v-model="form.is_public" active-text="在门户公开展示" /></el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="dialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submit">保存项目</el-button>
        </template>
      </el-dialog>
    </template>
  </AsyncState>
</template>
