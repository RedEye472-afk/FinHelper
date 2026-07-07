import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    css: false,
    // Vitest v4 + Windows + Node 24: tinypool threads worker иногда падает
    // с 'Worker exited unexpectedly'. Forks-pool стабильнее на этой связке.
    pool: 'forks',
    poolOptions: {
      forks: { singleFork: false, max: 1 },
    },
  },
})
