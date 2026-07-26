<script setup lang="ts">
import { RouterLink } from 'vue-router'

defineProps<{
  eyebrow?: string
  title: string
  body: string
  tags?: string[]
  meta?: string
  to?: string
}>()
</script>

<template>
  <component :is="to ? RouterLink : 'article'" :to="to" class="content-card" :class="{ 'is-link': to }">
    <div v-if="eyebrow" class="eyebrow">{{ eyebrow }}</div>
    <h3>{{ title }}</h3>
    <p>{{ body }}</p>
    <div v-if="tags?.length" class="tag-row"><el-tag v-for="tag in tags" :key="tag" effect="light" round>{{ tag }}</el-tag></div>
    <small v-if="meta">{{ meta }}</small>
  </component>
</template>
