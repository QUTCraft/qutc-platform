<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminUser } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(adminApi.getUsers)
const roleLabel = { owner: '所有者', administrator: '管理员', editor: '编辑者', member: '成员' }
const stateLabel = { active: '正常', invited: '待加入', disabled: '已停用' }
const dialogOpen = ref(false)
const saving = ref(false)
const editingUser = ref<AdminUser | null>(null)
const form = reactive({ role: 'member' as AdminUser['role'], state: 'active' as AdminUser['state'] })

function openEditor(user: AdminUser) {
  editingUser.value = user
  Object.assign(form, { role: user.role, state: user.state })
  dialogOpen.value = true
}

async function saveUser() {
  if (!editingUser.value) return
  saving.value = true
  try {
    await adminApi.updateUser(editingUser.value.id, form)
    ElMessage.success('成员信息已更新。')
    dialogOpen.value = false
    await refresh()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '成员信息保存失败。')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>成员与权限</h2>
          <p>管理组织成员及其在工作台中的协作权限与分配状态。</p>
        </div>
        <el-button round type="primary">+ 邀请成员</el-button>
      </section>

      <section class="admin-panel">
        <el-table :data="data.items" class="admin-table">
          <el-table-column label="成员" min-width="220">
            <template #default="scope">
              <div class="person-cell">
                <span class="avatar-bubble">{{ scope.row.name.slice(0, 1) }}</span>
                <div>
                  <strong>{{ scope.row.name }}</strong>
                  <small>{{ scope.row.email }}</small>
                </div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="角色" width="140">
            <template #default="scope">
              <el-tag effect="plain">{{ roleLabel[scope.row.role as keyof typeof roleLabel] }}</el-tag>
            </template>
          </el-table-column>

          <el-table-column label="状态" width="120">
            <template #default="scope">
              <el-tag :type="scope.row.state === 'active' ? 'success' : scope.row.state === 'invited' ? 'warning' : 'info'">
                {{ stateLabel[scope.row.state as keyof typeof stateLabel] }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column label="加入时间" width="150">
            <template #default="scope">{{ formatDate(scope.row.joined_at) }}</template>
          </el-table-column>

          <el-table-column width="110" align="right">
            <template #default="scope">
              <el-button text type="primary" size="small" @click="openEditor(scope.row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </section>

      <el-dialog v-model="dialogOpen" title="编辑成员" width="min(92vw, 440px)">
        <p v-if="editingUser">{{ editingUser.name }} · {{ editingUser.email }}</p>
        <el-form label-position="top">
          <el-form-item label="角色">
            <el-select v-model="form.role" style="width: 100%">
              <el-option label="成员" value="member" />
              <el-option label="编辑者" value="editor" />
              <el-option label="管理员" value="administrator" />
              <el-option label="所有者" value="owner" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="form.state" style="width: 100%">
              <el-option label="正常" value="active" />
              <el-option label="待加入" value="invited" />
              <el-option label="已停用" value="disabled" />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="dialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="saveUser">保存</el-button>
        </template>
      </el-dialog>
    </template>
  </AsyncState>
</template>
