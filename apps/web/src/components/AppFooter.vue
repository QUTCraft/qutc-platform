<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { organizationSlug } from '@/api/portal'
import { usePortalIdentity } from '@/composables/usePortalIdentity'

const { organization, loadPortalOrganization } = usePortalIdentity()
const organizationName = computed(() => organization.value?.name ?? (organizationSlug === 'qutcraft' ? 'QUTCraft Commons' : organizationSlug))
const filingNumber = computed(() => organization.value?.filing_number?.trim() ?? '')

onMounted(() => {
  void loadPortalOrganization().catch(() => undefined)
})
</script>

<template>
  <footer class="app-footer">
    <div class="app-footer-identity"><strong>{{ organizationName }}</strong><span> · 公共门户</span></div>
    <div class="app-footer-meta">
      <p>内容由独立管理端发布；门户仅消费公开 API 数据。</p>
      <span v-if="filingNumber" class="app-footer-filing">{{ filingNumber }}</span>
    </div>
  </footer>
</template>
