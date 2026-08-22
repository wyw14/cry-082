import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv, type ProxyOptions } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const environment = loadEnv(mode, process.cwd(), 'DUST_')
  const backend = environment.DUST_API_ORIGIN || 'http://127.0.0.1:8080'
  const proxiedPaths = ['/api/v1', '/healthz', '/readyz', '/metrics']
  const proxy = Object.fromEntries(
    proxiedPaths.map((path) => [path, { target: backend, changeOrigin: path === '/api/v1' } satisfies ProxyOptions]),
  )
  return {
    plugins: [vue()],
    resolve: {
      alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
    },
    server: {
      host: '0.0.0.0',
      port: 5173,
      strictPort: true,
      proxy,
    },
    build: {
      sourcemap: false,
      chunkSizeWarningLimit: 1100,
      rollupOptions: {
        output: {
          manualChunks(moduleID) {
            if (moduleID.includes('element-plus')) return 'element-plus'
            if (moduleID.includes('node_modules/vue') || moduleID.includes('node_modules/pinia')) return 'vue-runtime'
          },
        },
      },
    },
    test: {
      environment: 'jsdom',
      include: ['src/tests/**/*.spec.ts'],
      sequence: { concurrent: false },
    },
  }
})
