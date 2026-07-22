<script setup lang="ts">
import { computed, ref } from 'vue'
import AsyncState from '@/components/AsyncState.vue'
import ContentCard from '@/components/ContentCard.vue'
import { portalApi } from '@/api/portal'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const category = ref('全部')
const { data, error, loading, refresh } = useAsyncData(portalApi.getPosts)
const categories = computed(() => ['全部', ...new Set(data.value?.items.map((item) => item.category) ?? [])])
const filteredItems = computed(() => data.value?.items.filter((item) => category.value === '全部' || item.category === category.value) ?? [])
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
            :class="{ active: category === item }"
            @click="category = item"
          >
            {{ item }}
          </button>
        </aside>

        <div class="knowledge-list">
          <ContentCard
            v-for="post in filteredItems"
            :key="post.id"
            :eyebrow="post.category"
            :title="post.title"
            :body="post.excerpt"
            :meta="`${formatDate(post.published_at)} · ${post.reading_minutes} 分钟阅读`"
          />
          <el-empty v-if="filteredItems.length === 0" description="该分类下暂无公开动态。" />
        </div>
      </section>
    </template>
  </AsyncState>
</template>
