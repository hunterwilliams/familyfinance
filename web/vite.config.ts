import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const fileEnv = loadEnv(mode, process.cwd(), '')
  // Bracket access avoids esbuild rewriting process.env.FOO via NODE_ENV define.
  const apiTarget =
    process.env['DEV_API_TARGET'] ||
    process.env['VITE_DEV_API_TARGET'] ||
    fileEnv.DEV_API_TARGET ||
    fileEnv.VITE_DEV_API_TARGET ||
    'http://localhost:8080'
  const inDocker = Boolean(process.env['DEV_API_TARGET'] || fileEnv.DEV_API_TARGET)

  return {
    plugins: [react()],
    server: {
      host: '0.0.0.0',
      port: 5173,
      // Bind mounts on macOS don't emit inotify events into Linux containers.
      watch: inDocker ? { usePolling: true, interval: 1000 } : undefined,
      proxy: {
        '/api': {
          target: apiTarget,
          changeOrigin: true,
        },
        '/callback': {
          target: apiTarget,
          changeOrigin: true,
        },
      },
    },
  }
})
