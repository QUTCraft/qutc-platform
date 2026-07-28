<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api/admin'
import type { PortalConfiguration, PortalManifest, SmtpSettings } from '@/api/types'
import { clearPortalFallback } from '@/portal/runtime'

const allowedCapabilities: Array<{ value: PortalManifest['capabilities'][number]; label: string }> = [
  { value: 'organization.read', label: '组织公开资料' },
  { value: 'public_content.read', label: '已发布内容' },
  { value: 'projects.read', label: '公开项目' },
  { value: 'assets.read', label: '公开资源' },
  { value: 'knowledge.read', label: '公开知识库' },
  { value: 'server.status.read', label: '脱敏服务器状态' },
]

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

const smtp = reactive<SmtpSettings>({
  host: 'smtp.qutcraft.com',
  port: 465,
  sender_email: 'whitelist-bot@qutcraft.com',
  recipient_email: 'admin-whitelist@qutcraft.local',
  auth_code: '',
  enable_notification: true,
})

function saveSmtpSettings() {
  ElMessage.info('SMTP 持久化尚未接入；授权码不会在浏览器中保存。')
}

onMounted(loadPortalConfiguration)
</script>

<template>
  <section class="admin-page-heading">
    <div>
      <h2>系统设置</h2>
      <p>管理公开门户的基础属性以及新申请的邮件通知设置。</p>
    </div>
  </section>

  <section class="settings-layout">
    <div class="settings-main-column">
      <article v-loading="loading" class="admin-panel">
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

      <article class="admin-panel" style="margin-top: 20px;">
        <div class="panel-heading">
          <div>
            <h2>新申请邮件通知设置</h2>
          </div>
        </div>
        <el-form :model="smtp" label-position="top">
          <div class="form-grid">
            <el-form-item label="SMTP 服务器地址">
              <el-input v-model="smtp.host" placeholder="smtp.exmail.qq.com" />
            </el-form-item>
            <el-form-item label="端口">
              <el-input v-model.number="smtp.port" placeholder="465" />
            </el-form-item>
          </div>

          <div class="form-grid">
            <el-form-item label="发件人邮箱">
              <el-input v-model="smtp.sender_email" placeholder="noreply@qutcraft.com" />
            </el-form-item>
            <el-form-item label="管理员接收邮箱">
              <el-input v-model="smtp.recipient_email" placeholder="admin@qutcraft.com" />
            </el-form-item>
          </div>

          <el-form-item label="SMTP 授权码 / 密码">
            <el-input v-model="smtp.auth_code" type="password" show-password placeholder="输入授权码" />
          </el-form-item>

          <el-form-item>
            <el-switch v-model="smtp.enable_notification" active-text="当有新玩家申请加入时向管理员发送邮件通知" />
          </el-form-item>

          <el-button type="primary" round @click="saveSmtpSettings">保存邮件通知配置</el-button>
        </el-form>
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

@media (max-width: 720px) {
  .portal-active-summary {
    grid-template-columns: 1fr;
  }

  .portal-capabilities {
    grid-template-columns: 1fr;
  }
}
</style>
