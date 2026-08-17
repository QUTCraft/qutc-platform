<script setup lang="ts">
import { computed, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminApplication, AdminApplicationFilters } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const filters = reactive<Required<Pick<AdminApplicationFilters, 'page' | 'page_size'>> & AdminApplicationFilters>({
  page: 1,
  page_size: 10,
  status: '',
  type: '',
  query: '',
})

const { data, error, loading, refresh } = useAsyncData(async () => {
  const [applications, organization] = await Promise.all([
    adminApi.getApplications(filters),
    adminApi.getOrganization(),
  ])
  return { applications, organization }
})

const applications = computed(() => data.value?.applications.items ?? [])
const pendingCount = computed(() => applications.value.filter((item) => item.status === 'pending').length)
const decidedCount = computed(() => applications.value.length - pendingCount.value)

function applyFilters() {
  filters.page = 1
  refresh()
}

function resetFilters() {
  Object.assign(filters, { page: 1, page_size: 10, status: '', type: '', query: '' })
  refresh()
}

function changePage(page: number) {
  filters.page = page
  refresh()
}

function applicationStatusLabel(status: AdminApplication['status']) {
  return status === 'pending' ? '待处理' : status === 'approved' ? '已通过' : '已拒绝'
}

async function requestDecision(id: string, decision: 'approve' | 'reject') {
  let reason = ''
  try {
    const result = await ElMessageBox.prompt(
      decision === 'approve' ? '可以填写通过备注，留空也可继续。' : '请填写拒绝原因，该原因仅在管理后台显示。',
      decision === 'approve' ? '通过申请' : '拒绝申请',
      {
        confirmButtonText: decision === 'approve' ? '确认通过' : '确认拒绝',
        cancelButtonText: '取消',
        inputType: 'textarea',
        inputPlaceholder: decision === 'approve' ? '例如：资料完整，符合加入要求' : '请输入拒绝原因',
        inputValidator: (value) => {
          const normalized = value.trim()
          if (decision === 'reject' && !normalized) return '拒绝申请时必须填写原因。'
          if ([...normalized].length > 500) return '审核原因不能超过 500 个字符。'
          return true
        },
      },
    )
    reason = result.value.trim()
  } catch {
    return
  }

  try {
    await (decision === 'approve' ? adminApi.approveApplication(id, reason) : adminApi.rejectApplication(id, reason))
    ElMessage.success(decision === 'approve' ? '申请已通过并写入审计记录。' : '申请已拒绝并写入审计记录。')
    refresh()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '申请暂时无法处理。')
  }
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>申请审核</h2>
          <p>筛选并处理当前组织的加入申请。审核只更新平台内申请状态，并完整保留操作人与审核备注。</p>
        </div>
      </section>

      <section class="admin-two-column review-workspace">
        <article class="admin-panel">
          <div class="panel-heading">
            <div><h2>加入申请</h2></div>
            <el-tag>{{ data.applications.total }} 条结果</el-tag>
          </div>

          <div class="application-filters" aria-label="申请筛选">
            <el-input v-model="filters.query" clearable placeholder="搜索姓名、游戏 ID、邮箱或 QQ" @keyup.enter="applyFilters" />
            <el-select v-model="filters.status" aria-label="审批状态" placeholder="全部审批状态">
              <el-option label="全部审批状态" value="" />
              <el-option label="待处理" value="pending" />
              <el-option label="已通过" value="approved" />
              <el-option label="已拒绝" value="rejected" />
            </el-select>
            <el-select v-model="filters.type" aria-label="申请类型" placeholder="全部申请类型">
              <el-option label="全部申请类型" value="" />
              <el-option label="服务器加入申请" value="whitelist" />
              <el-option label="组织成员申请" value="membership" />
            </el-select>
            <div class="application-filter-actions">
              <el-button type="primary" @click="applyFilters">筛选</el-button>
              <el-button @click="resetFilters">重置</el-button>
            </div>
          </div>

          <div class="application-list">
            <article v-for="item in applications" :key="item.id" class="app-item-card">
              <div class="app-item-info">
                <div class="application-title-row">
                  <strong>{{ item.applicant }}</strong>
                  <el-tag size="small" :type="item.status === 'pending' ? 'warning' : item.status === 'approved' ? 'success' : 'info'">
                    {{ applicationStatusLabel(item.status) }}
                  </el-tag>
                </div>
                <small>{{ item.type === 'whitelist' ? '服务器加入申请' : '组织成员申请' }} · 提交于 {{ formatDate(item.submitted_at) }}</small>
                <div class="application-identifiers">
                  <span v-if="item.game_id">游戏 ID：{{ item.game_id }}</span>
                  <span v-if="item.class_name">班级：{{ item.class_name }}</span>
                  <span v-if="item.qq_number">QQ：{{ item.qq_number }}</span>
                  <span v-if="item.email">邮箱：{{ item.email }}</span>
                </div>
                <p>{{ item.note }}</p>
                <div v-if="item.decision_reason" class="application-decision-reason">
                  <strong>审核备注</strong>
                  <span>{{ item.decision_reason }}</span>
                </div>
              </div>
              <div v-if="item.status === 'pending'" class="app-item-actions">
                <el-button @click="requestDecision(item.id, 'reject')">拒绝</el-button>
                <el-button type="primary" @click="requestDecision(item.id, 'approve')">通过</el-button>
              </div>
            </article>

            <el-empty v-if="!applications.length" description="没有符合当前筛选条件的申请" />
          </div>

          <el-pagination
            v-if="data.applications.total > data.applications.page_size"
            class="application-pagination"
            background
            layout="prev, pager, next"
            :current-page="data.applications.page"
            :page-size="data.applications.page_size"
            :total="data.applications.total"
            @current-change="changePage"
          />
        </article>

        <article class="admin-panel review-summary-panel">
          <div class="panel-heading">
            <div><h2>审核边界</h2></div>
            <span class="review-state online"><i /> 人工审批</span>
          </div>
          <dl class="review-facts">
            <div><dt>当前组织</dt><dd>{{ data.organization.name }}</dd></div>
            <div><dt>当前页待处理</dt><dd>{{ pendingCount }} 条</dd></div>
            <div><dt>当前页已处理</dt><dd>{{ decidedCount }} 条</dd></div>
          </dl>
          <p>通过或拒绝只改变 CMS 内的申请状态并写入通知队列；通知投递失败不会回滚审核决定。</p>
        </article>
      </section>
    </template>
  </AsyncState>
</template>
