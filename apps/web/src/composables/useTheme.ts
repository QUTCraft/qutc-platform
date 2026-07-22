import { ref, onMounted, watch } from 'vue'
import { useMonetTheme } from './useMonetTheme'

export type ThemeMode = 'light' | 'dark' | 'system'

const theme = ref<ThemeMode>('light')
const isDark = ref<boolean>(false)

export function useTheme() {
  const monet = useMonetTheme()

  const updateDOM = () => {
    let effectiveDark = false
    if (theme.value === 'dark') {
      effectiveDark = true
    } else if (theme.value === 'system') {
      effectiveDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    } else {
      effectiveDark = false
    }

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

  const setTheme = (mode: ThemeMode) => {
    theme.value = mode
    localStorage.setItem('md3-theme', mode)
    updateDOM()
  }

  const toggleTheme = () => {
    setTheme(isDark.value ? 'light' : 'dark')
  }

  watch([monet.currentSeasonKey, monet.currentTimeKey], () => {
    updateDOM()
  })

  onMounted(() => {
    const saved = localStorage.getItem('md3-theme') as ThemeMode | null
    if (saved && ['light', 'dark', 'system'].includes(saved)) {
      theme.value = saved
    } else {
      theme.value = 'light'
    }
    updateDOM()

    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (theme.value === 'system') updateDOM()
    })
  })

  return {
    theme,
    isDark,
    setTheme,
    toggleTheme,
    monet,
  }
}
