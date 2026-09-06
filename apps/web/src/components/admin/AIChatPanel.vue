<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import { adminApi } from '@/api/admin'
import type { ChatMessage, EditorChatInput, PersonalAIConfiguration } from '@/api/personal-ai-types'
import MarkdownContent from '@/components/MarkdownContent.vue'
import { session } from '@/stores/session'

const props = defineProps<{ article?: string; allowInsert?: boolean; sidebar?: boolean }>()
const emit = defineEmits<{ close: []; insert: [text: string] }>()
const config = ref<PersonalAIConfiguration>()
const form = reactive({ base_url: '', model: '', api_key: '' })
const source = ref<EditorChatInput['source']>('personal')
const messages = ref<ChatMessage[]>([])
const question = ref('')
const includeArticle = ref(false)
const error = ref('')
const busy = ref(false)
const saving = ref(false)
const loading = ref(true)
const requestID = ref('')
let controller: AbortController | undefined
let generation = 0
const available = computed(() => source.value === 'personal' ? config.value?.api_key_configured : config.value?.organization_available)

function reset() {
  controller?.abort()
  generation++
  busy.value = false
  messages.value = []
  requestID.value = ''
  error.value = ''
}
async function load() {
  const version = generation
  loading.value = true
  try {
    const value = await adminApi.getPersonalAI()
    if (version !== generation) return
    config.value = value
    Object.assign(form, { base_url: value.base_url, model: value.model, api_key: '' })
    source.value = value.api_key_configured || !value.organization_available ? 'personal' : 'organization'
  } catch (cause) {
    if (version === generation) error.value = cause instanceof Error ? cause.message : '配置加载失败'
  } finally { if (version === generation) loading.value = false }
}
async function save() {
  const version = generation
  saving.value = true
  error.value = ''
  try {
    const value = await adminApi.savePersonalAI({ ...form })
    if (version === generation) { config.value = value; form.api_key = ''; reset() }
  } catch (cause) { if (version === generation) error.value = cause instanceof Error ? cause.message : '保存失败' }
  finally { saving.value = false }
}
async function remove() {
  try { await ElMessageBox.confirm('删除本账号保存的 AI 地址、模型和密钥？', '删除个人配置', { type: 'warning' }) } catch { return }
  const version = generation
  saving.value = true
  try {
    await adminApi.deletePersonalAI()
    if (version === generation) { reset(); await load() }
  } catch (cause) { if (version === generation) error.value = cause instanceof Error ? cause.message : '删除失败' }
  finally { saving.value = false }
}
async function send() {
  if (!question.value.trim() || !available.value || busy.value) return
  error.value = ''
  const next: ChatMessage[] = [...messages.value, { role: 'user', content: question.value.trim() }]
  if (next.length > 20) { error.value = '已达到本次对话上限，请清空后开始新对话。'; return }
  const version = generation
  controller = new AbortController()
  busy.value = true
  try {
    const result = await adminApi.chat({ source: source.value, messages: next, article: includeArticle.value ? props.article : undefined }, controller.signal)
    if (version !== generation) return
    messages.value = [...next, { role: 'assistant', content: result.markdown }]
    question.value = ''
    requestID.value = result.request_id
  } catch (cause) {
    if (version === generation && !controller.signal.aborted) error.value = cause instanceof Error ? cause.message : '请求失败，可再次发送。'
  } finally { if (version === generation) busy.value = false }
}
watch(source, () => { reset(); includeArticle.value = false })
watch(() => `${session.user?.id}/${session.user?.organization_id}`, () => {
  reset(); config.value = undefined; Object.assign(form, { base_url: '', model: '', api_key: '' }); question.value = ''; includeArticle.value = false
  void load()
}, { immediate: true })
onBeforeUnmount(() => { reset(); form.api_key = '' })
</script>

<template>
  <aside class="ai-chat-panel" :class="{ 'is-sidebar': sidebar }" aria-label="AI 咨询">
    <header><h2>AI 咨询</h2><el-button v-if="sidebar" text @click="emit('close')">收起</el-button></header>
    <p>对话仅保留在当前页面，不会自动保存或发布文章。AI 回答请自行核实。</p>
    <el-select v-model="source" aria-label="AI 来源" :disabled="loading || saving || busy">
      <el-option label="我的 AI 接口" value="personal" />
      <el-option label="组织 AI 接口" value="organization" :disabled="!config?.organization_available" />
    </el-select>
    <details v-if="source === 'personal'" :open="!config?.api_key_configured">
      <summary>个人接口配置 · {{ config?.api_key_configured ? '已保存密钥' : '尚未配置' }}</summary>
      <p>支持兼容 Chat Completions 的公开 HTTPS 接口。密钥在服务端加密保存，不回显；请仅使用信任的服务。个人额度为每小时 30 次。</p>
      <el-form label-position="top" @submit.prevent="save">
        <el-form-item label="API 调用地址"><el-input v-model="form.base_url" placeholder="https://api.example.com/v1" :maxlength="500" /></el-form-item>
        <el-form-item label="模型名称"><el-input v-model="form.model" :maxlength="120" /></el-form-item>
        <el-form-item label="API Key"><el-input v-model="form.api_key" type="password" autocomplete="new-password" :maxlength="4096" placeholder="留空保留原密钥；更换地址时必须重新填写" /></el-form-item>
        <div class="chat-actions"><el-button native-type="submit" :disabled="loading || busy" :loading="saving">保存个人配置</el-button><el-button v-if="config?.api_key_configured" text type="danger" :disabled="saving || busy" @click="remove">删除配置</el-button></div>
      </el-form>
    </details>
    <p v-else>使用组织服务并计入组织配置的个人小时额度；不会使用或覆盖你的个人密钥。</p>
    <div class="chat-history" aria-live="polite">
      <p v-if="!messages.length">可以请 AI 帮忙梳理思路、润色文章或回答问题。</p>
      <article v-for="(message, index) in messages" :key="index" :class="message.role">
        <strong>{{ message.role === 'user' ? '你' : 'AI' }}</strong>
        <MarkdownContent :markdown="message.content" />
        <el-button v-if="message.role === 'assistant' && allowInsert" text @click="emit('insert', message.content)">追加到正文</el-button>
      </article>
    </div>
    <small v-if="requestID">Request ID：{{ requestID }}</small>
    <el-checkbox v-if="article !== undefined" v-model="includeArticle" :disabled="busy">将当前文章发送给所选 AI 服务（含未保存内容）</el-checkbox>
    <el-input v-model="question" type="textarea" :rows="4" :maxlength="12000" aria-label="咨询问题" placeholder="输入问题；点击发送后才会调用接口" :disabled="busy" />
    <p v-if="error" role="alert" class="chat-error">{{ error }}</p>
    <div class="chat-actions">
      <el-button type="primary" :disabled="!available || !question.trim() || busy || saving || loading" @click="send">{{ busy ? '正在回答…' : '发送' }}</el-button>
      <el-button v-if="busy" @click="reset">停止</el-button>
      <el-button v-else :disabled="saving" @click="reset">清空对话</el-button>
    </div>
  </aside>
</template>

<style scoped>
.ai-chat-panel { display: flex; flex-direction: column; gap: 16px; padding: 24px; min-width: 0; color: var(--md-sys-color-on-surface); background: var(--md-sys-color-surface-container); border: 1px solid var(--md-sys-color-outline-variant); border-radius: 24px; }
.ai-chat-panel header, .chat-actions { display: flex; align-items: center; justify-content: space-between; gap: 8px; flex-wrap: wrap; }
h2, p { margin: 0; } p, small { color: var(--md-sys-color-on-surface-variant); overflow-wrap: anywhere; }
summary { cursor: pointer; padding: 12px 0; } details p { margin-bottom: 16px; }
.chat-history { display: grid; gap: 16px; } .chat-history article { min-width: 0; padding: 12px; background: var(--md-sys-color-surface); border-radius: 12px; overflow-wrap: anywhere; }
.chat-error { color: var(--md-sys-color-error); }
.ai-chat-panel :deep(.el-checkbox) { white-space: normal; height: auto; } .ai-chat-panel :deep(.el-checkbox__label) { white-space: normal; }
@media (min-width: 1400px) {
  .is-sidebar { position: fixed; z-index: 100; top: 96px; right: 16px; bottom: 16px; width: min(420px, calc(100vw - 32px)); overflow-y: auto; box-shadow: var(--md-elevation-2); }
}
</style>
