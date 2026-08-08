<script setup lang="ts">
withDefaults(defineProps<{ label?: string }>(), {
  label: '正在加载页面…',
})
</script>

<template>
  <div class="route-loading-state" role="status" aria-live="polite">
    <span class="route-loading-mark" aria-hidden="true" />
    <span>{{ label }}</span>
  </div>
</template>
