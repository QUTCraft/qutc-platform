<script setup lang="ts">
defineProps<{
  modelValue: boolean
  disabled?: boolean
  loading?: boolean
  onLabel?: string
  offLabel?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  change: [value: boolean]
}>()

function onChange(event: Event) {
  const checked = (event.target as HTMLInputElement).checked
  emit('update:modelValue', checked)
  emit('change', checked)
}
</script>

<template>
  <label class="md-switch" :class="{ 'is-on': modelValue, 'is-disabled': disabled, 'is-loading': loading }">
    <input
      class="md-switch-input"
      type="checkbox"
      role="switch"
      :checked="modelValue"
      :disabled="disabled || loading"
      :aria-checked="modelValue"
      @change="onChange"
    />
    <span class="md-switch-track" aria-hidden="true">
      <span class="md-switch-thumb" />
    </span>
    <span class="md-switch-text">{{ modelValue ? (onLabel ?? '公开') : (offLabel ?? '成员可见') }}</span>
  </label>
</template>
