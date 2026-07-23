<script setup lang="ts">
import { computed, ref } from 'vue'
import AsyncState from '@/components/AsyncState.vue'
import ContentCard from '@/components/ContentCard.vue'
import { portalApi } from '@/api/portal'
import type { Project } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const activeFilter = ref<'all' | Project['status']>('all')
const { data, error, loading, refresh } = useAsyncData(portalApi.getProjects)

const filteredItems = computed(() =>
  data.value?.items.filter((item) => activeFilter.value === 'all' || item.status === activeFilter.value) ?? []
)

const labels: Record<'all' | Project['status'], string> = {
  all: '全部',
  active: '进行中',
  research: '研究中',
  completed: '已完成',
}

function getCount(key: 'all' | Project['status']): number {
  if (!data.value) return 0
  if (key === 'all') return data.value.items.length
  return data.value.items.filter((item) => item.status === key).length
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
            v-for="project in filteredItems"
            :key="project.id"
            :eyebrow="`PROJECT / ${labels[project.status]}`"
            :title="project.title"
            :body="project.summary"
            :tags="project.tags"
            :meta="`公开更新时间：${formatDate(project.updated_at)}`"
          />
        </div>

        <el-empty v-if="filteredItems.length === 0" description="该筛选条件下暂时没有公开项目。" />
      </section>
    </template>
  </AsyncState>
</template>
