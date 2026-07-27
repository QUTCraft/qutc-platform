<script setup lang="ts">
import { computed, ref } from 'vue'
import AsyncState from '@/components/AsyncState.vue'
import ContentCard from '@/components/ContentCard.vue'
import { portalApi } from '@/api/portal'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const category = ref('全部')
const { data, error, loading, refresh } = useAsyncData(portalApi.getKnowledgeArticles)
const { data: directoryData } = useAsyncData(portalApi.getKnowledgeDirectories)
const categories = computed(() => ['全部', ...new Set([...(directoryData.value?.items.map((item) => item.name) ?? []), ...(data.value?.items.map((item) => item.category) ?? [])])])
const filteredItems = computed(() => data.value?.items.filter((item) => category.value === '全部' || item.category === category.value) ?? [])
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="page-intro">
        <div class="eyebrow">KNOWLEDGE BASE</div>
        <h1>公共知识库</h1>
        <p>对外公开的规则、经验与设计记录。草稿、协作批注和权限信息由独立后台管理。</p>
      </section>

      <section class="knowledge-layout">
        <aside class="knowledge-sidebar">
          <strong>文章分类</strong>
          <button
            v-for="item in categories"
            :key="item"
            type="button"
            :class="{ active: category === item }"
            @click="category = item"
          >
            {{ item }}
          </button>
        </aside>

        <div class="knowledge-list">
          <ContentCard
            v-for="article in filteredItems"
            :key="article.id"
            :eyebrow="article.category"
            :title="article.title"
            :body="article.summary"
            :meta="`${formatDate(article.updated_at)} · ${article.reading_minutes} 分钟阅读`"
            :to="`/knowledge/${article.id}`"
          />
          <el-empty v-if="filteredItems.length === 0" description="该分类下暂无公开文章。" />
        </div>
      </section>
    </template>
  </AsyncState>
</template>
