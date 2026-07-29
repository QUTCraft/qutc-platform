<script setup lang="ts">
import { computed, reactive } from 'vue'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminAuditEvent, AdminAuditFilters } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'

const filters = reactive<Required<Pick<AdminAuditFilters, 'page' | 'page_size'>> & AdminAuditFilters>({
  page: 1,
  page_size: 20,
  action: '',
  target_type: '',
  result: '',
  request_id: '',
  actor_user_id: '',
})

const { data, error, loading, refresh } = useAsyncData(async () => adminApi.getAuditEvents(filters))
const events = computed(() => data.value?.items ?? [])

function applyFilters() {
  filters.page = 1
  refresh()
}

function resetFilters() {
  Object.assign(filters, { page: 1, page_size: 20, action: '', target_type: '', result: '', request_id: '', actor_user_id: '' })
  refresh()
}

function changePage(page: number) {
  filters.page = page
  refresh()
}

function resultTagType(result: AdminAuditEvent['result']): 'success' | 'danger' | 'primary' {
  return result === 'success' ? 'success' : result === 'failed' ? 'danger' : 'primary'
}

function resultLabel(result: AdminAuditEvent['result']) {
  return result === 'success' ? '成功' : result === 'failed' ? '失败' : '已受理'
}

function targetTypeLabel(targetType: string) {
  const map: Record<string, string> = {
    content: '内容',
    application: '申请',
    invitation: '邀请',
    membership: '成员',
    portal_configuration: '门户配置',
    server: '服务器',
  }
  return map[targetType] ?? targetType
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit',
  }).format(new Date(value))
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>审计记录</h2>
          <p>查看组织范围内的关键写操作审计事件，可按操作、目标和 request_id 定位问题。</p>
        </div>
      </section>

      <article class="admin-panel">
        <div class="panel-heading">
          <div>
            <h2>审计事件</h2>
          </div>
          <el-tag>{{ data.total }} 条记录</el-tag>
        </div>

        <div class="application-filters" aria-label="审计筛选">
          <el-input
            v-model="filters.request_id"
            clearable
            placeholder="按 request_id 定位"
            @keyup.enter="applyFilters"
          />
          <el-select v-model="filters.target_type" aria-label="目标类型" placeholder="全部目标类型">
            <el-option label="全部目标类型" value="" />
            <el-option label="内容" value="content" />
            <el-option label="申请" value="application" />
            <el-option label="邀请" value="invitation" />
            <el-option label="成员" value="membership" />
            <el-option label="门户配置" value="portal_configuration" />
            <el-option label="服务器" value="server" />
          </el-select>
          <el-select v-model="filters.result" aria-label="结果" placeholder="全部结果">
            <el-option label="全部结果" value="" />
            <el-option label="成功" value="success" />
            <el-option label="已受理" value="accepted" />
            <el-option label="失败" value="failed" />
          </el-select>
          <el-input
            v-model="filters.action"
            clearable
            placeholder="操作类型，如 content.publish"
            @keyup.enter="applyFilters"
          />
          <el-input
            v-model="filters.actor_user_id"
            clearable
            placeholder="操作者用户 ID"
            @keyup.enter="applyFilters"
          />
          <div class="application-filter-actions">
            <el-button type="primary" @click="applyFilters">筛选</el-button>
            <el-button @click="resetFilters">重置</el-button>
          </div>
        </div>

        <el-table :data="events" class="audit-table" empty-text="没有符合筛选条件的审计事件">
          <el-table-column label="时间" width="180">
            <template #default="{ row }">
              <span class="audit-time">{{ formatDateTime(row.created_at) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作者" width="140" prop="actor_name" />
          <el-table-column label="操作" min-width="180">
            <template #default="{ row }">
              <div class="audit-action">
                <code>{{ row.action }}</code>
                <span class="audit-target">{{ targetTypeLabel(row.target_type) }}<template v-if="row.target_id"> · {{ row.target_id }}</template></span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="结果" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="resultTagType(row.result)">{{ resultLabel(row.result) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Request ID" min-width="180">
            <template #default="{ row }">
              <code class="audit-request-id">{{ row.request_id }}</code>
            </template>
          </el-table-column>
        </el-table>

        <el-pagination
          v-if="data.total > data.page_size"
          class="application-pagination"
          background
          layout="prev, pager, next"
          :current-page="data.page"
          :page-size="data.page_size"
          :total="data.total"
          @current-change="changePage"
        />
      </article>
    </template>
  </AsyncState>
</template>

<style scoped>
.audit-table {
  width: 100%;
}

.audit-time {
  font-variant-numeric: tabular-nums;
  color: var(--el-text-color-secondary);
}

.audit-action {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.audit-action code {
  font-size: 13px;
}

.audit-target {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.audit-request-id {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}

@media (max-width: 430px) {
  .audit-action {
    align-items: flex-start;
  }
}
</style>
