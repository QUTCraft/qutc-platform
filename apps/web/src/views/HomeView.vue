<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, Files, Notebook, Reading, Tools, VideoCamera } from '@element-plus/icons-vue'
import AsyncState from '@/components/AsyncState.vue'
import ContentCard from '@/components/ContentCard.vue'
import SectionHeading from '@/components/SectionHeading.vue'
import { portalApi } from '@/api/portal'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const router = useRouter()

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
const isQutcraftPortal = computed(() => data.value?.organization.slug === 'qutcraft')
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <!-- Dynamic Hero Aurora Mesh Backdrop & Stardust Stream -->
      <div class="hero-glow-backdrop" aria-hidden="true">
        <div class="mesh-blob blob-primary" />
        <div class="mesh-blob blob-secondary" />
        <div class="hero-stardust">
          <span
            v-for="s in 14"
            :key="s"
            class="stardust-pixel"
            :style="{
              '--delay': `${s * 180}ms`,
              '--x': `${((s * 7) % 94) + 3}%`,
              '--speed': `${3.5 + (s % 3) * 0.8}s`,
            }"
          />
        </div>
      </div>

      <!-- Hero Section -->
      <section class="portal-hero">
        <div class="hero-copy">
          <div class="hero-brand-pill">
            <span class="pulse-spark" /> {{ data.organization.short_name }} 官方门户
          </div>
          <h1 class="hero-gradient-title">{{ data.organization.tagline }}</h1>
          <p>{{ data.organization.introduction }}</p>

          <div class="hero-actions">
            <el-button
              v-if="isQutcraftPortal && data.serverStatus.apply_url"
              type="primary"
              size="large"
              round
              class="hero-apply-btn"
              @click="router.push({ name: 'apply' })"
            >
              申请加入服务器 <el-icon class="el-icon--right"><ArrowRight /></el-icon>
            </el-button>
            <RouterLink v-else-if="!isQutcraftPortal" to="/posts">
              <el-button type="primary" size="large" round class="hero-apply-btn">
                查看组织动态 <el-icon class="el-icon--right"><ArrowRight /></el-icon>
              </el-button>
            </RouterLink>
            <RouterLink to="/projects">
              <el-button size="large" round class="hero-secondary-btn">浏览公开项目</el-button>
            </RouterLink>
            <RouterLink to="/knowledge">
              <el-button size="large" round class="hero-secondary-btn">进入知识库</el-button>
            </RouterLink>
          </div>

          <div v-if="isQutcraftPortal" class="hero-stats-strip">
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
          <div v-else class="hero-stats-strip">
            <div class="stat-item">
              <strong>{{ data.posts.length }}</strong>
              <small>公开动态</small>
            </div>
            <div class="stat-divider" />
            <div class="stat-item">
              <strong>{{ data.projects.length }}</strong>
              <small>公开项目</small>
            </div>
            <div class="stat-divider" />
            <div class="stat-item">
              <strong>{{ data.knowledge.length }}</strong>
              <small>知识条目</small>
            </div>
          </div>
        </div>

        <!-- Live Server Status Glassmorphism Card -->
        <aside v-if="isQutcraftPortal" class="hero-status" aria-label="公开服务状态">
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
            @click="router.push({ name: 'apply' })"
          >
            立即提交白名单申请 →
          </el-button>
        </aside>
        <aside v-else class="hero-status" aria-label="组织公开概览">
          <div class="status-header">
            <div class="status-kicker">ORGANIZATION COMMONS</div>
            <span class="status-pill online">
              <span class="status-dot online" /> 公开门户运行中
            </span>
          </div>

          <h2>{{ data.organization.short_name }} 协作空间</h2>
          <dl class="status-dl">
            <div>
              <dt>内容发布</dt>
              <dd>{{ data.posts.length }} 条公开动态</dd>
            </div>
            <div>
              <dt>项目协作</dt>
              <dd>{{ data.projects.length }} 个公开项目</dd>
            </div>
            <div>
              <dt>知识沉淀</dt>
              <dd>{{ data.knowledge.length }} 篇公开资料</dd>
            </div>
          </dl>

          <RouterLink to="/projects">
            <el-button class="status-button" type="primary" round>查看公开协作 →</el-button>
          </RouterLink>
        </aside>
      </section>

      <!-- Portal Core Pillars Section -->
      <section v-if="isQutcraftPortal" class="pillars-section">
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
      <section v-else class="pillars-section">
        <div class="pillar-card">
          <div class="pillar-icon-box tone-primary">
            <el-icon><Reading /></el-icon>
          </div>
          <h3>公共内容运营</h3>
          <p>统一维护组织动态、公开资源与对外信息，让宣传内容始终有清晰来源和发布状态。</p>
        </div>

        <div class="pillar-card">
          <div class="pillar-icon-box tone-secondary">
            <el-icon><Tools /></el-icon>
          </div>
          <h3>项目协同推进</h3>
          <p>围绕负责人、成员和里程碑组织协作过程，使活动与长期项目都能够持续交接。</p>
        </div>

        <div class="pillar-card">
          <div class="pillar-icon-box tone-tertiary">
            <el-icon><Notebook /></el-icon>
          </div>
          <h3>组织知识沉淀</h3>
          <p>将制度、活动经验与复盘资料整理为可检索、可引用、可持续维护的公共知识。</p>
        </div>
      </section>

      <!-- Latest News Showcase Section -->
      <section class="portal-section">
        <SectionHeading
          eyebrow="LATEST ANNOUNCEMENTS"
          title="最新动态与公告"
          :description="isQutcraftPortal ? '社团发布的第一手活动通知、更新日志与公开通告。' : '组织公开发布的活动通知、服务进展与重要通告。'"
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
        <div v-if="!data.posts.length" class="portal-empty-state">
          <strong>动态内容正在整理中</strong>
          <span>发布后的社团公告与活动记录会出现在这里。</span>
        </div>
      </section>

      <!-- Open Projects Showcase Section -->
      <section class="portal-section">
        <SectionHeading
          eyebrow="CREATIVE LABS & PROJECTS"
          title="正在发生的项目"
          :description="isQutcraftPortal ? '由社团成员自主发起的建造、学术与开发工程。' : '由组织成员共同推进、对外公开的服务与协作项目。'"
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
        <div v-if="!data.projects.length" class="portal-empty-state">
          <strong>公开项目正在准备中</strong>
          <span>项目完成公开设置后会自动进入门户。</span>
        </div>
      </section>

      <!-- Resources & Knowledge Split Section -->
      <section class="split-section">
        <div class="surface-panel">
          <SectionHeading eyebrow="RESOURCES" title="共享资源" action-label="资源中心" action-to="/resources" />
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
            <div v-if="!data.resources.length" class="compact-empty-state">公开资源正在整理中。</div>
          </div>
        </div>

        <div class="surface-panel">
          <SectionHeading eyebrow="KNOWLEDGE" title="公共知识库" action-label="进入知识库" action-to="/knowledge" />
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
            <div v-if="!data.knowledge.length" class="compact-empty-state">知识条目正在整理中。</div>
          </div>
        </div>
      </section>
    </template>
  </AsyncState>
</template>
