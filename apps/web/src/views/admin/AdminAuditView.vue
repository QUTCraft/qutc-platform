<script setup lang="ts">
import { reactive, ref } from 'vue'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import { useAsyncData } from '@/composables/useAsyncData'

const page = ref(1)
const dateRange = ref<[string, string] | null>(null)
const filters = reactive({
  action: '',
  target_type: '',
  result: '',
  request_id: '',
})

const { data, error, loading, refresh } = useAsyncData(() => adminApi.getAuditEvents({
  page: page.value,
  page_size: 20,
  ...filters,
  date_from: dateRange.value?.[0],
  date_to: dateRange.value?.[1],
}))

async function search() {
  page.value = 1
  await refresh()
}

async function reset() {
  Object.assign(filters, { action: '', target_type: '', result: '', request_id: '' })
  dateRange.value = null
  await search()
}

async function changePage(value: number) {
  page.value = value
  await refresh()
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(new Date(value))
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>审计记录</h2>
          <p>按组织隔离查看管理操作，并通过 Request ID 关联 API 访问日志。</p>
        </div>
      </section>

      <section class="admin-panel">
        <form class="audit-filters" @submit.prevent="search">
          <el-input v-model="filters.action" clearable placeholder="操作，例如 content.published" />
          <el-input v-model="filters.target_type" clearable placeholder="对象类型，例如 content" />
          <el-select v-model="filters.result" clearable placeholder="执行结果">
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="已接受" value="accepted" />
          </el-select>
          <el-input v-model="filters.request_id" clearable placeholder="Request ID" />
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            value-format="YYYY-MM-DD"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
          />
          <div class="audit-filter-actions">
            <el-button native-type="button" round @click="reset">重置</el-button>
            <el-button native-type="submit" type="primary" round>查询</el-button>
          </div>
        </form>

        <el-table :data="data.items" class="admin-table" empty-text="当前筛选条件下没有审计记录">
          <el-table-column label="时间" width="180">
            <template #default="scope">{{ formatDateTime(scope.row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作者" min-width="150">
            <template #default="scope">
              <div class="audit-actor">
                <strong>{{ scope.row.actor_name || '未知用户' }}</strong>
                <small>{{ scope.row.actor_user_id }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="action" label="操作" min-width="190" />
          <el-table-column label="对象" min-width="170">
            <template #default="scope">
              <div class="audit-target">
                <el-tag effect="plain">{{ scope.row.target_type }}</el-tag>
                <small v-if="scope.row.target_id">{{ scope.row.target_id }}</small>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="结果" width="100">
            <template #default="scope">
              <el-tag :type="scope.row.result === 'success' ? 'success' : scope.row.result === 'failed' ? 'danger' : 'info'">
                {{ scope.row.result }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="Request ID" min-width="220">
            <template #default="scope">
              <span class="audit-request-id" :title="scope.row.request_id">{{ scope.row.request_id }}</span>
            </template>
          </el-table-column>
        </el-table>

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
    </template>
  </AsyncState>
</template>
