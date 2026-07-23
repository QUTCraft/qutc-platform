<script setup lang="ts">
import { computed } from 'vue'
import { ArrowRight, Files, Notebook, Reading, Tools, VideoCamera } from '@element-plus/icons-vue'
import AsyncState from '@/components/AsyncState.vue'
import ContentCard from '@/components/ContentCard.vue'
import SectionHeading from '@/components/SectionHeading.vue'
import { portalApi } from '@/api/portal'
import { useAsyncData } from '@/composables/useAsyncData'
import { usePageTransition } from '@/composables/usePageTransition'
import { formatDate } from '@/utils/format'

const { navigateToApply } = usePageTransition()

const { data, error, loading, refresh } = useAsyncData(async () => {
  const [organization, posts, projects, resources, knowledge, serverStatus] = await Promise.all([
    portalApi.getOrganization(),
    portalApi.getPosts(),
    portalApi.getProjects(),
    portalApi.getResources(),
    portalApi.getKnowledgeArticles(),
    portalApi.getServerStatus(),
  ])
  return { organization, posts: posts.items, projects: projects.items, resources: resources.items, knowledge: knowledge.items, serverStatus }
})

const heroNews = computed(() => data.value?.posts[0])
const statusLabel = computed(() => ({ online: '运行正常', maintenance: '维护中', offline: '离线' }[data.value?.serverStatus.state ?? 'offline']))
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <!-- Ambient Hero Ambient Glow Background -->
      <div class="hero-glow-backdrop" aria-hidden="true" />

      <!-- Hero Section -->
      <section class="portal-hero">
        <div class="hero-copy">
          <div class="hero-brand-pill">
            <span class="pulse-spark" /> {{ data.organization.short_name }} 官方门户
          </div>
          <h1>{{ data.organization.tagline }}</h1>
          <p>{{ data.organization.introduction }}</p>

          <div class="hero-actions">
            <el-button
              v-if="data.serverStatus.apply_url"
              type="primary"
              size="large"
              round
              class="hero-apply-btn"
              @click="(e: MouseEvent) => navigateToApply(e)"
            >
              申请加入服务器 <el-icon class="el-icon--right"><ArrowRight /></el-icon>
            </el-button>
            <RouterLink to="/projects">
              <el-button size="large" round class="hero-secondary-btn">浏览公开项目</el-button>
            </RouterLink>
            <RouterLink to="/knowledge">
              <el-button size="large" round class="hero-secondary-btn">进入知识库</el-button>
            </RouterLink>
          </div>

          <div class="hero-stats-strip">
            <div class="stat-item">
              <strong>30+</strong>
              <small>公开协作项目</small>
            </div>
            <div class="stat-divider" />
            <div class="stat-item">
              <strong>100+</strong>
              <small>沉淀知识条目</small>
            </div>
            <div class="stat-divider" />
            <div class="stat-item">
              <strong>24/7</strong>
              <small>Minecraft 生态服</small>
            </div>
          </div>
        </div>

        <!-- Live Server Status Glassmorphism Card -->
        <aside class="hero-status" aria-label="公开服务状态">
          <div class="status-header">
            <div class="status-kicker">LIVE SERVER NETWORK</div>
            <span class="status-pill" :class="data.serverStatus.state">
              <span class="status-dot" :class="data.serverStatus.state" /> {{ statusLabel }}
            </span>
          </div>

          <h2>{{ data.serverStatus.enabled ? data.serverStatus.label : 'QUTCraft 平台' }}</h2>

          <dl v-if="data.serverStatus.enabled" class="status-dl">
            <div>
              <dt>游戏版本</dt>
              <dd>{{ data.serverStatus.version }}</dd>
            </div>
            <div>
              <dt>在线玩家</dt>
              <dd>
                <strong>{{ data.serverStatus.online_players }}</strong> / {{ data.serverStatus.max_players }}
              </dd>
            </div>
            <div>
              <dt>状态实时同步</dt>
              <dd>{{ formatDate(data.serverStatus.updated_at) }}</dd>
            </div>
          </dl>

          <el-button
            v-if="data.serverStatus.apply_url"
            class="status-button"
            type="primary"
            round
            @click="(e: MouseEvent) => navigateToApply(e)"
          >
            立即提交白名单申请 →
          </el-button>
        </aside>
      </section>

      <!-- Portal Core Pillars Section -->
      <section class="pillars-section">
        <div class="pillar-card">
          <div class="pillar-icon-box tone-primary">
            <el-icon><Tools /></el-icon>
          </div>
          <h3>建筑与学术复原</h3>
          <p>1:1 比例数字孪生还原青岛理工大学校园，并开展数字建模与虚拟漫游研究。</p>
        </div>

        <div class="pillar-card">
          <div class="pillar-icon-box tone-secondary">
            <el-icon><Reading /></el-icon>
          </div>
          <h3>开源与技术研发</h3>
          <p>自研 Minecraft 自动化白名单系统、跨服即时同步与高性能服务端优化组件。</p>
        </div>

        <div class="pillar-card">
          <div class="pillar-icon-box tone-tertiary">
            <el-icon><Notebook /></el-icon>
          </div>
          <h3>知识与文化沉淀</h3>
          <p>积累完整的玩家教程、红石与建筑工坊指南，构建属于社团的长效知识库。</p>
        </div>
      </section>

      <!-- Latest News Showcase Section -->
      <section class="portal-section">
        <SectionHeading
          eyebrow="LATEST ANNOUNCEMENTS"
          title="最新动态与公告"
          description="社团发布的第一手活动通知、更新日志与公开通告。"
          action-label="查看全部动态"
          action-to="/posts"
        />
        <div class="news-layout">
          <article v-if="heroNews" class="lead-news">
            <div class="lead-news-top">
              <span class="category-badge">{{ heroNews.category }}</span>
              <small>{{ formatDate(heroNews.published_at) }} · {{ heroNews.reading_minutes }} 分钟阅读</small>
            </div>
            <h2>{{ heroNews.title }}</h2>
            <p>{{ heroNews.excerpt }}</p>
          </article>

          <div class="news-list">
            <article v-for="post in data.posts.slice(1)" :key="post.id" class="news-row">
              <div class="news-row-main">
                <span class="row-category">{{ post.category }}</span>
                <h3>{{ post.title }}</h3>
              </div>
              <small class="row-date">{{ formatDate(post.published_at) }}</small>
            </article>
          </div>
        </div>
      </section>

      <!-- Open Projects Showcase Section -->
      <section class="portal-section">
        <SectionHeading
          eyebrow="CREATIVE LABS & PROJECTS"
          title="正在发生的项目"
          description="由社团成员自主发起的建造、学术与开发工程。"
          action-label="查看全部项目"
          action-to="/projects"
        />
        <div class="card-grid">
          <ContentCard
            v-for="project in data.projects"
            :key="project.id"
            :eyebrow="`PROJECT / ${project.status.toUpperCase()}`"
            :title="project.title"
            :body="project.summary"
            :tags="project.tags"
            :meta="`公开更新时间：${formatDate(project.updated_at)}`"
          />
        </div>
      </section>

      <!-- Resources & Knowledge Split Section -->
      <section class="split-section">
        <div class="surface-panel">
          <SectionHeading eyebrow="RESOURCES" title="共享资源" action-label="资源中心 →" action-to="/resources" />
          <div class="compact-list">
            <RouterLink v-for="resource in data.resources" :key="resource.id" to="/resources" class="compact-item">
              <span class="icon-badge resource-icon-badge">
                <el-icon><Files /></el-icon>
              </span>
              <div class="item-body">
                <strong>{{ resource.title }}</strong>
                <small>{{ resource.description }}</small>
              </div>
            </RouterLink>
          </div>
        </div>

        <div class="surface-panel">
          <SectionHeading eyebrow="KNOWLEDGE" title="公共知识库" action-label="进入知识库 →" action-to="/knowledge" />
          <div class="compact-list">
            <RouterLink v-for="article in data.knowledge" :key="article.id" to="/knowledge" class="compact-item">
              <span class="icon-badge knowledge-icon-badge">
                <el-icon><Notebook /></el-icon>
              </span>
              <div class="item-body">
                <strong>{{ article.title }}</strong>
                <small>{{ article.category }} · {{ article.summary }}</small>
              </div>
            </RouterLink>
          </div>
        </div>
      </section>
    </template>
  </AsyncState>
</template>
