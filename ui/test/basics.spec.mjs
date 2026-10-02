import { data, expect, openPanel, status, test } from "./panel.mjs";

for (const lang of ["en", "uk"]) {
  test(`every tab renders without errors (${lang})`, async ({ page }) => {
    const p = await openPanel(page, data(status(lang)));
    const tabs = page.getByRole("tab");
    await expect(tabs).toHaveCount(5);
    for (let i = 0; i < 5; i++) {
      await tabs.nth(i).click();
      await expect(tabs.nth(i)).toHaveAttribute("aria-selected", "true");
    }
    expect(p.errors).toEqual([]);
  });

  test(`all tabs fit a 380 px panel (${lang})`, async ({ page }) => {
    await openPanel(page, data(status(lang)), { width: 380 });
    const bar = await page.locator(".tabs").boundingBox();
    const last = await page.getByRole("tab").last().boundingBox();
    expect(last.x + last.width).toBeLessThanOrEqual(bar.x + bar.width + 0.5);
  });
}

test("before setup the panel offers it", async ({ page }) => {
  const p = await openPanel(page, { type: "update", state: "not_initialized", locale: "en" });
  await page.getByRole("button", { name: "Set up Commit Hike" }).click();
  expect(await p.sent()).toContainEqual({ command: "setup" });
});

test("the panel's own errors reach the host, at most 20", async ({ page }) => {
  const p = await openPanel(page, data(status("en")));
  // The browser raises these for uncaught errors; the test clock would swallow real ones.
  await page.evaluate(() => {
    for (let i = 0; i < 25; i++) window.dispatchEvent(new ErrorEvent("error", { message: "boom " + i, lineno: 10, colno: 2 }));
    window.dispatchEvent(new PromiseRejectionEvent("unhandledrejection", { promise: Promise.resolve(), reason: new Error("late") }));
  });
  await expect.poll(async () => (await p.sent()).filter(m => m.command === "panelError").length).toBe(20);
  const first = (await p.sent()).find(m => m.command === "panelError");
  expect(first.message).toBe("boom 0 @ 10:2");
});
