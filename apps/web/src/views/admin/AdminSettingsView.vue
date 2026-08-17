<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api/admin'
import { resolveApiUrl } from '@/api/client'
import { organizationSlug } from '@/api/portal'
import type { IntegrationSettings, IntegrationSettingsUpdate, InvitationTemplate, NotificationOutbox, Organization, PortalConfiguration, PortalManifest } from '@/api/types'
import { usePortalIdentity } from '@/composables/usePortalIdentity'
import { clearPortalFallback } from '@/portal/runtime'

const allowedCapabilities: Array<{ value: PortalManifest['capabilities'][number]; label: string }> = [
  { value: 'organization.read', label: '组织公开资料' },
  { value: 'public_content.read', label: '已发布内容' },
  { value: 'projects.read', label: '公开项目' },
  { value: 'assets.read', label: '公开资源' },
  { value: 'knowledge.read', label: '公开知识库' },
]

const organization = reactive<Pick<Organization, 'name' | 'short_name' | 'tagline' | 'introduction' | 'contact_email' | 'filing_number' | 'logo_asset_id' | 'logo_url' | 'social_links' | 'is_public'>>({
  name: '', short_name: '', tagline: '', introduction: '', contact_email: '', filing_number: '', logo_asset_id: '', logo_url: '', social_links: [], is_public: true,
})
const organizationLoading = ref(false)
const organizationSaving = ref(false)
const logoUploading = ref(false)
const localLogoPreview = ref('')
const initialLoading = ref(true)
const { setPortalOrganization } = usePortalIdentity()
const organizationLogoPreview = computed(() => localLogoPreview.value || (organization.logo_url ? resolveApiUrl(organization.logo_url) : ''))

function releaseLocalLogoPreview() {
  if (localLogoPreview.value.startsWith('blob:')) URL.revokeObjectURL(localLogoPreview.value)
  localLogoPreview.value = ''
}

async function uploadOrganizationLogo(file: File) {
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) {
    ElMessage.error('门户 Logo 仅支持 PNG、JPEG 或 WebP 图片。')
    return false
  }
  if (file.size > 10 * 1024 * 1024) {
    ElMessage.error('门户 Logo 不能超过 10 MB。')
    return false
  }
  logoUploading.value = true
  try {
    const asset = await adminApi.uploadAsset(file)
    releaseLocalLogoPreview()
    localLogoPreview.value = URL.createObjectURL(file)
    organization.logo_asset_id = asset.id
    organization.logo_url = ''
    ElMessage.success('Logo 已上传，请保存组织资料后应用到门户。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '门户 Logo 上传失败。')
  } finally {
    logoUploading.value = false
  }
  return false
}

function removeOrganizationLogo() {
  releaseLocalLogoPreview()
  organization.logo_asset_id = ''
  organization.logo_url = ''
}

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
    const saved = await adminApi.updateOrganization({
      name: organization.name,
      short_name: organization.short_name,
      tagline: organization.tagline,
      introduction: organization.introduction,
      contact_email: organization.contact_email,
      filing_number: organization.filing_number,
      logo_asset_id: organization.logo_asset_id,
      social_links: organization.social_links.map((link) => ({ label: link.label.trim(), href: link.href.trim() })),
      is_public: organization.is_public,
    })
    Object.assign(organization, saved)
    releaseLocalLogoPreview()
    if (saved.slug === organizationSlug) setPortalOrganization(saved)
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

const integrationSettings = ref<IntegrationSettings | null>(null)
const integrationLoading = ref(false)
const integrationSaving = ref(false)
const integrationTesting = ref<'email' | 'storage' | ''>('')
const browserOrigin = window.location.origin
const integrationForm = reactive<IntegrationSettingsUpdate>({
  public_web_base_url: window.location.origin,
  email: {
    driver: 'disabled', host: '', port: 587, username: '', password: '', clear_password: false,
    from_address: '', from_name: '', security: 'starttls', timeout_seconds: 8,
  },
  storage: {
    driver: 'local', endpoint: '', access_key: '', secret_key: '', clear_access_key: false,
    clear_secret_key: false, bucket: '', region: '', use_ssl: false,
  },
})

function applyIntegrationSettings(settings: IntegrationSettings) {
  integrationSettings.value = settings
  Object.assign(integrationForm, {
    public_web_base_url: settings.public_web_base_url,
    email: {
      driver: settings.email.driver, host: settings.email.host, port: settings.email.port,
      username: settings.email.username, password: '', clear_password: false,
      from_address: settings.email.from_address, from_name: settings.email.from_name,
      security: settings.email.security, timeout_seconds: settings.email.timeout_seconds,
    },
    storage: {
      driver: settings.storage.driver, endpoint: settings.storage.endpoint, access_key: '', secret_key: '',
      clear_access_key: false, clear_secret_key: false, bucket: settings.storage.bucket,
      region: settings.storage.region, use_ssl: settings.storage.use_ssl,
    },
  })
}

const invitationTemplate = reactive<InvitationTemplate>({ subject_template: '', body_template: '', variables: [] })
const invitationTemplateLoading = ref(false)
const invitationTemplateSaving = ref(false)
const notificationItems = ref<NotificationOutbox[]>([])
const notificationLoading = ref(false)
const invitationTemplateVariablesLabel = computed(() => invitationTemplate.variables.map((item) => `{{${item}}}`).join('、') || '加载中')

async function loadIntegrationSettings() {
	  integrationLoading.value = true
  try {
	    applyIntegrationSettings(await adminApi.getIntegrationSettings())
  } catch (error) {
	    ElMessage.error(error instanceof Error ? error.message : '服务接入配置加载失败。')
  } finally {
	    integrationLoading.value = false
  }
}

async function saveIntegrationSettings(showSuccess = true) {
  integrationSaving.value = true
  try {
    const payload: IntegrationSettingsUpdate = {
      public_web_base_url: integrationForm.public_web_base_url,
      email: { ...integrationForm.email },
      storage: { ...integrationForm.storage },
    }
    applyIntegrationSettings(await adminApi.updateIntegrationSettings(payload))
    if (showSuccess) ElMessage.success('服务接入配置已加密保存并立即生效。')
    return true
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '服务接入配置保存失败。')
    return false
  } finally {
    integrationSaving.value = false
  }
}

async function testIntegration(section: 'email' | 'storage') {
  if (!await saveIntegrationSettings(false)) return
  integrationTesting.value = section
  try {
    await adminApi.testIntegration(section)
    ElMessage.success(section === 'email' ? 'SMTP 连接与身份验证成功，未发送测试邮件。' : '存储连接成功，目标存储桶可访问。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '连接验证失败。')
  } finally {
    integrationTesting.value = ''
  }
}

async function loadInvitationTemplate() {
  invitationTemplateLoading.value = true
  try {
    Object.assign(invitationTemplate, await adminApi.getInvitationTemplate())
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '邀请模板加载失败。')
  } finally {
    invitationTemplateLoading.value = false
  }
}

async function saveInvitationTemplate() {
  invitationTemplateSaving.value = true
  try {
    Object.assign(invitationTemplate, await adminApi.updateInvitationTemplate({
      subject_template: invitationTemplate.subject_template,
      body_template: invitationTemplate.body_template,
    }))
    ElMessage.success('邀请邮件模板已保存。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '邀请模板保存失败。')
  } finally {
    invitationTemplateSaving.value = false
  }
}

async function loadNotifications() {
  notificationLoading.value = true
  try {
    notificationItems.value = (await adminApi.getNotificationOutbox({ page: 1, page_size: 8 })).items
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '通知队列加载失败。')
  } finally {
    notificationLoading.value = false
  }
}

async function retryNotification(item: NotificationOutbox) {
  try {
    const updated = await adminApi.retryNotification(item.id)
    const index = notificationItems.value.findIndex((current) => current.id === updated.id)
    if (index >= 0) notificationItems.value[index] = updated
    ElMessage.success('通知已重新排队。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '通知重试失败。')
  }
}

async function loadInitialSettings() {
  initialLoading.value = true
  try {
    await Promise.all([
      loadOrganization(),
      loadPortalConfiguration(),
      loadIntegrationSettings(),
      loadInvitationTemplate(),
      loadNotifications(),
    ])
  } finally {
    initialLoading.value = false
  }
}

onMounted(() => {
  void loadInitialSettings()
})

onBeforeUnmount(releaseLocalLogoPreview)
</script>

<template>
  <section class="admin-page-heading">
    <div>
      <h2>系统设置</h2>
      <p>管理组织资料、门户和外部服务接入；常用配置无需编辑服务器文件。</p>
    </div>
  </section>

  <section class="settings-layout" :aria-busy="initialLoading">
    <div class="settings-main-column">
	  <article v-loading="organizationLoading && !initialLoading" class="admin-panel">
		<div class="panel-heading">
		  <div>
			<h2>组织公开资料</h2>
			<p>这里的内容由 Portal API 公开读取，修改后立即生效并写入审计。</p>
		  </div>
		  <el-switch v-model="organization.is_public" inline-prompt active-text="公开" inactive-text="隐藏" />
		</div>
		<el-form :model="organization" label-position="top">
		  <el-form-item label="官网 Logo">
			<div class="organization-logo-editor">
			  <div class="organization-logo-preview" :class="{ empty: !organizationLogoPreview }">
				<img v-if="organizationLogoPreview" :src="organizationLogoPreview" alt="当前官网 Logo 预览" />
				<strong v-else>Q</strong>
			  </div>
			  <div class="organization-logo-actions">
				<div>
				  <el-upload accept="image/png,image/jpeg,image/webp" :show-file-list="false" :before-upload="uploadOrganizationLogo">
					<el-button type="primary" plain round :loading="logoUploading" aria-label="上传官网 Logo">上传图片</el-button>
				  </el-upload>
				  <el-button v-if="organization.logo_asset_id || organizationLogoPreview" text type="danger" @click="removeOrganizationLogo">移除 Logo</el-button>
				</div>
				<small class="field-help">支持 PNG、JPEG、WebP，最大 10 MB；建议使用方形透明底图片。上传或移除后需点击“保存组织资料”。</small>
			  </div>
			</div>
		  </el-form-item>
		  <div class="form-grid">
			<el-form-item label="组织全称" required><el-input v-model="organization.name" maxlength="160" show-word-limit /></el-form-item>
			<el-form-item label="组织简称" required><el-input v-model="organization.short_name" maxlength="40" show-word-limit /></el-form-item>
		  </div>
		  <el-form-item label="门户标语"><el-input v-model="organization.tagline" maxlength="160" show-word-limit /></el-form-item>
		  <el-form-item label="组织介绍"><el-input v-model="organization.introduction" type="textarea" :rows="5" maxlength="2000" show-word-limit /></el-form-item>
		  <el-form-item label="公开联系邮箱"><el-input v-model="organization.contact_email" type="email" placeholder="contact@example.org" /></el-form-item>
		  <el-form-item label="网站备案号"><el-input v-model="organization.filing_number" maxlength="80" show-word-limit placeholder="例如：鲁ICP备XXXXXXXX号-X" /></el-form-item>
		  <small class="filing-help">备案号按原样显示在门户页脚；留空时不展示。</small>
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

	  <article v-loading="loading && !initialLoading" class="admin-panel" style="margin-top: 20px;">
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

      <article v-loading="integrationLoading && !initialLoading" class="admin-panel integration-panel" style="margin-top: 20px;">
        <div class="panel-heading">
          <div>
            <h2>服务接入</h2>
            <p>在网页中管理门户地址、邮件和文件存储；敏感凭据只加密保存，不会回传明文。</p>
          </div>
          <el-tag :type="integrationSettings?.source === 'web' ? 'success' : 'info'" round>
            {{ integrationSettings?.source === 'web' ? '网页配置生效中' : '沿用部署默认值' }}
          </el-tag>
        </div>

        <div class="same-origin-summary">
          <div>
            <span>浏览器 API</span>
            <strong>{{ browserOrigin }}/api</strong>
            <small>由 Web 容器同源代理自动接入，无需填写服务器 IP。</small>
          </div>
          <RouterLink to="/admin/ai" class="integration-link">配置 AI 模型接口 →</RouterLink>
        </div>

        <el-form :model="integrationForm" label-position="top" class="integration-form">
          <el-form-item label="门户公网地址" required>
            <el-input v-model="integrationForm.public_web_base_url" placeholder="https://cms.example.org" />
            <small class="field-help">用于邀请邮件中的链接。可填当前网站域名，不要包含 /invite 等路径。</small>
          </el-form-item>

          <section class="adapter-card">
            <div class="adapter-heading">
              <div><h3>邮件投递（SMTP）</h3><p>用于成员邀请和申请审批结果通知。</p></div>
              <el-tag :type="integrationSettings?.email.configured ? 'success' : 'info'" round>
                {{ integrationForm.email.driver === 'smtp' ? (integrationSettings?.email.configured ? '配置完整' : '待完善') : '未启用' }}
              </el-tag>
            </div>
            <el-form-item label="邮件服务">
              <el-radio-group v-model="integrationForm.email.driver">
                <el-radio-button value="disabled">暂不发送</el-radio-button>
                <el-radio-button value="smtp">SMTP</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <template v-if="integrationForm.email.driver === 'smtp'">
              <div class="form-grid integration-grid">
                <el-form-item label="SMTP 地址" required><el-input v-model="integrationForm.email.host" placeholder="smtp.example.org" /></el-form-item>
                <el-form-item label="端口" required><el-input-number v-model="integrationForm.email.port" :min="1" :max="65535" controls-position="right" /></el-form-item>
              </div>
              <div class="form-grid integration-grid">
                <el-form-item label="连接安全">
                  <el-select v-model="integrationForm.email.security">
                    <el-option label="STARTTLS（常用）" value="starttls" />
                    <el-option label="TLS" value="tls" />
                    <el-option label="无加密（仅内网）" value="none" />
                  </el-select>
                </el-form-item>
                <el-form-item label="超时（秒）"><el-input-number v-model="integrationForm.email.timeout_seconds" :min="2" :max="60" controls-position="right" /></el-form-item>
              </div>
              <div class="form-grid integration-grid">
                <el-form-item label="登录用户名"><el-input v-model="integrationForm.email.username" autocomplete="username" /></el-form-item>
                <el-form-item label="登录密码">
                  <el-input v-model="integrationForm.email.password" type="password" show-password autocomplete="new-password" :placeholder="integrationSettings?.email.password_configured ? `留空保留 ${integrationSettings.email.password_hint ?? '已保存密码'}` : '请输入 SMTP 密码'" />
                  <el-checkbox v-if="integrationSettings?.email.password_configured" v-model="integrationForm.email.clear_password">清除已保存密码</el-checkbox>
                </el-form-item>
              </div>
              <div class="form-grid integration-grid">
                <el-form-item label="发件邮箱" required><el-input v-model="integrationForm.email.from_address" type="email" placeholder="noreply@example.org" /></el-form-item>
                <el-form-item label="发件人名称"><el-input v-model="integrationForm.email.from_name" placeholder="社团名称" /></el-form-item>
              </div>
              <el-button plain round :loading="integrationTesting === 'email'" :disabled="integrationSaving" @click="testIntegration('email')">保存并验证 SMTP</el-button>
            </template>
          </section>

          <section class="adapter-card">
            <div class="adapter-heading">
              <div><h3>图片与文件存储</h3><p>本地存储开箱即用；MinIO、阿里云 OSS 等可通过 S3 兼容接口接入。</p></div>
              <el-tag :type="integrationSettings?.storage.configured ? 'success' : 'warning'" round>
                {{ integrationSettings?.storage.configured ? '配置完整' : '待完善' }}
              </el-tag>
            </div>
            <el-form-item label="存储方式">
              <el-radio-group v-model="integrationForm.storage.driver">
                <el-radio-button value="local">服务器本地</el-radio-button>
                <el-radio-button value="s3">S3 / MinIO</el-radio-button>
              </el-radio-group>
              <small v-if="integrationForm.storage.driver === 'local'" class="field-help">目录由部署安全管理，管理员无需填写服务器路径。</small>
            </el-form-item>
            <template v-if="integrationForm.storage.driver === 's3'">
              <div class="form-grid integration-grid">
                <el-form-item label="服务地址" required><el-input v-model="integrationForm.storage.endpoint" placeholder="minio.example.org:9000" /></el-form-item>
                <el-form-item label="存储桶" required><el-input v-model="integrationForm.storage.bucket" placeholder="qutcraft-media" /></el-form-item>
              </div>
              <div class="form-grid integration-grid">
                <el-form-item label="Access Key" required>
                  <el-input v-model="integrationForm.storage.access_key" type="password" show-password autocomplete="off" :placeholder="integrationSettings?.storage.access_key_configured ? `留空保留 ${integrationSettings.storage.access_key_hint ?? '已保存凭据'}` : '请输入 Access Key'" />
                  <el-checkbox v-if="integrationSettings?.storage.access_key_configured" v-model="integrationForm.storage.clear_access_key">清除已保存 Access Key</el-checkbox>
                </el-form-item>
                <el-form-item label="Secret Key" required>
                  <el-input v-model="integrationForm.storage.secret_key" type="password" show-password autocomplete="new-password" :placeholder="integrationSettings?.storage.secret_key_configured ? `留空保留 ${integrationSettings.storage.secret_key_hint ?? '已保存凭据'}` : '请输入 Secret Key'" />
                  <el-checkbox v-if="integrationSettings?.storage.secret_key_configured" v-model="integrationForm.storage.clear_secret_key">清除已保存 Secret Key</el-checkbox>
                </el-form-item>
              </div>
              <div class="form-grid integration-grid">
                <el-form-item label="区域（可选）"><el-input v-model="integrationForm.storage.region" placeholder="us-east-1" /></el-form-item>
                <el-form-item label="连接协议"><el-switch v-model="integrationForm.storage.use_ssl" inline-prompt active-text="HTTPS" inactive-text="HTTP" /></el-form-item>
              </div>
              <el-button plain round :loading="integrationTesting === 'storage'" :disabled="integrationSaving" @click="testIntegration('storage')">保存并验证存储</el-button>
            </template>
          </section>

          <div class="integration-actions">
            <el-button type="primary" round :loading="integrationSaving" @click="saveIntegrationSettings()">保存服务接入</el-button>
            <el-button round :loading="integrationLoading" @click="loadIntegrationSettings">放弃修改并重新读取</el-button>
          </div>
        </el-form>

        <div v-if="integrationSettings" class="managed-runtime">
          <h3>部署维护项</h3>
          <p>以下项目决定进程能否启动，不能在正在运行的网页里热修改。</p>
          <div class="managed-runtime-grid">
            <div v-for="item in integrationSettings.managed_runtime" :key="item.key" class="managed-runtime-item">
              <div><strong>{{ item.label }}</strong><el-tag size="small" :type="item.state === 'deferred' ? 'warning' : 'info'">{{ item.state === 'deferred' ? '已延期' : '部署维护' }}</el-tag></div>
              <span>{{ item.description }}</span>
            </div>
          </div>
        </div>
      </article>

      <article v-loading="invitationTemplateLoading && !initialLoading" class="admin-panel" style="margin-top: 20px;">
        <div class="panel-heading">
          <div>
            <h2>邀请邮件模板</h2>
            <p>留空即可恢复默认内容。可用变量：{{ invitationTemplateVariablesLabel }}。</p>
          </div>
        </div>
        <el-form label-position="top">
          <el-form-item label="邮件主题"><el-input v-model="invitationTemplate.subject_template" maxlength="255" show-word-limit placeholder="加入 {{organization}} 的成员邀请" /></el-form-item>
          <el-form-item label="邮件正文"><el-input v-model="invitationTemplate.body_template" type="textarea" :rows="8" maxlength="4000" show-word-limit placeholder="你好：&#10;&#10;你收到了加入 {{organization}} 的成员邀请……" /></el-form-item>
          <el-button type="primary" round :loading="invitationTemplateSaving" @click="saveInvitationTemplate">保存邀请模板</el-button>
        </el-form>
      </article>

      <article v-loading="notificationLoading && !initialLoading" class="admin-panel" style="margin-top: 20px;">
        <div class="panel-heading">
          <div>
            <h2>通知队列</h2>
            <p>申请审批结果通过 outbox 异步发送；失败项可手动重新排队。</p>
          </div>
          <el-button text @click="loadNotifications">刷新</el-button>
        </div>
        <el-empty v-if="notificationItems.length === 0" description="暂无审批通知记录" :image-size="72" />
        <div v-else class="notification-list">
          <div v-for="item in notificationItems" :key="item.id" class="notification-row">
            <div>
              <strong>{{ item.recipient_email }}</strong>
              <span>{{ item.event_type }} · {{ item.attempts }} 次尝试</span>
            </div>
            <el-tag size="small" :type="item.status === 'sent' ? 'success' : item.status === 'failed' ? 'danger' : 'info'">{{ item.status }}</el-tag>
            <el-button v-if="item.status === 'failed' || item.status === 'disabled'" text type="primary" @click="retryNotification(item)">重试</el-button>
          </div>
        </div>
      </article>
    </div>

    <aside class="admin-panel settings-note">
      <h2>门户安全边界</h2>
      <p>Manifest 只能声明公开读取能力，不能访问成员、审批、审计或任何管理接口。</p>
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

.social-links-editor {
  display: grid;
  gap: 10px;
  width: 100%;
}

.organization-logo-editor {
  display: flex;
  align-items: center;
  gap: 18px;
  width: 100%;
  padding: 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 18px;
  background: var(--el-fill-color-light);
}

.organization-logo-preview {
  display: grid;
  flex: 0 0 88px;
  width: 88px;
  height: 88px;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  border-radius: 22px;
  background: var(--el-bg-color-overlay);
}

.organization-logo-preview.empty {
  color: var(--el-color-primary);
  background: linear-gradient(135deg, var(--el-color-primary-light-7), var(--el-color-warning-light-7));
  font-size: 32px;
}

.organization-logo-preview img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.organization-logo-actions {
  flex: 1;
  min-width: 0;
}

.organization-logo-actions > div {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.filing-help {
  display: block;
  margin: -8px 0 18px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.55;
}

.social-link-row {
  display: grid;
  grid-template-columns: minmax(140px, .4fr) minmax(240px, 1fr) auto;
  gap: 10px;
}

.same-origin-summary {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  margin-bottom: 22px;
  padding: 16px 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 16px;
  background: var(--el-fill-color-light);
}

.same-origin-summary span,
.same-origin-summary strong,
.same-origin-summary small {
  display: block;
}

.same-origin-summary span,
.same-origin-summary small,
.field-help,
.adapter-heading p,
.managed-runtime > p,
.managed-runtime-item span {
  color: var(--el-text-color-secondary);
}

.same-origin-summary strong {
  margin: 4px 0;
  overflow-wrap: anywhere;
}

.integration-link {
  flex: 0 0 auto;
  color: var(--el-color-primary);
  font-weight: 700;
  text-decoration: none;
}

.integration-form {
  display: grid;
  gap: 16px;
}

.adapter-card {
  padding: 20px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 18px;
  background: var(--el-fill-color-lighter);
}

.adapter-heading {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 14px;
  margin-bottom: 18px;
}

.adapter-heading h3,
.adapter-heading p,
.managed-runtime h3,
.managed-runtime > p {
  margin: 0;
}

.adapter-heading p,
.managed-runtime > p {
  margin-top: 5px;
}

.integration-grid :deep(.el-input-number),
.integration-grid :deep(.el-select) {
  width: 100%;
}

.field-help {
  display: block;
  width: 100%;
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.55;
}

.integration-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.integration-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.managed-runtime {
  margin-top: 26px;
  padding-top: 22px;
  border-top: 1px solid var(--el-border-color-light);
}

.managed-runtime-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.managed-runtime-item {
  padding: 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 14px;
  background: var(--el-fill-color-light);
}

.managed-runtime-item > div {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  align-items: center;
  margin-bottom: 7px;
}

.managed-runtime-item span {
  font-size: 12px;
  line-height: 1.6;
}

.notification-list {
  display: grid;
  gap: 8px;
}

.notification-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: 12px;
  align-items: center;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 14px;
  background: var(--el-fill-color-lighter);
}

.notification-row strong,
.notification-row span {
  display: block;
}

.notification-row span {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
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

  .organization-logo-editor {
    align-items: flex-start;
    flex-direction: column;
  }

  .same-origin-summary,
  .adapter-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .managed-runtime-grid {
    grid-template-columns: 1fr;
  }

  .adapter-card {
    padding: 16px;
  }

  .notification-row {
    grid-template-columns: 1fr auto;
  }

  .notification-row > :last-child {
    grid-column: 2;
  }
}
</style>
