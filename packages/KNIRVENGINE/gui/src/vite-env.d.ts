/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_KNIRVORACLE_URL: string
  readonly VITE_KNIRVORACLE_API_KEY: string
  readonly VITE_WEBSOCKET_URL: string
  readonly VITE_NODE_ENV: string
  readonly VITE_DEBUG: string
  readonly VITE_APP_VERSION: string
  readonly VITE_AGENTIC_ENGINE_DEMO_MODE: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
