<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Check, Close, DocumentAdd, MagicStick, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api/admin'
import type { AdminContent, AIAgentCatalog, AIAgentRun, AIConfiguration, AIKnowledgeResult } from '@/api/types'
import { renderMarkdown } from '@/utils/markdown'

const props = defineProps<{
  modelValue: boolean
  currentTitle: string
  currentExcerpt: string
  currentBody: string
  contentType: AdminContent['type']
  category: string
  knowledgeDirectoryId: string
  published: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  apply: [proposal: { title: string; excerpt: string; body: string }]
  created: [content: AdminContent]
}>()

const configuration = ref<AIConfiguration | null>(null)
const catalog = ref<AIAgentCatalog | null>(null)
const loading = ref(false)
const searching = ref(false)
const generating = ref(false)
const creatingDraft = ref(false)
const query = ref('')
const task = ref('')
const knowledgeResults = ref<AIKnowledgeResult[]>([])
const selectedSourceIds = ref<string[]>([])
const sourceRegistry = ref<Record<string, AIKnowledgeResult>>({})
const run = ref<AIAgentRun | null>(null)
const resultTab = ref<'proposal' | 'compare' | 'citations'>('proposal')
let pollGeneration = 0
let disposed = false

const open = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})
const provider = computed(() => configuration.value?.provider)
const providerLabel = computed(() => {
  if (provider.value?.mode === 'real') return '真实模型'
  if (provider.value?.mode === 'mock') return '开发 Mock'
  return '未启用'
})
const maxSources = computed(() => configuration.value?.max_sources ?? 10)
const selectedSources = computed(() => {
  return selectedSourceIds.value
    .map((id) => sourceRegistry.value[id])
    .filter((item): item is AIKnowledgeResult => Boolean(item))
})
const canGenerate = computed(() => {
  return configuration.value?.enabled &&
    provider.value?.enabled &&
    task.value.trim().length > 0 &&
    selectedSourceIds.value.length > 0 &&
    !generating.value
})
const terminal = computed(() => run.value && ['succeeded', 'failed', 'canceled'].includes(run.value.status))
const generatedTitle = computed(() => {
  const heading = run.value?.output_markdown.match(/^#\s+(.+)$/m)?.[1]?.trim()
  return run.value?.output_title.trim() || heading || props.currentTitle.trim() || '智能体生成草稿'
})
const generatedExcerpt = computed(() => run.value?.output_excerpt.trim() || props.currentExcerpt.trim())
const proposalMarkdown = computed(() => {
  const currentRun = run.value
  if (!currentRun?.output_markdown) return ''
  const markdown = currentRun.output_markdown.trim()
  const missingCitations = currentRun.citations.filter((citation) => !markdown.includes(citation.source_id))
  if (missingCitations.length === 0) return markdown
  const references = missingCitations.map((citation) => {
    return `- **${citation.title}**（知识内容 ID：\`${citation.source_id}\`，引用版本：${new Date(citation.source_updated_at).toLocaleString('zh-CN')}）`
  })
  return `${markdown}\n\n## 引用资料\n\n${references.join('\n')}`
})
const proposalHtml = computed(() => renderMarkdown(proposalMarkdown.value))
const currentHtml = computed(() => renderMarkdown(props.currentBody))
const comparisonSummary = computed(() => {
  const before = props.currentBody.trim()
  const after = proposalMarkdown.value.trim()
  return {
    beforeCharacters: before.length,
    afterCharacters: after.length,
    beforeLines: before ? before.split(/\r?\n/).length : 0,
    afterLines: after ? after.split(/\r?\n/).length : 0,
  }
})

watch(() => props.modelValue, (value) => {
  document.documentElement.classList.toggle('ai-content-drawer-open', value)
  if (!value) {
    pollGeneration += 1
    generating.value = false
    return
  }
  if (!task.value.trim()) {
    task.value = props.currentTitle.trim()
      ? `根据选定知识资料，为“${props.currentTitle.trim()}”生成一份结构清晰、适合门户发布的 Markdown 内容提案。`
      : '根据选定知识资料生成一份结构清晰、适合门户发布的 Markdown 内容提案。'
  }
  if (!configuration.value) void loadFoundation()
  if (run.value && !['succeeded', 'failed', 'canceled'].includes(run.value.status)) {
    const generation = ++pollGeneration
    generating.value = true
    void pollRun(run.value.id, generation)
      .catch((error) => ElMessage.error(error instanceof Error ? error.message : '智能体状态刷新失败。'))
      .finally(() => {
        if (generation === pollGeneration) generating.value = false
      })
  }
})

async function loadFoundation() {
  loading.value = true
  try {
    const [nextConfiguration, nextCatalog] = await Promise.all([
      adminApi.getAIConfiguration(),
      adminApi.getAIAgents(),
    ])
    configuration.value = nextConfiguration
    catalog.value = nextCatalog
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '智能体状态加载失败。')
  } finally {
    loading.value = false
  }
}

async function searchKnowledge() {
  const keyword = query.value.trim()
  if (!keyword) {
    ElMessage.warning('请输入知识资料关键词。')
    return
  }
  searching.value = true
  try {
    knowledgeResults.value = await adminApi.searchAIKnowledge({ query: keyword, limit: 20 })
    for (const item of knowledgeResults.value) sourceRegistry.value[item.id] = item
    if (knowledgeResults.value.length === 0) ElMessage.info('当前组织中没有匹配的知识资料。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '知识资料检索失败。')
  } finally {
    searching.value = false
  }
}

function toggleSource(id: string) {
  const index = selectedSourceIds.value.indexOf(id)
  if (index >= 0) {
    selectedSourceIds.value.splice(index, 1)
    return
  }
  if (selectedSourceIds.value.length >= maxSources.value) {
    ElMessage.warning(`当前组织策略每次最多选择 ${maxSources.value} 条资料。`)
    return
  }
  selectedSourceIds.value.push(id)
}

async function generateProposal() {
  if (!canGenerate.value) {
    ElMessage.warning('请确认智能体已启用，并填写任务、选择至少一条知识资料。')
    return
  }
  generating.value = true
  run.value = null
  resultTab.value = 'proposal'
  const generation = ++pollGeneration
  try {
    const created = await adminApi.createAIRun({
      agent_key: 'content-copilot',
      task: task.value.trim(),
      context_refs: selectedSourceIds.value.map((id) => ({ type: 'content', id })),
      output_mode: 'proposal',
    })
    run.value = created
    if (!['succeeded', 'failed', 'canceled'].includes(created.status)) {
      await pollRun(created.id, generation)
    }
    if (run.value?.status === 'succeeded') ElMessage.success('内容提案已生成，请预览并人工确认。')
    else if (run.value?.status === 'failed') ElMessage.error(run.value.failure_message || '智能体生成失败。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '智能体运行创建失败。')
  } finally {
    if (generation === pollGeneration) generating.value = false
  }
}

async function pollRun(id: string, generation: number) {
  while (!disposed && open.value && generation === pollGeneration) {
    await new Promise((resolve) => window.setTimeout(resolve, 700))
    if (disposed || !open.value || generation !== pollGeneration) return
    const latest = await adminApi.getAIRun(id)
    run.value = latest
    if (['succeeded', 'failed', 'canceled'].includes(latest.status)) return
  }
}

async function cancelRun() {
  if (!run.value || terminal.value) return
  try {
    run.value = await adminApi.cancelAIRun(run.value.id)
    pollGeneration += 1
    generating.value = false
    ElMessage.info('本次智能体运行已取消。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '运行取消失败。')
  }
}

async function applyProposal() {
  if (run.value?.status !== 'succeeded') return
  try {
    await ElMessageBox.confirm(
      props.currentBody.trim()
        ? '生成结果将替换当前编辑器中的标题、摘要和正文，但不会自动保存或发布。'
        : '生成结果将写入当前编辑器，但不会自动保存或发布。',
      '人工确认应用提案',
      { confirmButtonText: '应用到编辑器', cancelButtonText: '继续预览', type: 'warning' },
    )
  } catch {
    return
  }
  emit('apply', {
    title: generatedTitle.value,
    excerpt: generatedExcerpt.value,
    body: proposalMarkdown.value,
  })
  open.value = false
  ElMessage.success('已应用到编辑器，请继续人工修改并保存草稿。')
}

async function createDraft() {
  if (run.value?.status !== 'succeeded') return
  if (props.contentType === 'knowledge' && !props.knowledgeDirectoryId) {
    ElMessage.warning('创建知识库草稿前，请先在编辑器中选择知识库目录。')
    return
  }
  try {
    await ElMessageBox.confirm(
      `将以“${generatedTitle.value}”创建一篇新的${contentTypeLabel(props.contentType)}草稿。该操作不会发布内容。`,
      '人工确认创建草稿',
      { confirmButtonText: '创建草稿', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  creatingDraft.value = true
  try {
    const content = await adminApi.createContent({
      title: generatedTitle.value,
      type: props.contentType,
      category: props.category.trim() || 'AI 内容提案',
      knowledge_directory_id: props.contentType === 'knowledge' ? props.knowledgeDirectoryId : '',
      excerpt: generatedExcerpt.value,
      body: proposalMarkdown.value,
    })
    emit('created', content)
    open.value = false
    ElMessage.success('AI 提案已创建为草稿，仍需人工检查后才能发布。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '草稿创建失败。')
  } finally {
    creatingDraft.value = false
  }
}

function contentTypeLabel(type: AdminContent['type']) {
  return ({ news: '动态', resource: '资源', knowledge: '知识库' })[type]
}

function statusLabel(status: AIAgentRun['status']) {
  return ({ queued: '排队中', running: '生成中', succeeded: '已完成', failed: '失败', canceled: '已取消' })[status]
}

function statusType(status: AIAgentRun['status']) {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'canceled') return 'info'
  return 'warning'
}

onBeforeUnmount(() => {
  disposed = true
  pollGeneration += 1
  document.documentElement.classList.remove('ai-content-drawer-open')
})
</script>

<template>
  <el-drawer v-model="open" class="ai-content-drawer" size="min(96vw, 1080px)" destroy-on-close>
    <template #header>
      <div class="ai-drawer-title">
        <span><MagicStick /> CONTENT COPILOT</span>
        <h2>从知识资料生成内容</h2>
        <p>智能体只读取你明确选择的当前组织资料；生成结果必须人工确认，并且永远不会自动发布。</p>
      </div>
    </template>

    <div v-loading="loading" class="ai-content-flow">
      <section class="ai-flow-panel ai-provider-strip">
        <div>
          <span>模型模式</span>
          <strong>{{ providerLabel }}</strong>
        </div>
        <div>
          <span>可用智能体</span>
          <strong>{{ catalog?.agents[0]?.name ?? '未加载' }}</strong>
        </div>
        <div>
          <span>资料上限</span>
          <strong>{{ maxSources }} 条 / 次</strong>
        </div>
        <el-button text :icon="Refresh" :loading="loading" @click="loadFoundation">刷新</el-button>
      </section>

      <el-alert
        v-if="provider?.mode === 'mock'"
        title="当前使用开发 Mock，结果仅用于验证流程，不能作为真实模型能力演示。"
        type="warning"
        :closable="false"
        show-icon
      />
      <el-alert
        v-else-if="configuration && (!configuration.enabled || !provider?.enabled)"
        :title="!configuration.enabled ? '当前组织已停用智能体，请由所有者在智能体配置页启用。' : '模型供应商尚未启用或配置不完整。'"
        type="info"
        :closable="false"
        show-icon
      />

      <section class="ai-flow-panel">
        <div class="ai-section-heading">
          <div>
            <span>STEP 1</span>
            <h3>选择授权知识资料</h3>
          </div>
          <el-tag effect="plain">{{ selectedSourceIds.length }} / {{ maxSources }}</el-tag>
        </div>
        <div class="ai-search-row">
          <el-input
            v-model="query"
            clearable
            maxlength="80"
            placeholder="搜索知识标题、分类、摘要或正文"
            @keyup.enter="searchKnowledge"
          >
            <template #prefix><el-icon><Search /></el-icon></template>
          </el-input>
          <el-button type="primary" :loading="searching" @click="searchKnowledge">检索</el-button>
        </div>
        <div v-if="selectedSources.length" class="selected-source-list" aria-label="已选知识资料">
          <span>已选：</span>
          <el-tag
            v-for="item in selectedSources"
            :key="item.id"
            closable
            effect="plain"
            @close="toggleSource(item.id)"
          >
            {{ item.title }}
          </el-tag>
        </div>
        <div v-if="knowledgeResults.length" class="knowledge-result-list">
          <button
            v-for="item in knowledgeResults"
            :key="item.id"
            type="button"
            class="knowledge-result-card"
            :class="{ selected: selectedSourceIds.includes(item.id) }"
            @click="toggleSource(item.id)"
          >
            <span class="selection-mark"><Check v-if="selectedSourceIds.includes(item.id)" /></span>
            <span class="knowledge-result-copy">
              <strong>{{ item.title }}</strong>
              <small>{{ item.excerpt || '暂无摘要' }}</small>
              <span>{{ item.status }} · 更新于 {{ new Date(item.updated_at).toLocaleString('zh-CN') }}</span>
            </span>
          </button>
        </div>
        <el-empty v-else-if="!searching" description="输入关键词检索当前组织内有权读取的知识资料" :image-size="80" />
      </section>

      <section class="ai-flow-panel">
        <div class="ai-section-heading">
          <div>
            <span>STEP 2</span>
            <h3>描述生成任务</h3>
          </div>
        </div>
        <el-input
          v-model="task"
          type="textarea"
          :rows="4"
          maxlength="1000"
          show-word-limit
          placeholder="说明目标读者、内容目的、结构和语气。不要在这里输入密码或成员隐私。"
        />
        <div class="ai-generate-actions">
          <el-button type="primary" :icon="MagicStick" :loading="generating" :disabled="!canGenerate" @click="generateProposal">
            生成 Markdown 提案
          </el-button>
          <el-button v-if="run && !terminal" :icon="Close" @click="cancelRun">取消运行</el-button>
          <span v-if="run">
            <el-tag :type="statusType(run.status)" effect="plain">{{ statusLabel(run.status) }}</el-tag>
            <small>{{ run.mode }} · {{ run.model }} · {{ run.request_id }}</small>
          </span>
        </div>
      </section>

      <section v-if="run?.status === 'succeeded'" class="ai-flow-panel ai-result-panel">
        <div class="ai-section-heading">
          <div>
            <span>STEP 3</span>
            <h3>预览、对比并人工确认</h3>
          </div>
          <el-tag type="success" effect="plain">{{ run.citations.length }} 条固定引用</el-tag>
        </div>

        <el-radio-group v-model="resultTab" class="ai-result-tabs">
          <el-radio-button value="proposal">生成结果</el-radio-button>
          <el-radio-button value="compare">与当前内容对比</el-radio-button>
          <el-radio-button value="citations">引用资料</el-radio-button>
        </el-radio-group>

        <div v-if="resultTab === 'proposal'" class="ai-markdown-preview" v-html="proposalHtml" />
        <div v-else-if="resultTab === 'compare'" class="ai-comparison">
          <article>
            <header>
              <strong>当前编辑器</strong>
              <span>{{ comparisonSummary.beforeCharacters }} 字符 · {{ comparisonSummary.beforeLines }} 行</span>
            </header>
            <div v-if="currentBody.trim()" class="ai-markdown-preview" v-html="currentHtml" />
            <el-empty v-else description="当前编辑器为空" :image-size="72" />
          </article>
          <article>
            <header>
              <strong>生成提案</strong>
              <span>{{ comparisonSummary.afterCharacters }} 字符 · {{ comparisonSummary.afterLines }} 行</span>
            </header>
            <div class="ai-markdown-preview" v-html="proposalHtml" />
          </article>
        </div>
        <div v-else class="ai-citation-list">
          <article v-for="citation in run.citations" :key="citation.id">
            <div>
              <strong>{{ citation.title }}</strong>
              <code>{{ citation.source_id }}</code>
            </div>
            <p>{{ citation.excerpt || '暂无摘要' }}</p>
            <span>引用版本：{{ new Date(citation.source_updated_at).toLocaleString('zh-CN') }}</span>
          </article>
        </div>

        <div class="ai-confirmation-note">
          <strong>人工确认边界</strong>
          <p>“应用到编辑器”不会保存；“创建新草稿”只创建 draft。两种操作都不会审核、发布或修改原知识资料。</p>
        </div>
        <div class="ai-result-actions">
          <el-button :disabled="published" @click="applyProposal">应用到当前编辑器</el-button>
          <el-button type="primary" :icon="DocumentAdd" :loading="creatingDraft" @click="createDraft">确认并创建新草稿</el-button>
        </div>
      </section>

      <el-alert
        v-else-if="run?.status === 'failed'"
        :title="run.failure_message || '模型运行失败，请调整任务后重试。'"
        :description="`错误码：${run.failure_code || 'ai.run_failed'}；Request ID：${run.request_id}`"
        type="error"
        :closable="false"
        show-icon
      />
    </div>
  </el-drawer>
</template>

<style scoped>
:global(html.ai-content-drawer-open) {
  overflow: hidden;
}

.ai-drawer-title {
  display: grid;
  gap: 4px;
}

.ai-drawer-title > span,
.ai-section-heading span {
  color: var(--md-sys-color-primary);
  font: 800 11px var(--md-font-mono);
  letter-spacing: 0.1em;
}

.ai-drawer-title > span {
  display: flex;
  align-items: center;
  gap: 6px;
}

.ai-drawer-title svg {
  width: 15px;
}

.ai-drawer-title h2,
.ai-section-heading h3 {
  margin: 0;
}

.ai-drawer-title p {
  margin: 0;
  color: var(--md-sys-color-on-surface-variant);
  font-size: 13px;
}

.ai-content-flow {
  display: grid;
  gap: 16px;
  padding-bottom: 32px;
}

.ai-flow-panel {
  padding: 20px;
  background: var(--md-sys-color-surface-container);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-lg);
}

.ai-provider-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr)) auto;
  align-items: center;
  gap: 14px;
}

.ai-provider-strip > div {
  display: grid;
  gap: 3px;
}

.ai-provider-strip span,
.ai-generate-actions small,
.ai-comparison header span,
.ai-citation-list article > span {
  color: var(--md-sys-color-on-surface-variant);
  font-size: 11px;
}

.ai-section-heading,
.ai-search-row,
.ai-generate-actions,
.ai-result-actions,
.ai-comparison header,
.ai-citation-list article > div {
  display: flex;
  align-items: center;
}

.ai-section-heading {
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.ai-section-heading > div {
  display: grid;
  gap: 4px;
}

.ai-search-row {
  gap: 10px;
}

.knowledge-result-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.selected-source-list {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
}

.selected-source-list > span {
  color: var(--md-sys-color-on-surface-variant);
  font-size: 12px;
}

.knowledge-result-card {
  display: grid;
  grid-template-columns: 26px minmax(0, 1fr);
  gap: 10px;
  padding: 14px;
  color: var(--md-sys-color-on-surface);
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-md);
  cursor: pointer;
  text-align: left;
}

.knowledge-result-card:hover,
.knowledge-result-card.selected {
  background: var(--md-sys-color-primary-container);
  border-color: var(--md-sys-color-primary);
}

.selection-mark {
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  color: var(--md-sys-color-on-primary);
  background: var(--md-sys-color-surface-container-highest);
  border: 1px solid var(--md-sys-color-outline);
  border-radius: 50%;
}

.selected .selection-mark {
  background: var(--md-sys-color-primary);
  border-color: var(--md-sys-color-primary);
}

.selection-mark svg {
  width: 13px;
}

.knowledge-result-copy {
  display: grid;
  min-width: 0;
  gap: 5px;
}

.knowledge-result-copy strong,
.knowledge-result-copy small,
.knowledge-result-copy span {
  overflow: hidden;
  text-overflow: ellipsis;
}

.knowledge-result-copy small {
  color: var(--md-sys-color-on-surface-variant);
  white-space: nowrap;
}

.knowledge-result-copy span {
  color: var(--md-sys-color-primary);
  font-size: 11px;
}

.ai-generate-actions {
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 14px;
}

.ai-generate-actions > span {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
}

.ai-generate-actions small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-result-tabs {
  margin-bottom: 14px;
}

.ai-markdown-preview {
  min-height: 180px;
  padding: 20px;
  overflow: auto;
  color: var(--md-sys-color-on-surface);
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-md);
  line-height: 1.75;
  overflow-wrap: anywhere;
}

.ai-markdown-preview :deep(h1),
.ai-markdown-preview :deep(h2),
.ai-markdown-preview :deep(h3) {
  margin: 1.1em 0 0.5em;
}

.ai-markdown-preview :deep(h1:first-child) {
  margin-top: 0;
}

.ai-markdown-preview :deep(code) {
  padding: 2px 5px;
  color: var(--md-sys-color-on-primary-container);
  background: var(--md-sys-color-primary-container);
  border-radius: 5px;
}

.ai-markdown-preview :deep(a) {
  color: var(--md-sys-color-primary);
}

.ai-comparison {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.ai-comparison > article {
  min-width: 0;
}

.ai-comparison header {
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.ai-citation-list {
  display: grid;
  gap: 10px;
}

.ai-citation-list article {
  padding: 14px 16px;
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-md);
}

.ai-citation-list article > div {
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 8px;
}

.ai-citation-list code {
  color: var(--md-sys-color-primary);
  font-size: 11px;
}

.ai-citation-list p {
  margin: 8px 0;
  color: var(--md-sys-color-on-surface-variant);
}

.ai-confirmation-note {
  margin-top: 14px;
  padding: 14px 16px;
  color: var(--md-sys-color-on-tertiary-container);
  background: var(--md-sys-color-tertiary-container);
  border-radius: var(--md-shape-md);
}

.ai-confirmation-note p {
  margin: 4px 0 0;
  font-size: 12px;
}

.ai-result-actions {
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 14px;
}

@media (max-width: 760px) {
  .ai-provider-strip,
  .knowledge-result-list,
  .ai-comparison {
    grid-template-columns: 1fr;
  }

  .ai-provider-strip > .el-button {
    justify-self: start;
  }

  .ai-search-row {
    align-items: stretch;
    flex-direction: column;
  }

  .ai-result-actions > * {
    flex: 1 1 auto;
  }
}
</style>
