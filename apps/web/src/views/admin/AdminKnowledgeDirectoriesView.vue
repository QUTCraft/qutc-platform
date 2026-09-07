<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminContent, AdminKnowledgeDirectory } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(async () => {
  const [directories, content] = await Promise.all([
    adminApi.getKnowledgeDirectories({ page: 1, page_size: 100 }),
    adminApi.getContent({ page: 1, page_size: 100 }),
  ])
  return { directories, articles: content.items.filter((item) => item.type === 'knowledge') }
})
const dialogOpen = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const deletingArticleId = ref('')
const deletingDirectoryId = ref('')
const formRef = ref<FormInstance>()
const form = reactive<Omit<AdminKnowledgeDirectory, 'id' | 'updated_at'>>({ parent_id: '', name: '', slug: '', description: '', sort_order: 10, is_public: true })
const rules: FormRules = {
  name: [{ required: true, message: '请填写目录名称', trigger: 'blur' }],
  slug: [{ required: true, message: '请填写目录标识', trigger: 'blur' }],
}
const articleCountByDirectory = computed(() => {
  const counts = new Map<string, number>()
  for (const article of data.value?.articles ?? []) {
    if (!article.knowledge_directory_id) continue
    counts.set(article.knowledge_directory_id, (counts.get(article.knowledge_directory_id) ?? 0) + 1)
  }
  return counts
})
const childCountByDirectory = computed(() => {
  const counts = new Map<string, number>()
  for (const directory of data.value?.directories.items ?? []) {
    if (!directory.parent_id) continue
    counts.set(directory.parent_id, (counts.get(directory.parent_id) ?? 0) + 1)
  }
  return counts
})

function resetForm() {
  editingId.value = null
  Object.assign(form, { parent_id: '', name: '', slug: '', description: '', sort_order: ((data.value?.directories.items.length ?? 0) + 1) * 10, is_public: true })
  dialogOpen.value = true
}

function directoryName(id?: string | null) {
  return data.value?.directories.items.find((item) => item.id === id)?.name ?? '未关联目录'
}

function contentStatusLabel(status: 'draft' | 'review' | 'published' | 'archived') {
  return ({ draft: '草稿', review: '待审核', published: '已发布', archived: '已下线' })[status]
}

function contentStatusType(status: 'draft' | 'review' | 'published' | 'archived'): 'info' | 'warning' | 'success' {
  if (status === 'published') return 'success'
  if (status === 'review') return 'warning'
  return 'info'
}

function editDirectory(item: AdminKnowledgeDirectory) {
  editingId.value = item.id
  Object.assign(form, { parent_id: item.parent_id, name: item.name, slug: item.slug, description: item.description, sort_order: item.sort_order, is_public: item.is_public })
  dialogOpen.value = true
}

function directoryDeleteBlockReason(item: AdminKnowledgeDirectory) {
  const articleCount = articleCountByDirectory.value.get(item.id) ?? 0
  const childCount = childCountByDirectory.value.get(item.id) ?? 0
  if (articleCount > 0 && childCount > 0) return '该目录下仍有知识文章和子目录，请先移动或删除。'
  if (articleCount > 0) return '该目录下仍有知识文章，请先移动或删除。'
  if (childCount > 0) return '该目录下仍有子目录，请先移动或删除。'
  return ''
}

async function removeArticle(item: AdminContent) {
  if (!item.can_delete) {
    ElMessage.warning('已发布文章需要先下线后才能删除。')
    return
  }
  try {
    await ElMessageBox.confirm('确定永久删除这篇知识文章？修订与审核记录会一并清除，且无法恢复。', '删除知识文章', {
      confirmButtonText: '永久删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }
  deletingArticleId.value = item.id
  try {
    await adminApi.deleteContent(item.id)
    ElMessage.success('知识文章已删除。')
    await refresh()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '知识文章删除失败。')
  } finally {
    deletingArticleId.value = ''
  }
}

async function removeDirectory(item: AdminKnowledgeDirectory) {
  const reason = directoryDeleteBlockReason(item)
  if (reason) {
    ElMessage.warning(reason)
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除知识库目录“${item.name}”？仅空目录可以删除，此操作无法恢复。`, '删除知识库目录', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }
  deletingDirectoryId.value = item.id
  try {
    await adminApi.deleteKnowledgeDirectory(item.id)
    ElMessage.success('知识库目录已删除。')
    await refresh()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '知识库目录删除失败。')
  } finally {
    deletingDirectoryId.value = ''
  }
}

async function submit() {
  if (!formRef.value || !(await formRef.value.validate().catch(() => false))) return
  submitting.value = true
  try {
    if (editingId.value) await adminApi.updateKnowledgeDirectory(editingId.value, form)
    else await adminApi.createKnowledgeDirectory(form)
    ElMessage.success(editingId.value ? '知识库目录已更新。' : '知识库目录已创建。')
    dialogOpen.value = false
    await refresh()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '目录保存失败。')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>知识库</h2>
          <p>在同一处维护知识文章与分类目录；活动策划会读取这里未下线的文章作为可追溯依据。</p>
        </div>
        <div class="heading-actions">
          <RouterLink to="/admin/content/new?type=knowledge"><el-button type="primary" round>+ 新建知识文章</el-button></RouterLink>
          <el-button round @click="resetForm">+ 新建目录</el-button>
        </div>
      </section>

      <section class="admin-panel knowledge-section">
        <div class="section-heading">
          <div><h3>知识文章</h3><p>草稿、待审核和已发布文章均可作为内部智能体依据；已下线文章不会进入检索。</p></div>
          <span>{{ data.articles.length }} 篇</span>
        </div>
        <el-table v-if="data.articles.length" :data="data.articles" class="admin-table">
          <el-table-column prop="title" label="文章标题" min-width="240" />
          <el-table-column label="目录" min-width="150"><template #default="scope">{{ directoryName(scope.row.knowledge_directory_id) }}</template></el-table-column>
          <el-table-column prop="category" label="分类" min-width="130" />
          <el-table-column label="状态" width="110"><template #default="scope"><el-tag :type="contentStatusType(scope.row.status)" effect="plain">{{ contentStatusLabel(scope.row.status) }}</el-tag></template></el-table-column>
          <el-table-column label="更新时间" width="150"><template #default="scope">{{ formatDate(scope.row.updated_at) }}</template></el-table-column>
          <el-table-column label="操作" width="190" fixed="right">
            <template #default="scope">
              <RouterLink :to="`/admin/content/${scope.row.id}/edit`"><el-button text type="primary">编辑</el-button></RouterLink>
              <el-tooltip :content="scope.row.status === 'published' ? '已发布文章请先下线后再删除' : '永久删除知识文章及其修订与审核记录'" :disabled="Boolean(scope.row.can_delete)" placement="top">
                <span>
                  <el-popconfirm
                    title="确定永久删除这篇知识文章？"
                    confirm-button-text="永久删除"
                    cancel-button-text="取消"
                    width="280"
                    :disabled="!scope.row.can_delete"
                    @confirm="removeArticle(scope.row)"
                  >
                    <template #reference>
                      <el-button text type="danger" :disabled="!scope.row.can_delete || deletingArticleId === scope.row.id" :loading="deletingArticleId === scope.row.id">删除</el-button>
                    </template>
                  </el-popconfirm>
                </span>
              </el-tooltip>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-else description="还没有知识文章">
          <RouterLink to="/admin/content/new?type=knowledge"><el-button type="primary" round>新建第一篇知识文章</el-button></RouterLink>
        </el-empty>
      </section>

      <section class="admin-panel knowledge-section">
        <div class="section-heading">
          <div><h3>分类目录</h3><p>知识文章必须关联目录；只有公开目录中的已发布文章会显示在门户。</p></div>
          <span>{{ data.directories.total }} 个</span>
        </div>
        <el-table v-if="data.directories.items.length" :data="data.directories.items" class="admin-table">
          <el-table-column prop="name" label="目录名称" min-width="180" />
          <el-table-column prop="slug" label="标识" width="160" />
          <el-table-column prop="description" label="说明" min-width="260" />
          <el-table-column label="门户状态" width="120">
            <template #default="scope">
              <el-tag :type="scope.row.is_public ? 'success' : 'info'">{{ scope.row.is_public ? '公开' : '内部' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="sort_order" label="排序" width="90" />
          <el-table-column label="更新时间" width="150">
            <template #default="scope">{{ formatDate(scope.row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="190" fixed="right">
            <template #default="scope">
              <el-button text type="primary" @click="editDirectory(scope.row)">编辑</el-button>
              <el-tooltip :content="directoryDeleteBlockReason(scope.row)" :disabled="!directoryDeleteBlockReason(scope.row)" placement="top">
                <span>
                  <el-popconfirm
                    title="确定删除该知识库目录？仅空目录可以删除，且无法恢复。"
                    confirm-button-text="删除"
                    cancel-button-text="取消"
                    width="280"
                    :disabled="Boolean(directoryDeleteBlockReason(scope.row))"
                    @confirm="removeDirectory(scope.row)"
                  >
                    <template #reference>
                      <el-button text type="danger" :disabled="Boolean(directoryDeleteBlockReason(scope.row)) || deletingDirectoryId === scope.row.id" :loading="deletingDirectoryId === scope.row.id">删除</el-button>
                    </template>
                  </el-popconfirm>
                </span>
              </el-tooltip>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-else description="暂无知识库目录"><el-button type="primary" round @click="resetForm">新建第一个目录</el-button></el-empty>
      </section>

      <el-dialog v-model="dialogOpen" :title="editingId ? '编辑知识库目录' : '新建知识库目录'" width="min(92vw, 560px)">
        <el-form ref="formRef" :model="form" :rules="rules" label-position="top">
          <div class="form-grid">
            <el-form-item label="目录名称" prop="name"><el-input v-model="form.name" placeholder="例如：技术规范" /></el-form-item>
            <el-form-item label="目录标识" prop="slug"><el-input v-model="form.slug" placeholder="例如：technology" /></el-form-item>
          </div>
          <el-form-item label="目录说明"><el-input v-model="form.description" type="textarea" :rows="3" maxlength="500" /></el-form-item>
          <div class="form-grid">
            <el-form-item label="父目录标识"><el-input v-model="form.parent_id" placeholder="暂留空表示一级目录" /></el-form-item>
            <el-form-item label="排序"><el-input-number v-model="form.sort_order" :min="0" :max="9999" /></el-form-item>
          </div>
          <el-form-item><el-switch v-model="form.is_public" active-text="在门户公开显示" /></el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="dialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submit">保存目录</el-button>
        </template>
      </el-dialog>
    </template>
  </AsyncState>
</template>

<style scoped>
.heading-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.heading-actions .el-button { margin: 0; }
.knowledge-section { display: grid; gap: 16px; }
.section-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; }
.section-heading h3 { margin: 0 0 5px; font-size: 1.15rem; }
.section-heading p { margin: 0; color: var(--md-sys-color-on-surface-variant); line-height: 1.55; }
.section-heading > span { flex: 0 0 auto; padding: 6px 10px; color: var(--md-sys-color-on-secondary-container); border-radius: 999px; background: var(--md-sys-color-secondary-container); font-size: .78rem; }
@media (max-width: 640px) {
  .heading-actions { width: 100%; }
  .heading-actions a, .heading-actions .el-button { flex: 1 1 auto; }
  .section-heading { flex-direction: column; gap: 10px; }
}
</style>
