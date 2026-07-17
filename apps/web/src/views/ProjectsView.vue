<script setup lang="ts">
import { computed, ref } from 'vue'
import AsyncState from '@/components/AsyncState.vue'
import ContentCard from '@/components/ContentCard.vue'
import SectionHeading from '@/components/SectionHeading.vue'
import { portalApi } from '@/api/portal'
import type { Project } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const activeFilter = ref<'all' | Project['status']>('all')
const { data, error, loading, refresh } = useAsyncData(portalApi.getProjects)
const filteredItems = computed(() => data.value?.items.filter((item) => activeFilter.value === 'all' || item.status === activeFilter.value) ?? [])
const labels: Record<'all' | Project['status'], string> = { all: '全部', active: '进行中', research: '研究中', completed: '已完成' }
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh"><template v-if="data"><section class="page-intro"><div class="eyebrow">PROJECTS</div><h1>正在发生的项目</h1><p>门户只展示已公开的项目概览、进展和参与入口。项目成员、私有资料与管理操作属于独立后台。</p></section><section><div class="filter-row" aria-label="项目状态筛选"><el-radio-group v-model="activeFilter"><el-radio-button v-for="(label, key) in labels" :key="key" :value="key">{{ label }}</el-radio-button></el-radio-group></div><div class="card-grid"><ContentCard v-for="project in filteredItems" :key="project.id" :eyebrow="`PROJECT / ${labels[project.status]}`" :title="project.title" :body="project.summary" :tags="project.tags" :meta="`公开更新时间：${formatDate(project.updated_at)}`" /></div><el-empty v-if="filteredItems.length === 0" description="这个筛选条件下暂时没有公开项目。" /></section></template></AsyncState>
</template>
