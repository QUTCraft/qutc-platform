<script setup lang="ts">
import { computed, ref } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { portalApi } from '@/api/portal'
import type { Resource } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatBytes, formatDate } from '@/utils/format'

const keyword = ref('')
const kind = ref<'all' | Resource['kind']>('all')
const { data, error, loading, refresh } = useAsyncData(portalApi.getResources)
const kinds: Array<'all' | Resource['kind']> = ['all', 'document', 'template', 'package', 'video']
const kindLabels: Record<'all' | Resource['kind'], string> = { all: '全部', document: '文档', template: '模板', package: '资源包', video: '影音' }
const filteredItems = computed(() => data.value?.items.filter((item) => (kind.value === 'all' || item.kind === kind.value) && `${item.title}${item.description}`.toLowerCase().includes(keyword.value.toLowerCase())) ?? [])
const displayKind = (value: Resource['kind']) => kindLabels[value]
const showDownloadHint = (title: string) => ElMessage.info(`将通过受控下载地址获取「${title}」。`)
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="page-intro">
        <div class="eyebrow">RESOURCE LIBRARY</div>
        <h1>共享资源</h1>
        <p>从活动模板到开发资料，公开资源通过受控下载链接分发。文件权限与上传流程由独立管理端处理。</p>
      </section>

      <section class="resource-toolbar">
        <el-input
          v-model="keyword"
          placeholder="搜索资源关键词..."
          :prefix-icon="Search"
          clearable
          size="large"
        />
        <el-segmented
          v-model="kind"
          :options="kinds.map((value) => ({ label: kindLabels[value], value }))"
          size="large"
        />
      </section>

      <section class="resource-table surface-panel">
        <el-table :data="filteredItems" style="width: 100%">
          <el-table-column prop="title" label="资源">
            <template #default="scope">
              <strong>{{ scope.row.title }}</strong>
              <p class="table-description">{{ scope.row.description }}</p>
            </template>
          </el-table-column>
          <el-table-column label="类型" width="120">
            <template #default="scope">
              <el-tag round>{{ displayKind(scope.row.kind) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="大小" width="120">
            <template #default="scope">
              {{ formatBytes(scope.row.size_bytes) }}
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="140">
            <template #default="scope">
              {{ formatDate(scope.row.updated_at) }}
            </template>
          </el-table-column>
          <el-table-column label="" width="110" align="right">
            <template #default="scope">
              <el-button type="primary" size="small" round class="resource-download-btn" @click="showDownloadHint(scope.row.title)">
                下载
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="filteredItems.length === 0" description="没有找到匹配的公开资源。" />
      </section>
    </template>
  </AsyncState>
</template>
