<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminInvitation, AdminInvitationSummary, AdminMembershipWriteState, AdminUser, BatchInvitationResponse, InvitationRole } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const page = ref(1)
const { data, error, loading, refresh } = useAsyncData(() => adminApi.getUsers({ page: page.value }))
const invitationPage = ref(1)
const {
  data: invitationData,
  error: invitationError,
  loading: invitationsLoading,
  refresh: refreshInvitations,
} = useAsyncData(() => adminApi.getInvitations({ page: invitationPage.value, page_size: 10, status: 'pending' }))
const roleLabel = { owner: '所有者', administrator: '管理员', editor: '编辑者', member: '成员' }
const stateLabel = { active: '正常', invited: '待加入', disabled: '已停用' }
const dialogOpen = ref(false)
const saving = ref(false)
const editingUser = ref<AdminUser | null>(null)
const form = reactive({ role: 'member' as AdminUser['role'], state: 'active' as AdminMembershipWriteState })
const inviteDialogOpen = ref(false)
const inviting = ref(false)
const retryingEmail = ref(false)
const revokingInvitationID = ref('')
const inviteResult = ref<AdminInvitation | null>(null)
const inviteForm = reactive<{ email: string; role: InvitationRole; expires_in_hours: number }>({ email: '', role: 'member', expires_in_hours: 168 })
const inviteRules: FormRules = {
  email: [{ required: true, type: 'email', message: '请输入有效邮箱账号', trigger: 'blur' }],
}
const inviteFormRef = ref<FormInstance>()
const inviteLink = computed(() => inviteResult.value ? new URL(inviteResult.value.invite_url, window.location.origin).toString() : '')
const batchDialogOpen = ref(false)
const batchInviting = ref(false)
const batchResult = ref<BatchInvitationResponse | null>(null)
const batchForm = reactive<{ emails: string; role: InvitationRole; expires_in_hours: number }>({ emails: '', role: 'member', expires_in_hours: 168 })
const batchEmails = computed(() => {
  const seen = new Set<string>()
  return batchForm.emails
    .split(/[\n,;，；]+/)
    .map((email) => email.trim().toLowerCase())
    .filter((email) => {
      if (!email || seen.has(email)) return false
      seen.add(email)
      return true
    })
})
const successfulBatchInvitations = computed(() => batchResult.value?.results.flatMap((item) => item.invitation ? [item.invitation] : []) ?? [])

async function changePage(value: number) {
  page.value = value
  await refresh()
}

async function changeInvitationPage(value: number) {
  invitationPage.value = value
  await refreshInvitations()
}

function openEditor(user: AdminUser) {
  editingUser.value = user
  Object.assign(form, { role: user.role, state: user.state === 'disabled' ? 'disabled' : 'active' })
  dialogOpen.value = true
}

function openInvite() {
  inviteResult.value = null
  Object.assign(inviteForm, { email: '', role: 'member', expires_in_hours: 168 })
  inviteDialogOpen.value = true
}

async function createInvitation() {
  if (!inviteFormRef.value || !(await inviteFormRef.value.validate().catch(() => false))) return
  inviting.value = true
  try {
    inviteResult.value = await adminApi.createInvitation(inviteForm)
    if (inviteResult.value.delivery.status === 'sent') {
      ElMessage.success('邀请链接已创建，邮件已发送。')
    } else if (inviteResult.value.delivery.status === 'failed') {
      ElMessage.warning('邀请已创建，但邮件发送失败；链接仍可正常使用。')
    } else {
      ElMessage.success('邀请链接已创建，请复制给对方。')
    }
    await refresh()
    await refreshInvitations()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '邀请创建失败。')
  } finally {
    inviting.value = false
  }
}

async function retryInvitationEmail(invitationID = inviteResult.value?.id) {
  if (!invitationID) return
  retryingEmail.value = true
  try {
    inviteResult.value = await adminApi.retryInvitationEmail(invitationID)
    inviteDialogOpen.value = true
    if (inviteResult.value.delivery.status === 'sent') {
      ElMessage.success('新邀请链接已生成，邮件已发送；旧链接已失效。')
    } else {
      ElMessage.warning('已生成新链接，但邮件仍未发送成功；请复制链接手动发送。')
    }
    await refreshInvitations()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '邮件重试失败。')
  } finally {
    retryingEmail.value = false
  }
}

function openBatchInvite() {
  batchResult.value = null
  Object.assign(batchForm, { emails: '', role: 'member', expires_in_hours: 168 })
  batchDialogOpen.value = true
}

async function createBatchInvitations() {
  if (batchEmails.value.length < 1 || batchEmails.value.length > 20) {
    ElMessage.warning('请输入 1 到 20 个不重复的邮箱地址。')
    return
  }
  batchInviting.value = true
  try {
    batchResult.value = await adminApi.createBatchInvitations({
      invitations: batchEmails.value.map((email) => ({ email, role: batchForm.role, expires_in_hours: batchForm.expires_in_hours })),
    })
    if (batchResult.value.failed === 0) ElMessage.success(`已创建 ${batchResult.value.succeeded} 条邀请。`)
    else ElMessage.warning(`批量处理完成：${batchResult.value.succeeded} 条成功，${batchResult.value.failed} 条失败。`)
    await refresh()
    await refreshInvitations()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '批量邀请失败。')
  } finally {
    batchInviting.value = false
  }
}

function absoluteInviteURL(invitation: AdminInvitation) {
  return new URL(invitation.invite_url, window.location.origin).toString()
}

async function copyBatchInviteLink(invitation: AdminInvitation) {
  await navigator.clipboard.writeText(absoluteInviteURL(invitation))
  ElMessage.success(`已复制 ${invitation.email} 的邀请链接。`)
}

async function copyAllBatchInviteLinks() {
  if (successfulBatchInvitations.value.length === 0) return
  await navigator.clipboard.writeText(successfulBatchInvitations.value.map((invitation) => `${invitation.email}\t${absoluteInviteURL(invitation)}`).join('\n'))
  ElMessage.success(`已复制 ${successfulBatchInvitations.value.length} 条邀请链接。`)
}

async function revokeInvitation(invitation: Pick<AdminInvitationSummary, 'id' | 'email'>) {
  try {
    await ElMessageBox.confirm(
      `撤销发往 ${invitation.email} 的邀请后，现有链接会立即失效，且不能恢复。`,
      '确认撤销邀请',
      { confirmButtonText: '确认撤销', cancelButtonText: '取消', type: 'warning' },
    )
  } catch {
    return
  }
  revokingInvitationID.value = invitation.id
  try {
    await adminApi.revokeInvitation(invitation.id)
    if (inviteResult.value?.id === invitation.id) inviteResult.value.status = 'revoked'
    ElMessage.success('邀请已撤销，原链接现已失效。')
    await refreshInvitations()
    await refresh()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '邀请撤销失败。')
  } finally {
    revokingInvitationID.value = ''
  }
}

async function copyInviteLink() {
  if (!inviteResult.value) return
  const link = new URL(inviteResult.value.invite_url, window.location.origin).toString()
  await navigator.clipboard.writeText(link)
  ElMessage.success('邀请链接已复制。')
}

async function saveUser() {
  if (!editingUser.value) return
  if (form.state === 'disabled' && editingUser.value.state !== 'disabled') {
    try {
      await ElMessageBox.confirm(
        '停用后，该成员现有 Access Token 会立即失效，Refresh Token 也会被撤销；重新启用后需要再次登录。',
        '确认停用成员',
        { confirmButtonText: '确认停用', cancelButtonText: '取消', type: 'warning' },
      )
    } catch {
      return
    }
  }
  saving.value = true
  try {
    await adminApi.updateUser(editingUser.value.id, form)
    ElMessage.success('成员信息已更新。')
    dialogOpen.value = false
    await refresh()
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '成员信息保存失败。')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>成员与权限</h2>
          <p>管理组织成员及其在工作台中的协作权限与分配状态。</p>
        </div>
        <div class="heading-actions">
          <el-button round @click="openBatchInvite">批量邀请</el-button>
          <el-button round type="primary" @click="openInvite">+ 邀请成员</el-button>
        </div>
      </section>

      <section class="admin-panel member-panel">
        <el-table :data="data.items" class="admin-table">
          <el-table-column label="成员" min-width="220">
            <template #default="scope">
              <div class="person-cell">
                <span class="avatar-bubble">{{ scope.row.name.slice(0, 1) }}</span>
                <div>
                  <strong>{{ scope.row.name }}</strong>
                  <small>{{ scope.row.email }}</small>
                </div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="角色" width="140">
            <template #default="scope">
              <el-tag effect="plain">{{ roleLabel[scope.row.role as keyof typeof roleLabel] }}</el-tag>
            </template>
          </el-table-column>

          <el-table-column label="状态" width="120">
            <template #default="scope">
              <el-tag :type="scope.row.state === 'active' ? 'success' : scope.row.state === 'invited' ? 'warning' : 'info'">
                {{ stateLabel[scope.row.state as keyof typeof stateLabel] }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column label="加入时间" width="150">
            <template #default="scope">{{ formatDate(scope.row.joined_at) }}</template>
          </el-table-column>

          <el-table-column width="110" align="right">
            <template #default="scope">
              <el-button v-if="scope.row.state !== 'invited'" text type="primary" size="small" @click="openEditor(scope.row)">编辑</el-button>
              <span v-else class="form-help">等待接受</span>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination
          v-if="data.total > data.page_size"
          class="application-pagination"
          background
          layout="total, prev, pager, next"
          :current-page="data.page"
          :page-size="data.page_size"
          :total="data.total"
          @current-change="changePage"
        />
      </section>

      <section class="admin-section-heading invitation-heading">
        <div>
          <h3>待处理邀请</h3>
          <p>查看尚未接受的邀请；撤销后原链接会立即失效。</p>
        </div>
        <el-tag round effect="plain">{{ invitationData?.total ?? 0 }} 条待处理</el-tag>
      </section>

      <AsyncState :loading="invitationsLoading" :error="invitationError" @retry="refreshInvitations">
        <template v-if="invitationData">
          <section class="admin-panel invitation-panel">
            <el-empty v-if="invitationData.items.length === 0" description="当前没有待处理邀请" :image-size="72" />
            <el-table v-else :data="invitationData.items" class="admin-table">
              <el-table-column label="邀请邮箱" min-width="220" prop="email" />
              <el-table-column label="角色" width="130">
                <template #default="scope">{{ roleLabel[scope.row.role as keyof typeof roleLabel] }}</template>
              </el-table-column>
              <el-table-column label="邮件" width="120">
                <template #default="scope">
                  <el-tag :type="scope.row.delivery.status === 'sent' ? 'success' : scope.row.delivery.status === 'failed' ? 'warning' : 'info'" effect="plain">
                    {{ scope.row.delivery.status === 'sent' ? '已发送' : scope.row.delivery.status === 'failed' ? '发送失败' : '未启用' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="有效期至" width="170">
                <template #default="scope">{{ formatDate(scope.row.expires_at) }}</template>
              </el-table-column>
              <el-table-column label="操作" width="190" align="right" fixed="right">
                <template #default="scope">
                  <el-button
                    v-if="scope.row.delivery.status === 'failed'"
                    text
                    type="primary"
                    :loading="retryingEmail"
                    @click="retryInvitationEmail(scope.row.id)"
                  >
                    重试邮件
                  </el-button>
                  <el-button
                    text
                    type="danger"
                    :loading="revokingInvitationID === scope.row.id"
                    @click="revokeInvitation(scope.row)"
                  >
                    撤销
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            <el-pagination
              v-if="invitationData.total > invitationData.page_size"
              class="application-pagination"
              background
              layout="total, prev, pager, next"
              :current-page="invitationData.page"
              :page-size="invitationData.page_size"
              :total="invitationData.total"
              @current-change="changeInvitationPage"
            />
          </section>
        </template>
      </AsyncState>

      <el-dialog v-model="dialogOpen" title="编辑成员" width="min(92vw, 440px)">
        <p v-if="editingUser">{{ editingUser.name }} · {{ editingUser.email }}</p>
        <el-form label-position="top">
          <el-form-item label="角色">
            <el-select v-model="form.role" style="width: 100%">
              <el-option label="成员" value="member" />
              <el-option label="编辑者" value="editor" />
              <el-option label="管理员" value="administrator" />
              <el-option label="所有者" value="owner" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="form.state" style="width: 100%">
              <el-option label="正常" value="active" />
              <el-option label="已停用" value="disabled" />
            </el-select>
          </el-form-item>
          <p class="form-help">“待加入”只由邀请流程产生，不能手动设置。停用会立即终止该成员的受保护访问。</p>
        </el-form>
        <template #footer>
          <el-button @click="dialogOpen = false">取消</el-button>
          <el-button type="primary" :loading="saving" @click="saveUser">保存</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="inviteDialogOpen" :title="inviteResult ? '邀请链接已创建' : '邀请新成员'" width="min(92vw, 520px)">
        <template v-if="inviteResult">
          <el-alert
            :title="inviteResult.delivery.status === 'sent' ? '邀请邮件已发送' : inviteResult.delivery.status === 'failed' ? '邀请已创建，但邮件发送失败' : '邮件投递未启用'"
            :type="inviteResult.delivery.status === 'sent' ? 'success' : inviteResult.delivery.status === 'failed' ? 'warning' : 'info'"
            :closable="false"
            show-icon
          >
            {{
              inviteResult.delivery.status === 'sent'
                ? `已发送至 ${inviteResult.email}；下方链接仍只展示本次。`
                : inviteResult.delivery.status === 'failed'
                  ? '邀请本身不受影响，可复制链接手动发送，或轮换链接后重试邮件。'
                  : '服务端未配置 SMTP，请复制下方链接发送给成员。'
            }}
          </el-alert>
          <div class="invite-link-box">
            <el-input :model-value="inviteLink" readonly />
            <el-button type="primary" @click="copyInviteLink">复制链接</el-button>
          </div>
          <p class="form-help">{{ inviteResult.email }} · {{ roleLabel[inviteResult.role] }} · 有效期至 {{ formatDate(inviteResult.expires_at) }}</p>
          <p v-if="inviteResult.delivery.last_error" class="form-help delivery-error">
            投递失败：{{ inviteResult.delivery.last_error }}
          </p>
          <p v-if="inviteResult.delivery.attempts" class="form-help">
            邮件尝试 {{ inviteResult.delivery.attempts }} 次
            <template v-if="inviteResult.delivery.last_attempt_at"> · {{ formatDate(inviteResult.delivery.last_attempt_at) }}</template>
          </p>
        </template>
        <el-form v-else ref="inviteFormRef" :model="inviteForm" :rules="inviteRules" label-position="top">
          <el-form-item label="成员邮箱" prop="email"><el-input v-model="inviteForm.email" type="email" autocomplete="email" placeholder="member@example.com" /></el-form-item>
          <el-form-item label="加入角色">
            <el-select v-model="inviteForm.role" style="width: 100%">
              <el-option label="成员" value="member" />
              <el-option label="编辑者" value="editor" />
              <el-option label="管理员" value="administrator" />
            </el-select>
          </el-form-item>
          <el-form-item label="有效期（小时）"><el-input-number v-model="inviteForm.expires_in_hours" :min="1" :max="720" /></el-form-item>
          <p class="form-help">默认 7 天；同一组织同一邮箱只能存在一个待处理邀请，不能通过邀请授予所有者角色。</p>
        </el-form>
        <template #footer>
          <el-button @click="inviteDialogOpen = false">关闭</el-button>
          <el-button
            v-if="inviteResult?.delivery.status === 'failed'"
            type="warning"
            plain
            :loading="retryingEmail"
            @click="retryInvitationEmail"
          >
            轮换链接并重试邮件
          </el-button>
          <el-button
            v-if="inviteResult?.status === 'pending'"
            type="danger"
            plain
            :loading="revokingInvitationID === inviteResult.id"
            @click="revokeInvitation(inviteResult)"
          >
            撤销邀请
          </el-button>
          <el-button v-if="!inviteResult" type="primary" :loading="inviting" @click="createInvitation">创建邀请</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="batchDialogOpen" :title="batchResult ? '批量邀请结果' : '批量邀请成员'" width="min(94vw, 760px)">
        <template v-if="batchResult">
          <el-alert
            :title="`处理完成：${batchResult.succeeded} 条成功，${batchResult.failed} 条失败`"
            :type="batchResult.failed === 0 ? 'success' : batchResult.succeeded === 0 ? 'error' : 'warning'"
            :closable="false"
            show-icon
          >
            成功项的链接只在本次结果中展示。关闭窗口前请完成复制；邮件投递状态可在待处理邀请列表继续查看。
          </el-alert>
          <div v-if="successfulBatchInvitations.length" class="batch-result-actions">
            <el-button type="primary" plain @click="copyAllBatchInviteLinks">复制全部成功链接</el-button>
          </div>
          <el-table :data="batchResult.results" class="batch-result-table">
            <el-table-column label="#" width="54">
              <template #default="scope">{{ scope.row.index + 1 }}</template>
            </el-table-column>
            <el-table-column label="邮箱" min-width="210" prop="email" />
            <el-table-column label="结果" min-width="210">
              <template #default="scope">
                <el-tag v-if="scope.row.succeeded" type="success" effect="plain">已创建</el-tag>
                <span v-else class="batch-error">{{ scope.row.error?.message ?? '处理失败' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="邮件" width="110">
              <template #default="scope">
                <span v-if="!scope.row.invitation">—</span>
                <span v-else>{{ scope.row.invitation.delivery.status === 'sent' ? '已发送' : scope.row.invitation.delivery.status === 'failed' ? '失败' : '未启用' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" align="right" fixed="right">
              <template #default="scope">
                <el-button v-if="scope.row.invitation" text type="primary" @click="copyBatchInviteLink(scope.row.invitation)">复制链接</el-button>
              </template>
            </el-table-column>
          </el-table>
        </template>
        <el-form v-else label-position="top">
          <el-form-item label="成员邮箱">
            <el-input
              v-model="batchForm.emails"
              type="textarea"
              :rows="8"
              resize="vertical"
              placeholder="每行一个邮箱，也可使用逗号或分号分隔&#10;member-a@example.com&#10;member-b@example.com"
            />
          </el-form-item>
          <p class="form-help batch-count" :class="{ 'batch-count-invalid': batchEmails.length > 20 }">
            已识别 {{ batchEmails.length }} 个不重复邮箱，单次最多 20 个。
          </p>
          <div class="batch-options">
            <el-form-item label="统一加入角色">
              <el-select v-model="batchForm.role">
                <el-option label="成员" value="member" />
                <el-option label="编辑者" value="editor" />
                <el-option label="管理员" value="administrator" />
              </el-select>
            </el-form-item>
            <el-form-item label="有效期（小时）">
              <el-input-number v-model="batchForm.expires_in_hours" :min="1" :max="720" />
            </el-form-item>
          </div>
          <p class="form-help">每条邀请独立处理；已有成员、待处理邀请或错误邮箱不会阻断其他记录。</p>
        </el-form>
        <template #footer>
          <el-button @click="batchDialogOpen = false">关闭</el-button>
          <el-button v-if="batchResult" @click="batchResult = null">继续邀请</el-button>
          <el-button v-else type="primary" :loading="batchInviting" @click="createBatchInvitations">开始批量邀请</el-button>
        </template>
      </el-dialog>
    </template>
  </AsyncState>
</template>

<style scoped>
.heading-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.invitation-heading {
  align-items: flex-end;
  display: flex;
  justify-content: space-between;
  margin: 28px 4px 12px;
}

.invitation-heading h3,
.invitation-heading p {
  margin: 0;
}

.invitation-heading p {
  color: var(--md-sys-color-on-surface-variant);
  margin-top: 6px;
}

.invitation-panel {
  overflow: hidden;
}

.batch-options {
  display: grid;
  gap: 16px;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
}

.batch-options :deep(.el-select),
.batch-options :deep(.el-input-number) {
  width: 100%;
}

.batch-count {
  margin: -10px 0 18px;
}

.batch-count-invalid,
.batch-error {
  color: var(--el-color-danger);
}

.batch-result-actions {
  display: flex;
  justify-content: flex-end;
  margin: 16px 0 10px;
}

.batch-result-table {
  width: 100%;
}

.delivery-error {
  color: var(--el-color-warning-dark-2);
  overflow-wrap: anywhere;
}

@media (max-width: 540px) {
  .heading-actions {
    width: 100%;
  }

  .heading-actions :deep(.el-button) {
    flex: 1;
    margin-left: 0;
  }

  .batch-options {
    grid-template-columns: 1fr;
  }

  .invitation-heading {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
  }

  .invite-link-box {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
