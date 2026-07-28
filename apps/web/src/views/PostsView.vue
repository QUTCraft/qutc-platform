<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import AsyncState from '@/components/AsyncState.vue'
import ContentCard from '@/components/ContentCard.vue'
import { portalApi } from '@/api/portal'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const category = ref('全部')
const page = ref(1)
const { data, error, loading, refresh } = useAsyncData(() => portalApi.getPosts({
  page: page.value,
  category: category.value === '全部' ? undefined : category.value,
}))
const { data: categoryData } = useAsyncData(() => portalApi.getPosts({ page_size: 100 }))
const categories = computed(() => ['全部', ...new Set(categoryData.value?.items.map((item) => item.category).filter(Boolean) ?? [])])

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
        <div class="eyebrow">PUBLIC UPDATES</div>
        <h1>社团动态</h1>
        <p>这里记录社团公开发布的公告、活动与阶段性进展。项目详情请前往项目页查看。</p>
      </section>

      <section class="knowledge-layout">
        <aside class="knowledge-sidebar">
          <strong>动态分类</strong>
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
            v-for="post in data.items"
            :key="post.id"
            :eyebrow="post.category"
            :title="post.title"
            :body="post.excerpt"
            :meta="`${formatDate(post.published_at)} · ${post.reading_minutes} 分钟阅读`"
            :to="`/posts/${post.id}`"
          />
          <el-empty v-if="data.items.length === 0" description="该分类下暂无公开动态。" />
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
