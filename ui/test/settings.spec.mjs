import { data, expect, openPanel, status, test } from "./panel.mjs";

async function openSettings(page) {
  await page.getByRole("button", { name: "Menu" }).click();
  await page.getByRole("menuitem", { name: /Settings/ }).click();
}

test("every setting is on the page and talks to the host", async ({ page }) => {
  const p = await openPanel(page, data(status("en")));
  await openSettings(page);
  await expect(page.locator(".set-row h3")).toHaveText(["Language", "Difficulty", "Days off", "Notifications", "Less motion", "Holidays in the scene", "Weather and wildlife", "More contrast", "Your hiker", "Backup", "Something wrong?"]);
  await page.locator(".set-row", { hasText: "Notifications" }).getByRole("radio", { name: "Only milestones" }).click();
  await page.locator(".set-row", { hasText: "More contrast" }).getByRole("radio", { name: "On" }).click();
  await page.locator(".set-row", { hasText: "Holidays in the scene" }).getByRole("radio", { name: "Off" }).click();
  await page.locator(".set-row", { hasText: "Weather and wildlife" }).getByRole("radio", { name: "Off" }).click();
  await page.getByRole("button", { name: "Export…" }).click();
  await page.getByRole("button", { name: "Import…" }).click();
  await page.getByRole("button", { name: "Copy a report for the developer" }).click();
  expect(await p.sent()).toEqual([
    { command: "setSettings", notifications: "milestones" },
    { command: "setSettings", high_contrast: "on" },
    { command: "setSettings", festive: "off" },
    { command: "setSettings", ambient: "off" },
    { command: "exportProgress" },
    { command: "importProgress" },
    { command: "copyDiagnostics" },
  ]);
});

test("the last working day can't be made a day off", async ({ page }) => {
  const st = status("en");
  st.rest_days = [0, 1, 2, 3, 4, 6]; // only Friday works
  const p = await openPanel(page, data(st));
  await openSettings(page);
  const days = page.locator(".choice.days button");
  await expect(days.nth(4)).toBeDisabled(); // Friday
  await days.nth(0).click(); // Monday becomes a working day again
  expect(await p.sent()).toEqual([{ command: "setRestDays", days: [0, 2, 3, 4, 6] }]);
});

test("the host can open the page once per token, and Back returns to the tab", async ({ page }) => {
  const st = status("en");
  const p = await openPanel(page, data(st));
  await page.getByRole("tab", { name: "Places" }).click();
  await p.update(data(st, { open_view: "settings", open_token: 7 }));
  await expect(page.locator(".settings")).toBeVisible();
  await page.getByRole("button", { name: "← Back" }).click();
  await expect(page.getByRole("tab", { name: "Places" })).toHaveAttribute("aria-selected", "true");
  await p.update(data(st, { open_view: "settings", open_token: 7 })); // same token: stays closed
  await expect(page.locator(".settings")).toHaveCount(0);
});
