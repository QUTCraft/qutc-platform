<script setup lang="ts">
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(adminApi.getUsers)
const roleLabel = { owner: '所有者', administrator: '管理员', editor: '编辑者', member: '成员' }
const stateLabel = { active: '正常', invited: '待加入', disabled: '已停用' }
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
            <template #default>
              <el-button text type="primary" size="small">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </section>
    </template>
  </AsyncState>
</template>
