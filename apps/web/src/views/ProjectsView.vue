<script setup lang="ts">
import { ref, watch } from 'vue'
import AsyncState from '@/components/AsyncState.vue'
import ContentCard from '@/components/ContentCard.vue'
import { portalApi } from '@/api/portal'
import type { Project } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const activeFilter = ref<'all' | Project['status']>('all')
const page = ref(1)
const { data, error, loading, refresh } = useAsyncData(() => portalApi.getProjects({
  page: page.value,
  status: activeFilter.value === 'all' ? undefined : activeFilter.value,
}))
const { data: countData } = useAsyncData(() => portalApi.getProjects({ page_size: 100 }))

const labels: Record<'all' | Project['status'], string> = {
  all: '全部',
  active: '进行中',
  research: '研究中',
  completed: '已完成',
}

function getCount(key: 'all' | Project['status']): number {
  if (!countData.value) return 0
  if (key === 'all') return countData.value.total
  return countData.value.items.filter((item) => item.status === key).length
}

watch(activeFilter, async () => {
  page.value = 1
  await refresh()
})

async function changePage(value: number) {
  page.value = value
  await refresh()
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="projects-header">
        <div class="projects-header-text">
          <div class="eyebrow">PROJECTS & LABS</div>
          <h1>正在发生的项目</h1>
          <p>门户展示已公开的项目概览、研究课题与构建成果。</p>
        </div>

        <div class="projects-filter-wrapper">
          <div class="filter-pills" role="tablist" aria-label="项目状态筛选">
            <button
              v-for="(label, key) in labels"
              :key="key"
              type="button"
              role="tab"
              :aria-selected="activeFilter === key"
              class="filter-pill"
              :class="[`pill-${key}`, { active: activeFilter === key }]"
              @click="activeFilter = key"
            >
              <span class="pill-dot" />
              <span class="pill-label">{{ label }}</span>
              <span class="pill-count">{{ getCount(key) }}</span>
            </button>
          </div>
        </div>
      </section>

      <section class="projects-content">
        <div class="card-grid">
          <ContentCard
            v-for="project in data.items"
            :key="project.id"
            :eyebrow="`PROJECT / ${labels[project.status]}`"
            :title="project.title"
            :body="project.summary"
            :tags="project.tags"
            :meta="`公开更新时间：${formatDate(project.updated_at)}`"
          />
        </div>

        <el-empty v-if="data.items.length === 0" description="该筛选条件下暂时没有公开项目。" />
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
