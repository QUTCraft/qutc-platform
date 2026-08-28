import { computed, onMounted, ref } from 'vue'

/**
 * useAsyncData 将异步 request 包装成统一的 Vue 响应式状态。
 * data 保存最后一次成功结果，error 保存本次失败原因；刷新期间保留旧 data，避免页面闪烁。
 */
export function useAsyncData<T>(request: () => Promise<T>) {
  const data = ref<T>() // 最近一次成功响应；初次加载前为 undefined。
  const error = ref<Error>() // 最近一次失败原因；每次新请求开始时清空。
  const pending = ref(true) // 请求是否正在执行，是 loading / refreshing 两种展示状态的共同来源。
  // Keep the existing view mounted while a filter, pagination, or retry request
  // is in flight. Consumers can still use `refreshing` for a small busy state.
  const loading = computed(() => pending.value && data.value === undefined)
  const refreshing = computed(() => pending.value && data.value !== undefined)

  // refresh 可由组件手动重试或变更筛选条件后调用；错误不会丢弃之前成功的数据。
  const refresh = async () => {
    pending.value = true
    error.value = undefined
    try {
      data.value = await request()
    } catch (cause) {
      error.value = cause instanceof Error ? cause : new Error('请求失败，请稍后重试。')
    } finally {
      pending.value = false
    }
  }

  // 首次挂载即发起请求，调用方无需重复编写生命周期钩子。
  onMounted(refresh)
  return { data, error, loading, refreshing, refresh }
}
