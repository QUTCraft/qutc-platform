<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CircleCheck, CircleClose, CopyDocument, Delete, Document, FolderOpened, Picture, Refresh, UploadFilled } from '@element-plus/icons-vue'
import { adminApi } from '@/api/admin'
import { resolveApiUrl } from '@/api/client'
import type { AdminContent, IntegrationSettings, MediaAsset, Page } from '@/api/types'
import { session } from '@/stores/session'
import { formatBytes, formatDate } from '@/utils/format'

const maxAssetSize = 10 * 1024 * 1024
const acceptedExtensions = new Set(['.png', '.jpg', '.jpeg', '.webp', '.pdf', '.zip', '.mp4'])
const acceptedMimeTypes = new Set(['image/png', 'image/jpeg', 'image/webp', 'application/pdf', 'application/zip', 'video/mp4', 'application/x-zip-compressed'])

type UploadState = 'waiting' | 'uploading' | 'success' | 'error'
interface UploadTask {
  id: string
  file: File
  contentId?: string
  state: UploadState
  asset?: MediaAsset
  error?: string
}

const fileInput = ref<HTMLInputElement>()
const data = ref<Page<MediaAsset>>()
const contentItems = ref<AdminContent[]>([])
const integrationSettings = ref<IntegrationSettings | null>(null)
const loading = ref(true)
const refreshing = ref(false)
const loadError = ref<Error>()
const page = ref(1)
const query = ref('')
const selectedContentId = ref('')
const uploadQueue = ref<UploadTask[]>([])
const uploading = ref(false)
const dropActive = ref(false)
const deletingId = ref('')

const canManageAssets = computed(() => session.user?.roles.some((role) => role === 'owner' || role === 'administrator') ?? false)
const storageDriverLabel = computed(() => integrationSettings.value?.storage.driver === 's3' ? 'S3 / MinIO' : '服务器本地存储')
const storageConfigured = computed(() => integrationSettings.value?.storage.configured ?? true)
const storageDescription = computed(() => {
  const storage = integrationSettings.value?.storage
  if (!storage) return '沿用部署配置；上传仍由 QUTCraft API 统一代理。'
  if (storage.driver === 's3') return storage.endpoint ? `${storage.endpoint} · ${storage.bucket || '未填写存储桶'}` : '已选择 S3 / MinIO，请到系统设置完成连接信息。'
  return '文件保存到服务器维护的本地媒体目录。'
})
const waitingCount = computed(() => uploadQueue.value.filter((item) => item.state === 'waiting' || item.state === 'uploading').length)

function extensionOf(file: File) {
  const name = file.name.toLowerCase()
  const index = name.lastIndexOf('.')
  return index >= 0 ? name.slice(index) : ''
}

function isAllowedFile(file: File) {
  return acceptedMimeTypes.has(file.type) || acceptedExtensions.has(extensionOf(file))
}

function isImage(asset: MediaAsset) {
  return asset.mime_type.startsWith('image/')
}

function contentLabel(contentId?: string | null) {
  if (!contentId) return '未关联内容（仅后台）'
  return contentItems.value.find((item) => item.id === contentId)?.title ?? `内容 ${contentId.slice(0, 8)}`
}

async function loadAssets() {
  data.value = await adminApi.getAssets({ page: page.value, page_size: 20, query: query.value.trim() })
}

async function loadWorkspace() {
  loading.value = true
  loadError.value = undefined
  try {
    const [assetPage, contentPage] = await Promise.all([
      adminApi.getAssets({ page: page.value, page_size: 20, query: query.value.trim() }),
      adminApi.getContent({ page: 1, page_size: 100 }),
    ])
    data.value = assetPage
    contentItems.value = contentPage.items
    try {
      integrationSettings.value = await adminApi.getIntegrationSettings()
    } catch {
      // Editors can upload with asset:upload even when they cannot view the
      // organization-level integration settings. The storage card is optional.
      integrationSettings.value = null
    }
  } catch (error) {
    loadError.value = error instanceof Error ? error : new Error('媒体资源列表加载失败。')
  } finally {
    loading.value = false
  }
}

async function refreshAssets() {
  refreshing.value = true
  try {
    await loadAssets()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '媒体资源列表刷新失败。')
  } finally {
    refreshing.value = false
  }
}

async function changePage(value: number) {
  page.value = value
  await refreshAssets()
}

async function searchAssets() {
  page.value = 1
  await refreshAssets()
}

function openFilePicker() {
  fileInput.value?.click()
}

function queueFiles(files: File[]) {
  const accepted: UploadTask[] = []
  const rejected: string[] = []
  files.forEach((file, index) => {
    if (!isAllowedFile(file)) {
      rejected.push(`${file.name}：格式不在允许列表中`)
      return
    }
    if (file.size > maxAssetSize) {
      rejected.push(`${file.name}：超过 10 MB 限制`)
      return
    }
    accepted.push({
      id: `${Date.now()}-${index}-${file.name}`,
      file,
      contentId: selectedContentId.value || undefined,
      state: 'waiting',
    })
  })
  if (rejected.length) ElMessage.warning(rejected.slice(0, 3).join('；') + (rejected.length > 3 ? '；其余文件未加入队列。' : '。'))
  if (!accepted.length) return
  uploadQueue.value.push(...accepted)
  void processUploadQueue()
}

function handleFileInput(event: Event) {
  const input = event.target as HTMLInputElement
  queueFiles(Array.from(input.files ?? []))
  input.value = ''
}

function handleDrop(event: DragEvent) {
  dropActive.value = false
  queueFiles(Array.from(event.dataTransfer?.files ?? []))
}

async function processUploadQueue() {
  if (uploading.value) return
  uploading.value = true
  try {
    for (const task of uploadQueue.value) {
      if (task.state !== 'waiting') continue
      task.state = 'uploading'
      task.error = undefined
      try {
        task.asset = await adminApi.uploadAsset(task.file, task.contentId)
        task.state = 'success'
      } catch (error) {
        task.state = 'error'
        task.error = error instanceof Error ? error.message : '上传失败。'
      }
    }
    await refreshAssets()
    const successCount = uploadQueue.value.filter((item) => item.state === 'success').length
    const failedCount = uploadQueue.value.filter((item) => item.state === 'error').length
    if (successCount && !failedCount) ElMessage.success(`已上传 ${successCount} 个文件。`)
    else if (failedCount) ElMessage.warning(`上传完成：${successCount} 个成功，${failedCount} 个失败。`)
  } finally {
    uploading.value = false
  }
}

function clearCompleted() {
  uploadQueue.value = uploadQueue.value.filter((item) => item.state !== 'success')
}

function downloadHref(asset: MediaAsset) {
  return resolveApiUrl(asset.download_url)
}

async function copyDownloadLink(asset: MediaAsset) {
  const link = downloadHref(asset)
  try {
    if (navigator.clipboard?.writeText) await navigator.clipboard.writeText(link)
    else {
      const textarea = document.createElement('textarea')
      textarea.value = link
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      textarea.remove()
    }
    ElMessage.success('管理下载链接已复制。')
  } catch {
    ElMessage.error('复制失败，请直接使用下载按钮。')
  }
}

async function deleteAsset(asset: MediaAsset) {
  if (asset.content_id || !canManageAssets.value) return
  try {
    await ElMessageBox.confirm(`确定删除“${asset.original_name}”？删除后无法恢复。`, '删除媒体资源', { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' })
    deletingId.value = asset.id
    await adminApi.deleteAsset(asset.id)
    ElMessage.success('媒体资源已删除。')
    await refreshAssets()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error instanceof Error ? error.message : '媒体资源删除失败。')
  } finally {
    deletingId.value = ''
  }
}

function uploadStateLabel(state: UploadState) {
  return { waiting: '等待上传', uploading: '上传中', success: '已完成', error: '失败' }[state]
}

onMounted(loadWorkspace)
</script>

<template>
  <div v-if="loading" class="async-pending" role="status" aria-live="polite">资源工作区加载中</div>
  <el-result v-else-if="loadError" icon="error" title="资源工作区无法加载" :sub-title="loadError.message">
    <template #extra><el-button type="primary" round @click="loadWorkspace">重试</el-button></template>
  </el-result>
  <template v-else>
    <section class="admin-page-heading asset-page-heading">
      <div>
        <h2>资源文件</h2>
        <p>批量上传图片与文件，统一交给当前组织的媒体存储；MinIO 凭据只在服务端加密保存。</p>
      </div>
      <div class="asset-heading-actions">
        <el-button round :loading="refreshing" @click="refreshAssets"><el-icon><Refresh /></el-icon>刷新列表</el-button>
        <RouterLink to="/admin/content/new"><el-button type="primary" round>上传并写入内容</el-button></RouterLink>
      </div>
    </section>

    <section class="asset-workspace-grid">
      <article class="admin-panel asset-upload-panel">
        <div class="panel-heading">
          <div>
            <h2>快捷上传</h2>
            <p>支持 PNG、JPEG、WebP、PDF、ZIP、MP4，单文件最大 10 MB。</p>
          </div>
          <el-tag type="info" effect="plain" round>{{ waitingCount ? `${waitingCount} 个处理中` : '可批量选择' }}</el-tag>
        </div>

        <label
          class="asset-dropzone"
          :class="{ 'is-active': dropActive, 'is-busy': uploading }"
          @dragenter.prevent="dropActive = true"
          @dragover.prevent="dropActive = true"
          @dragleave.prevent="dropActive = false"
          @drop.prevent="handleDrop"
          @click="openFilePicker"
        >
          <span class="asset-dropzone-icon"><el-icon><UploadFilled /></el-icon></span>
          <strong>{{ uploading ? '正在上传文件…' : '拖拽文件到这里，或点击选择' }}</strong>
          <small>文件会经过 API 鉴权、类型检查和大小检查后写入媒体存储</small>
          <input ref="fileInput" class="visually-hidden" type="file" multiple accept="image/png,image/jpeg,image/webp,application/pdf,application/zip,video/mp4" @change="handleFileInput" />
        </label>

        <div class="asset-association-field">
          <label for="asset-content">关联门户内容（可选）</label>
          <el-select id="asset-content" v-model="selectedContentId" clearable filterable placeholder="不关联，仅保留在后台" style="width: 100%">
            <el-option v-for="item in contentItems" :key="item.id" :label="`${item.title} · ${item.status}`" :value="item.id" />
          </el-select>
          <small>关联并发布内容后，门户才会公开提供文件；不关联的文件只能从后台下载。</small>
        </div>

        <div v-if="uploadQueue.length" class="asset-upload-queue">
          <div class="asset-queue-heading">
            <strong>本次上传</strong>
            <el-button text size="small" :disabled="uploading" @click="clearCompleted">清除已完成</el-button>
          </div>
          <div v-for="task in uploadQueue" :key="task.id" class="asset-upload-row">
            <span class="asset-queue-file"><el-icon><Document /></el-icon><strong>{{ task.file.name }}</strong></span>
            <small>{{ formatBytes(task.file.size) }}</small>
            <el-tag v-if="task.state === 'success'" type="success" size="small"><el-icon><CircleCheck /></el-icon>{{ uploadStateLabel(task.state) }}</el-tag>
            <el-tag v-else-if="task.state === 'error'" type="danger" size="small" :title="task.error"><el-icon><CircleClose /></el-icon>{{ uploadStateLabel(task.state) }}</el-tag>
            <el-tag v-else size="small" effect="plain">{{ uploadStateLabel(task.state) }}</el-tag>
          </div>
        </div>
      </article>

      <article class="admin-panel asset-storage-panel">
        <div class="panel-heading">
          <div>
            <h2>当前存储</h2>
            <p>网页不直连 MinIO，所有文件都由 QUTCraft API 代理。</p>
          </div>
          <span class="storage-health-dot" :class="{ 'is-ready': storageConfigured }" />
        </div>
        <div class="storage-summary">
          <span class="storage-summary-icon"><el-icon><FolderOpened /></el-icon></span>
          <strong>{{ storageDriverLabel }}</strong>
          <span class="storage-state">{{ storageConfigured ? '配置已就绪' : '待完成配置' }}</span>
        </div>
        <p class="storage-description">{{ storageDescription }}</p>
        <ul class="asset-safety-list">
          <li>Access Key 与 Secret Key 永不回传浏览器。</li>
          <li>管理下载链接需要登录，并自动隔离组织数据。</li>
          <li>关联内容后，发布页会生成受控门户下载地址。</li>
        </ul>
        <RouterLink to="/admin/settings" class="storage-settings-link">
          <el-button plain round>到系统设置配置 MinIO <el-icon class="el-icon--right"><FolderOpened /></el-icon></el-button>
        </RouterLink>
      </article>
    </section>

    <section class="admin-panel asset-list-panel">
      <div class="panel-heading asset-list-heading">
        <div>
          <h2>已上传文件</h2>
          <p>共 {{ data?.total ?? 0 }} 个文件。删除仅对未关联内容的文件开放，避免误删门户资源。</p>
        </div>
        <el-form class="asset-search-form" @submit.prevent="searchAssets">
          <el-input v-model="query" clearable placeholder="搜索文件名" aria-label="搜索文件名" @keyup.enter="searchAssets" />
          <el-button type="primary" plain @click="searchAssets">搜索</el-button>
        </el-form>
      </div>

      <el-table v-if="data" :data="data.items" class="admin-table asset-table" empty-text="暂无上传文件">
        <el-table-column label="文件" min-width="300">
          <template #default="scope">
            <div class="asset-name-cell">
              <span class="asset-type-icon"><el-icon><Picture v-if="isImage(scope.row)" /><Document v-else /></el-icon></span>
              <div><strong>{{ scope.row.original_name }}</strong><small>{{ scope.row.mime_type }}</small></div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="大小" width="110"><template #default="scope">{{ formatBytes(scope.row.size_bytes) }}</template></el-table-column>
        <el-table-column label="关联内容" min-width="190"><template #default="scope"><span class="asset-content-label">{{ contentLabel(scope.row.content_id) }}</span></template></el-table-column>
        <el-table-column label="下载" width="90"><template #default="scope">{{ scope.row.download_count }}</template></el-table-column>
        <el-table-column label="上传时间" width="150"><template #default="scope">{{ scope.row.created_at ? formatDate(scope.row.created_at) : '—' }}</template></el-table-column>
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="scope">
            <a :href="downloadHref(scope.row)" class="asset-action-link" target="_blank" rel="noopener">下载</a>
            <el-button text type="primary" @click="copyDownloadLink(scope.row)"><el-icon><CopyDocument /></el-icon>复制链接</el-button>
            <el-button v-if="canManageAssets && !scope.row.content_id" text type="danger" aria-label="删除未关联文件" title="删除未关联文件" :loading="deletingId === scope.row.id" @click="deleteAsset(scope.row)"><el-icon><Delete /></el-icon></el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-if="data && data.total > data.page_size"
        class="application-pagination"
        background
        layout="total, prev, pager, next"
        :current-page="data.page"
        :page-size="data.page_size"
        :total="data.total"
        @current-change="changePage"
      />
    </section>
  </template>
</template>

<style scoped>
.asset-page-heading {
  align-items: center;
}

.asset-heading-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.asset-workspace-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) minmax(290px, 0.65fr);
  gap: 20px;
  margin-bottom: 20px;
}

.asset-upload-panel,
.asset-storage-panel,
.asset-list-panel {
  min-width: 0;
}

.asset-dropzone {
  display: flex;
  min-height: 220px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 28px;
  color: var(--md-sys-color-on-surface);
  background: var(--md-sys-color-surface-container-high);
  border: 1px dashed var(--md-sys-color-outline);
  border-radius: var(--md-shape-lg);
  cursor: pointer;
  text-align: center;
  transition: border-color 180ms ease, background-color 180ms ease, transform 180ms ease;
}

.asset-dropzone:hover,
.asset-dropzone.is-active {
  background: var(--md-sys-color-primary-container);
  border-color: var(--md-sys-color-primary);
  transform: translateY(-1px);
}

.asset-dropzone.is-busy {
  cursor: wait;
  opacity: 0.78;
}

.asset-dropzone-icon {
  display: grid;
  width: 52px;
  height: 52px;
  place-items: center;
  color: var(--md-sys-color-on-primary-container);
  background: var(--md-sys-color-primary);
  border-radius: 50%;
  font-size: 24px;
}

.asset-dropzone strong {
  font-size: 17px;
}

.asset-dropzone small,
.asset-association-field small,
.storage-description,
.asset-name-cell small {
  color: var(--md-sys-color-on-surface-variant);
}

.asset-association-field {
  display: grid;
  gap: 8px;
  margin-top: 20px;
}

.asset-association-field label {
  font-weight: 700;
}

.asset-upload-queue {
  display: grid;
  gap: 8px;
  margin-top: 20px;
  padding-top: 18px;
  border-top: 1px solid var(--md-sys-color-outline-variant);
}

.asset-queue-heading,
.asset-upload-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.asset-queue-heading {
  justify-content: space-between;
}

.asset-upload-row {
  min-width: 0;
  padding: 8px 10px;
  background: var(--md-sys-color-surface-container-high);
  border-radius: var(--md-shape-sm);
  font-size: 13px;
}

.asset-upload-row small {
  margin-left: auto;
  color: var(--md-sys-color-on-surface-variant);
}

.asset-queue-file {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 7px;
}

.asset-queue-file strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.storage-summary {
  display: grid;
  grid-template-columns: 42px 1fr auto;
  align-items: center;
  gap: 10px;
  padding: 16px;
  background: var(--md-sys-color-surface-container-high);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-md);
}

.storage-summary-icon {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  color: var(--md-sys-color-on-primary-container);
  background: var(--md-sys-color-primary);
  border-radius: var(--md-shape-sm);
}

.storage-state {
  color: var(--md-sys-color-primary);
  font-size: 12px;
  font-weight: 700;
}

.storage-health-dot {
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: var(--md-sys-color-error);
  box-shadow: 0 0 0 4px var(--md-sys-color-error-container);
}

.storage-health-dot.is-ready {
  background: var(--md-sys-color-primary);
  box-shadow: 0 0 0 4px var(--md-sys-color-primary-container);
}

.storage-description {
  min-height: 42px;
  margin: 14px 0;
  font-size: 13px;
  line-height: 1.7;
}

.asset-safety-list {
  display: grid;
  gap: 9px;
  margin: 0 0 20px;
  padding-left: 18px;
  color: var(--md-sys-color-on-surface-variant);
  font-size: 13px;
  line-height: 1.55;
}

.storage-settings-link {
  display: inline-flex;
}

.asset-list-heading {
  align-items: flex-end;
}

.asset-search-form {
  display: flex;
  width: min(360px, 100%);
  gap: 8px;
}

.asset-search-form .el-input {
  min-width: 0;
}

.asset-name-cell {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.asset-name-cell > div {
  display: grid;
  min-width: 0;
  gap: 4px;
}

.asset-name-cell strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-name-cell small {
  font-size: 12px;
}

.asset-type-icon {
  display: grid;
  flex: 0 0 36px;
  width: 36px;
  height: 36px;
  place-items: center;
  color: var(--md-sys-color-primary);
  background: var(--md-sys-color-primary-container);
  border-radius: var(--md-shape-sm);
}

.asset-content-label {
  color: var(--md-sys-color-on-surface-variant);
  font-size: 13px;
}

.asset-action-link {
  margin-right: 10px;
  color: var(--md-sys-color-primary);
  font-size: 13px;
  font-weight: 700;
}

.asset-list-panel :deep(.el-table) {
  --el-table-bg-color: transparent;
  --el-table-tr-bg-color: transparent;
  --el-table-row-hover-bg-color: var(--md-sys-color-surface-container-high);
  --el-table-header-bg-color: var(--md-sys-color-surface-container-high);
  --el-table-border-color: var(--md-sys-color-outline-variant);
  color: var(--md-sys-color-on-surface);
}

.asset-list-panel :deep(.el-table th.el-table__cell),
.asset-list-panel :deep(.el-table td.el-table__cell) {
  background: transparent;
  border-bottom-color: var(--md-sys-color-outline-variant);
}

.asset-list-panel :deep(.el-table th.el-table__cell) {
  background: var(--md-sys-color-surface-container-high);
  color: var(--md-sys-color-on-surface-variant);
}

@media (max-width: 900px) {
  .asset-workspace-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .asset-page-heading,
  .asset-list-heading {
    align-items: flex-start;
  }

  .asset-heading-actions,
  .asset-search-form {
    width: 100%;
  }

  .asset-heading-actions > *,
  .asset-search-form .el-button {
    flex: 1;
  }

  .asset-upload-row {
    flex-wrap: wrap;
  }

  .asset-upload-row small {
    margin-left: 0;
  }
}
</style>
