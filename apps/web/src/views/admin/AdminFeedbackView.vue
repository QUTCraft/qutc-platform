<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminFeedback, AdminFeedbackFilters } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { hasPermission } from '@/stores/session'

const filters = reactive<Required<Pick<AdminFeedbackFilters, 'page' | 'page_size'>> & AdminFeedbackFilters>({
  page: 1,
  page_size: 10,
  status: '',
  kind: '',
  query: '',
})

const canUpdate = hasPermission('application:approve')
const updatingId = ref('')

const { data, error, loading, refresh } = useAsyncData(async () => {
  const [feedback, organization] = await Promise.all([
    adminApi.getFeedback(filters),
    adminApi.getOrganization(),
  ])
  return { feedback, organization }
})

const items = computed(() => data.value?.feedback.items ?? [])
const openCount = computed(() => items.value.filter((item) => item.status === 'open').length)
const resolvedCount = computed(() => items.value.length - openCount.value)

function applyFilters() {
  filters.page = 1
  refresh()
}

function resetFilters() {
  Object.assign(filters, { page: 1, page_size: 10, status: '', kind: '', query: '' })
  refresh()
}

function changePage(page: number) {
  filters.page = page
  refresh()
}

function kindLabel(kind: AdminFeedback['kind']) {
  return kind === 'feature' ? '功能建议' : '网页缺陷'
}

function statusLabel(status: AdminFeedback['status']) {
  return status === 'resolved' ? '已处理' : '待处理'
}

function formatSubmittedAt(value: string) {
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(new Date(value))
}

async function updateStatus(item: AdminFeedback, status: AdminFeedback['status']) {
  updatingId.value = item.id
  try {
    await adminApi.updateFeedback(item.id, { status })
    ElMessage.success(status === 'resolved' ? '报告已标记为已处理。' : '报告已重新打开。')
    await refresh()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '问题报告暂时无法更新。')
  } finally {
    updatingId.value = ''
  }
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>问题报告</h2>
          <p>查看门户和后台提交的网页缺陷、功能建议全文。邮件只作提醒，完整内容保存在这里。</p>
        </div>
      </section>

      <section class="admin-two-column review-workspace">
        <article class="admin-panel">
          <div class="panel-heading">
            <div><h2>收件箱</h2></div>
            <el-tag>{{ data.feedback.total }} 条结果</el-tag>
          </div>

          <div class="application-filters" aria-label="问题报告筛选">
            <el-input v-model="filters.query" clearable placeholder="搜索标题、描述、联系人或页面路径" @keyup.enter="applyFilters" />
            <el-select v-model="filters.status" aria-label="处理状态" placeholder="全部处理状态">
              <el-option label="全部处理状态" value="" />
              <el-option label="待处理" value="open" />
              <el-option label="已处理" value="resolved" />
            </el-select>
            <el-select v-model="filters.kind" aria-label="报告类型" placeholder="全部报告类型">
              <el-option label="全部报告类型" value="" />
              <el-option label="网页缺陷" value="bug" />
              <el-option label="功能建议" value="feature" />
            </el-select>
            <div class="application-filter-actions">
              <el-button type="primary" @click="applyFilters">筛选</el-button>
              <el-button @click="resetFilters">重置</el-button>
            </div>
          </div>

          <div class="feedback-list">
            <article v-for="item in items" :key="item.id" class="feedback-item-card">
              <div class="feedback-item-header">
                <div class="application-title-row">
                  <strong>{{ item.title }}</strong>
                  <el-tag size="small" :type="item.kind === 'bug' ? 'danger' : 'primary'">{{ kindLabel(item.kind) }}</el-tag>
                  <el-tag size="small" :type="item.status === 'open' ? 'warning' : 'success'">{{ statusLabel(item.status) }}</el-tag>
                </div>
                <div v-if="canUpdate" class="app-item-actions">
                  <el-button
                    v-if="item.status === 'open'"
                    type="primary"
                    :loading="updatingId === item.id"
                    @click="updateStatus(item, 'resolved')"
                  >
                    标记已处理
                  </el-button>
                  <el-button
                    v-else
                    :loading="updatingId === item.id"
                    @click="updateStatus(item, 'open')"
                  >
                    重新打开
                  </el-button>
                </div>
              </div>
              <small>提交于 {{ formatSubmittedAt(item.submitted_at) }}</small>
              <div class="application-identifiers">
                <span>联系人：{{ item.contact_name }}</span>
                <span>邮箱：{{ item.contact_email }}</span>
                <span>页面：{{ item.page_url || '未填写' }}</span>
              </div>
              <p class="feedback-description">{{ item.description }}</p>
            </article>

            <el-empty v-if="!items.length" description="没有符合当前筛选条件的问题报告" />
          </div>

          <el-pagination
            v-if="data.feedback.total > data.feedback.page_size"
            class="application-pagination"
            background
            layout="prev, pager, next"
            :current-page="data.feedback.page"
            :page-size="data.feedback.page_size"
            :total="data.feedback.total"
            @current-change="changePage"
          />
        </article>

        <article class="admin-panel review-summary-panel">
          <div class="panel-heading">
            <div><h2>处理说明</h2></div>
            <span class="review-state online"><i /> 工作台原文</span>
          </div>
          <dl class="review-facts">
            <div><dt>当前组织</dt><dd>{{ data.organization.name }}</dd></div>
            <div><dt>当前页待处理</dt><dd>{{ openCount }} 条</dd></div>
            <div><dt>当前页已处理</dt><dd>{{ resolvedCount }} 条</dd></div>
          </dl>
          <p>访客提交后仍会发邮件提醒；这里保存完整标题、描述和联系方式，方便直接处理而不必打开邮箱。</p>
        </article>
      </section>
    </template>
  </AsyncState>
</template>
