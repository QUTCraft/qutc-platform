<script setup lang="ts">
defineProps<{ eyebrow: string; title: string; description?: string; actionLabel?: string; actionTo?: string }>()
</script>

<template>
  <div class="section-heading">
    <div><div class="eyebrow">{{ eyebrow }}</div><h2>{{ title }}</h2><p v-if="description">{{ description }}</p></div>
    <RouterLink v-if="actionLabel && actionTo" class="text-link" :to="actionTo">{{ actionLabel }} →</RouterLink>
  </div>
</template>
