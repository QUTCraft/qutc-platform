<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api/admin'
import type { EmailAdapterStatus, Organization, PortalConfiguration, PortalManifest } from '@/api/types'
import { clearPortalFallback } from '@/portal/runtime'

const allowedCapabilities: Array<{ value: PortalManifest['capabilities'][number]; label: string }> = [
  { value: 'organization.read', label: '组织公开资料' },
  { value: 'public_content.read', label: '已发布内容' },
  { value: 'projects.read', label: '公开项目' },
  { value: 'assets.read', label: '公开资源' },
  { value: 'knowledge.read', label: '公开知识库' },
  { value: 'server.status.read', label: '脱敏服务器状态' },
]

const organization = reactive<Pick<Organization, 'name' | 'short_name' | 'tagline' | 'introduction' | 'contact_email' | 'social_links' | 'is_public'>>({
  name: '', short_name: '', tagline: '', introduction: '', contact_email: '', social_links: [], is_public: true,
})
const organizationLoading = ref(false)
const organizationSaving = ref(false)

async function loadOrganization() {
  organizationLoading.value = true
  try {
    Object.assign(organization, await adminApi.getOrganization())
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '组织资料加载失败。')
  } finally {
    organizationLoading.value = false
  }
}

async function saveOrganization() {
  organizationSaving.value = true
  try {
    Object.assign(organization, await adminApi.updateOrganization({
      ...organization,
      social_links: organization.social_links.map((link) => ({ label: link.label.trim(), href: link.href.trim() })),
    }))
    ElMessage.success('组织公开资料已保存并立即应用到门户。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '组织资料保存失败。')
  } finally {
    organizationSaving.value = false
  }
}

function addSocialLink() {
  if (organization.social_links.length < 12) organization.social_links.push({ label: '', href: '' })
}

function removeSocialLink(index: number) {
  organization.social_links.splice(index, 1)
}

const manifest = reactive<PortalManifest>({
  schema: 'qutc.portal/v1',
  id: 'qutcraft-md3',
  version: '0.1.0',
  display_name: 'QUTCraft MD3 Portal',
  entry: '/index.html',
  theme: { mode: 'md3' },
  capabilities: allowedCapabilities.map((item) => item.value),
  fallback: 'md3',
})
const configuration = ref<PortalConfiguration | null>(null)
const loading = ref(false)
const saving = ref(false)
const enabling = ref(false)
const restoring = ref(false)
const usesCustomPortal = computed(() => {
  const entry = configuration.value?.active_manifest?.entry
  return Boolean(entry && entry !== '/index.html' && entry !== '/')
})
const activeLabel = computed(() => configuration.value?.active_manifest
  ? `${configuration.value.active_manifest.display_name} · v${configuration.value.active_manifest.version}`
  : '默认 MD3 回退门户')

function applyManifest(value: PortalManifest) {
  Object.assign(manifest, value, {
    theme: { ...value.theme },
    capabilities: [...value.capabilities],
  })
}

async function loadPortalConfiguration() {
  loading.value = true
  try {
    configuration.value = await adminApi.getPortalConfiguration()
    if (configuration.value.draft_manifest) applyManifest(configuration.value.draft_manifest)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '门户配置加载失败。')
  } finally {
    loading.value = false
  }
}

async function savePortalDraft() {
  saving.value = true
  try {
    const payload: PortalManifest = {
      ...manifest,
      theme: manifest.theme.mode === 'custom'
        ? { mode: 'custom', tokens: manifest.theme.tokens?.trim() }
        : { mode: 'md3' },
      capabilities: [...manifest.capabilities],
      integrity: manifest.integrity?.trim() || undefined,
    }
    configuration.value = await adminApi.savePortalDraft(payload)
    ElMessage.success('门户草稿已保存，当前线上门户未改变。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '门户草稿保存失败。')
  } finally {
    saving.value = false
  }
}

async function enablePortal() {
  enabling.value = true
  try {
    configuration.value = await adminApi.enablePortalConfiguration()
    ElMessage.success('门户配置已启用。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '门户配置启用失败。')
  } finally {
    enabling.value = false
  }
}

async function restoreDefaultPortal() {
  try {
    await ElMessageBox.confirm(
      '这会保存并立即启用默认 MD3 Manifest，当前自定义门户将停止生效。已有配置记录不会被删除。',
      '恢复默认门户',
      { confirmButtonText: '恢复默认 MD3', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  restoring.value = true
  try {
    const defaultManifest = createDefaultManifest()
    configuration.value = await adminApi.restoreDefaultPortal()
    applyManifest(defaultManifest)
    clearPortalFallback()
    ElMessage.success('默认 MD3 门户已恢复并持久化。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '默认门户恢复失败。')
  } finally {
    restoring.value = false
  }
}

function previewPortal() {
  window.open(manifest.entry, '_blank', 'noopener,noreferrer')
}

function createDefaultManifest(): PortalManifest {
  return {
    schema: 'qutc.portal/v1',
    id: 'qutc-md3',
    version: '0.1.0',
    display_name: 'QUTCraft MD3 Portal',
    entry: '/index.html',
    theme: { mode: 'md3' },
    capabilities: allowedCapabilities.map((item) => item.value),
    fallback: 'md3',
  }
}

async function importManifest(file: File) {
  try {
    const value = JSON.parse(await file.text()) as PortalManifest
    applyManifest(value)
    ElMessage.success('Manifest 已载入编辑器，保存前不会影响线上门户。')
  } catch {
    ElMessage.error('文件不是有效的 Manifest JSON。')
  }
  return false
}

const emailStatus = ref<EmailAdapterStatus | null>(null)
const emailStatusLoading = ref(false)

async function loadEmailStatus() {
  emailStatusLoading.value = true
  try {
    emailStatus.value = await adminApi.getEmailAdapterStatus()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '邮件适配器状态加载失败。')
  } finally {
    emailStatusLoading.value = false
  }
}

onMounted(() => {
	void loadOrganization()
  void loadPortalConfiguration()
  void loadEmailStatus()
})
</script>

<template>
  <section class="admin-page-heading">
    <div>
      <h2>系统设置</h2>
	  <p>管理组织公开资料、门户配置，并检查服务端邀请邮件适配器的运行状态。</p>
    </div>
  </section>

  <section class="settings-layout">
    <div class="settings-main-column">
	  <article v-loading="organizationLoading" class="admin-panel">
		<div class="panel-heading">
		  <div>
			<h2>组织公开资料</h2>
			<p>这里的内容由 Portal API 公开读取，修改后立即生效并写入审计。</p>
		  </div>
		  <el-switch v-model="organization.is_public" inline-prompt active-text="公开" inactive-text="隐藏" />
		</div>
		<el-form :model="organization" label-position="top">
		  <div class="form-grid">
			<el-form-item label="组织全称" required><el-input v-model="organization.name" maxlength="160" show-word-limit /></el-form-item>
			<el-form-item label="组织简称" required><el-input v-model="organization.short_name" maxlength="40" show-word-limit /></el-form-item>
		  </div>
		  <el-form-item label="门户标语"><el-input v-model="organization.tagline" maxlength="160" show-word-limit /></el-form-item>
		  <el-form-item label="组织介绍"><el-input v-model="organization.introduction" type="textarea" :rows="5" maxlength="2000" show-word-limit /></el-form-item>
		  <el-form-item label="公开联系邮箱"><el-input v-model="organization.contact_email" type="email" placeholder="contact@example.org" /></el-form-item>
		  <el-form-item label="公开链接">
			<div class="social-links-editor">
			  <div v-for="(link, index) in organization.social_links" :key="index" class="social-link-row">
				<el-input v-model="link.label" maxlength="40" placeholder="名称，如 GitHub" />
				<el-input v-model="link.href" placeholder="https://..." />
				<el-button text type="danger" @click="removeSocialLink(index)">移除</el-button>
			  </div>
			  <el-button plain round :disabled="organization.social_links.length >= 12" @click="addSocialLink">添加链接</el-button>
			</div>
		  </el-form-item>
		  <el-button type="primary" round :loading="organizationSaving" @click="saveOrganization">保存组织资料</el-button>
		</el-form>
	  </article>

	  <article v-loading="loading" class="admin-panel" style="margin-top: 20px;">
        <div class="panel-heading">
          <div>
            <h2>门户 Manifest</h2>
            <p>编辑并保存草稿后，可先预览再单独启用；草稿不会改变当前线上门户。</p>
          </div>
          <el-tag :type="usesCustomPortal ? 'success' : 'info'" round>
            {{ usesCustomPortal ? '自定义门户生效中' : '默认 MD3 生效中' }}
          </el-tag>
        </div>
        <div class="portal-active-summary">
          <span>当前生效</span>
          <strong>{{ activeLabel }}</strong>
          <small v-if="configuration?.activated_at">
            {{ new Date(configuration.activated_at).toLocaleString('zh-CN') }}
          </small>
        </div>
        <el-form :model="manifest" label-position="top">
          <div class="form-grid">
            <el-form-item label="Schema">
              <el-input v-model="manifest.schema" disabled />
            </el-form-item>
            <el-form-item label="门户 ID">
              <el-input v-model="manifest.id" placeholder="qutcraft-md3" />
            </el-form-item>
          </div>
          <div class="form-grid">
            <el-form-item label="显示名称">
              <el-input v-model="manifest.display_name" />
            </el-form-item>
            <el-form-item label="版本（SemVer）">
              <el-input v-model="manifest.version" placeholder="0.1.0" />
            </el-form-item>
          </div>
          <el-form-item label="同源入口">
            <el-input v-model="manifest.entry" placeholder="/index.html" />
          </el-form-item>
          <el-form-item label="主题模式">
            <el-radio-group v-model="manifest.theme.mode">
              <el-radio-button value="md3">默认 MD3</el-radio-button>
              <el-radio-button value="custom">自定义 Token</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="manifest.theme.mode === 'custom'" label="主题 Token 同源路径">
            <el-input v-model="manifest.theme.tokens" placeholder="/portals/example/theme.json" />
          </el-form-item>
          <el-form-item label="公开能力">
            <el-checkbox-group v-model="manifest.capabilities" class="portal-capabilities">
              <el-checkbox v-for="item in allowedCapabilities" :key="item.value" :value="item.value">
                {{ item.label }}
              </el-checkbox>
            </el-checkbox-group>
          </el-form-item>
          <el-form-item label="资源完整性（可选）">
            <el-input v-model="manifest.integrity" placeholder="sha384-..." />
          </el-form-item>
          <div class="portal-config-actions">
            <el-button round @click="previewPortal">预览入口</el-button>
            <el-upload accept=".json,application/json" :show-file-list="false" :before-upload="importManifest">
              <el-button round>导入 Manifest</el-button>
            </el-upload>
            <el-button type="primary" round :loading="saving" @click="savePortalDraft">保存草稿</el-button>
            <el-button type="success" round :loading="enabling" :disabled="!configuration?.draft_manifest" @click="enablePortal">
              启用此草稿
            </el-button>
            <el-button type="warning" plain round :loading="restoring" @click="restoreDefaultPortal">
              恢复默认 MD3
            </el-button>
          </div>
        </el-form>
      </article>

      <article v-loading="emailStatusLoading" class="admin-panel" style="margin-top: 20px;">
        <div class="panel-heading">
          <div>
            <h2>邀请邮件投递</h2>
            <p>SMTP 凭据仅通过 API 服务环境变量配置，不会传输到浏览器或写入前端存储。</p>
          </div>
          <el-tag :type="emailStatus?.enabled ? 'success' : 'info'" round>
            {{ emailStatus?.enabled ? '已启用' : '未启用' }}
          </el-tag>
        </div>
        <el-alert
          v-if="emailStatus && !emailStatus.enabled"
          title="邮件未启用，成员邀请仍可通过复制链接完成"
          type="info"
          :closable="false"
          show-icon
        />
        <el-descriptions v-if="emailStatus" class="email-status-details" :column="1" border>
          <el-descriptions-item label="驱动">{{ emailStatus.driver }}</el-descriptions-item>
          <el-descriptions-item label="配置完整性">{{ emailStatus.configured ? '已通过启动校验' : '未配置' }}</el-descriptions-item>
          <el-descriptions-item v-if="emailStatus.from_address" label="发件人">
            {{ emailStatus.from_name ? `${emailStatus.from_name} · ` : '' }}{{ emailStatus.from_address }}
          </el-descriptions-item>
          <el-descriptions-item v-if="emailStatus.security" label="传输安全">{{ emailStatus.security }}</el-descriptions-item>
        </el-descriptions>
        <div class="email-status-actions">
          <el-button round :loading="emailStatusLoading" @click="loadEmailStatus">刷新状态</el-button>
          <span>修改部署环境变量并重启 API 后生效。</span>
        </div>
      </article>
    </div>

    <aside class="admin-panel settings-note">
      <h2>门户安全边界</h2>
      <p>Manifest 只能声明公开读取能力，不能访问成员、审批、审计、管理接口或服务器命令。</p>
      <p>入口和主题资源必须来自本站同源路径；加载失败时始终回退到默认 MD3 门户。</p>
      <p>预览只打开草稿声明的入口。只有点击“启用此草稿”后，新配置才会成为运行时生效版本。</p>
      <p>若自定义入口持续异常，可点击“恢复默认 MD3”永久恢复；访问 <code>/?portal=md3</code> 只对当前访问临时强制回退。</p>
    </aside>
  </section>
</template>

<style scoped>
.portal-active-summary {
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 12px;
  align-items: center;
  margin-bottom: 20px;
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 16px;
  background: var(--el-fill-color-light);
}

.portal-active-summary span,
.portal-active-summary small {
  color: var(--el-text-color-secondary);
}

.portal-capabilities {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  width: 100%;
}

.portal-config-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.portal-config-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.email-status-details {
  margin-top: 16px;
}

.social-links-editor {
  display: grid;
  gap: 10px;
  width: 100%;
}

.social-link-row {
  display: grid;
  grid-template-columns: minmax(140px, .4fr) minmax(240px, 1fr) auto;
  gap: 10px;
}

.email-status-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  margin-top: 16px;
  color: var(--el-text-color-secondary);
}

@media (max-width: 720px) {
  .portal-active-summary {
    grid-template-columns: 1fr;
  }

  .portal-capabilities {
    grid-template-columns: 1fr;
  }

  .social-link-row {
	grid-template-columns: 1fr;
  }
}
</style>
