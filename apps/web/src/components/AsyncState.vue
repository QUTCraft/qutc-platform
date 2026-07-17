<script setup lang="ts">
defineProps<{ loading: boolean; error?: Error }>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div v-if="loading" class="loading-state" aria-live="polite"><el-skeleton :rows="5" animated /></div>
  <el-result v-else-if="error" icon="error" title="暂时无法加载内容" :sub-title="error.message">
    <template #extra><el-button type="primary" round @click="emit('retry')">重试</el-button></template>
  </el-result>
  <slot v-else />
</template>
