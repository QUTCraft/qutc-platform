<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import AsyncState from '@/components/AsyncState.vue'
import { invitationApi } from '@/api/invitations'
import { session } from '@/stores/session'
import { useAsyncData } from '@/composables/useAsyncData'

const route = useRoute()
const router = useRouter()
const token = computed(() => String(route.params.token))
const { data, error, loading, refresh } = useAsyncData(() => invitationApi.get(token.value))
const accepting = ref(false)
const accepted = ref(false)

async function accept() {
  if (!session.user) {
    await router.push({ name: 'login', query: { redirect: route.fullPath } })
    return
  }
  accepting.value = true
  try {
    await invitationApi.accept(token.value)
    accepted.value = true
    ElMessage.success('邀请已接受，欢迎加入组织。')
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : '邀请接受失败。')
  } finally {
    accepting.value = false
  }
}
</script>

<template>
  <AsyncState :loading="loading" :error="error" @retry="refresh">
    <template v-if="data">
      <section class="invite-page surface-panel">
        <p class="eyebrow">QUTCraft Commons · 成员邀请</p>
        <h1>{{ data.organization_name }} 邀请你加入</h1>
        <p class="invite-description">邀请邮箱：{{ data.email }}<br />加入角色：{{ data.role === 'administrator' ? '管理员' : data.role === 'editor' ? '编辑者' : '成员' }}</p>
        <el-alert v-if="data.status === 'accepted' || accepted" title="邀请已使用" type="success" :closable="false" show-icon>当前邀请已经完成，无需重复操作。</el-alert>
        <el-alert v-else-if="data.status !== 'pending'" title="邀请不可用" type="warning" :closable="false" show-icon>该邀请已过期、撤销或失效。</el-alert>
        <template v-else>
          <p class="invite-description">请使用邀请邮箱登录或注册后接受邀请。邀请有效期至 {{ new Date(data.expires_at).toLocaleString() }}。</p>
          <div class="invite-actions">
            <el-button type="primary" round :loading="accepting" @click="accept">{{ session.user ? '接受邀请' : '登录并接受' }}</el-button>
            <RouterLink v-if="!session.user" class="button button-secondary" :to="{ name: 'register', query: { invitation: token } }">使用邀请注册</RouterLink>
          </div>
        </template>
      </section>
    </template>
  </AsyncState>
</template>

<style scoped>
.invite-page { width: min(620px, 100%); margin: 72px auto; padding: 40px; }
.invite-page h1 { margin: 12px 0 20px; font-size: clamp(2rem, 5vw, 3.4rem); line-height: 1.12; }
.invite-description { color: var(--md-sys-color-on-surface-variant); line-height: 1.8; }
.invite-actions { display: flex; align-items: center; gap: 12px; margin-top: 28px; }
.button-secondary { display: inline-flex; align-items: center; min-height: 40px; padding: 0 18px; border: 1px solid var(--md-sys-color-outline); border-radius: 999px; color: var(--md-sys-color-on-surface); text-decoration: none; font-weight: 700; }
@media (max-width: 640px) { .invite-page { margin: 24px 0; padding: 24px; } .invite-actions { align-items: stretch; flex-direction: column; } }
</style>
