<script setup lang="ts">
import { computed } from 'vue'
import { ElMessage } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import ContentCard from '@/components/ContentCard.vue'
import SectionHeading from '@/components/SectionHeading.vue'
import { portalApi } from '@/api/portal'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(async () => {
  const [organization, posts, projects, resources, knowledge, serverStatus] = await Promise.all([
    portalApi.getOrganization(), portalApi.getPosts(), portalApi.getProjects(), portalApi.getResources(), portalApi.getKnowledgeArticles(), portalApi.getServerStatus(),
  ])
  return { organization, posts: posts.items, projects: projects.items, resources: resources.items, knowledge: knowledge.items, serverStatus }
})

const heroNews = computed(() => data.value?.posts[0])
const statusLabel = computed(() => ({ online: '在线', maintenance: '维护中', offline: '离线' }[data.value?.serverStatus.state ?? 'offline']))
const showApplyHint = () => ElMessage.info('白名单申请将提交至公开申请接口。')
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="portal-hero">
        <div class="hero-copy">
          <div class="eyebrow">{{ data.organization.short_name }} / PUBLIC PORTAL</div>
          <h1>{{ data.organization.tagline }}</h1>
          <p>{{ data.organization.introduction }}</p>
          <div class="hero-actions"><RouterLink to="/projects"><el-button type="primary" size="large" round>浏览公开项目</el-button></RouterLink><RouterLink to="/knowledge"><el-button size="large" round>进入知识库</el-button></RouterLink></div>
        </div>
        <aside class="hero-status" aria-label="公开服务状态">
          <div class="status-kicker">PUBLIC STATUS</div>
          <h2>{{ data.serverStatus.enabled ? data.serverStatus.label : '社团公开门户' }}</h2>
          <div class="status-row"><span class="status-dot" :class="data.serverStatus.state" />{{ statusLabel }}</div>
          <dl v-if="data.serverStatus.enabled"><div><dt>版本</dt><dd>{{ data.serverStatus.version }}</dd></div><div><dt>在线玩家</dt><dd>{{ data.serverStatus.online_players }} / {{ data.serverStatus.max_players }}</dd></div><div><dt>最近同步</dt><dd>{{ formatDate(data.serverStatus.updated_at) }}</dd></div></dl>
          <el-button v-if="data.serverStatus.apply_url" class="status-button" round @click="showApplyHint">申请加入服务器</el-button>
        </aside>
      </section>

      <section class="portal-section">
        <SectionHeading eyebrow="LATEST NEWS" title="最近发生的事" description="社团公开发布的动态、活动与公告。" action-label="全部项目" action-to="/projects" />
        <div class="news-layout"><article v-if="heroNews" class="lead-news"><div class="eyebrow">{{ heroNews.category }}</div><h2>{{ heroNews.title }}</h2><p>{{ heroNews.excerpt }}</p><small>{{ formatDate(heroNews.published_at) }} · {{ heroNews.reading_minutes }} 分钟阅读</small></article><div class="news-list"><article v-for="post in data.posts.slice(1)" :key="post.id"><div><span>{{ post.category }}</span><h3>{{ post.title }}</h3></div><small>{{ formatDate(post.published_at) }}</small></article></div></div>
      </section>

      <section class="portal-section">
        <SectionHeading eyebrow="OPEN PROJECTS" title="正在发生的项目" description="所有项目均由社团成员维护，并以公开状态展示进展。" action-label="查看全部" action-to="/projects" />
        <div class="card-grid"><ContentCard v-for="project in data.projects" :key="project.id" :eyebrow="`PROJECT / ${project.status.toUpperCase()}`" :title="project.title" :body="project.summary" :tags="project.tags" :meta="`更新于 ${formatDate(project.updated_at)}`" /></div>
      </section>

      <section class="split-section">
        <div class="surface-panel"><SectionHeading eyebrow="RESOURCES" title="共享资源" action-label="资源中心" action-to="/resources" /><div class="compact-list"><RouterLink v-for="resource in data.resources" :key="resource.id" to="/resources"><span class="resource-badge">{{ resource.kind.toUpperCase() }}</span><span><strong>{{ resource.title }}</strong><small>{{ resource.description }}</small></span></RouterLink></div></div>
        <div class="surface-panel"><SectionHeading eyebrow="KNOWLEDGE" title="公共经验库" action-label="进入知识库" action-to="/knowledge" /><div class="compact-list"><RouterLink v-for="article in data.knowledge" :key="article.id" to="/knowledge"><span class="article-index">{{ article.reading_minutes }}m</span><span><strong>{{ article.title }}</strong><small>{{ article.category }} · {{ article.summary }}</small></span></RouterLink></div></div>
      </section>
    </template>
  </AsyncState>
</template>
