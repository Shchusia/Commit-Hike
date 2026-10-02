// Browser tests for ui/panel/panel.html: `task ui:test`.
import { defineConfig } from "@playwright/test";
import { browserPath } from "./test/browser.mjs";

export default defineConfig({
  testDir: "./test",
  testMatch: "*.spec.mjs",
  globalSetup: "./test/fixtures.mjs",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["list"], ["github"]] : "list",
  globalTeardown: "./test/coverage-report.mjs",
  use: {
    browserName: "chromium", deviceScaleFactor: 1, timezoneId: "UTC", locale: "en-US",
    launchOptions: { executablePath: browserPath() || undefined }, // see test/browser.mjs
  },
});
