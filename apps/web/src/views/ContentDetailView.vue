<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AsyncState from '@/components/AsyncState.vue'
import { resolveApiUrl } from '@/api/client'
import { portalApi } from '@/api/portal'
import type { PublicContentDetail } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatBytes, formatDate } from '@/utils/format'

const props = defineProps<{ contentType: PublicContentDetail['type'] }>()
const route = useRoute()
const { data, error, loading, refresh } = useAsyncData(() => portalApi.getContentDetail(String(route.params.id)))
const typeLabels: Record<PublicContentDetail['type'], string> = { news: '社团动态', resource: '共享资源', knowledge: '公共知识库' }
const paragraphs = computed(() => data.value?.body.split(/\n{2,}/).filter(Boolean) ?? [])
const backPath = computed(() => props.contentType === 'news' ? '/posts' : props.contentType === 'resource' ? '/resources' : '/knowledge')
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <article class="content-detail-page">
        <RouterLink :to="backPath" class="detail-back">← 返回{{ typeLabels[props.contentType] }}</RouterLink>
        <header class="content-detail-header">
          <div class="eyebrow">{{ data.category }} · {{ typeLabels[data.type] }}</div>
          <h1>{{ data.title }}</h1>
          <p class="content-detail-excerpt">{{ data.excerpt }}</p>
          <small>{{ formatDate(data.published_at ?? data.updated_at) }} · {{ data.reading_minutes }} 分钟阅读</small>
        </header>

        <section class="content-detail-body">
          <p v-for="(paragraph, index) in paragraphs" :key="index">{{ paragraph }}</p>
        </section>

        <aside v-if="data.type === 'resource'" class="content-detail-asset surface-panel">
          <div>
            <strong>{{ data.asset?.original_name ?? '尚未关联可下载文件' }}</strong>
            <small v-if="data.asset">{{ data.asset.mime_type }} · {{ formatBytes(data.asset.size_bytes) }}</small>
            <small v-else>请等待管理端上传资源文件。</small>
          </div>
          <a v-if="data.download_url" class="button button-primary" :href="resolveApiUrl(data.download_url)" download>下载资源</a>
          <span v-else class="asset-unavailable">暂无文件</span>
        </aside>
      </article>
    </template>
  </AsyncState>
</template>

<style scoped>
.content-detail-page {
  width: min(860px, 100%);
  margin: 0 auto;
  padding: 32px 0 72px;
}

.detail-back {
  display: inline-flex;
  margin-bottom: 34px;
  color: var(--md-sys-color-primary);
  font-weight: 700;
}

.content-detail-header {
  padding-bottom: 30px;
  border-bottom: 1px solid color-mix(in srgb, var(--md-sys-color-outline) 24%, transparent);
}

.content-detail-header h1 {
  margin: 12px 0 18px;
  font-size: clamp(2rem, 5vw, 4rem);
  line-height: 1.08;
}

.content-detail-header small,
.content-detail-asset small {
  color: var(--md-sys-color-on-surface-variant);
}

.content-detail-excerpt {
  max-width: 700px;
  margin: 0 0 18px;
  color: var(--md-sys-color-on-surface-variant);
  font-size: 1.12rem;
  line-height: 1.8;
}

.content-detail-body {
  padding: 34px 0;
  color: var(--md-sys-color-on-surface);
  font-size: 1.08rem;
  line-height: 2;
  white-space: pre-wrap;
}

.content-detail-body p {
  margin: 0 0 22px;
}

.content-detail-asset {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 20px 24px;
}

.content-detail-asset div {
  display: grid;
  gap: 6px;
}

.button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 42px;
  padding: 0 20px;
  border-radius: 999px;
  text-decoration: none;
  font-weight: 700;
}

.button-primary {
  background: var(--md-sys-color-primary);
  color: var(--md-sys-color-on-primary);
}

.asset-unavailable {
  color: var(--md-sys-color-on-surface-variant);
  font-weight: 700;
}

@media (max-width: 640px) {
  .content-detail-page { padding-top: 20px; }
  .content-detail-asset { align-items: flex-start; flex-direction: column; }
}
</style>
