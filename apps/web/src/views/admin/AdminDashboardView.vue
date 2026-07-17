<script setup lang="ts">
import { computed } from 'vue'
import { Bell, Connection, DocumentChecked, User } from '@element-plus/icons-vue'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(adminApi.getDashboard)
const metricIcons = [User, DocumentChecked, Bell, Connection]
const serverLabel = computed(() => data.value?.server.state === 'online' ? '服务器连接正常' : '服务器暂不可用')
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh"><template v-if="data">
    <section class="admin-intro"><div><p>早上好，BBKarasu</p><span>这里汇总公开门户、成员协作和服务器适配的待办。</span></div><small>数据同步于 {{ formatDate(data.updated_at) }}</small></section>
    <section class="metric-grid"><article v-for="(metric, index) in data.metrics" :key="metric.label" class="metric-card" :class="`tone-${metric.tone}`"><el-icon><component :is="metricIcons[index]" /></el-icon><p>{{ metric.label }}</p><strong>{{ metric.value }}</strong><small>{{ metric.change ?? '本周期无变化' }}</small></article></section>
    <section class="admin-two-column"><article class="admin-panel"><div class="panel-heading"><div><p class="eyebrow">TO REVIEW</p><h2>待处理申请</h2></div><RouterLink to="/admin/reviews">查看全部</RouterLink></div><div class="review-list"><div v-for="application in data.pending_applications" :key="application.id"><span class="avatar">{{ application.applicant.slice(0, 1) }}</span><div><strong>{{ application.applicant }}</strong><small>{{ application.type === 'whitelist' ? '服务器白名单' : '成员申请' }} · {{ application.note }}</small></div><RouterLink to="/admin/reviews"><el-button text type="primary">处理</el-button></RouterLink></div></div></article>
      <article class="admin-panel server-panel"><div class="server-panel-top"><div><p class="eyebrow">SERVER ADAPTER</p><h2>{{ data.server.label }}</h2></div><span class="server-state" :class="data.server.state"><i />{{ serverLabel }}</span></div><p>{{ data.server.online_players }} / {{ data.server.max_players }} 名玩家在线。RCON 命令只会在审核后由受限接口执行。</p><RouterLink to="/admin/reviews"><el-button type="primary" round>进入审核与服务器</el-button></RouterLink></article></section>
    <section class="admin-panel"><div class="panel-heading"><div><p class="eyebrow">CONTENT FLOW</p><h2>最近编辑</h2></div><RouterLink to="/admin/content">管理内容</RouterLink></div><el-table :data="data.recent_content" class="admin-table"><el-table-column prop="title" label="内容" min-width="220" /><el-table-column prop="type" label="类型" width="110"><template #default="scope"><el-tag effect="plain">{{ scope.row.type }}</el-tag></template></el-table-column><el-table-column prop="author" label="编辑者" width="120" /><el-table-column prop="status" label="状态" width="120"><template #default="scope"><el-tag :type="scope.row.status === 'published' ? 'success' : scope.row.status === 'review' ? 'warning' : 'info'" effect="light">{{ scope.row.status === 'published' ? '已发布' : scope.row.status === 'review' ? '待审核' : '草稿' }}</el-tag></template></el-table-column><el-table-column label="更新时间" width="150"><template #default="scope">{{ formatDate(scope.row.updated_at) }}</template></el-table-column></el-table></section>
  </template></AsyncState>
</template>
