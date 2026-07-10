import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/simulation",
  fullyParallel: false,
  workers: 1,
  timeout: 45_000,
  expect: { timeout: 12_000 },
  reporter: [["list"]],
  use: {
    baseURL: process.env.SIM_UI_E2E_URL ?? "http://127.0.0.1:8090",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
