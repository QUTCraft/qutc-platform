/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_MODE?: 'mock' | 'remote'
  readonly VITE_API_BASE_URL?: string
  readonly VITE_ORGANIZATION_SLUG?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
