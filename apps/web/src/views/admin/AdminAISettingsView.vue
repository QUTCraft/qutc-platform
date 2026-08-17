<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api/admin'
import type { AIAgentCatalog, AIConfiguration, AIConfigurationUpdate } from '@/api/types'
import { session } from '@/stores/session'

const configuration = ref<AIConfiguration | null>(null)
const catalog = ref<AIAgentCatalog | null>(null)
const loading = ref(false)
const saving = ref(false)

const form = reactive({
	provider: 'disabled' as AIConfigurationUpdate['provider'],
	base_url: '',
	api_key: '',
	model: '',
	enabled: true,
  run_limit_per_hour: 20,
  request_timeout_seconds: 30,
  max_sources: 10,
  max_context_characters: 30000,
})

const canEdit = computed(() => session.user?.roles.includes('owner') ?? false)
const provider = computed(() => configuration.value?.provider)
const providerLabel = computed(() => {
  if (provider.value?.mode === 'real') return '真实模型'
  return '未启用'
})
const providerTagType = computed<'success' | 'warning' | 'info'>(() => {
  if (provider.value?.mode === 'real') return 'success'
  return 'info'
})
const providerDisplay = computed(() => provider.value?.mode === 'real' ? 'OpenAI Compatible' : '未启用')

function applyConfiguration(value: AIConfiguration) {
  configuration.value = value
	Object.assign(form, {
		provider: value.provider_config?.driver === 'openai_compatible' ? 'openai_compatible' : 'disabled',
		base_url: value.provider_config?.base_url ?? '',
		api_key: '',
		model: value.provider_config?.model ?? value.provider.model ?? '',
		enabled: value.enabled,
    run_limit_per_hour: value.run_limit_per_hour,
    request_timeout_seconds: value.request_timeout_seconds,
    max_sources: value.max_sources,
    max_context_characters: value.max_context_characters,
  })
}

async function loadConfiguration() {
  loading.value = true
  try {
    const [nextConfiguration, nextCatalog] = await Promise.all([
      adminApi.getAIConfiguration(),
      adminApi.getAIAgents(),
    ])
    applyConfiguration(nextConfiguration)
    catalog.value = nextCatalog
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '智能体配置加载失败。')
  } finally {
    loading.value = false
  }
}

async function saveConfiguration() {
  if (!canEdit.value) {
    ElMessage.warning('只有组织所有者可以修改智能体策略。')
    return
  }
  if (configuration.value?.enabled && !form.enabled) {
    try {
      await ElMessageBox.confirm(
        '停用后，新建智能体运行会立即被拒绝；已有终态记录不会删除。',
        '停用组织智能体',
        { confirmButtonText: '确认停用', cancelButtonText: '取消', type: 'warning' },
      )
    } catch {
      form.enabled = true
      return
    }
  }
  saving.value = true
  try {
    const updated = await adminApi.updateAIConfiguration({ ...form })
    applyConfiguration(updated)
    ElMessage.success('智能体策略已保存，并将应用于后续运行。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '智能体配置保存失败。')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void loadConfiguration()
})
</script>

<template>
  <section class="admin-page-heading">
    <div>
      <h2>智能体配置</h2>
      <p>管理组织级运行策略，检查模型供应商状态；模型凭据始终留在服务端。</p>
    </div>
    <el-button round :loading="loading" @click="loadConfiguration">刷新状态</el-button>
  </section>

  <section class="settings-layout ai-settings-layout" :aria-busy="loading">
    <div class="settings-main-column">
      <article class="admin-panel">
        <div class="panel-heading">
          <div>
            <h2>模型供应商</h2>
            <p>这里显示当前组织实际使用的驱动；API Key 只显示脱敏状态，不会返回原文。</p>
          </div>
          <el-tag :type="providerTagType" round>{{ providerLabel }}</el-tag>
        </div>

        <el-form :model="form" label-position="top" class="ai-provider-form">
          <div class="ai-provider-form-heading">
            <div>
              <strong>网页端连接配置</strong>
              <p>组织所有者可以在这里填写 OpenAI 兼容接口。服务端会自动请求接口根地址下的 <code>/chat/completions</code>。</p>
            </div>
            <el-tag v-if="configuration?.provider_config?.source === 'organization'" type="success" round>组织配置</el-tag>
            <el-tag v-else round>服务端默认</el-tag>
          </div>
          <div class="form-grid">
            <el-form-item label="模型驱动">
              <el-select v-model="form.provider" :disabled="!canEdit" class="full-width-control">
                <el-option label="未启用" value="disabled" />
                <el-option label="OpenAI 兼容接口" value="openai_compatible" />
              </el-select>
              <small>生产页面只允许停用或接入真实的 OpenAI 兼容接口。</small>
            </el-form-item>
            <el-form-item label="模型名称">
              <el-input v-model="form.model" :disabled="!canEdit || form.provider !== 'openai_compatible'" placeholder="例如：agnes-2.0-flash" />
              <small>填写供应商实际支持的 model 标识。</small>
            </el-form-item>
          </div>
          <el-form-item label="API 调用链接">
            <el-input v-model="form.base_url" :disabled="!canEdit || form.provider !== 'openai_compatible'" placeholder="例如：https://api.example.com/v1" />
            <small>只填写接口根地址，不要追加 <code>/chat/completions</code>；生产环境必须使用 HTTPS。</small>
          </el-form-item>
          <el-form-item label="API Key">
            <el-input
              v-model="form.api_key"
              :disabled="!canEdit || form.provider !== 'openai_compatible'"
              type="password"
              show-password
              autocomplete="new-password"
              :placeholder="configuration?.provider_config?.api_key_configured ? `已配置 ${configuration.provider_config.api_key_hint || '密钥'}，留空保持不变` : '请输入供应商 API Key'"
            />
            <small>密钥只在提交时发送到 API，服务端使用应用密钥加密保存；GET 接口永远不会返回原文。</small>
          </el-form-item>
        </el-form>
        <el-alert
          v-if="!provider?.enabled"
          title="模型供应商未启用；可以保存组织策略，但创建运行会返回 503"
          type="info"
          :closable="false"
          show-icon
        />

        <div class="ai-provider-metrics">
          <div>
            <span>驱动</span>
            <strong>{{ providerDisplay }}</strong>
          </div>
          <div>
            <span>模型</span>
            <strong>{{ provider?.model || '未配置' }}</strong>
          </div>
          <div>
            <span>启动校验</span>
            <strong>{{ provider?.configured ? '已通过' : '未通过' }}</strong>
          </div>
        </div>

        <div class="deployment-note">
          <div>
            <strong>配置优先级</strong>
            <p>当前组织的网页配置优先于服务端环境变量；留空 API Key 会保留已保存的密钥。</p>
          </div>
          <div class="deployment-variables" aria-label="模型服务端环境变量">
            <code>AI_PROVIDER</code>
            <code>AI_BASE_URL</code>
            <code>AI_API_KEY</code>
            <code>AI_MODEL</code>
          </div>
        </div>
      </article>

      <article class="admin-panel ai-policy-panel">
        <div class="panel-heading">
          <div>
            <h2>组织运行策略</h2>
            <p>配置保存在当前组织范围内，并在每次创建运行时重新读取。</p>
          </div>
          <el-tag :type="form.enabled ? 'success' : 'info'" round>
            {{ form.enabled ? '允许新运行' : '已停用' }}
          </el-tag>
        </div>

        <el-alert
          v-if="!canEdit"
          title="当前账户可以查看配置，但只有组织所有者可以保存修改"
          type="info"
          :closable="false"
          show-icon
        />

        <el-form :model="form" label-position="top" class="ai-policy-form">
          <div class="ai-enable-row">
            <div>
              <strong>启用组织智能体</strong>
              <p>关闭后拒绝新运行，不删除历史结果、引用或审计。</p>
            </div>
            <el-switch v-model="form.enabled" :disabled="!canEdit" />
          </div>

          <div class="form-grid">
            <el-form-item label="每用户每小时运行额度">
              <el-input-number
                v-model="form.run_limit_per_hour"
                :min="1"
                :max="200"
                controls-position="right"
                :disabled="!canEdit"
              />
              <small>按当前用户与当前组织统计，范围 1—200。</small>
            </el-form-item>
            <el-form-item label="单次模型请求超时">
              <el-input-number
                v-model="form.request_timeout_seconds"
                :min="5"
                :max="120"
                controls-position="right"
                :disabled="!canEdit"
              />
              <small>单位为秒，超时后运行进入 failed。</small>
            </el-form-item>
          </div>

          <div class="form-grid">
            <el-form-item label="单次最多引用资料">
              <el-input-number
                v-model="form.max_sources"
                :min="1"
                :max="10"
                controls-position="right"
                :disabled="!canEdit"
              />
              <small>只允许当前组织中有权读取的知识内容。</small>
            </el-form-item>
            <el-form-item label="单次上下文正文上限">
              <el-input-number
                v-model="form.max_context_characters"
                :min="1000"
                :max="100000"
                :step="1000"
                controls-position="right"
                :disabled="!canEdit"
              />
              <small>按字符计，范围 1,000—100,000。</small>
            </el-form-item>
          </div>

          <div class="ai-policy-actions">
            <el-button type="primary" round :loading="saving" :disabled="!canEdit" @click="saveConfiguration">
              保存智能体配置
            </el-button>
            <span v-if="configuration?.updated_at">
              最近更新：{{ new Date(configuration.updated_at).toLocaleString('zh-CN') }}
            </span>
          </div>
        </el-form>
      </article>

      <article class="admin-panel">
        <div class="panel-heading">
          <div>
            <h2>可用智能体</h2>
            <p>当前定义由服务端管理，工具白名单不会由页面或模型自行扩展。</p>
          </div>
          <el-tag round>{{ catalog?.agents.length ?? 0 }} 个</el-tag>
        </div>
        <div class="agent-definition-list">
          <section v-for="agent in catalog?.agents ?? []" :key="agent.id" class="agent-definition-card">
            <div>
              <span class="agent-key">{{ agent.key }}</span>
              <h3>{{ agent.name }}</h3>
              <p>{{ agent.purpose }}</p>
            </div>
            <dl>
              <div>
                <dt>策略版本</dt>
                <dd>{{ agent.system_policy_version }}</dd>
              </div>
              <div>
                <dt>模型档案</dt>
                <dd>{{ agent.model_profile }}</dd>
              </div>
            </dl>
            <div class="agent-tools">
              <span>允许工具</span>
              <el-tag v-for="tool in agent.allowed_tool_keys" :key="tool" size="small" effect="plain">
                {{ tool }}
              </el-tag>
            </div>
          </section>
          <el-empty v-if="catalog && catalog.agents.length === 0" description="当前组织没有启用的智能体定义" />
        </div>
      </article>
    </div>

    <aside class="admin-panel settings-note ai-security-note">
      <h2>配置边界</h2>
      <p>模型地址、模型名和 API Key 可以由组织所有者在本页配置；API Key 会在服务端加密保存，页面只显示脱敏状态。</p>
      <p>组织策略只影响后续运行；运行创建后会固定引用版本、模型模式和 Prompt 版本。</p>
      <p>有效权限仍是用户 RBAC 与智能体工具白名单的交集。关闭限制不会授予发布、审批、角色修改或外部执行权限。</p>
      <p>生产页面不提供模拟模型入口；停用供应商时不会删除已有运行记录。</p>
    </aside>
  </section>
</template>

<style scoped>
.ai-settings-layout {
  align-items: start;
}

.settings-main-column {
  display: grid;
  gap: 20px;
}

.ai-provider-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-top: 18px;
}

.ai-provider-form {
  margin-top: 20px;
  padding-top: 18px;
  border-top: 1px solid var(--md-sys-color-outline-variant);
}

.ai-provider-form-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 16px;
}

.ai-provider-form-heading p {
  margin: 5px 0 0;
  color: var(--md-sys-color-on-surface-variant);
  font-size: 13px;
  line-height: 1.55;
}

.ai-provider-form :deep(.el-form-item) {
  margin-bottom: 16px;
}

.ai-provider-form :deep(.el-input),
.ai-provider-form :deep(.el-select) {
  width: 100%;
}

.full-width-control {
  width: 100%;
}

.ai-provider-metrics > div {
  display: grid;
  gap: 6px;
  padding: 16px;
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-lg);
  background: var(--md-sys-color-surface-container-low);
}

.ai-provider-metrics span,
.ai-policy-actions span {
  color: var(--md-sys-color-on-surface-variant);
  font-size: 12px;
}

.ai-provider-metrics strong {
  overflow-wrap: anywhere;
}

.deployment-note {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-top: 16px;
  padding: 16px;
  border-radius: var(--md-shape-lg);
  background: var(--md-sys-color-surface-container);
}

.deployment-note p,
.ai-enable-row p {
  margin: 4px 0 0;
  color: var(--md-sys-color-on-surface-variant);
  font-size: 13px;
}

.deployment-variables {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

.deployment-variables code,
.ai-security-note code {
  padding: 4px 8px;
  border-radius: var(--md-shape-sm);
  color: var(--md-sys-color-primary);
  background: var(--md-sys-color-primary-container);
  font-size: 11px;
}

.ai-policy-panel {
  position: relative;
}

.ai-policy-form {
  margin-top: 18px;
}

.ai-enable-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 20px;
  padding: 16px 18px;
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-lg);
  background: var(--md-sys-color-surface-container-low);
}

.ai-policy-form :deep(.el-input-number) {
  width: 100%;
}

.ai-policy-form small {
  display: block;
  margin-top: 7px;
  color: var(--md-sys-color-on-surface-variant);
  line-height: 1.45;
}

.ai-policy-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}

.agent-definition-list {
  display: grid;
  gap: 12px;
}

.agent-definition-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(220px, 0.42fr);
  gap: 18px 24px;
  padding: 18px;
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-lg);
  background: var(--md-sys-color-surface-container-low);
}

.agent-key {
  color: var(--md-sys-color-primary);
  font: 800 11px var(--md-font-mono);
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.agent-definition-card h3 {
  margin: 6px 0;
  font-size: 18px;
}

.agent-definition-card p {
  margin: 0;
  color: var(--md-sys-color-on-surface-variant);
  line-height: 1.6;
}

.agent-definition-card dl {
  display: grid;
  gap: 10px;
  margin: 0;
}

.agent-definition-card dl div {
  display: grid;
  gap: 3px;
}

.agent-definition-card dt {
  color: var(--md-sys-color-on-surface-variant);
  font-size: 11px;
}

.agent-definition-card dd {
  margin: 0;
  font-weight: 700;
}

.agent-tools {
  display: flex;
  flex-wrap: wrap;
  grid-column: 1 / -1;
  align-items: center;
  gap: 7px;
  padding-top: 14px;
  border-top: 1px solid var(--md-sys-color-outline-variant);
}

.agent-tools > span {
  margin-right: 4px;
  color: var(--md-sys-color-on-surface-variant);
  font-size: 12px;
}

@media (max-width: 760px) {
  .ai-provider-metrics,
  .agent-definition-card {
    grid-template-columns: 1fr;
  }

  .deployment-note {
    align-items: flex-start;
    flex-direction: column;
  }

  .deployment-variables {
    justify-content: flex-start;
  }
}
</style>
