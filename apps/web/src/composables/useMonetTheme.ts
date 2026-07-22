import { computed } from 'vue'

export interface MonetPalette {
  primary: string
  onPrimary: string
  primaryContainer: string
  onPrimaryContainer: string
  secondary: string
  secondaryContainer: string
  tertiary: string
  tertiaryContainer: string
  surfaceContainerLowest: string
  surfaceContainerLow: string
  surfaceContainer: string
  surfaceContainerHigh: string
  onSurface: string
  onSurfaceVariant: string
  outlineVariant: string
}

export interface SeasonConfig {
  id: 'spring' | 'summer' | 'autumn' | 'winter'
  name: string
  icon: string
  light: MonetPalette
  dark: MonetPalette
}

export const MONET_SEASONS: Record<'spring' | 'summer' | 'autumn' | 'winter', SeasonConfig> = {
  spring: {
    id: 'spring',
    name: '春 · 樱纷复苏',
    icon: '🌸',
    light: {
      primary: '#d81b60',
      onPrimary: '#ffffff',
      primaryContainer: '#ffe0e8',
      onPrimaryContainer: '#4a0018',
      secondary: '#2e7d32',
      secondaryContainer: '#c8e6c9',
      tertiary: '#f4511e',
      tertiaryContainer: '#ffccbc',
      surfaceContainerLowest: '#ffffff',
      surfaceContainerLow: '#fff4f7',
      surfaceContainer: '#ffebf1',
      surfaceContainerHigh: '#f8d7e2',
      onSurface: '#23191c',
      onSurfaceVariant: '#544347',
      outlineVariant: '#f3b7c8',
    },
    dark: {
      primary: '#ff80ab',
      onPrimary: '#4a0024',
      primaryContainer: '#680036',
      onPrimaryContainer: '#ffe0e8',
      secondary: '#a5d6a7',
      secondaryContainer: '#1b5e20',
      tertiary: '#ffab91',
      tertiaryContainer: '#bf360c',
      surfaceContainerLowest: '#140a0f',
      surfaceContainerLow: '#1a1114',
      surfaceContainer: '#24181d',
      surfaceContainerHigh: '#302027',
      onSurface: '#f4e0e5',
      onSurfaceVariant: '#d8c1c7',
      outlineVariant: '#5c3140',
    },
  },
  summer: {
    id: 'summer',
    name: '夏 · 碧海潮生',
    icon: '🌿',
    light: {
      primary: '#00897b',
      onPrimary: '#ffffff',
      primaryContainer: '#b2dfdb',
      onPrimaryContainer: '#002521',
      secondary: '#0288d1',
      secondaryContainer: '#b3e5fc',
      tertiary: '#ff6f00',
      tertiaryContainer: '#ffe0b2',
      surfaceContainerLowest: '#ffffff',
      surfaceContainerLow: '#f0fbf9',
      surfaceContainer: '#e2f6f3',
      surfaceContainerHigh: '#ceefe9',
      onSurface: '#162120',
      onSurfaceVariant: '#415350',
      outlineVariant: '#9fe0d5',
    },
    dark: {
      primary: '#80cbd2',
      onPrimary: '#003732',
      primaryContainer: '#004d40',
      onPrimaryContainer: '#b2dfdb',
      secondary: '#81d4fa',
      secondaryContainer: '#01579b',
      tertiary: '#ffcc80',
      tertiaryContainer: '#e65100',
      surfaceContainerLowest: '#071211',
      surfaceContainerLow: '#0f1817',
      surfaceContainer: '#162221',
      surfaceContainerHigh: '#21302e',
      onSurface: '#e0f2f1',
      onSurfaceVariant: '#b2ccc9',
      outlineVariant: '#2a544f',
    },
  },
  autumn: {
    id: 'autumn',
    name: '秋 · 枫丹金秋',
    icon: '🍁',
    light: {
      primary: '#d84315',
      onPrimary: '#ffffff',
      primaryContainer: '#ffccbc',
      onPrimaryContainer: '#3e0c00',
      secondary: '#f57f17',
      secondaryContainer: '#fff9c4',
      tertiary: '#6d4c41',
      tertiaryContainer: '#d7ccc8',
      surfaceContainerLowest: '#ffffff',
      surfaceContainerLow: '#fff7f2',
      surfaceContainer: '#ffefe5',
      surfaceContainerHigh: '#fbe1d3',
      onSurface: '#241a16',
      onSurfaceVariant: '#56443c',
      outlineVariant: '#f5beab',
    },
    dark: {
      primary: '#ff9e80',
      onPrimary: '#4a0b00',
      primaryContainer: '#7f270b',
      onPrimaryContainer: '#ffccbc',
      secondary: '#fff59d',
      secondaryContainer: '#6d3c00',
      tertiary: '#bcaaa4',
      tertiaryContainer: '#4e342e',
      surfaceContainerLowest: '#140d09',
      surfaceContainerLow: '#1c130e',
      surfaceContainer: '#271b15',
      surfaceContainerHigh: '#36271e',
      onSurface: '#fbece5',
      onSurfaceVariant: '#debcae',
      outlineVariant: '#5e3a2d',
    },
  },
  winter: {
    id: 'winter',
    name: '冬 · 霜雪微芒',
    icon: '❄️',
    light: {
      primary: '#1e88e5',
      onPrimary: '#ffffff',
      primaryContainer: '#bbdefb',
      onPrimaryContainer: '#001c40',
      secondary: '#5e35b1',
      secondaryContainer: '#d1c4e9',
      tertiary: '#00acc1',
      tertiaryContainer: '#b2ebf2',
      surfaceContainerLowest: '#ffffff',
      surfaceContainerLow: '#f2f7fd',
      surfaceContainer: '#e6f0fa',
      surfaceContainerHigh: '#d4e4f7',
      onSurface: '#161e27',
      onSurfaceVariant: '#434e5c',
      outlineVariant: '#accceb',
    },
    dark: {
      primary: '#90caf9',
      onPrimary: '#001b3a',
      primaryContainer: '#0d47a1',
      onPrimaryContainer: '#bbdefb',
      secondary: '#b39ddb',
      secondaryContainer: '#311b92',
      tertiary: '#80deea',
      tertiaryContainer: '#006064',
      surfaceContainerLowest: '#070c14',
      surfaceContainerLow: '#101722',
      surfaceContainer: '#182130',
      surfaceContainerHigh: '#222d40',
      onSurface: '#e3edfb',
      onSurfaceVariant: '#b6c8e3',
      outlineVariant: '#314766',
    },
  },
}

export function useMonetTheme() {
  // Automatically derive season strictly based on local month
  const currentSeasonKey = computed(() => {
    const month = new Date().getMonth() + 1
    if (month >= 3 && month <= 5) return 'spring'
    if (month >= 6 && month <= 8) return 'summer'
    if (month >= 9 && month <= 11) return 'autumn'
    return 'winter'
  })

  // Automatically derive day/night strictly based on local time hour
  const currentTimeKey = computed(() => {
    const hour = new Date().getHours()
    return hour >= 7 && hour < 19 ? 'day' : 'night'
  })

  const currentSeasonConfig = computed(() => MONET_SEASONS[currentSeasonKey.value])

  const applyMonetColors = (isDarkSystem: boolean) => {
    const timeKey = currentTimeKey.value
    const seasonConfig = currentSeasonConfig.value
    const root = document.documentElement

    const isNight = isDarkSystem || timeKey === 'night'
    const palette = isNight ? seasonConfig.dark : seasonConfig.light

    root.setAttribute('data-monet-time', timeKey)

    // Primary & Accent Colors
    root.style.setProperty('--md-sys-color-primary', palette.primary)
    root.style.setProperty('--md-sys-color-on-primary', palette.onPrimary)
    root.style.setProperty('--md-sys-color-primary-container', palette.primaryContainer)
    root.style.setProperty('--md-sys-color-on-primary-container', palette.onPrimaryContainer)
    root.style.setProperty('--md-sys-color-secondary', palette.secondary)
    root.style.setProperty('--md-sys-color-secondary-container', palette.secondaryContainer)
    root.style.setProperty('--md-sys-color-tertiary', palette.tertiary)
    root.style.setProperty('--md-sys-color-tertiary-container', palette.tertiaryContainer)

    // Dynamic Surface Container Colors & High-Contrast Typography
    root.style.setProperty('--md-sys-color-surface-container-lowest', palette.surfaceContainerLowest)
    root.style.setProperty('--md-sys-color-surface-container-low', palette.surfaceContainerLow)
    root.style.setProperty('--md-sys-color-surface-container', palette.surfaceContainer)
    root.style.setProperty('--md-sys-color-surface-container-high', palette.surfaceContainerHigh)
    root.style.setProperty('--md-sys-color-on-surface', palette.onSurface)
    root.style.setProperty('--md-sys-color-on-surface-variant', palette.onSurfaceVariant)
    root.style.setProperty('--md-sys-color-outline-variant', palette.outlineVariant)

    root.style.setProperty('--el-color-primary', palette.primary)
  }

  return {
    currentSeasonKey,
    currentTimeKey,
    currentSeasonConfig,
    applyMonetColors,
  }
}
