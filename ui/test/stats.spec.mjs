import { data, expect, openPanel, pngSize, status, test } from "./panel.mjs";

test("the 14-day chart shows a tooltip on hover and with the keyboard", async ({ page }) => {
  await openPanel(page, data(status("en")));
  await page.getByRole("tab", { name: "Stats" }).click();
  const tip = page.locator(".chart-box .chart-tip").first();
  await page.locator(".bar-hit").last().hover();
  await expect(tip).toContainText("Today");
  await expect(tip).toContainText(/commit/);
  await page.locator(".bar-hit").first().focus();
  const first = await tip.textContent();
  await page.keyboard.press("ArrowRight");
  await expect(tip).not.toHaveText(first);
});

// A known history: work on 2025-06-02…06 (Mon–Fri) and 2026-03-02…13 except
// the weekend in between; days off are Saturday and Sunday.
function history() {
  const days = [];
  const add = (iso, m, commits) => days.push({ date: iso, m, commits });
  for (let d = 2; d <= 6; d++) add(`2025-06-0${d}`, 1000, 2);
  for (const d of [2, 3, 4, 5, 6, 9, 10, 11, 12, 13]) add(`2026-03-${String(d).padStart(2, "0")}`, 1500 + d * 10, 3);
  return days;
}

test("the year in review adds up and switches years", async ({ page }) => {
  const st = status("en");
  st.history = history();
  st.rest_days = [0, 6];
  await openPanel(page, data(st));
  await page.getByRole("tab", { name: "Stats" }).click();
  const year = page.locator(".year");
  await expect(year.locator(".section-title")).toHaveText("2026 on the trail");
  const stats = await year.locator(".stat").allInnerTexts();
  expect(stats.join(" | ")).toContain("30\nCommits"); // 10 days × 3
  expect(stats.join(" | ")).toContain("10\nActive days");
  expect(stats.join(" | ")).toContain("10 days\nLongest streak"); // the weekend off doesn't break it
  await expect(year.locator(".cal rect.cal-d")).toHaveCount(365);

  await page.getByRole("button", { name: "Previous year" }).click();
  await expect(year.locator(".section-title")).toHaveText("2025 on the trail");
  await expect(page.getByRole("button", { name: "Previous year" })).toBeDisabled();
  expect((await year.locator(".stat").allInnerTexts()).join(" | ")).toContain("5 days\nLongest streak");
});

test("days off show next to the streak", async ({ page }) => {
  const st = status("en");
  st.rest_days = [6, 0];
  await openPanel(page, data(st));
  await page.getByRole("tab", { name: "Stats" }).click();
  await expect(page.locator(".stats .stat").nth(1)).toContainText("days off: Sat, Sun");
});

test("the year postcard and the badge go to the host", async ({ page }) => {
  const st = status("en");
  st.history = history();
  const p = await openPanel(page, data(st));
  await page.getByRole("tab", { name: "Stats" }).click();
  await page.getByRole("button", { name: "✉ Year postcard" }).click();
  await page.getByRole("button", { name: /Badge for your GitHub profile/ }).click();
  const sent = await p.sent();
  const card = sent.find(m => m.command === "savePostcard");
  expect(card.name).toBe("commit-hike-year-2026.png");
  expect(pngSize(card.data)).toEqual({ png: true, width: 1200, height: 630 });
  expect(sent).toContainEqual({ command: "saveBadge" });
  require: {
    // keep the picture for a look: ui/test-results/year-postcard.png
    const fs = await import("node:fs");
    fs.mkdirSync("test-results", { recursive: true });
    fs.writeFileSync("test-results/year-postcard.png", Buffer.from(card.data.split(",")[1], "base64"));
  }
});

test("two quiet weeks say so, and the year is still there", async ({ page }) => {
  const st = status("en");
  st.global.daily = st.global.daily.map(d => ({ ...d, m: 0, commits: 0 }));
  st.history = history();
  await openPanel(page, data(st));
  await page.getByRole("tab", { name: "Stats" }).click();
  await expect(page.getByText("No commits in the last two weeks yet.")).toBeVisible();
  await expect(page.locator(".year")).toBeVisible();
});
