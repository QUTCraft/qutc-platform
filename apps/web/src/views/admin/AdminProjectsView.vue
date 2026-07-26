<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminProject, AdminProjectMember, AdminProjectMilestone, AdminUser } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(adminApi.getProjects)
const dialogOpen = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const form = reactive({ title: '', summary: '', status: 'research' as AdminProject['status'], tags: '', is_public: true })
const statusLabel: Record<AdminProject['status'], string> = { active: '进行中', research: '研究中', completed: '已完成' }

const workspaceOpen = ref(false)
const workspaceTab = ref<'members' | 'milestones'>('members')
const workspaceLoading = ref(false)
const selectedProject = ref<AdminProject | null>(null)
const projectMembers = ref<AdminProjectMember[]>([])
const projectMilestones = ref<AdminProjectMilestone[]>([])
const organizationUsers = ref<AdminUser[]>([])

const memberDialogOpen = ref(false)
const memberSubmitting = ref(false)
const editingMember = ref<AdminProjectMember | null>(null)
const memberForm = reactive({ user_id: '', role: 'member' as Exclude<AdminProjectMember['role'], 'owner'> })
const memberRoleLabel: Record<AdminProjectMember['role'], string> = { owner: '负责人', lead: '项目主力', contributor: '贡献者', member: '成员' }
const memberCandidates = computed(() => organizationUsers.value.filter((user) => user.state === 'active' && !projectMembers.value.some((member) => member.user_id === user.id)))

const milestoneDialogOpen = ref(false)
const milestoneSubmitting = ref(false)
const editingMilestone = ref<AdminProjectMilestone | null>(null)
const milestoneForm = reactive({ title: '', status: 'planned' as AdminProjectMilestone['status'], due_at: '' })
const milestoneStatusLabel: Record<AdminProjectMilestone['status'], string> = { planned: '未开始', active: '进行中', completed: '已完成' }
const milestoneStatusType: Record<AdminProjectMilestone['status'], '' | 'success' | 'warning'> = { planned: '', active: 'warning', completed: 'success' }

function resetForm() {
  editingId.value = null
  Object.assign(form, { title: '', summary: '', status: 'research', tags: '', is_public: true })
  dialogOpen.value = true
}

function editProject(item: AdminProject) {
  editingId.value = item.id
  Object.assign(form, { title: item.title, summary: item.summary, status: item.status, tags: item.tags.join(', '), is_public: item.is_public })
  dialogOpen.value = true
}

async function submit() {
  submitting.value = true
  try {
    const payload = { ...form, tags: form.tags.split(',').map((tag) => tag.trim()).filter(Boolean) }
    if (editingId.value) await adminApi.updateProject(editingId.value, payload)
    else await adminApi.createProject(payload)
    ElMessage.success(editingId.value ? '项目已保存。' : '项目已创建。')
    dialogOpen.value = false
    await refresh()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '项目保存失败。')
  } finally {
    submitting.value = false
  }
}

async function loadProjectWorkspace() {
  if (!selectedProject.value) return
  workspaceLoading.value = true
  try {
    const [members, milestones, users] = await Promise.all([adminApi.getProjectMembers(selectedProject.value.id), adminApi.getProjectMilestones(selectedProject.value.id), adminApi.getUsers()])
    projectMembers.value = members.items
    projectMilestones.value = milestones.items
    organizationUsers.value = users.items
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '项目协作数据加载失败。')
  } finally {
    workspaceLoading.value = false
  }
}

async function openProjectWorkspace(project: AdminProject, tab: 'members' | 'milestones' = 'members') {
  selectedProject.value = project
  workspaceTab.value = tab
  workspaceOpen.value = true
  await loadProjectWorkspace()
}

async function refreshProjectSummary() {
  await refresh()
  if (!selectedProject.value) return
  const updated = data.value?.items.find((project) => project.id === selectedProject.value?.id)
  if (updated) selectedProject.value = updated
}

function openMemberEditor(member?: AdminProjectMember) {
  editingMember.value = member ?? null
  memberForm.user_id = member?.user_id ?? ''
  memberForm.role = member?.role === 'owner' ? 'member' : (member?.role ?? 'member')
  memberDialogOpen.value = true
}

async function saveMember() {
  if (!selectedProject.value || !memberForm.user_id) {
    ElMessage.warning('请选择要加入项目的组织成员。')
    return
  }
  memberSubmitting.value = true
  try {
    if (editingMember.value) await adminApi.updateProjectMember(selectedProject.value.id, editingMember.value.user_id, { role: memberForm.role })
    else await adminApi.addProjectMember(selectedProject.value.id, { user_id: memberForm.user_id, role: memberForm.role })
    ElMessage.success(editingMember.value ? '项目成员角色已更新。' : '项目成员已添加。')
    memberDialogOpen.value = false
    await loadProjectWorkspace()
    await refreshProjectSummary()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '项目成员保存失败。')
  } finally {
    memberSubmitting.value = false
  }
}

async function removeMember(member: AdminProjectMember) {
  if (!selectedProject.value) return
  try {
    await adminApi.removeProjectMember(selectedProject.value.id, member.user_id)
    ElMessage.success('项目成员已移除。')
    await loadProjectWorkspace()
    await refreshProjectSummary()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '项目成员移除失败。')
  }
}

function openMilestoneEditor(milestone?: AdminProjectMilestone) {
  editingMilestone.value = milestone ?? null
  Object.assign(milestoneForm, { title: milestone?.title ?? '', status: milestone?.status ?? 'planned', due_at: milestone?.due_at ?? '' })
  milestoneDialogOpen.value = true
}

async function saveMilestone() {
  if (!selectedProject.value || !milestoneForm.title.trim()) {
    ElMessage.warning('请填写里程碑标题。')
    return
  }
  milestoneSubmitting.value = true
  try {
    const payload = { title: milestoneForm.title.trim(), status: milestoneForm.status, due_at: milestoneForm.due_at }
    if (editingMilestone.value) await adminApi.updateProjectMilestone(selectedProject.value.id, editingMilestone.value.id, payload)
    else await adminApi.createProjectMilestone(selectedProject.value.id, payload)
    ElMessage.success(editingMilestone.value ? '里程碑已保存。' : '里程碑已创建。')
    milestoneDialogOpen.value = false
    await loadProjectWorkspace()
    await refreshProjectSummary()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '里程碑保存失败。')
  } finally {
    milestoneSubmitting.value = false
  }
}

async function removeMilestone(milestone: AdminProjectMilestone) {
  if (!selectedProject.value) return
  try {
    await adminApi.removeProjectMilestone(selectedProject.value.id, milestone.id)
    ElMessage.success('里程碑已删除。')
    await loadProjectWorkspace()
    await refreshProjectSummary()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '里程碑删除失败。')
  }
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>项目管理</h2>
          <p>维护项目负责人、成员协作、里程碑和门户公开范围。</p>
        </div>
        <el-button type="primary" round @click="resetForm">+ 新建项目</el-button>
      </section>

      <section class="admin-panel">
        <el-table :data="data.items" class="admin-table">
          <el-table-column prop="title" label="项目" min-width="240" />
          <el-table-column label="状态" width="120">
            <template #default="scope"><el-tag>{{ statusLabel[scope.row.status as AdminProject['status']] }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="owner" label="负责人" width="140" />
          <el-table-column prop="member_count" label="成员" width="90" />
          <el-table-column prop="milestone_count" label="里程碑" width="100" />
          <el-table-column label="公开" width="100">
            <template #default="scope"><el-tag :type="scope.row.is_public ? 'success' : 'info'">{{ scope.row.is_public ? '已公开' : '私有' }}</el-tag></template>
          </el-table-column>
          <el-table-column label="更新时间" width="150">
            <template #default="scope">{{ formatDate(scope.row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="230" fixed="right">
            <template #default="scope">
              <el-button text type="primary" @click="editProject(scope.row)">编辑</el-button>
              <el-button text type="primary" @click="openProjectWorkspace(scope.row, 'members')">成员</el-button>
              <el-button text type="primary" @click="openProjectWorkspace(scope.row, 'milestones')">里程碑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </section>

      <el-dialog v-model="dialogOpen" :title="editingId ? '编辑项目' : '新建项目'" width="min(92vw, 560px)">
        <el-form :model="form" label-position="top">
          <el-form-item label="项目名称"><el-input v-model="form.title" maxlength="160" /></el-form-item>
          <el-form-item label="项目简介"><el-input v-model="form.summary" type="textarea" :rows="3" maxlength="500" show-word-limit /></el-form-item>
          <div class="form-grid">
            <el-form-item label="项目状态">
              <el-select v-model="form.status" style="width: 100%">
                <el-option label="进行中" value="active" />
                <el-option label="研究中" value="research" />
                <el-option label="已完成" value="completed" />
              </el-select>
            </el-form-item>
            <el-form-item label="标签"><el-input v-model="form.tags" placeholder="用逗号分隔" /></el-form-item>
          </div>
          <el-form-item><el-switch v-model="form.is_public" active-text="在门户公开展示" /></el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="dialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="submitting" @click="submit">保存项目</el-button>
        </template>
      </el-dialog>

      <el-drawer v-model="workspaceOpen" :title="selectedProject ? `项目协作 · ${selectedProject.title}` : '项目协作'" size="min(92vw, 840px)">
        <template v-if="selectedProject">
          <div class="project-workspace-summary">
            <span>{{ statusLabel[selectedProject.status] }}</span>
            <strong>{{ selectedProject.summary || '暂未填写项目简介。' }}</strong>
            <small>负责人：{{ selectedProject.owner }} · {{ selectedProject.is_public ? '门户已公开' : '仅组织内部可见' }}</small>
          </div>

          <el-tabs v-model="workspaceTab">
            <el-tab-pane label="项目成员" name="members">
              <div class="project-drawer-toolbar">
                <p>项目负责人不能被移除或改为其他项目角色。</p>
                <el-button type="primary" round @click="openMemberEditor()">+ 添加成员</el-button>
              </div>
              <div v-loading="workspaceLoading">
                <el-table :data="projectMembers" class="admin-table" empty-text="还没有项目成员">
                  <el-table-column label="成员" min-width="220">
                    <template #default="scope">
                      <div class="person-cell">
                        <span class="avatar-bubble">{{ scope.row.name.slice(0, 1) }}</span>
                        <div><strong>{{ scope.row.name }}</strong><small>{{ scope.row.email }}</small></div>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="角色" width="130">
                    <template #default="scope"><el-tag :type="scope.row.role === 'owner' ? 'success' : ''" effect="plain">{{ memberRoleLabel[scope.row.role as AdminProjectMember['role']] }}</el-tag></template>
                  </el-table-column>
                  <el-table-column label="加入时间" width="145">
                    <template #default="scope">{{ formatDate(scope.row.assigned_at) }}</template>
                  </el-table-column>
                  <el-table-column label="操作" width="150" align="right">
                    <template #default="scope">
                      <template v-if="scope.row.role !== 'owner'">
                        <el-button text type="primary" size="small" @click="openMemberEditor(scope.row)">编辑</el-button>
                        <el-popconfirm title="确认移除该项目成员？" confirm-button-text="移除" cancel-button-text="取消" @confirm="removeMember(scope.row)">
                          <template #reference><el-button text type="danger" size="small">移除</el-button></template>
                        </el-popconfirm>
                      </template>
                      <span v-else class="muted-note">负责人</span>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </el-tab-pane>

            <el-tab-pane label="项目里程碑" name="milestones">
              <div class="project-drawer-toolbar">
                <p>用里程碑记录项目阶段、截止日期和交付状态。</p>
                <el-button type="primary" round @click="openMilestoneEditor()">+ 新建里程碑</el-button>
              </div>
              <div v-loading="workspaceLoading">
                <el-table :data="projectMilestones" class="admin-table" empty-text="还没有里程碑">
                  <el-table-column prop="title" label="里程碑" min-width="240" />
                  <el-table-column label="状态" width="130">
                    <template #default="scope"><el-tag :type="milestoneStatusType[scope.row.status as AdminProjectMilestone['status']]" effect="plain">{{ milestoneStatusLabel[scope.row.status as AdminProjectMilestone['status']] }}</el-tag></template>
                  </el-table-column>
                  <el-table-column label="截止时间" width="150">
                    <template #default="scope">{{ scope.row.due_at ? formatDate(scope.row.due_at) : '未设置' }}</template>
                  </el-table-column>
                  <el-table-column label="操作" width="145" align="right">
                    <template #default="scope">
                      <el-button text type="primary" size="small" @click="openMilestoneEditor(scope.row)">编辑</el-button>
                      <el-popconfirm title="确认删除该里程碑？" confirm-button-text="删除" cancel-button-text="取消" @confirm="removeMilestone(scope.row)">
                        <template #reference><el-button text type="danger" size="small">删除</el-button></template>
                      </el-popconfirm>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </el-tab-pane>
          </el-tabs>
        </template>
      </el-drawer>

      <el-dialog v-model="memberDialogOpen" :title="editingMember ? '编辑项目成员' : '添加项目成员'" width="min(92vw, 460px)">
        <el-form label-position="top">
          <el-form-item label="组织成员" required>
            <el-select v-model="memberForm.user_id" :disabled="Boolean(editingMember)" filterable style="width: 100%" placeholder="选择活跃成员">
              <el-option v-for="user in memberCandidates" :key="user.id" :label="`${user.name} · ${user.email}`" :value="user.id" />
            </el-select>
            <p v-if="!editingMember && !memberCandidates.length" class="form-help">没有可添加的活跃成员，或所有成员已经加入该项目。</p>
          </el-form-item>
          <el-form-item label="项目角色">
            <el-select v-model="memberForm.role" style="width: 100%">
              <el-option label="成员" value="member" />
              <el-option label="贡献者" value="contributor" />
              <el-option label="项目主力" value="lead" />
            </el-select>
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="memberDialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="memberSubmitting" @click="saveMember">保存成员</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="milestoneDialogOpen" :title="editingMilestone ? '编辑里程碑' : '新建里程碑'" width="min(92vw, 500px)">
        <el-form label-position="top">
          <el-form-item label="里程碑标题" required><el-input v-model="milestoneForm.title" maxlength="160" placeholder="例如：完成前端 API 对接" /></el-form-item>
          <div class="form-grid">
            <el-form-item label="状态">
              <el-select v-model="milestoneForm.status" style="width: 100%">
                <el-option label="未开始" value="planned" />
                <el-option label="进行中" value="active" />
                <el-option label="已完成" value="completed" />
              </el-select>
            </el-form-item>
            <el-form-item label="截止时间">
              <el-date-picker v-model="milestoneForm.due_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ss[Z]" placeholder="可选" style="width: 100%" />
            </el-form-item>
          </div>
        </el-form>
        <template #footer>
          <el-button @click="milestoneDialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="milestoneSubmitting" @click="saveMilestone">保存里程碑</el-button>
        </template>
      </el-dialog>
    </template>
  </AsyncState>
</template>

<style scoped>
.project-workspace-summary {
  display: grid;
  gap: 8px;
  margin-bottom: 18px;
  padding: 16px 18px;
  background: var(--md-sys-color-surface-container);
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: var(--md-shape-lg);
}

.project-workspace-summary span,
.project-workspace-summary small,
.project-drawer-toolbar p,
.muted-note {
  color: var(--md-sys-color-on-surface-variant);
}

.project-workspace-summary span {
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.project-workspace-summary strong {
  color: var(--md-sys-color-on-surface);
  font-size: 16px;
}

.project-drawer-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.project-drawer-toolbar p {
  margin: 0;
  font-size: 13px;
}

.muted-note {
  font-size: 12px;
}

@media (max-width: 640px) {
  .project-drawer-toolbar {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
