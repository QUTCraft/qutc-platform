import { computed, onMounted, ref } from 'vue'

export function useAsyncData<T>(request: () => Promise<T>) {
  const data = ref<T>()
  const error = ref<Error>()
  const pending = ref(true)
  // Keep the existing view mounted while a filter, pagination, or retry request
  // is in flight. Consumers can still use `refreshing` for a small busy state.
  const loading = computed(() => pending.value && data.value === undefined)
  const refreshing = computed(() => pending.value && data.value !== undefined)

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

  onMounted(refresh)
  return { data, error, loading, refreshing, refresh }
}
