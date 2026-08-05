<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import type { AdminApplication, AdminApplicationFilters, ApplicationServerSync } from '@/api/types'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const filters = reactive<Required<Pick<AdminApplicationFilters, 'page' | 'page_size'>> & AdminApplicationFilters>({
  page: 1,
  page_size: 10,
  status: '',
  type: '',
  server_sync_status: '',
  query: '',
})

const { data, error, loading, refresh } = useAsyncData(async () => {
  const [applications, server, organization] = await Promise.all([adminApi.getApplications(filters), adminApi.getServerStatus(), adminApi.getOrganization()])
  return { applications, server, organization }
})
const command = reactive({ value: 'list' })
const running = ref(false)
const retryingId = ref('')
const applications = computed(() => data.value?.applications.items ?? [])
const isQutcraftOrganization = computed(() => data.value?.organization.slug === 'qutcraft')

function applyFilters() {
  filters.page = 1
  refresh()
}

function resetFilters() {
  Object.assign(filters, { page: 1, page_size: 10, status: '', type: '', server_sync_status: '', query: '' })
  refresh()
}

function changePage(page: number) {
  filters.page = page
  refresh()
}

function applicationStatusLabel(status: AdminApplication['status']) {
  return status === 'pending' ? '待处理' : status === 'approved' ? '已通过' : '已拒绝'
}

function syncStatusLabel(status: ApplicationServerSync['status']) {
  return status === 'succeeded' ? '同步完成' : status === 'failed' ? '同步失败' : '同步中'
}

function syncTagType(status: ApplicationServerSync['status']): 'success' | 'danger' | 'warning' {
  return status === 'succeeded' ? 'success' : status === 'failed' ? 'danger' : 'warning'
}

async function requestDecision(id: string, decision: 'approve' | 'reject') {
  let reason = ''
  try {
    const result = await ElMessageBox.prompt(
      decision === 'approve' ? '可以填写通过备注，留空也可继续。' : '请填写拒绝原因，该原因仅在管理后台显示。',
      decision === 'approve' ? '通过申请' : '拒绝申请',
      {
        confirmButtonText: decision === 'approve' ? '确认通过' : '确认拒绝',
        cancelButtonText: '取消',
        inputType: 'textarea',
        inputPlaceholder: decision === 'approve' ? '例如：资料完整，符合加入要求' : '请输入拒绝原因',
        inputValidator: (value) => {
          const normalized = value.trim()
          if (decision === 'reject' && !normalized) return '拒绝申请时必须填写原因。'
          if ([...normalized].length > 500) return '审核原因不能超过 500 个字符。'
          return true
        },
      },
    )
    reason = result.value.trim()
  } catch {
    return
  }

  const result = await (decision === 'approve' ? adminApi.approveApplication(id, reason) : adminApi.rejectApplication(id, reason))
  if (result.server_sync) {
    const label = result.server_sync.status === 'succeeded' ? '同步已完成' : result.server_sync.status === 'failed' ? '同步失败，可稍后重试' : '等待同步'
    ElMessage.success(`申请已通过；${label}（${result.server_sync.mode === 'mock' ? 'Mock' : 'RCON'}）。`)
  } else {
    ElMessage.success(decision === 'approve' ? '申请已通过。' : '申请已拒绝。')
  }
  refresh()
}

async function retryServerSync(id: string) {
  retryingId.value = id
  try {
    const result = await adminApi.retryApplicationServerSync(id)
    if (result.status === 'succeeded') {
      ElMessage.success(`服务器同步已完成（第 ${result.attempts} 次尝试）。`)
    } else {
      ElMessage.warning('服务器同步仍然失败，请检查适配器状态后重试。')
    }
    refresh()
  } finally {
    retryingId.value = ''
  }
}

async function runCommand() {
  if (!command.value.trim()) return
  running.value = true
  try {
    const result = await adminApi.runServerCommand(command.value)
    ElMessage.success(result.message)
    refresh()
  } finally {
    running.value = false
  }
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="admin-page-heading">
        <div>
          <h2>{{ isQutcraftOrganization ? '审核与服务器' : '申请审核' }}</h2>
          <p>{{ isQutcraftOrganization ? '筛选并审批加入申请，查看与受限服务器适配器分离保存的同步结果。' : '筛选并处理当前组织的成员申请，审核事实与组织操作均会保留审计记录。' }}</p>
        </div>
      </section>

      <section class="admin-two-column review-workspace">
        <article class="admin-panel">
          <div class="panel-heading">
            <div>
              <h2>{{ isQutcraftOrganization ? '白名单与成员申请' : '成员申请' }}</h2>
            </div>
            <el-tag>{{ data.applications.total }} 条结果</el-tag>
          </div>

          <div class="application-filters" aria-label="申请筛选">
            <el-input
              v-model="filters.query"
              clearable
              :placeholder="isQutcraftOrganization ? '搜索姓名、游戏 ID、邮箱或 QQ' : '搜索姓名、班级或邮箱'"
              @keyup.enter="applyFilters"
            />
            <el-select v-model="filters.status" aria-label="审批状态" placeholder="全部审批状态">
              <el-option label="全部审批状态" value="" />
              <el-option label="待处理" value="pending" />
              <el-option label="已通过" value="approved" />
              <el-option label="已拒绝" value="rejected" />
            </el-select>
            <el-select v-model="filters.type" aria-label="申请类型" placeholder="全部申请类型">
              <el-option label="全部申请类型" value="" />
              <el-option v-if="isQutcraftOrganization" label="服务器白名单" value="whitelist" />
              <el-option label="成员申请" value="membership" />
            </el-select>
            <el-select v-if="isQutcraftOrganization" v-model="filters.server_sync_status" aria-label="服务器同步状态" placeholder="全部同步状态">
              <el-option label="全部同步状态" value="" />
              <el-option label="无同步任务" value="none" />
              <el-option label="同步中" value="pending" />
              <el-option label="同步完成" value="succeeded" />
              <el-option label="同步失败" value="failed" />
            </el-select>
            <div class="application-filter-actions">
              <el-button type="primary" @click="applyFilters">筛选</el-button>
              <el-button @click="resetFilters">重置</el-button>
            </div>
          </div>

          <div class="application-list">
            <article v-for="item in applications" :key="item.id" class="app-item-card">
              <div class="app-item-info">
                <div class="application-title-row">
                  <strong>{{ item.applicant }}</strong>
                  <el-tag size="small" :type="item.status === 'pending' ? 'warning' : item.status === 'approved' ? 'success' : 'info'">
                    {{ applicationStatusLabel(item.status) }}
                  </el-tag>
                </div>
                <small>{{ item.type === 'whitelist' ? '服务器白名单' : '成员申请' }} · 提交于 {{ formatDate(item.submitted_at) }}</small>
                <div class="application-identifiers">
                  <span v-if="item.game_id">游戏 ID：{{ item.game_id }}</span>
                  <span v-if="item.class_name">班级：{{ item.class_name }}</span>
                  <span v-if="item.qq_number">QQ：{{ item.qq_number }}</span>
                  <span v-if="item.email">邮箱：{{ item.email }}</span>
                </div>
                <p>{{ item.note }}</p>
                <div v-if="item.decision_reason" class="application-decision-reason">
                  <strong>审核备注</strong>
                  <span>{{ item.decision_reason }}</span>
                </div>
                <div v-if="item.server_sync" class="application-sync" :class="`is-${item.server_sync.status}`">
                  <div>
                    <el-tag size="small" :type="syncTagType(item.server_sync.status)">
                      {{ syncStatusLabel(item.server_sync.status) }}
                    </el-tag>
                    <span>{{ item.server_sync.mode === 'mock' ? 'Mock 模拟' : item.server_sync.adapter }}</span>
                    <span>尝试 {{ item.server_sync.attempts }} 次</span>
                    <span>请求于 {{ formatDate(item.server_sync.requested_at) }}</span>
                  </div>
                  <p v-if="item.server_sync.last_error">{{ item.server_sync.last_error }}</p>
                  <p v-else-if="item.server_sync.message">{{ item.server_sync.message }}</p>
                </div>
              </div>
              <div v-if="item.status === 'pending'" class="app-item-actions">
                <el-button @click="requestDecision(item.id, 'reject')">拒绝</el-button>
                <el-button type="primary" @click="requestDecision(item.id, 'approve')">通过</el-button>
              </div>
              <div v-else class="app-item-actions">
                <el-button
                  v-if="item.server_sync?.status === 'failed'"
                  :loading="retryingId === item.id"
                  @click="retryServerSync(item.id)"
                >
                  重试同步
                </el-button>
              </div>
            </article>

            <el-empty v-if="!applications.length" description="没有符合当前筛选条件的申请" />
          </div>

          <el-pagination
            v-if="data.applications.total > data.applications.page_size"
            class="application-pagination"
            background
            layout="prev, pager, next"
            :current-page="data.applications.page"
            :page-size="data.applications.page_size"
            :total="data.applications.total"
            @current-change="changePage"
          />
        </article>

        <article v-if="isQutcraftOrganization" class="admin-panel command-panel">
          <div class="panel-heading">
            <div>
              <h2>{{ data.server.label }}</h2>
            </div>
            <span class="server-state" :class="data.server.state">
              <i />{{ data.server.mode === 'mock' ? 'Mock 模式' : data.server.state === 'online' ? '已连接' : '未连接' }}
            </span>
          </div>

          <dl class="server-facts">
            <div>
              <dt>适配器</dt>
              <dd>{{ data.server.adapter }} · {{ data.server.mode === 'mock' ? '模拟执行' : '真实 RCON' }}</dd>
            </div>
            <div>
              <dt>在线玩家</dt>
              <dd>{{ data.server.online_players }} / {{ data.server.max_players }}</dd>
            </div>
            <div>
              <dt>上次命令执行</dt>
              <dd>{{ data.server.last_command_at ? formatDate(data.server.last_command_at) : '暂无' }}</dd>
            </div>
          </dl>

          <div class="command-input-row">
            <el-input
              v-model="command.value"
              class="command-input"
              aria-label="管理命令"
              placeholder="输入控制台命令，例如：list"
            />
            <el-button
              type="primary"
              :loading="running"
              :disabled="!data.server.enabled"
              @click="runCommand"
            >
              {{ data.server.mode === 'mock' ? '模拟命令' : '执行命令' }}
            </el-button>
          </div>
        </article>
        <article v-else class="admin-panel command-panel">
          <div class="panel-heading">
            <div>
              <h2>审核边界</h2>
            </div>
            <span class="server-state online"><i /> 组织内审批</span>
          </div>

          <dl class="server-facts">
            <div>
              <dt>当前组织</dt>
              <dd>{{ data.organization.name }}</dd>
            </div>
            <div>
              <dt>申请范围</dt>
              <dd>成员加入与活动协作</dd>
            </div>
            <div>
              <dt>外部执行</dt>
              <dd>未配置，不会触发服务器命令</dd>
            </div>
          </dl>

          <p>通用组织的成员申请只更新平台内审批状态；Minecraft ServerAdapter 是 QUTCraft 场景的可选扩展。</p>
        </article>
      </section>
    </template>
  </AsyncState>
</template>
