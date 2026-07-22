<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(async () => {
  const [applications, server] = await Promise.all([adminApi.getApplications(), adminApi.getServerStatus()])
  return { applications: applications.items, server }
})
const command = reactive({ value: 'list' })
const running = ref(false)
const pending = computed(() => data.value?.applications.filter((item) => item.status === 'pending') ?? [])

async function decide(id: string, decision: 'approve' | 'reject') {
  await (decision === 'approve' ? adminApi.approveApplication(id) : adminApi.rejectApplication(id))
  ElMessage.success(decision === 'approve' ? '申请已通过。' : '申请已拒绝。')
  refresh()
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
          <h2>审核与服务器</h2>
          <p>在此快速审批玩家加入申请，并向 Minecraft 服务器发送管理命令。</p>
        </div>
      </section>

      <section class="admin-two-column review-workspace">
        <article class="admin-panel">
          <div class="panel-heading">
            <div>
              <h2>白名单与成员申请</h2>
            </div>
            <el-tag type="warning">{{ pending.length }} 待处理</el-tag>
          </div>

          <div class="application-list">
            <article v-for="item in pending" :key="item.id" class="app-item-card">
              <div class="app-item-info">
                <strong>{{ item.applicant }}</strong>
                <small>{{ item.type === 'whitelist' ? '服务器白名单' : '成员申请' }} · {{ formatDate(item.submitted_at) }}</small>
                <p>{{ item.note }}</p>
              </div>
              <div class="app-item-actions">
                <el-button @click="decide(item.id, 'reject')">拒绝</el-button>
                <el-button type="primary" @click="decide(item.id, 'approve')">通过</el-button>
              </div>
            </article>

            <el-empty v-if="!pending.length" description="目前没有待处理的加入申请" />
          </div>
        </article>

        <article class="admin-panel command-panel">
          <div class="panel-heading">
            <div>
              <h2>{{ data.server.label }}</h2>
            </div>
            <span class="server-state" :class="data.server.state">
              <i />{{ data.server.state === 'online' ? '已连接' : '未连接' }}
            </span>
          </div>

          <dl class="server-facts">
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
              :disabled="data.server.state !== 'online'"
              @click="runCommand"
            >
              执行命令
            </el-button>
          </div>
        </article>
      </section>
    </template>
  </AsyncState>
</template>
