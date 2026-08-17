<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CircleCheck, CircleClose, CopyDocument, Delete, Document, FolderOpened, Picture, Refresh, UploadFilled } from '@element-plus/icons-vue'
import { adminApi } from '@/api/admin'
import { resolveApiUrl } from '@/api/client'
import type { AdminContent, IntegrationSettings, MediaAsset, Page, Resource } from '@/api/types'
import { usePortalIdentity } from '@/composables/usePortalIdentity'
import { session } from '@/stores/session'
import { formatBytes, formatDate } from '@/utils/format'

const maxAssetSize = 10 * 1024 * 1024
const acceptedExtensions = new Set(['.png', '.jpg', '.jpeg', '.webp', '.pdf', '.zip', '.mp4'])
const acceptedMimeTypes = new Set(['image/png', 'image/jpeg', 'image/webp', 'application/pdf', 'application/zip', 'video/mp4', 'application/x-zip-compressed'])
const { clearPortalOrganization } = usePortalIdentity()

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
const archivingId = ref('')
const republishingId = ref('')
const publishDialogVisible = ref(false)
const publishTarget = ref<MediaAsset>()
const publishing = ref(false)
const publishForm = ref<{ title: string; kind: Resource['kind']; description: string }>({
  title: '',
  kind: 'document',
  description: '',
})

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

function linkedContent(asset: MediaAsset) {
  if (!asset.content_id) return undefined
  return contentItems.value.find((item) => item.id === asset.content_id)
}

function syncContent(content: AdminContent) {
  const existingIndex = contentItems.value.findIndex((item) => item.id === content.id)
  const nextContent = { ...content }
  if (existingIndex >= 0) {
    contentItems.value = contentItems.value.map((item, index) => index === existingIndex ? nextContent : item)
  } else {
    contentItems.value = [nextContent, ...contentItems.value]
  }
}

async function resolveLinkedContent(asset: MediaAsset) {
  if (!asset.content_id) return undefined
  const cached = linkedContent(asset)
  if (cached) return cached
  const content = await adminApi.getContentById(asset.content_id)
  syncContent(content)
  return content
}

function publicResourcePath(asset: MediaAsset) {
  const content = linkedContent(asset)
  return content?.type === 'resource' && content.status === 'published' ? `/resources/${content.id}` : ''
}

function contentStatusLabel(status?: AdminContent['status']) {
  return status ? ({ draft: '草稿', review: '待审核', published: '已公开', archived: '已下线' }[status]) : ''
}

function inferredKind(asset: MediaAsset): Resource['kind'] {
  if (asset.mime_type.startsWith('video/')) return 'video'
  if (asset.mime_type.includes('zip')) return 'package'
  return 'document'
}

function titleFromFilename(filename: string) {
  return filename.replace(/\.[^.]+$/, '').trim() || filename
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
    if (selectedContentId.value && uploadQueue.value.some((item) => item.state === 'success' && item.contentId === selectedContentId.value)) {
      selectedContentId.value = ''
    }
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
  if (!canManageAssets.value) return
  try {
    const content = await resolveLinkedContent(asset)
    if (content?.status === 'published') {
      ElMessage.warning('文件仍在门户公开，请先下架后再删除。')
      return
    }
    const linkedNotice = content?.type === 'resource'
      ? '若这是该门户资源的最后一个文件，对应的非公开资源记录和版本历史也会一并删除。'
      : content
        ? '关联的文章或知识记录会保留为非公开状态，重新发布前需要上传新的文件。'
        : ''
    await ElMessageBox.confirm(`确定永久删除“${asset.original_name}”？MinIO / 本地存储中的文件也会被删除，无法恢复。${linkedNotice}`, '永久删除文件', { type: 'warning', confirmButtonText: '永久删除', cancelButtonText: '取消' })
    deletingId.value = asset.id
    const removed = await adminApi.deleteAsset(asset.id)
    if (removed.cleared_logo) clearPortalOrganization()
    uploadQueue.value = uploadQueue.value.filter((task) => task.asset?.id !== removed.id)
    if (removed.removed_content_id) {
      contentItems.value = contentItems.value.filter((item) => item.id !== removed.removed_content_id)
      if (selectedContentId.value === removed.removed_content_id) selectedContentId.value = ''
    }
    if (removed.detached_content_id) {
      contentItems.value = contentItems.value.map((item) => item.id === removed.detached_content_id ? { ...item, asset: null } : item)
      if (selectedContentId.value === removed.detached_content_id) selectedContentId.value = ''
    }
    ElMessage.success(removed.cleared_logo
      ? '文件已永久删除，官网 Logo 已恢复为默认标识。'
      : removed.removed_content_id
        ? '文件及对应门户资源记录已永久删除。'
        : '文件及其存储对象已永久删除。')
    await refreshAssets()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error instanceof Error ? error.message : '媒体资源删除失败。')
  } finally {
    deletingId.value = ''
  }
}

async function archiveAsset(asset: MediaAsset) {
  if (!canManageAssets.value) return
  try {
    const content = await resolveLinkedContent(asset)
    if (!content || content.status !== 'published') return
    await ElMessageBox.confirm(`下架“${content.title}”后，门户将立即停止展示和提供公开下载，原文件仍会保留。`, '下架门户资源', { type: 'warning', confirmButtonText: '确认下架', cancelButtonText: '取消' })
    archivingId.value = asset.id
    syncContent(await adminApi.archiveContent(content.id))
    ElMessage.success('资源已下架，文件仍保留在后台。')
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error instanceof Error ? error.message : '资源下架失败。')
  } finally {
    archivingId.value = ''
  }
}

async function republishAsset(asset: MediaAsset) {
  if (!canManageAssets.value) return
  try {
    const content = await resolveLinkedContent(asset)
    if (!content || content.status === 'published') return
    republishingId.value = asset.id
    syncContent(await adminApi.publishContent(content.id))
    ElMessage.success('资源已重新上架。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '资源上架失败。')
  } finally {
    republishingId.value = ''
  }
}

function openPublishDialog(asset: MediaAsset) {
  if (asset.content_id || !canManageAssets.value) return
  publishTarget.value = asset
  publishForm.value = {
    title: titleFromFilename(asset.original_name),
    kind: inferredKind(asset),
    description: `公开文件：${asset.original_name}`,
  }
  publishDialogVisible.value = true
}

async function publishAsset() {
  const asset = publishTarget.value
  if (!asset) return
  if (!publishForm.value.title.trim()) {
    ElMessage.warning('请填写门户中展示的资源标题。')
    return
  }
  publishing.value = true
  try {
    const content = await adminApi.publishAssetAsResource(asset.id, {
      title: publishForm.value.title.trim(),
      kind: publishForm.value.kind,
      description: publishForm.value.description.trim(),
    })
    syncContent(content)
    await refreshAssets()
    publishDialogVisible.value = false
    publishTarget.value = undefined
    ElMessage.success('文件已归档并发布到门户资源中心。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '资源归档发布失败。')
  } finally {
    publishing.value = false
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
          <p>共 {{ data?.total ?? 0 }} 个文件。公开资源需先下架；下架后可保留内容记录并永久删除存储文件。</p>
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
        <el-table-column label="归档状态" min-width="210">
          <template #default="scope">
            <div class="asset-content-state">
              <RouterLink v-if="publicResourcePath(scope.row)" :to="publicResourcePath(scope.row)">{{ contentLabel(scope.row.content_id) }}</RouterLink>
              <span v-else class="asset-content-label">{{ contentLabel(scope.row.content_id) }}</span>
              <small v-if="linkedContent(scope.row)">{{ contentStatusLabel(linkedContent(scope.row)?.status) }}</small>
              <small v-else>尚未公开</small>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="下载" width="90"><template #default="scope">{{ scope.row.download_count }}</template></el-table-column>
        <el-table-column label="上传时间" width="150"><template #default="scope">{{ scope.row.created_at ? formatDate(scope.row.created_at) : '—' }}</template></el-table-column>
        <el-table-column label="操作" width="470" fixed="right">
          <template #default="scope">
            <div class="asset-row-actions">
              <a :href="downloadHref(scope.row)" class="asset-action-link" target="_blank" rel="noopener">下载</a>
              <el-button link type="primary" @click="copyDownloadLink(scope.row)"><el-icon><CopyDocument /></el-icon>复制链接</el-button>
              <el-button v-if="canManageAssets && !scope.row.content_id" link type="primary" @click="openPublishDialog(scope.row)">归档到门户</el-button>
              <RouterLink v-if="publicResourcePath(scope.row)" :to="publicResourcePath(scope.row)" class="asset-action-link">查看门户</RouterLink>
              <RouterLink v-else-if="scope.row.content_id" :to="`/admin/content/${scope.row.content_id}/edit`" class="asset-action-link">编辑内容</RouterLink>
              <el-button v-if="canManageAssets && linkedContent(scope.row)?.status === 'published'" link type="warning" :loading="archivingId === scope.row.id" @click="archiveAsset(scope.row)">下架</el-button>
              <el-button v-else-if="canManageAssets && scope.row.content_id" link type="success" :loading="republishingId === scope.row.id" @click="republishAsset(scope.row)">重新上架</el-button>
              <el-button v-if="canManageAssets" link type="danger" aria-label="永久删除文件" :title="linkedContent(scope.row)?.status === 'published' ? '请先下架再删除' : '永久删除文件'" :disabled="linkedContent(scope.row)?.status === 'published'" :loading="deletingId === scope.row.id" @click="deleteAsset(scope.row)"><el-icon><Delete /></el-icon>删除文件</el-button>
            </div>
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

    <el-dialog
      v-model="publishDialogVisible"
      class="asset-publish-dialog"
      title="归档到门户资源中心"
      width="min(560px, calc(100vw - 32px))"
      :close-on-click-modal="!publishing"
      :close-on-press-escape="!publishing"
      :show-close="!publishing"
    >
      <div v-if="publishTarget" class="asset-publish-content">
        <div class="asset-publish-file">
          <span class="asset-type-icon"><el-icon><Picture v-if="isImage(publishTarget)" /><Document v-else /></el-icon></span>
          <div><strong>{{ publishTarget.original_name }}</strong><small>{{ formatBytes(publishTarget.size_bytes) }} · {{ publishTarget.mime_type }}</small></div>
        </div>
        <div class="asset-publish-notice" role="status">
          <strong>公开发布</strong>
          <span>发布后，所有访客都能在门户资源中心查看并下载此文件。</span>
        </div>
        <el-form label-position="top" @submit.prevent="publishAsset">
          <el-form-item label="门户标题" required>
            <el-input v-model="publishForm.title" maxlength="160" show-word-limit placeholder="例如：社团招新资料包" />
          </el-form-item>
          <el-form-item label="资源类型" required>
            <el-select v-model="publishForm.kind" style="width: 100%">
              <el-option label="文档" value="document" />
              <el-option label="模板" value="template" />
              <el-option label="资源包" value="package" />
              <el-option label="视频" value="video" />
            </el-select>
          </el-form-item>
          <el-form-item label="公开说明">
            <el-input v-model="publishForm.description" type="textarea" :rows="4" maxlength="500" show-word-limit placeholder="说明文件内容、版本和适用范围" />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button round :disabled="publishing" @click="publishDialogVisible = false">取消</el-button>
        <el-button type="primary" round :loading="publishing" @click="publishAsset">归档并发布</el-button>
      </template>
    </el-dialog>
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

.visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
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

.asset-content-state {
  display: grid;
  gap: 3px;
}

.asset-content-state a {
  overflow: hidden;
  color: var(--md-sys-color-primary);
  font-size: 13px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-content-state small {
  color: var(--md-sys-color-on-surface-variant);
  font-size: 12px;
}

.asset-publish-content {
  display: grid;
  gap: 18px;
}

.asset-publish-file {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
  padding: 14px;
  background: var(--md-sys-color-surface-container-high);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-md);
}

.asset-publish-file > div {
  display: grid;
  min-width: 0;
  gap: 4px;
}

.asset-publish-file strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-publish-file small {
  color: var(--md-sys-color-on-surface-variant);
}

.asset-publish-notice {
  display: grid;
  gap: 3px;
  padding: 12px 14px;
  color: var(--md-sys-color-on-primary-container);
  background: color-mix(in srgb, var(--md-sys-color-primary-container) 72%, var(--md-sys-color-surface-container-high));
  border: 1px solid var(--md-sys-color-primary);
  border-radius: var(--md-shape-md);
  font-size: 13px;
  line-height: 1.55;
}

.asset-row-actions {
  display: flex;
  min-height: 32px;
  flex-wrap: wrap;
  align-items: center;
  gap: 2px 10px;
}

.asset-action-link,
.asset-row-actions :deep(.el-button) {
  display: inline-flex;
  height: 30px;
  align-items: center;
  justify-content: center;
  margin: 0;
  padding: 0 3px;
  color: var(--md-sys-color-primary);
  font-size: 13px;
  font-weight: 700;
  line-height: 30px;
  white-space: nowrap;
}

.asset-row-actions :deep(.el-button + .el-button) {
  margin-left: 0;
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
