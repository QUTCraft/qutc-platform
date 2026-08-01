<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { Check, MagicStick, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api/admin'
import type { ActivityPlan, AIConfiguration, AIKnowledgeResult } from '@/api/types'
import { renderMarkdown } from '@/utils/markdown'

const step = ref(0)
const loadingFoundation = ref(false)
const searching = ref(false)
const generating = ref(false)
const approving = ref(false)
const configuration = ref<AIConfiguration | null>(null)
const query = ref('')
const knowledgeResults = ref<AIKnowledgeResult[]>([])
const selectedSourceIds = ref<string[]>([])
const sourceRegistry = ref<Record<string, AIKnowledgeResult>>({})
const plan = ref<ActivityPlan | null>(null)
const selectedActions = ref<string[]>([])
let pollGeneration = 0
let disposed = false

const form = reactive({
  title: '',
  objective: '',
  audience: '',
  venue: '',
  dateRange: [] as string[],
  expectedParticipants: 0,
  budget: '',
  constraints: '',
})

const selectedSources = computed(() => selectedSourceIds.value.map((id) => sourceRegistry.value[id]).filter(Boolean))
const maxSources = computed(() => configuration.value?.max_sources ?? 10)
const providerLabel = computed(() => {
  if (configuration.value?.provider.mode === 'real') return '真实模型'
  if (configuration.value?.provider.mode === 'mock') return '开发 Mock'
  return '未启用'
})
const planHtml = computed(() => renderMarkdown(plan.value?.run.output_markdown ?? ''))
const activityAgentAvailable = computed(() => configuration.value?.enabled && configuration.value.provider.enabled)

onMounted(() => void loadFoundation())
onBeforeUnmount(() => {
  disposed = true
  pollGeneration += 1
})

async function loadFoundation() {
  loadingFoundation.value = true
  try {
    const [nextConfiguration, catalog] = await Promise.all([adminApi.getAIConfiguration(), adminApi.getAIAgents()])
    configuration.value = nextConfiguration
    if (!catalog.agents.some((agent) => agent.key === 'activity-planner')) {
      ElMessage.warning('当前组织尚未初始化活动策划智能体，请重启 API 完成定义初始化。')
    }
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '智能体状态加载失败。')
  } finally {
    loadingFoundation.value = false
  }
}

function continueToSources() {
  if (!form.title.trim() || !form.objective.trim() || !form.audience.trim()) {
    ElMessage.warning('请至少填写活动名称、目标和目标受众。')
    return
  }
  if (form.dateRange.length === 2 && new Date(form.dateRange[1]).getTime() <= new Date(form.dateRange[0]).getTime()) {
    ElMessage.warning('结束时间必须晚于开始时间。')
    return
  }
  step.value = 1
}

async function searchKnowledge() {
  const keyword = query.value.trim()
  if (!keyword) {
    ElMessage.warning('请输入活动规范、历史活动或相关知识关键词。')
    return
  }
  searching.value = true
  try {
    knowledgeResults.value = await adminApi.searchAIKnowledge({ query: keyword, limit: 20 })
    for (const item of knowledgeResults.value) sourceRegistry.value[item.id] = item
    if (!knowledgeResults.value.length) ElMessage.info('没有匹配资料，请先在知识库录入活动规范或历史材料。')
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
    ElMessage.warning(`每次最多选择 ${maxSources.value} 条资料。`)
    return
  }
  selectedSourceIds.value.push(id)
}

async function generatePlan() {
  if (!activityAgentAvailable.value) {
    ElMessage.warning('请先在智能体配置中启用可用模型。')
    return
  }
  if (!selectedSourceIds.value.length) {
    ElMessage.warning('至少选择一条组织知识，确保方案有可核对的依据。')
    return
  }
  generating.value = true
  step.value = 2
  const generation = ++pollGeneration
  try {
    plan.value = await adminApi.createActivityPlan({
      title: form.title.trim(),
      objective: form.objective.trim(),
      audience: form.audience.trim(),
      venue: form.venue.trim(),
      starts_at: form.dateRange[0] || undefined,
      ends_at: form.dateRange[1] || undefined,
      expected_participants: form.expectedParticipants,
      budget: form.budget.trim(),
      constraints: form.constraints.trim(),
      context_refs: selectedSourceIds.value.map((id) => ({ type: 'content', id })),
    })
    await pollPlan(plan.value.id, generation)
    if (plan.value?.status === 'ready') {
      selectedActions.value = plan.value.proposed_actions.map((action) => action.key)
      ElMessage.success('活动方案已生成，请核对引用并逐项批准建议操作。')
    } else if (plan.value?.status === 'failed') {
      ElMessage.error(plan.value.run.failure_message || '活动方案生成失败。')
    }
  } catch (error) {
    step.value = 1
    ElMessage.error(error instanceof Error ? error.message : '活动策划任务创建失败。')
  } finally {
    if (generation === pollGeneration) generating.value = false
  }
}

async function pollPlan(id: string, generation: number) {
  while (!disposed && generation === pollGeneration) {
    const latest = await adminApi.getActivityPlan(id)
    plan.value = latest
    if (['ready', 'failed', 'canceled', 'applied'].includes(latest.status)) return
    await new Promise((resolve) => window.setTimeout(resolve, 700))
  }
}

function normalizeActionSelection() {
  const milestoneKeys = plan.value?.proposed_actions.filter((action) => action.kind === 'milestone').map((action) => action.key) ?? []
  if (milestoneKeys.some((key) => selectedActions.value.includes(key)) && !selectedActions.value.includes('create_project')) {
    selectedActions.value.push('create_project')
    ElMessage.info('里程碑必须归属项目，已同时选择“创建项目”。')
  }
}

async function approveActions() {
  if (!plan.value || plan.value.status !== 'ready' || !selectedActions.value.length) {
    ElMessage.warning('请至少选择一项需要执行的建议操作。')
    return
  }
  try {
    await ElMessageBox.confirm(
      `将执行 ${selectedActions.value.length} 项操作。项目保持非公开，公告只创建为草稿，不会自动发布。`,
      '人工批准智能体建议',
      { confirmButtonText: '批准并执行', cancelButtonText: '继续审查', type: 'warning' },
    )
  } catch {
    return
  }
  approving.value = true
  try {
    plan.value = await adminApi.approveActivityPlan(plan.value.id, selectedActions.value)
    ElMessage.success('批准完成，所选业务对象已创建并写入审计。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '建议操作执行失败。')
  } finally {
    approving.value = false
  }
}

function resetPlanner() {
  pollGeneration += 1
  step.value = 0
  plan.value = null
  selectedActions.value = []
  selectedSourceIds.value = []
  Object.assign(form, { title: '', objective: '', audience: '', venue: '', dateRange: [], expectedParticipants: 0, budget: '', constraints: '' })
}

function actionKindLabel(kind: string) {
  return ({ project: '项目', milestone: '里程碑', content: '内容草稿' } as Record<string, string>)[kind] ?? kind
}

function formatDate(value?: string | null) {
  return value ? new Date(value).toLocaleString('zh-CN') : '日期待定'
}
</script>

<template>
  <div class="activity-planner-page" v-loading="loadingFoundation">
    <section class="admin-page-heading activity-heading">
      <div>
        <span class="eyebrow">COMMONS AGENT · SERVICE INNOVATION</span>
        <h2>AI 校园活动策划</h2>
        <p>从组织知识生成带引用的活动方案；所有项目、里程碑和公告草稿必须由人逐项批准。</p>
      </div>
      <el-tag effect="plain" round>{{ providerLabel }}</el-tag>
    </section>

    <el-alert
      v-if="configuration?.provider.mode === 'mock'"
      title="当前使用开发 Mock，只用于验证工作流；比赛正式演示必须连接真实模型。"
      type="warning"
      :closable="false"
      show-icon
    />

    <section class="planner-shell">
      <el-steps :active="step" finish-status="success" align-center class="planner-steps">
        <el-step title="活动需求" description="目标、受众与约束" />
        <el-step title="选择依据" description="组织知识与历史资料" />
        <el-step title="审查执行" description="带引用方案与人工批准" />
      </el-steps>

      <div v-if="step === 0" class="planner-stage">
        <div class="stage-heading">
          <div><span>STEP 01</span><h3>描述这次校园活动</h3></div>
          <p>结构化信息会与选定知识共同交给智能体，不会把组织外数据混入上下文。</p>
        </div>
        <el-form label-position="top" class="brief-grid">
          <el-form-item label="活动名称" required><el-input v-model="form.title" maxlength="160" show-word-limit placeholder="例如：校园开源创作工作坊" /></el-form-item>
          <el-form-item label="目标受众" required><el-input v-model="form.audience" maxlength="500" placeholder="例如：全校对开源与内容创作感兴趣的学生" /></el-form-item>
          <el-form-item label="活动目标" required class="wide"><el-input v-model="form.objective" type="textarea" :rows="3" maxlength="1000" show-word-limit placeholder="说明希望解决的问题和活动价值" /></el-form-item>
          <el-form-item label="时间范围" class="wide"><el-date-picker v-model="form.dateRange" type="datetimerange" value-format="YYYY-MM-DDTHH:mm:ss[Z]" start-placeholder="开始时间" end-placeholder="结束时间" style="width: 100%" /></el-form-item>
          <el-form-item label="场地"><el-input v-model="form.venue" maxlength="300" placeholder="待定也可以" /></el-form-item>
          <el-form-item label="预计人数"><el-input-number v-model="form.expectedParticipants" :min="0" :max="100000" style="width: 100%" /></el-form-item>
          <el-form-item label="预算"><el-input v-model="form.budget" maxlength="200" placeholder="例如：500 元，或暂未确定" /></el-form-item>
          <el-form-item label="限制与风险"><el-input v-model="form.constraints" maxlength="2000" placeholder="审批、安全、设备、天气或人员限制" /></el-form-item>
        </el-form>
        <div class="stage-actions"><el-button type="primary" round @click="continueToSources">下一步：选择活动依据</el-button></div>
      </div>

      <div v-else-if="step === 1" class="planner-stage">
        <div class="stage-heading">
          <div><span>STEP 02</span><h3>选择可信的组织资料</h3></div>
          <p>至少选择一条知识。引用会固定快照，后续可以核对方案依据。</p>
        </div>
        <div class="knowledge-search">
          <el-input v-model="query" clearable placeholder="搜索活动规范、场地要求或历史活动" @keyup.enter="searchKnowledge"><template #prefix><el-icon><Search /></el-icon></template></el-input>
          <el-button type="primary" :icon="Search" :loading="searching" @click="searchKnowledge">检索</el-button>
        </div>
        <div v-if="knowledgeResults.length" class="knowledge-grid">
          <button v-for="item in knowledgeResults" :key="item.id" type="button" class="knowledge-card" :class="{ selected: selectedSourceIds.includes(item.id) }" @click="toggleSource(item.id)">
            <span class="knowledge-check"><el-icon v-if="selectedSourceIds.includes(item.id)"><Check /></el-icon></span>
            <strong>{{ item.title }}</strong>
            <p>{{ item.excerpt }}</p>
            <small>{{ new Date(item.updated_at).toLocaleDateString('zh-CN') }}</small>
          </button>
        </div>
        <el-empty v-else description="检索并选择当前组织的知识资料" />
        <div class="selection-summary">已选择 <strong>{{ selectedSources.length }}</strong> / {{ maxSources }} 条资料</div>
        <div class="stage-actions spread">
          <el-button round @click="step = 0">返回修改需求</el-button>
          <el-button type="primary" round :icon="MagicStick" :loading="generating" @click="generatePlan">生成活动方案</el-button>
        </div>
      </div>

      <div v-else class="planner-stage result-stage">
        <div v-if="generating || plan?.status === 'generating'" class="generation-state">
          <el-icon class="is-loading"><Refresh /></el-icon><h3>智能体正在生成活动方案</h3><p>正在读取所选资料、组织流程并生成风险与执行建议。</p>
        </div>

        <template v-else-if="plan?.status === 'ready' || plan?.status === 'applied'">
          <div class="stage-heading">
            <div><span>STEP 03</span><h3>{{ plan.title }}</h3></div>
            <el-tag :type="plan.status === 'applied' ? 'success' : 'warning'" effect="plain">{{ plan.status === 'applied' ? '已人工批准' : '等待人工批准' }}</el-tag>
          </div>
          <div class="result-grid">
            <article class="plan-preview markdown-body" v-html="planHtml" />
            <aside class="plan-review">
              <section class="review-block">
                <span class="block-label">固定引用</span>
                <div v-for="citation in plan.run.citations" :key="citation.id" class="citation-item">
                  <strong>{{ citation.title }}</strong><p>{{ citation.excerpt }}</p><small>引用版本：{{ formatDate(citation.source_updated_at) }}</small>
                </div>
              </section>
              <section class="review-block">
                <span class="block-label">建议操作</span>
                <el-checkbox-group v-model="selectedActions" :disabled="plan.status === 'applied'" @change="normalizeActionSelection">
                  <el-checkbox v-for="action in plan.proposed_actions" :key="action.key" :value="action.key" class="action-option">
                    <span><strong>{{ action.title }}</strong><small>{{ actionKindLabel(action.kind) }} · {{ action.description }}<template v-if="action.due_at"> · {{ formatDate(action.due_at) }}</template></small></span>
                  </el-checkbox>
                </el-checkbox-group>
              </section>
              <el-alert title="批准只会创建非公开项目、里程碑和 CMS 草稿；不会自动发布，也不会执行审批或外部命令。" type="info" :closable="false" show-icon />
              <el-button v-if="plan.status === 'ready'" type="primary" size="large" round :loading="approving" style="width: 100%" @click="approveActions">批准所选操作</el-button>
              <div v-else class="created-links">
                <RouterLink v-if="plan.project_id" to="/admin/projects"><el-button type="primary" plain round>查看已创建项目</el-button></RouterLink>
                <RouterLink v-if="plan.announcement_content_id" :to="`/admin/content/${plan.announcement_content_id}/edit`"><el-button type="primary" plain round>检查公告草稿</el-button></RouterLink>
              </div>
            </aside>
          </div>
          <div class="stage-actions"><el-button round @click="resetPlanner">策划另一场活动</el-button></div>
        </template>

        <div v-else class="generation-state failed">
          <h3>本次活动方案未生成</h3><p>{{ plan?.run.failure_message || '模型暂时不可用，请检查配置后重试。' }}</p><el-button type="primary" round @click="step = 1">返回并重试</el-button>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.activity-planner-page { display: grid; gap: 20px; }
.activity-heading { align-items: flex-end; }
.eyebrow, .stage-heading span, .block-label { color: var(--md-sys-color-primary); font-size: .74rem; font-weight: 800; letter-spacing: .12em; }
.planner-shell { overflow: hidden; border: 1px solid var(--md-sys-color-outline-variant); border-radius: 28px; background: var(--md-sys-color-surface-container-low); }
.planner-steps { padding: 28px 34px; border-bottom: 1px solid var(--md-sys-color-outline-variant); background: var(--md-sys-color-surface-container); }
.planner-stage { padding: 32px; }
.stage-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; margin-bottom: 28px; }
.stage-heading h3 { margin: 5px 0 0; font-size: clamp(1.4rem, 2vw, 2rem); }
.stage-heading p { max-width: 600px; margin: 0; color: var(--md-sys-color-on-surface-variant); }
.brief-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 18px; }
.brief-grid .wide { grid-column: 1 / -1; }
.stage-actions { display: flex; justify-content: flex-end; margin-top: 20px; }
.stage-actions.spread { justify-content: space-between; }
.knowledge-search { display: grid; grid-template-columns: 1fr auto; gap: 12px; }
.knowledge-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin-top: 20px; }
.knowledge-card { position: relative; min-height: 150px; padding: 20px; text-align: left; color: inherit; border: 1px solid var(--md-sys-color-outline-variant); border-radius: 20px; background: var(--md-sys-color-surface-container); cursor: pointer; }
.knowledge-card.selected { border-color: var(--md-sys-color-primary); background: var(--md-sys-color-primary-container); }
.knowledge-card strong { display: block; padding-right: 32px; font-size: 1rem; }
.knowledge-card p { margin: 10px 0; color: var(--md-sys-color-on-surface-variant); display: -webkit-box; overflow: hidden; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }
.knowledge-card small { color: var(--md-sys-color-on-surface-variant); }
.knowledge-check { position: absolute; top: 16px; right: 16px; display: grid; width: 24px; height: 24px; place-items: center; border: 1px solid var(--md-sys-color-outline); border-radius: 50%; }
.selected .knowledge-check { color: var(--md-sys-color-on-primary); border-color: var(--md-sys-color-primary); background: var(--md-sys-color-primary); }
.selection-summary { margin-top: 18px; color: var(--md-sys-color-on-surface-variant); }
.result-grid { display: grid; grid-template-columns: minmax(0, 1.25fr) minmax(330px, .75fr); gap: 20px; align-items: start; }
.plan-preview, .plan-review { border: 1px solid var(--md-sys-color-outline-variant); border-radius: 22px; background: var(--md-sys-color-surface-container); }
.plan-preview { min-height: 600px; padding: 30px; }
.plan-review { position: sticky; top: 92px; display: grid; gap: 16px; padding: 22px; }
.review-block { display: grid; gap: 10px; }
.citation-item { padding: 12px; border-radius: 14px; background: var(--md-sys-color-surface-container-high); }
.citation-item p { margin: 6px 0; color: var(--md-sys-color-on-surface-variant); font-size: .88rem; }
.citation-item small { color: var(--md-sys-color-on-surface-variant); }
.action-option { width: 100%; height: auto; margin: 0 0 8px; padding: 12px; border: 1px solid var(--md-sys-color-outline-variant); border-radius: 14px; }
.action-option :deep(.el-checkbox__label) { min-width: 0; white-space: normal; }
.action-option span { display: grid; gap: 4px; }
.action-option small { color: var(--md-sys-color-on-surface-variant); line-height: 1.45; }
.generation-state { display: grid; min-height: 520px; place-items: center; align-content: center; text-align: center; }
.generation-state .el-icon { color: var(--md-sys-color-primary); font-size: 48px; }
.generation-state h3 { margin: 16px 0 4px; }
.generation-state p { max-width: 560px; color: var(--md-sys-color-on-surface-variant); }
.created-links { display: flex; flex-wrap: wrap; gap: 10px; }
.markdown-body :deep(h1) { margin-top: 0; }
.markdown-body :deep(h2) { margin-top: 28px; padding-bottom: 8px; border-bottom: 1px solid var(--md-sys-color-outline-variant); }
.markdown-body :deep(p), .markdown-body :deep(li) { line-height: 1.8; }
@media (max-width: 900px) {
  .result-grid { grid-template-columns: 1fr; }
  .plan-review { position: static; }
}
@media (max-width: 640px) {
  .planner-steps { padding: 20px 10px; }
  .planner-steps :deep(.el-step__description) { display: none; }
  .planner-stage { padding: 20px 16px; }
  .brief-grid, .knowledge-grid { grid-template-columns: 1fr; }
  .brief-grid .wide { grid-column: auto; }
  .stage-heading { flex-direction: column; }
  .knowledge-search { grid-template-columns: 1fr; }
  .stage-actions.spread { align-items: stretch; flex-direction: column-reverse; gap: 10px; }
  .stage-actions.spread .el-button { width: 100%; margin: 0; }
  .plan-preview { padding: 20px; }
}
</style>
