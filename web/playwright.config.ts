import { defineConfig, devices } from '@playwright/test'

const origin = 'http://127.0.0.1:18573'
// A second binary on the same database, signed in as an account without ROLE_ADMIN.
const viewerOrigin = 'http://127.0.0.1:18574'
const database = process.env.DATABASE_URL ?? 'postgres://app:app@localhost:55432/app?sslmode=disable'
const VIEWER_SPEC = /viewer\.spec\.ts/

// E2E runs the real binary, with the web build embedded, against the compose Postgres: the same
// artifact the image ships. `task e2e` builds it first.
export default defineConfig({
  testDir: 'tests/e2e',
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  // A flaky test is a bug to fix, not to retry.
  retries: 0,
  // CI writes a blob report, which the job then turns into an HTML report.
  reporter: process.env.CI ? [['blob'], ['github'], ['list']] : 'list',
  use: { baseURL: origin, trace: 'retain-on-failure' },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'] }, testIgnore: VIEWER_SPEC },
    { name: 'phone', use: { ...devices['Pixel 7'] }, testIgnore: VIEWER_SPEC },
    { name: 'viewer', use: { ...devices['Desktop Chrome'], baseURL: viewerOrigin }, testMatch: VIEWER_SPEC },
  ],
  // Playwright starts the servers one after the other, so the first has migrated the database
  // before the second starts.
  webServer: [
    {
      command: '../bin/findability-explorer',
      url: `${origin}/readyz`,
      reuseExistingServer: !process.env.CI,
      timeout: 60_000,
      env: {
        ADDR: new URL(origin).host,
        DATABASE_URL: database,
        // No forward-auth in front of a local run; this identity stands in for it.
        DEV_USER: 'e2e',
        // The dev user is an administrator, so the settings page can be reached too.
        DEV_ROLES: 'ROLE_ADMIN',
      },
    },
    {
      command: '../bin/findability-explorer',
      url: `${viewerOrigin}/readyz`,
      reuseExistingServer: !process.env.CI,
      timeout: 60_000,
      env: { ADDR: new URL(viewerOrigin).host, DATABASE_URL: database, DEV_USER: 'e2e-viewer', DEV_ROLES: '' },
    },
  ],
})
