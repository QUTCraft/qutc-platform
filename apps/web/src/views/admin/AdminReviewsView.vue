<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import AsyncState from '@/components/AsyncState.vue'
import { adminApi } from '@/api/admin'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatDate } from '@/utils/format'

const { data, error, loading, refresh } = useAsyncData(async () => { const [applications, server] = await Promise.all([adminApi.getApplications(), adminApi.getServerStatus()]); return { applications: applications.items, server } })
const command = reactive({ value: 'list' })
const running = ref(false)
const pending = computed(() => data.value?.applications.filter((item) => item.status === 'pending') ?? [])
async function decide(id: string, decision: 'approve' | 'reject') { await (decision === 'approve' ? adminApi.approveApplication(id) : adminApi.rejectApplication(id)); ElMessage.success(decision === 'approve' ? '申请已通过并记录审计日志。' : '申请已拒绝。'); refresh() }
async function runCommand() { if (!command.value.trim()) return; running.value = true; try { const result = await adminApi.runServerCommand(command.value); ElMessage.success(result.message); refresh() } finally { running.value = false } }
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh"><template v-if="data"><section class="admin-page-heading"><div><p class="eyebrow">APPROVALS & SERVER ADAPTER</p><h2>审核与服务器</h2><p>审批与 RCON 命令均走受认证、可审计的管理接口，不会暴露给公开门户。</p></div></section><section class="admin-two-column review-workspace"><article class="admin-panel"><div class="panel-heading"><div><p class="eyebrow">PENDING</p><h2>白名单与成员申请</h2></div><el-tag type="warning">{{ pending.length }} 待处理</el-tag></div><div class="application-list"><article v-for="item in pending" :key="item.id"><div><strong>{{ item.applicant }}</strong><small>{{ item.type === 'whitelist' ? '服务器白名单' : '成员申请' }} · {{ formatDate(item.submitted_at) }}</small><p>{{ item.note }}</p></div><div><el-button @click="decide(item.id, 'reject')">拒绝</el-button><el-button type="primary" @click="decide(item.id, 'approve')">通过</el-button></div></article><el-empty v-if="!pending.length" description="没有待处理申请" /></div></article><article class="admin-panel command-panel"><div class="panel-heading"><div><p class="eyebrow">RESTRICTED RCON</p><h2>{{ data.server.label }}</h2></div><span class="server-state" :class="data.server.state"><i />{{ data.server.state === 'online' ? '已连接' : '未连接' }}</span></div><dl class="server-facts"><div><dt>在线玩家</dt><dd>{{ data.server.online_players }} / {{ data.server.max_players }}</dd></div><div><dt>上次命令</dt><dd>{{ data.server.last_command_at ? formatDate(data.server.last_command_at) : '暂无' }}</dd></div></dl><el-alert title="模拟环境仅记录命令，不会连接实际 Minecraft 服务器。" type="info" :closable="false" show-icon /><el-input v-model="command.value" class="command-input" aria-label="RCON 命令" placeholder="例如：list" /><el-button type="primary" :loading="running" :disabled="data.server.state !== 'online'" @click="runCommand">执行受限命令</el-button></article></section></template></AsyncState>
</template>
