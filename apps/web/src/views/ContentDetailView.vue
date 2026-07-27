<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import AsyncState from '@/components/AsyncState.vue'
import { ApiClientError, resolveApiUrl } from '@/api/client'
import { portalApi } from '@/api/portal'
import type { PublicContentDetail } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatBytes, formatDate } from '@/utils/format'
import { renderMarkdown } from '@/utils/markdown'

const props = defineProps<{ contentType: PublicContentDetail['type'] }>()
const route = useRoute()
const { data, error, loading, refresh } = useAsyncData(() => portalApi.getContentDetail(String(route.params.id)))
const typeLabels: Record<PublicContentDetail['type'], string> = { news: '社团动态', resource: '共享资源', knowledge: '公共知识库' }
const content = computed(() => data.value?.type === props.contentType ? data.value : undefined)
const notFound = computed(() => !loading.value && ((error.value instanceof ApiClientError && error.value.status === 404) || (!!data.value && !content.value)))
const bodyHtml = computed(() => renderMarkdown(content.value?.body ?? '', true))
const backPath = computed(() => props.contentType === 'news' ? '/posts' : props.contentType === 'resource' ? '/resources' : '/knowledge')

watch(() => route.params.id, () => refresh())
</script>

<template>
  <el-result v-if="notFound" icon="warning" title="内容不存在或尚未发布" sub-title="公开门户只展示已发布内容，链接可能已下线或已失效。">
    <template #extra>
      <RouterLink :to="backPath"><el-button type="primary" round>返回{{ typeLabels[props.contentType] }}</el-button></RouterLink>
    </template>
  </el-result>
  <AsyncState v-else :loading="loading" :error="error" @retry="refresh">
    <template v-if="content">
      <article class="content-detail-page">
        <RouterLink :to="backPath" class="detail-back">← 返回{{ typeLabels[props.contentType] }}</RouterLink>
        <header class="content-detail-header">
          <div class="eyebrow">{{ content.category }} · {{ typeLabels[content.type] }}</div>
          <h1>{{ content.title }}</h1>
          <p class="content-detail-excerpt">{{ content.excerpt }}</p>
          <small>{{ formatDate(content.published_at ?? content.updated_at) }} · {{ content.reading_minutes }} 分钟阅读</small>
        </header>

        <section class="content-detail-body" v-html="bodyHtml" />

        <aside v-if="content.type === 'resource'" class="content-detail-asset surface-panel">
          <div>
            <strong>{{ content.asset?.original_name ?? '尚未关联可下载文件' }}</strong>
            <small v-if="content.asset">{{ content.asset.mime_type }} · {{ formatBytes(content.asset.size_bytes) }}</small>
            <small v-else>请等待管理端上传资源文件。</small>
          </div>
          <a v-if="content.download_url" class="button button-primary" :href="resolveApiUrl(content.download_url)" download>下载资源</a>
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
}

.content-detail-body :deep(h1),
.content-detail-body :deep(h2),
.content-detail-body :deep(h3),
.content-detail-body :deep(h4) {
  margin: 34px 0 14px;
  line-height: 1.25;
}

.content-detail-body :deep(h1) { font-size: 2rem; }
.content-detail-body :deep(h2) { font-size: 1.6rem; }
.content-detail-body :deep(h3) { font-size: 1.3rem; }

.content-detail-body :deep(p),
.content-detail-body :deep(ul),
.content-detail-body :deep(ol),
.content-detail-body :deep(blockquote),
.content-detail-body :deep(pre),
.content-detail-body :deep(table) {
  margin: 0 0 22px;
}

.content-detail-body :deep(ul),
.content-detail-body :deep(ol) {
  padding-left: 1.5em;
}

.content-detail-body :deep(a) {
  color: var(--md-sys-color-primary);
  font-weight: 700;
}

.content-detail-body :deep(blockquote) {
  padding: 12px 18px;
  border-left: 4px solid var(--md-sys-color-primary);
  background: var(--md-sys-color-surface-container);
  color: var(--md-sys-color-on-surface-variant);
}

.content-detail-body :deep(code) {
  padding: 2px 6px;
  border-radius: 6px;
  background: var(--md-sys-color-surface-container-high);
  font-size: 0.92em;
}

.content-detail-body :deep(pre) {
  overflow-x: auto;
  padding: 16px 18px;
  border-radius: 14px;
  background: var(--md-sys-color-surface-container-highest);
}

.content-detail-body :deep(pre code) {
  padding: 0;
  background: transparent;
}

.content-detail-body :deep(img) {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 24px auto;
  border-radius: 16px;
  box-shadow: 0 12px 28px color-mix(in srgb, var(--md-sys-color-shadow) 18%, transparent);
}

.content-detail-body :deep(hr) {
  margin: 32px 0;
  border: 0;
  border-top: 1px solid color-mix(in srgb, var(--md-sys-color-outline) 24%, transparent);
}

.content-detail-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
}

.content-detail-body :deep(th),
.content-detail-body :deep(td) {
  padding: 10px 12px;
  border: 1px solid var(--md-sys-color-outline-variant);
  text-align: left;
}

.content-detail-body :deep(th) {
  background: var(--md-sys-color-surface-container);
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
