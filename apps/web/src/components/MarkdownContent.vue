<script setup lang="ts">
import { computed, onBeforeUnmount } from 'vue'
import { renderMarkdown } from '@/utils/markdown'

const props = withDefaults(defineProps<{
  markdown: string
  publicAssets?: boolean
}>(), {
  publicAssets: false,
})

const html = computed(() => renderMarkdown(props.markdown, props.publicAssets))
const feedbackTimers = new Set<number>()

async function writeClipboard(text: string) {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // HTTP deployments and restrictive browser policies can reject the
      // asynchronous Clipboard API. Keep the user-gesture fallback below.
    }
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.readOnly = true
  textarea.style.position = 'fixed'
  textarea.style.inset = '-9999px auto auto -9999px'
  document.body.appendChild(textarea)
  textarea.select()
  const copied = document.execCommand('copy')
  textarea.remove()
  if (!copied) throw new Error('浏览器拒绝了复制操作。')
}

function showCopyFeedback(button: HTMLButtonElement, message: string, copied: boolean) {
  const label = button.querySelector<HTMLElement>('.markdown-copy-label')
  if (!label) return
  label.textContent = message
  button.setAttribute('aria-label', message === '已复制' ? '代码已复制' : message)
  button.classList.toggle('is-copied', copied)

  const timer = window.setTimeout(() => {
    feedbackTimers.delete(timer)
    if (!button.isConnected) return
    label.textContent = '复制'
    button.setAttribute('aria-label', '复制代码')
    button.classList.remove('is-copied')
  }, 1600)
  feedbackTimers.add(timer)
}

async function handleClick(event: MouseEvent) {
  if (!(event.target instanceof Element)) return
  const button = event.target.closest<HTMLButtonElement>('[data-markdown-copy]')
  if (!button) return
  const code = button.closest('.markdown-code-block')?.querySelector('code')?.textContent
  if (code === undefined) return

  try {
    await writeClipboard(code)
    showCopyFeedback(button, '已复制', true)
  } catch {
    showCopyFeedback(button, '复制失败', false)
  }
}

onBeforeUnmount(() => {
  for (const timer of feedbackTimers) window.clearTimeout(timer)
  feedbackTimers.clear()
})
</script>

<template>
  <div class="markdown-content" @click="handleClick" v-html="html" />
</template>

<style scoped>
.markdown-content :deep(.markdown-code-block) {
  overflow: hidden;
  margin: 0 0 22px;
  border: 1px solid var(--md-sys-color-outline-variant);
  border-radius: 16px;
  background: var(--md-sys-color-surface-container-high);
  box-shadow: 0 8px 20px color-mix(in srgb, var(--md-sys-color-shadow) 8%, transparent);
}

.markdown-content :deep(.markdown-code-toolbar) {
  display: flex;
  min-height: 42px;
  align-items: center;
  gap: 10px;
  padding: 0 10px 0 16px;
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
  background: var(--md-sys-color-surface-container);
}

.markdown-content :deep(.markdown-code-language) {
  color: var(--md-sys-color-on-surface);
  font-family: ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', monospace;
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.markdown-content :deep(.markdown-code-lines) {
  margin-right: auto;
  color: var(--md-sys-color-on-surface-variant);
  font-size: 0.76rem;
}

.markdown-content :deep(.markdown-copy-button) {
  min-width: 62px;
  min-height: 30px;
  padding: 0 12px;
  color: var(--md-sys-color-primary);
  border: 1px solid color-mix(in srgb, var(--md-sys-color-primary) 45%, transparent);
  border-radius: 999px;
  background: transparent;
  font: inherit;
  font-size: 0.78rem;
  font-weight: 750;
  cursor: pointer;
  transition: background-color 140ms ease, color 140ms ease, border-color 140ms ease;
}

.markdown-content :deep(.markdown-copy-button:hover),
.markdown-content :deep(.markdown-copy-button:focus-visible),
.markdown-content :deep(.markdown-copy-button.is-copied) {
  color: var(--md-sys-color-on-primary-container);
  border-color: var(--md-sys-color-primary-container);
  background: var(--md-sys-color-primary-container);
  outline: none;
}

.markdown-content :deep(.markdown-code-block details) {
  margin: 0;
}

.markdown-content :deep(.markdown-code-block summary) {
  display: flex;
  min-height: 44px;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
  color: var(--md-sys-color-primary);
  background: var(--md-sys-color-surface-container-low);
  font-size: 0.86rem;
  font-weight: 750;
  cursor: pointer;
  list-style: none;
  user-select: none;
}

.markdown-content :deep(.markdown-code-block summary::-webkit-details-marker) {
  display: none;
}

.markdown-content :deep(.markdown-code-block summary::before) {
  content: '›';
  font-size: 1.15rem;
  transform: rotate(0deg);
  transition: transform 140ms ease;
}

.markdown-content :deep(.markdown-code-block details[open] summary::before) {
  transform: rotate(90deg);
}

.markdown-content :deep(.markdown-fold-open),
.markdown-content :deep(.markdown-code-block details[open] .markdown-fold-closed) {
  display: none;
}

.markdown-content :deep(.markdown-code-block details[open] .markdown-fold-open) {
  display: inline;
}

.markdown-content :deep(.markdown-fold-lines) {
  margin-left: auto;
  color: var(--md-sys-color-on-surface-variant);
  font-size: 0.76rem;
  font-weight: 600;
}

.markdown-content :deep(.markdown-code-block pre) {
  overflow-x: auto;
  margin: 0;
  padding: 18px 20px;
  border: 0;
  border-radius: 0;
  background: var(--md-sys-color-surface-container-highest);
  box-shadow: none;
  line-height: 1.65;
  tab-size: 2;
}

.markdown-content :deep(.markdown-code-block pre code) {
  padding: 0;
  color: var(--md-sys-color-on-surface);
  background: transparent;
  font-size: 0.9em;
  white-space: pre;
}

@media (max-width: 640px) {
  .markdown-content :deep(.markdown-code-toolbar) {
    padding-left: 12px;
  }

  .markdown-content :deep(.markdown-code-block pre) {
    padding: 14px;
  }
}
</style>
