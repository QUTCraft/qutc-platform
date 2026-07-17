import { onMounted, ref } from 'vue'

export function useAsyncData<T>(request: () => Promise<T>) {
  const data = ref<T>()
  const error = ref<Error>()
  const loading = ref(true)

  const refresh = async () => {
    loading.value = true
    error.value = undefined
    try {
      data.value = await request()
    } catch (cause) {
      error.value = cause instanceof Error ? cause : new Error('请求失败，请稍后重试。')
    } finally {
      loading.value = false
    }
  }

  onMounted(refresh)
  return { data, error, loading, refresh }
}
