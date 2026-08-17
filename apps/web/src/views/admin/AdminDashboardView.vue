<script setup lang="ts">
import { Bell, DocumentChecked, Folder, User } from '@element-plus/icons-vue'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(adminApi.getDashboard)
const metricIcons = [User, DocumentChecked, Bell, Folder]
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-intro">
        <div>
          <h2>工作台概览</h2>
          <p>这里汇总当前组织的成员、内容、申请与项目协作状态。</p>
        </div>
        <small>上次同步：{{ formatDate(data.updated_at) }}</small>
      </section>

      <section class="metric-grid">
        <article v-for="(metric, index) in data.metrics" :key="metric.label" class="metric-card" :class="`tone-${metric.tone}`">
          <el-icon><component :is="metricIcons[index]" /></el-icon>
          <p>{{ metric.label }}</p>
          <strong>{{ metric.value }}</strong>
          <small>{{ metric.change ?? '本周期无变化' }}</small>
        </article>
      </section>

      <section class="admin-two-column">
        <article class="admin-panel">
          <div class="panel-heading">
            <div>
              <h2>待处理申请</h2>
            </div>
            <RouterLink to="/admin/reviews" class="text-link">查看全部 →</RouterLink>
          </div>
          <div class="review-list">
            <div v-for="application in data.pending_applications" :key="application.id" class="review-item">
              <span class="avatar-bubble">{{ application.applicant.slice(0, 1) }}</span>
              <div class="review-item-content">
                <strong>{{ application.applicant }}</strong>
                <small>{{ application.type === 'whitelist' ? '服务器白名单' : '成员申请' }} · {{ application.note }}</small>
              </div>
              <RouterLink to="/admin/reviews">
                <el-button text type="primary">处理</el-button>
              </RouterLink>
            </div>
            <el-empty v-if="!data.pending_applications.length" description="暂无待处理申请" />
          </div>
        </article>

        <article class="admin-panel activity-panel">
          <div class="activity-panel-top">
            <h2>AI 活动运营</h2>
            <span class="review-state online"><i /> 人工审批保护</span>
          </div>
          <p>从组织知识生成带引用的活动方案，再由成员逐项评分、批准并进入项目与内容工作流。</p>
          <RouterLink to="/admin/activity-planner">
            <el-button type="primary" round style="margin-top: 16px;">进入活动策划工作台</el-button>
          </RouterLink>
        </article>
      </section>

      <section class="admin-panel">
        <div class="panel-heading">
          <div>
            <h2>最近编辑内容</h2>
          </div>
          <RouterLink to="/admin/content" class="text-link">管理内容 →</RouterLink>
        </div>
        <el-table :data="data.recent_content" class="admin-table">
          <el-table-column prop="title" label="内容标题" min-width="220" />
          <el-table-column prop="type" label="类型" width="110">
            <template #default="scope">
              <el-tag effect="plain">{{ scope.row.type }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="author" label="编辑者" width="120" />
          <el-table-column prop="status" label="状态" width="120">
            <template #default="scope">
              <el-tag :type="scope.row.status === 'published' ? 'success' : scope.row.status === 'review' ? 'warning' : 'info'" effect="light">
                {{ scope.row.status === 'published' ? '已发布' : scope.row.status === 'review' ? '待审核' : '草稿' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="150">
            <template #default="scope">{{ formatDate(scope.row.updated_at) }}</template>
          </el-table-column>
        </el-table>
      </section>
    </template>
  </AsyncState>
</template>
