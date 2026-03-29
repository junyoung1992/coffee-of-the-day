import { defineConfig, devices } from '@playwright/test'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const currentDir = path.dirname(fileURLToPath(import.meta.url))
const rootDir = path.resolve(currentDir, '..')
const e2eDBPath = path.join(os.tmpdir(), `coffee-of-the-day-e2e-${process.pid}.db`)

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:4173',
    trace: 'on-first-retry',
  },
  webServer: [
    {
      command: 'go run ./cmd/server',
      cwd: path.join(rootDir, 'backend'),
      env: {
        ...process.env,
        PORT: '18080',
        DB_PATH: e2eDBPath,
      },
      url: 'http://127.0.0.1:18080/health',
      reuseExistingServer: false,
      timeout: 120_000,
    },
    {
      command: 'npm run dev -- --host 127.0.0.1 --port 4173',
      cwd: path.join(rootDir, 'frontend'),
      env: {
        ...process.env,
        VITE_API_BASE_URL: 'http://127.0.0.1:18080/api/v1',
      },
      url: 'http://127.0.0.1:4173',
      reuseExistingServer: false,
      timeout: 120_000,
    },
  ],
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
