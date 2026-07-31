<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminInvitation, AdminMembershipWriteState, AdminUser, InvitationRole } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const page = ref(1)
const { data, error, loading, refresh } = useAsyncData(() => adminApi.getUsers({ page: page.value }))
const roleLabel = { owner: '所有者', administrator: '管理员', editor: '编辑者', member: '成员' }
const stateLabel = { active: '正常', invited: '待加入', disabled: '已停用' }
const dialogOpen = ref(false)
const saving = ref(false)
const editingUser = ref<AdminUser | null>(null)
const form = reactive({ role: 'member' as AdminUser['role'], state: 'active' as AdminMembershipWriteState })
const inviteDialogOpen = ref(false)
const inviting = ref(false)
const retryingEmail = ref(false)
const inviteResult = ref<AdminInvitation | null>(null)
const inviteForm = reactive<{ email: string; role: InvitationRole; expires_in_hours: number }>({ email: '', role: 'member', expires_in_hours: 168 })
const inviteRules: FormRules = {
  email: [{ required: true, type: 'email', message: '请输入有效邮箱账号', trigger: 'blur' }],
}
const inviteFormRef = ref<FormInstance>()
const inviteLink = computed(() => inviteResult.value ? new URL(inviteResult.value.invite_url, window.location.origin).toString() : '')

async function changePage(value: number) {
  page.value = value
  await refresh()
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
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '邀请创建失败。')
  } finally {
    inviting.value = false
  }
}

async function retryInvitationEmail() {
  if (!inviteResult.value) return
  retryingEmail.value = true
  try {
    inviteResult.value = await adminApi.retryInvitationEmail(inviteResult.value.id)
    if (inviteResult.value.delivery.status === 'sent') {
      ElMessage.success('新邀请链接已生成，邮件已发送；旧链接已失效。')
    } else {
      ElMessage.warning('已生成新链接，但邮件仍未发送成功；请复制链接手动发送。')
    }
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '邮件重试失败。')
  } finally {
    retryingEmail.value = false
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
        <el-button round type="primary" @click="openInvite">+ 邀请成员</el-button>
      </section>

      <section class="admin-panel">
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
              <el-button text type="primary" size="small" @click="openEditor(scope.row)">编辑</el-button>
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
          <el-button v-if="!inviteResult" type="primary" :loading="inviting" @click="createInvitation">创建邀请</el-button>
        </template>
      </el-dialog>
    </template>
  </AsyncState>
</template>

<style scoped>
.delivery-error {
  color: var(--el-color-warning-dark-2);
  overflow-wrap: anywhere;
}

@media (max-width: 540px) {
  .invite-link-box {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
