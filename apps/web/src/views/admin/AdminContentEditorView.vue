<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Check, EditPen, Link, MagicStick, Paperclip, Picture, Upload, View } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '@/api/admin'
import type { AdminContent, ContentRevision } from '@/api/types'
import AIContentAssistant from '@/components/admin/AIContentAssistant.vue'
import MarkdownContent from '@/components/MarkdownContent.vue'
import { useAsyncData } from '@/composables/useAsyncData'

type ContentType = AdminContent['type']

const route = useRoute()
const router = useRouter()
const { data: directoryData } = useAsyncData(() => adminApi.getKnowledgeDirectories({ page_size: 100 }))

const loading = ref(true)
const loadError = ref<Error | null>(null)
const saving = ref(false)
const publishing = ref(false)
const assetUploading = ref(false)
const status = ref<AdminContent['status']>('draft')
const bodyEditor = ref<HTMLTextAreaElement>()
const markdownFileInput = ref<HTMLInputElement>()
const imageFileInput = ref<HTMLInputElement>()
const attachmentFileInput = ref<HTMLInputElement>()
const assetPreviewUrls = ref<Record<string, string>>({})
const aiAssistantOpen = ref(false)
const revisions = ref<ContentRevision[]>([])
const revisionsLoading = ref(false)
const selectedRevision = ref<ContentRevision | null>(null)
const revisionDialogOpen = ref(false)

const form = reactive({
  title: '',
  type: 'news' as ContentType,
  category: '',
  knowledge_directory_id: '',
  excerpt: '',
  body: '',
})

const contentId = computed(() => typeof route.params.id === 'string' ? route.params.id : '')
const isNew = computed(() => !contentId.value)
const isPublished = computed(() => status.value === 'published')
const statusLabel = computed(() => ({ draft: '草稿', review: '待审核', published: '已发布', archived: '已下线' })[status.value])
const previewMarkdown = computed(() => {
  let markdown = form.body
  for (const [source, preview] of Object.entries(assetPreviewUrls.value)) markdown = markdown.split(source).join(preview)
  return markdown
})

function resetForm() {
  Object.assign(form, { title: '', type: 'news', category: '', knowledge_directory_id: '', excerpt: '', body: '' })
  status.value = 'draft'
}

function loadItem(item: AdminContent) {
  Object.assign(form, {
    title: item.title,
    type: item.type,
    category: item.category ?? '',
    knowledge_directory_id: item.knowledge_directory_id ?? '',
    excerpt: item.excerpt ?? '',
    body: item.body ?? '',
  })
  status.value = item.status
}

async function loadContent() {
  loading.value = true
  loadError.value = null
  try {
    if (isNew.value) {
      resetForm()
      return
    }
    const item = await adminApi.getContentById(contentId.value)
    loadItem(item)
    await loadRevisions()
  } catch (error) {
    loadError.value = error instanceof Error ? error : new Error('内容暂时无法加载。')
  } finally {
    loading.value = false
  }
}

async function loadRevisions() {
  if (!contentId.value) {
    revisions.value = []
    return
  }
  revisionsLoading.value = true
  try {
    revisions.value = (await adminApi.getContentRevisions(contentId.value, { page: 1, page_size: 12 })).items
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '内容修订历史加载失败。')
  } finally {
    revisionsLoading.value = false
  }
}

async function inspectRevision(revision: ContentRevision) {
  try {
    selectedRevision.value = await adminApi.getContentRevision(contentId.value, revision.id)
    revisionDialogOpen.value = true
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '修订版本加载失败。')
  }
}

async function restoreRevision(revision: ContentRevision) {
  try {
    await ElMessageBox.confirm(`恢复到 v${revision.version} 会生成新的草稿修订版本，当前内容不会直接发布。`, '恢复内容版本', { confirmButtonText: '恢复为草稿', cancelButtonText: '取消', type: 'warning' })
  } catch {
    return
  }
  try {
    const restored = await adminApi.restoreContentRevision(contentId.value, revision.id)
    loadItem(restored)
    await loadRevisions()
    ElMessage.success('已恢复为新的草稿版本。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '修订版本恢复失败。')
  }
}

onMounted(loadContent)
watch(contentId, loadContent)

function validateForm() {
  if (!form.title.trim()) {
    ElMessage.warning('请先填写内容标题。')
    return false
  }
  if (form.type === 'knowledge' && !form.knowledge_directory_id) {
    ElMessage.warning('知识库文章必须选择目录。')
    return false
  }
  if (isPublished.value) {
    ElMessage.warning('已发布内容不能直接编辑，请先下线。')
    return false
  }
  return true
}

async function persist(showMessage = true) {
  if (!validateForm()) return null
  saving.value = true
  try {
    const payload = {
      title: form.title,
      type: form.type,
      category: form.category,
      knowledge_directory_id: form.type === 'knowledge' ? form.knowledge_directory_id : '',
      excerpt: form.excerpt,
      body: form.body,
    }
    const saved = contentId.value ? await adminApi.updateContent(contentId.value, payload) : await adminApi.createContent(payload)
    status.value = saved.status
    if (isNew.value) await router.replace({ name: 'admin-content-edit', params: { id: saved.id } })
    if (showMessage) ElMessage.success('内容草稿已保存。')
    return saved
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '内容保存失败。')
    return null
  } finally {
    saving.value = false
  }
}

async function saveDraft() {
  await persist()
}

async function publish() {
  const saved = await persist(false)
  if (!saved) return
  publishing.value = true
  try {
    const published = await adminApi.publishContent(saved.id)
    status.value = published.status
    ElMessage.success('内容已发布到门户。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '内容发布失败。')
  } finally {
    publishing.value = false
  }
}

async function archive() {
  if (!contentId.value || status.value !== 'published') return
  publishing.value = true
  try {
    const archived = await adminApi.archiveContent(contentId.value)
    status.value = archived.status
    ElMessage.success('内容已下线。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '内容下线失败。')
  } finally {
    publishing.value = false
  }
}

function insertMarkdown(before: string, after = '', placeholder = '文本') {
  const target = bodyEditor.value
  const value = form.body
  const start = target?.selectionStart ?? value.length
  const end = target?.selectionEnd ?? value.length
  const selected = value.slice(start, end) || placeholder
  form.body = value.slice(0, start) + before + selected + after + value.slice(end)
  void nextTick(() => {
    if (!target) return
    const cursorStart = start + before.length
    target.focus()
    target.setSelectionRange(cursorStart, cursorStart + selected.length)
  })
}

function currentEditorSelection() {
  const target = bodyEditor.value
  return {
    start: target?.selectionStart ?? form.body.length,
    end: target?.selectionEnd ?? form.body.length,
  }
}

function insertUploadedMarkdown(markdown: string, selection: { start: number; end: number }) {
  const target = bodyEditor.value
  const value = form.body
  const start = Math.min(selection.start, value.length)
  const end = Math.min(Math.max(selection.end, start), value.length)
  form.body = value.slice(0, start) + markdown + value.slice(end)
  void nextTick(() => {
    if (!target) return
    target.focus()
    target.setSelectionRange(start + markdown.length, start + markdown.length)
  })
}

function importMarkdown(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  void file.text().then((content) => {
    form.body = content
    if (!form.title.trim()) {
      const heading = content.match(/^#\s+(.+)$/m)?.[1]?.trim()
      if (heading) form.title = heading
    }
    ElMessage.success(`已导入 ${file.name}。`)
  }).catch(() => ElMessage.error('Markdown 文件读取失败。'))
}

function triggerImageUpload() {
  imageFileInput.value?.click()
}

function triggerAttachmentUpload() {
  attachmentFileInput.value?.click()
}

function handleImageUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (file) void uploadAndInsert(file, 'image')
}

function handleAttachmentUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (file) void uploadAndInsert(file, 'file')
}

async function uploadAndInsert(file: File, kind: 'image' | 'file') {
  if (assetUploading.value) return
  const selection = currentEditorSelection()
  const saved = await persist(false)
  if (!saved) return
  assetUploading.value = true
  try {
    const asset = await adminApi.uploadAsset(file, saved.id)
    if (!asset.download_url) throw new Error('媒体服务没有返回可用地址。')
    const previewUrl = URL.createObjectURL(file)
    assetPreviewUrls.value[asset.download_url] = previewUrl
    const label = file.name.replace(/\.[^.]+$/, '') || file.name
    const markdown = kind === 'image' ? `![${label}](${asset.download_url})` : `[${file.name}](${asset.download_url})`
    insertUploadedMarkdown(markdown, selection)
    ElMessage.success(kind === 'image' ? '图片已上传并插入 Markdown。' : '文件已上传并插入 Markdown。')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '附件上传失败。')
  } finally {
    assetUploading.value = false
  }
}

function goBack() {
  void router.push('/admin/content')
}

function applyAIProposal(proposal: { title: string; excerpt: string; body: string }) {
  Object.assign(form, proposal)
}

function openCreatedDraft(content: AdminContent) {
  loadItem(content)
  void router.replace({ name: 'admin-content-edit', params: { id: content.id } })
}

onBeforeUnmount(() => {
  for (const url of Object.values(assetPreviewUrls.value)) URL.revokeObjectURL(url)
})
</script>

<template>
  <div v-if="loading" class="async-pending" role="status" aria-live="polite">内容加载中</div>
  <el-result v-else-if="loadError" icon="error" title="内容无法加载" :sub-title="loadError.message">
    <template #extra>
      <el-button type="primary" @click="loadContent">重试</el-button>
      <el-button @click="goBack">返回内容管理</el-button>
    </template>
  </el-result>
  <section v-else class="content-editor-page">
    <header class="content-editor-header">
      <div class="content-editor-heading">
        <el-button text class="editor-back-button" @click="goBack">
          <el-icon><ArrowLeft /></el-icon>
          内容管理
        </el-button>
        <div class="editor-title-row">
          <h2>{{ isNew ? '新建内容' : '编辑内容' }}</h2>
          <el-tag :type="status === 'published' ? 'success' : status === 'archived' ? 'info' : 'warning'" effect="plain">{{ statusLabel }}</el-tag>
        </div>
        <p>使用标准 Markdown 编写正文，右侧预览会随输入即时更新。</p>
      </div>
      <div class="content-editor-actions">
        <el-button class="editor-ai-button" :icon="MagicStick" @click="aiAssistantOpen = true">从知识生成</el-button>
        <label class="editor-import-button">
          <el-icon><Upload /></el-icon>
          导入 Markdown
          <input ref="markdownFileInput" type="file" accept=".md,.markdown,text/markdown,text/plain" @change="importMarkdown" />
        </label>
        <el-button :loading="saving" :disabled="isPublished" @click="saveDraft"><el-icon><Check /></el-icon>保存草稿</el-button>
        <el-button v-if="status !== 'published'" type="primary" :loading="publishing || saving" @click="publish">发布</el-button>
        <el-button v-else type="danger" :loading="publishing" @click="archive">下线</el-button>
      </div>
    </header>

    <section class="content-editor-meta editor-surface">
      <div class="editor-meta-main">
        <label class="editor-field editor-field-title">
          <span>标题</span>
          <el-input v-model="form.title" :disabled="isPublished" size="large" placeholder="例如：暑期建筑活动报名" maxlength="160" />
        </label>
        <label class="editor-field">
          <span>摘要</span>
          <el-input v-model="form.excerpt" :disabled="isPublished" type="textarea" :rows="2" maxlength="500" show-word-limit placeholder="发布后显示在门户卡片中" />
        </label>
      </div>
      <div class="editor-meta-options">
        <label class="editor-field">
          <span>内容类型</span>
          <el-radio-group v-model="form.type" class="content-type-selector" :disabled="isPublished">
            <el-radio-button value="news">动态</el-radio-button>
            <el-radio-button value="resource">资源</el-radio-button>
            <el-radio-button value="knowledge">知识库</el-radio-button>
          </el-radio-group>
        </label>
        <label v-if="form.type === 'knowledge'" class="editor-field">
          <span>知识库目录</span>
          <el-select v-model="form.knowledge_directory_id" :disabled="isPublished" filterable placeholder="选择目录">
            <el-option v-for="directory in directoryData?.items ?? []" :key="directory.id" :label="directory.name" :value="directory.id" />
          </el-select>
        </label>
        <label v-else class="editor-field">
          <span>分类</span>
          <el-input v-model="form.category" :disabled="isPublished" maxlength="64" placeholder="例如：公告、活动" />
        </label>
      </div>
    </section>

    <section class="content-editor-workspace">
      <article class="editor-surface editor-pane">
        <div class="editor-pane-heading">
          <div>
            <span class="editor-pane-kicker"><EditPen /> MARKDOWN</span>
            <h3>正文编辑</h3>
          </div>
          <span class="editor-character-count">{{ form.body.length }} 字符</span>
        </div>
        <div class="markdown-toolbar" aria-label="Markdown 工具栏">
          <button type="button" :disabled="isPublished" title="标题" @click="insertMarkdown('## ', '', '小标题')">H₂</button>
          <button type="button" :disabled="isPublished" title="粗体" @click="insertMarkdown('**', '**', '重点文本')"><strong>B</strong></button>
          <button type="button" :disabled="isPublished" title="斜体" @click="insertMarkdown('*', '*', '强调文本')"><em>I</em></button>
          <button type="button" :disabled="isPublished" title="引用" @click="insertMarkdown('> ', '', '引用内容')">❞</button>
          <button type="button" :disabled="isPublished" title="代码" @click="insertMarkdown('`', '`', 'code')">&lt;/&gt;</button>
          <button type="button" :disabled="isPublished" title="链接" @click="insertMarkdown('[', '](https://)', '链接文本')"><el-icon><Link /></el-icon></button>
          <span class="toolbar-divider" />
          <button type="button" :disabled="isPublished || assetUploading" title="插入图片" @click="triggerImageUpload"><el-icon><Picture /></el-icon></button>
          <button type="button" :disabled="isPublished || assetUploading" title="插入文件" @click="triggerAttachmentUpload"><el-icon><Paperclip /></el-icon></button>
          <input ref="imageFileInput" class="visually-hidden" type="file" accept="image/png,image/jpeg,image/webp" @change="handleImageUpload" />
          <input ref="attachmentFileInput" class="visually-hidden" type="file" accept="application/pdf,application/zip,video/mp4" @change="handleAttachmentUpload" />
        </div>
        <textarea ref="bodyEditor" v-model="form.body" class="markdown-textarea" :disabled="isPublished" spellcheck="false" placeholder="# 文章标题

在这里编写标准 Markdown 正文……" />
        <div class="editor-attachment-tip">
          <el-icon><Paperclip /></el-icon>
          <span>图片与文件会先保存草稿，再上传到媒体服务并自动插入 Markdown 链接。</span>
        </div>
      </article>

      <article class="editor-surface editor-pane preview-pane">
        <div class="editor-pane-heading">
          <div>
            <span class="editor-pane-kicker"><View /> LIVE PREVIEW</span>
            <h3>实时预览</h3>
          </div>
          <span class="preview-mode-label">门户正文样式</span>
        </div>
        <MarkdownContent v-if="form.body.trim()" class="markdown-preview" :markdown="previewMarkdown" />
        <el-empty v-else description="开始输入 Markdown，预览会显示在这里。" :image-size="96" />
      </article>
    </section>

    <section class="editor-surface revision-history-panel">
      <div class="editor-pane-heading">
        <div>
          <span class="editor-pane-kicker"><View /> VERSION HISTORY</span>
          <h3>修订历史</h3>
        </div>
        <el-button text :loading="revisionsLoading" @click="loadRevisions">刷新</el-button>
      </div>
      <el-empty v-if="!revisionsLoading && revisions.length === 0" description="保存后会在这里生成版本快照" :image-size="72" />
      <div v-else class="revision-list">
        <div v-for="revision in revisions" :key="revision.id" class="revision-row">
          <div>
            <strong>v{{ revision.version }} · {{ revision.title }}</strong>
            <span>{{ revision.reason }} · {{ new Date(revision.created_at).toLocaleString('zh-CN') }}</span>
          </div>
          <div class="revision-actions">
            <el-button text @click="inspectRevision(revision)">查看</el-button>
            <el-button v-if="!isPublished" text type="primary" @click="restoreRevision(revision)">恢复</el-button>
          </div>
        </div>
      </div>
    </section>

    <el-dialog v-model="revisionDialogOpen" :title="selectedRevision ? `修订 v${selectedRevision.version}` : '修订详情'" width="min(860px, 92vw)">
      <div v-if="selectedRevision" class="revision-dialog-content">
        <p>{{ selectedRevision.excerpt || '无摘要' }}</p>
        <pre>{{ selectedRevision.body }}</pre>
      </div>
    </el-dialog>

    <AIContentAssistant
      v-model="aiAssistantOpen"
      :current-title="form.title"
      :current-excerpt="form.excerpt"
      :current-body="form.body"
      :content-type="form.type"
      :category="form.category"
      :knowledge-directory-id="form.knowledge_directory_id"
      :published="isPublished"
      @apply="applyAIProposal"
      @created="openCreatedDraft"
    />
  </section>
</template>

<style scoped>
.content-editor-page {
  width: 100%;
}

.content-editor-loading {
  padding: 32px;
  background: var(--md-sys-color-surface-container);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-lg);
}

.content-editor-header,
.content-editor-heading,
.content-editor-actions,
.editor-title-row,
.editor-pane-heading,
.editor-attachment-tip {
  display: flex;
  align-items: center;
}

.content-editor-header {
  justify-content: space-between;
  align-items: flex-end;
  gap: 24px;
  margin-bottom: 24px;
}

.content-editor-heading {
  align-items: flex-start;
  flex-direction: column;
  gap: 8px;
}

.editor-back-button {
  margin-left: -12px;
  color: var(--md-sys-color-primary) !important;
}

.editor-title-row {
  gap: 12px;
}

.editor-title-row h2 {
  margin: 0;
  font-size: clamp(24px, 3vw, 34px);
  font-weight: 850;
  letter-spacing: -0.04em;
}

.content-editor-heading p {
  margin: 0;
  color: var(--md-sys-color-on-surface-variant);
  font-size: 14px;
}

.content-editor-actions {
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.editor-import-button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 40px;
  padding: 0 16px;
  color: var(--md-sys-color-primary);
  background: var(--md-sys-color-surface-container-high);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-full);
  cursor: pointer;
  font-size: 14px;
  font-weight: 700;
  transition: all 180ms ease;
}

.editor-import-button:hover {
  color: var(--md-sys-color-on-primary-container);
  background: var(--md-sys-color-primary-container);
  border-color: var(--md-sys-color-primary);
}

.editor-ai-button {
  color: var(--md-sys-color-on-tertiary-container) !important;
  background: var(--md-sys-color-tertiary-container) !important;
  border-color: color-mix(in srgb, var(--md-sys-color-tertiary) 42%, transparent) !important;
}

.editor-ai-button:hover {
  border-color: var(--md-sys-color-tertiary) !important;
}

.editor-import-button input {
  display: none;
}

.editor-surface {
  background: var(--md-sys-color-surface-container);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-lg);
  box-shadow: var(--md-elevation-1);
}

.content-editor-meta {
  display: grid;
  grid-template-columns: minmax(0, 1.5fr) minmax(280px, 0.8fr);
  gap: 22px;
  margin-bottom: 22px;
  padding: 24px;
}

.editor-meta-main,
.editor-meta-options {
  display: grid;
  gap: 16px;
}

.editor-field {
  display: grid;
  gap: 8px;
  min-width: 0;
}

.editor-field > span {
  color: var(--md-sys-color-on-surface-variant);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.editor-field-title :deep(.el-input__inner) {
  font-size: 18px;
  font-weight: 700;
}

.editor-meta-options {
  grid-template-columns: 1fr;
  align-content: start;
}

.editor-meta-options :deep(.el-select) {
  width: 100%;
}

.content-editor-workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 22px;
  align-items: stretch;
}

.revision-history-panel {
  margin-top: 20px;
  padding: 20px;
}

.revision-list {
  display: grid;
  gap: 8px;
}

.revision-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 16px;
  align-items: center;
  padding: 12px 14px;
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-md);
  background: var(--md-sys-color-surface-container-low);
}

.revision-row strong,
.revision-row span {
  display: block;
}

.revision-row span {
  margin-top: 4px;
  color: var(--md-sys-color-on-surface-variant);
  font-size: 12px;
}

.revision-actions {
  display: flex;
  gap: 4px;
}

.revision-dialog-content p {
  color: var(--md-sys-color-on-surface-variant);
}

.revision-dialog-content pre {
  max-height: 55vh;
  overflow: auto;
  padding: 16px;
  white-space: pre-wrap;
  word-break: break-word;
  border-radius: 12px;
  background: var(--md-sys-color-surface-container-low);
}

.editor-pane {
  display: flex;
  min-width: 0;
  min-height: 680px;
  flex-direction: column;
  overflow: hidden;
}

.editor-pane-heading {
  justify-content: space-between;
  gap: 16px;
  padding: 22px 24px 16px;
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
}

.editor-pane-heading h3 {
  margin: 3px 0 0;
  font-size: 20px;
  font-weight: 800;
}

.editor-pane-kicker {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--md-sys-color-primary);
  font: 800 11px var(--md-font-mono);
  letter-spacing: 0.12em;
}

.editor-pane-kicker :deep(.el-icon) {
  font-size: 14px;
}

.editor-character-count,
.preview-mode-label {
  color: var(--md-sys-color-on-surface-variant);
  font-size: 12px;
  white-space: nowrap;
}

.markdown-toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 50px;
  padding: 8px 16px;
  background: var(--md-sys-color-surface-container-high);
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
}

.markdown-toolbar button {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  color: var(--md-sys-color-on-surface-variant);
  background: transparent;
  border: 0;
  border-radius: var(--md-shape-sm);
  cursor: pointer;
  font-size: 13px;
  font-weight: 700;
}

.markdown-toolbar button:hover:not(:disabled) {
  color: var(--md-sys-color-on-primary-container);
  background: var(--md-sys-color-primary-container);
}

.markdown-toolbar button:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.toolbar-divider {
  width: 1px;
  height: 22px;
  margin: 0 5px;
  background: var(--md-sys-color-outline-variant);
}

.markdown-textarea {
  display: block;
  width: 100%;
  min-height: 540px;
  flex: 1;
  padding: 22px 24px;
  color: var(--md-sys-color-on-surface);
  background: transparent;
  border: 0;
  outline: 0;
  resize: vertical;
  font: 14px/1.75 var(--md-font-mono);
  tab-size: 2;
}

.markdown-textarea:focus {
  background: color-mix(in srgb, var(--md-sys-color-primary-container) 22%, transparent);
}

.markdown-textarea::placeholder {
  color: var(--md-sys-color-on-surface-variant);
  opacity: 0.72;
}

.editor-attachment-tip {
  gap: 8px;
  padding: 12px 24px;
  color: var(--md-sys-color-on-surface-variant);
  border-top: 1px solid var(--md-sys-color-outline-variant);
  font-size: 12px;
}

.editor-attachment-tip :deep(.el-icon) {
  flex: 0 0 auto;
  color: var(--md-sys-color-primary);
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

.markdown-preview {
  min-height: 540px;
  flex: 1;
  overflow: auto;
  padding: 26px 28px 40px;
  color: var(--md-sys-color-on-surface);
  line-height: 1.8;
  overflow-wrap: anywhere;
}

.markdown-preview :deep(h1),
.markdown-preview :deep(h2),
.markdown-preview :deep(h3),
.markdown-preview :deep(h4) {
  margin: 1.3em 0 0.55em;
  color: var(--md-sys-color-on-surface);
  line-height: 1.25;
}

.markdown-preview :deep(h1) {
  padding-bottom: 12px;
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  font-size: 30px;
}

.markdown-preview :deep(h2) { font-size: 24px; }
.markdown-preview :deep(h3) { font-size: 20px; }

.markdown-preview :deep(p),
.markdown-preview :deep(ul),
.markdown-preview :deep(ol),
.markdown-preview :deep(blockquote),
.markdown-preview :deep(pre),
.markdown-preview :deep(table) {
  margin: 0 0 16px;
}

.markdown-preview :deep(a) {
  color: var(--md-sys-color-primary);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.markdown-preview :deep(blockquote) {
  padding: 10px 16px;
  color: var(--md-sys-color-on-surface-variant);
  background: var(--md-sys-color-surface-container-high);
  border-left: 4px solid var(--md-sys-color-primary);
  border-radius: 0 var(--md-shape-sm) var(--md-shape-sm) 0;
}

.markdown-preview :deep(code) {
  padding: 2px 6px;
  color: var(--md-sys-color-on-primary-container);
  background: var(--md-sys-color-primary-container);
  border-radius: 5px;
  font: 0.9em var(--md-font-mono);
}

.markdown-preview :deep(pre) {
  padding: 16px;
  overflow: auto;
  color: var(--md-sys-color-on-surface);
  background: var(--md-sys-color-surface-container-highest);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-md);
}

.markdown-preview :deep(pre code) {
  padding: 0;
  color: inherit;
  background: transparent;
}

.markdown-preview :deep(img) {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 18px auto;
  border-radius: var(--md-shape-md);
  box-shadow: var(--md-elevation-1);
}

.markdown-preview :deep(hr) {
  margin: 24px 0;
  border: 0;
  border-top: 1px solid var(--md-sys-color-outline-variant);
}

.markdown-preview :deep(table) {
  width: 100%;
  border-collapse: collapse;
}

.markdown-preview :deep(th),
.markdown-preview :deep(td) {
  padding: 8px 10px;
  border: 1px solid var(--md-sys-color-outline-variant);
  text-align: left;
}

.markdown-preview :deep(th) {
  background: var(--md-sys-color-surface-container-high);
}

@media (max-width: 1100px) {
  .content-editor-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .content-editor-actions {
    justify-content: flex-start;
  }

  .content-editor-meta {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 850px) {
  .content-editor-workspace {
    grid-template-columns: 1fr;
  }

  .editor-pane {
    min-height: 560px;
  }

  .markdown-textarea,
  .markdown-preview {
    min-height: 420px;
  }
}

@media (max-width: 560px) {
  .content-editor-header {
    gap: 16px;
  }

  .content-editor-actions {
    width: 100%;
  }

  .content-editor-actions > *,
  .editor-import-button {
    flex: 1 1 auto;
    justify-content: center;
  }

  .content-editor-meta,
  .editor-pane-heading {
    padding-inline: 16px;
  }

  .markdown-toolbar {
    padding-inline: 10px;
  }

  .markdown-textarea,
  .markdown-preview {
    padding-inline: 16px;
  }

  .editor-attachment-tip {
    align-items: flex-start;
    padding-inline: 16px;
  }
}
</style>
