import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/browser",
  workers: 1,
  retries: 0,
  use: {
    baseURL: "http://127.0.0.1:5410",
    launchOptions: { executablePath: process.env.BROWSER_EXECUTABLE },
    trace: "retain-on-failure",
  },
  webServer: {
    command: "node tests/server.mjs",
    url: "http://127.0.0.1:5410/readyz",
    timeout: 120_000,
    reuseExistingServer: false,
  },
});
