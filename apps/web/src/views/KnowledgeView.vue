<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import AsyncState from '@/components/AsyncState.vue'
import ContentCard from '@/components/ContentCard.vue'
import { portalApi } from '@/api/portal'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const category = ref('全部')
const page = ref(1)
const { data, error, loading, refresh } = useAsyncData(() => portalApi.getKnowledgeArticles({
  page: page.value,
  category: category.value === '全部' ? undefined : category.value,
}))
const { data: directoryData } = useAsyncData(() => portalApi.getKnowledgeDirectories({ page_size: 100 }))
const { data: categoryData } = useAsyncData(() => portalApi.getKnowledgeArticles({ page_size: 100 }))
const allCategories = computed(() => ['全部', ...new Set([...(directoryData.value?.items.map((item) => item.name) ?? []), ...(categoryData.value?.items.map((item) => item.category).filter(Boolean) ?? [])])])

watch(category, async () => {
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
      <section class="page-intro">
        <div class="eyebrow">KNOWLEDGE BASE</div>
        <h1>公共知识库</h1>
        <p>对外公开的规则、经验与设计记录。草稿、协作批注和权限信息由独立后台管理。</p>
      </section>

      <section class="knowledge-layout">
        <aside class="knowledge-sidebar">
          <strong>文章分类</strong>
          <button
            v-for="item in allCategories"
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
            v-for="article in data.items"
            :key="article.id"
            :eyebrow="article.category"
            :title="article.title"
            :body="article.summary"
            :meta="`${formatDate(article.updated_at)} · ${article.reading_minutes} 分钟阅读`"
            :to="`/knowledge/${article.id}`"
          />
          <el-empty v-if="data.items.length === 0" description="该分类下暂无公开文章。" />
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
        </div>
      </section>
    </template>
  </AsyncState>
</template>
