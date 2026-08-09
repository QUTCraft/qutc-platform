<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminContent } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const page = ref(1)
const { data, error, loading, refresh } = useAsyncData(() => adminApi.getContent({ page: page.value }))

async function changePage(value: number) {
  page.value = value
  await refresh()
}

async function changeStatus(id: string, action: 'publish' | 'archive') {
  try {
    if (action === 'publish') await adminApi.publishContent(id)
    else await adminApi.archiveContent(id)
    ElMessage.success(action === 'publish' ? '内容已发布到门户。' : '内容已下线。')
    await refresh()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '操作失败。')
  }
}

function typeLabel(type: AdminContent['type']) {
  return { news: '动态', resource: '资源', knowledge: '知识库' }[type]
}

function statusLabel(status: AdminContent['status']) {
  return { draft: '草稿', review: '待审核', published: '已发布', archived: '已下线' }[status]
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>内容工作区</h2>
          <p>统一管理门户动态、资源与知识库条目；正文使用标准 Markdown 编写。</p>
        </div>
        <div class="content-heading-actions">
          <RouterLink to="/admin/assets">
            <el-button plain round>快捷上传文件</el-button>
          </RouterLink>
          <RouterLink to="/admin/content/new">
            <el-button type="primary" round>+ 新建内容</el-button>
          </RouterLink>
        </div>
      </section>

      <section class="admin-panel">
        <el-table :data="data.items" class="admin-table">
          <el-table-column prop="title" label="标题" min-width="260" />
          <el-table-column label="类型" width="120">
            <template #default="scope">
              <el-tag effect="plain">{{ typeLabel(scope.row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="author" label="负责人" width="130" />
          <el-table-column label="状态" width="120">
            <template #default="scope">
              <el-tag :type="scope.row.status === 'published' ? 'success' : scope.row.status === 'review' ? 'warning' : scope.row.status === 'archived' ? 'info' : 'info'">
                {{ statusLabel(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="最后修改" width="150">
            <template #default="scope">{{ formatDate(scope.row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="240" fixed="right">
            <template #default="scope">
              <RouterLink :to="`/admin/content/${scope.row.id}/edit`">
                <el-button text type="primary">编辑</el-button>
              </RouterLink>
              <el-button v-if="scope.row.status !== 'published'" text type="success" @click="changeStatus(scope.row.id, 'publish')">发布</el-button>
              <el-button v-else text type="danger" @click="changeStatus(scope.row.id, 'archive')">下线</el-button>
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

<style scoped>
.content-heading-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

@media (max-width: 640px) {
  .content-heading-actions {
    width: 100%;
  }
}
</style>
