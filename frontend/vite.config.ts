/// <reference types="vitest" />
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    // Go embed 대상 디렉토리(web/static)로 빌드 결과물을 출력한다.
    outDir: '../web/static',
  },
  server: {
    // 로컬 개발 시 /api 요청을 백엔드로 프록시해 브라우저 입장에서 동일 origin으로 동작한다.
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
  },
})
