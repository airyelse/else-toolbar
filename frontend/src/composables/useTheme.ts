import { ref, computed, watch, onMounted, onUnmounted } from 'vue'

export type ThemeMode = 'system' | 'light' | 'dark' | 'amoled'

const STORAGE_KEY = 'else-toolbox-theme'

const mode = ref<ThemeMode>(loadStoredMode())

function loadStoredMode(): ThemeMode {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'system' || stored === 'light' || stored === 'dark' || stored === 'amoled') return stored
  } catch {}
  return 'system'
}

const prefersDark = ref(false)
let mediaQuery: MediaQueryList | null = null

function onMediaChange(e: MediaQueryListEvent) {
  prefersDark.value = e.matches
}

function applyTheme() {
  const html = document.documentElement
  html.classList.remove('dark', 'amoled')

  if (mode.value === 'dark') {
    html.classList.add('dark')
  } else if (mode.value === 'amoled') {
    html.classList.add('amoled')
  } else if (mode.value === 'system') {
    // system 只在 light/dark 之间切换，不自动 amoled
    if (prefersDark.value) {
      html.classList.add('dark')
    }
  }
  // mode === 'light' → no dark class
}

watch(() => [mode.value, prefersDark.value], () => applyTheme(), { immediate: true })

watch(mode, (val) => {
  try {
    localStorage.setItem(STORAGE_KEY, val)
  } catch {}
})

export function useTheme() {
  onMounted(() => {
    mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    prefersDark.value = mediaQuery.matches
    mediaQuery.addEventListener('change', onMediaChange)
  })

  onUnmounted(() => {
    mediaQuery?.removeEventListener('change', onMediaChange)
  })

  function setMode(m: ThemeMode) {
    mode.value = m
  }

  function cycleMode() {
    const next: Record<ThemeMode, ThemeMode> = {
      system: 'light',
      light: 'dark',
      dark: 'amoled',
      amoled: 'system',
    }
    mode.value = next[mode.value]
  }

  const modeLabel = computed(() => {
    const labels: Record<ThemeMode, string> = {
      system: '跟随系统',
      light: '浅色',
      dark: '深色',
      amoled: 'AMOLED',
    }
    return labels[mode.value]
  })

  const modeIcon = computed(() => {
    const icons: Record<ThemeMode, string> = {
      system: 'Monitor',
      light: 'Sunny',
      dark: 'Moon',
      amoled: 'Cellphone',
    }
    return icons[mode.value]
  })

  return {
    mode,
    setMode,
    cycleMode,
    modeLabel,
    modeIcon,
  }
}
