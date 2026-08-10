import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useMonetTheme } from './useMonetTheme'

const isDark = ref<boolean>(false)

export function useTheme() {
  const monet = useMonetTheme()
  let refreshTimer: number | undefined

  const updateDOM = () => {
    monet.refreshClock()
    const effectiveDark = monet.currentTimeKey.value === 'night'

    isDark.value = effectiveDark
    if (effectiveDark) {
      document.documentElement.classList.add('dark')
      document.documentElement.setAttribute('data-theme', 'dark')
    } else {
      document.documentElement.classList.remove('dark')
      document.documentElement.setAttribute('data-theme', 'light')
    }

    // Apply Monet Dynamic Colors for active Season & Time of Day
    monet.applyMonetColors(effectiveDark)
  }

  onMounted(() => {
    updateDOM()
    // Re-evaluate the clock so the palette changes automatically at the next
    // day/night boundary and also picks up a season change without a reload.
    refreshTimer = window.setInterval(updateDOM, 60_000)
  })

  onBeforeUnmount(() => {
    if (refreshTimer !== undefined) window.clearInterval(refreshTimer)
  })

  return {
    isDark,
    monet,
  }
}
