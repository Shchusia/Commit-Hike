import { data, expect, openPanel, status, test } from "./panel.mjs";

function passport(st) {
  const day = 86400;
  st.passport = [
    { route_id: "old-trail", route_name: "Old Trail", waypoint_id: "a", name: "First Hut", kind: "hut", reached_at: 1780000000 },
    { route_id: "old-trail", route_name: "Old Trail", waypoint_id: "b", name: "High Peak", kind: "peak", elevation_m: 2100, reached_at: 1780000000 + 3 * day },
    { route_id: st.global.route.id, route_name: st.global.route.name, waypoint_id: "s", name: "Start Gate", kind: "start", reached_at: 1770000000 },
  ];
  st.route_stops = { "old-trail": 2, [st.global.route.id]: 9 };
  return st;
}

test("the passport shows the current trail first, then the others", async ({ page }) => {
  const st = passport(status("en"));
  const p = await openPanel(page, data(st));
  await page.getByRole("tab", { name: "Places" }).click();
  await page.getByRole("radio", { name: /Passport · 3/ }).click();
  await expect(page.locator(".pass-head .nm")).toHaveText([st.global.route.name, "Old Trail"]);
  await expect(page.locator(".pass-head .meta")).toHaveText(["1 of 9", "✓ 2 of 2"]);
  await expect(page.locator(".stamp")).toHaveCount(3);
  await expect(page.locator(".stamp").nth(2)).toHaveAttribute("title", "High Peak · 2,100 m");
  expect(p.errors).toEqual([]);
});

test("an empty passport explains itself", async ({ page }) => {
  const st = status("en");
  st.passport = [];
  await openPanel(page, data(st));
  await page.getByRole("tab", { name: "Places" }).click();
  await page.getByRole("radio", { name: "🎫 Passport" }).click();
  await expect(page.getByText(/Stamps for the stops you reach/)).toBeVisible();
});

test("a day without commits is a camp; a day off is a rest", async ({ page }) => {
  const st = status("en");
  st.today_m = 0;
  const p = await openPanel(page, data(st));
  await expect(page.locator(".scene .hud .camp")).toHaveText("⛺ Camp");
  st.rest_days = [0, 1, 2, 3, 4, 5, 6].filter(d => d === new Date("2026-09-28T13:00:00Z").getDay()); // the test clock's weekday
  await p.update(data(st));
  await expect(page.locator(".scene .hud .camp")).toHaveText("⛺ Day off");
  st.today_m = 1200;
  await p.update(data(st));
  await expect(page.locator(".scene .hud .camp")).toHaveCount(0);
});

test("a finished route of a series offers the next one", async ({ page }) => {
  const st = status("en");
  st.global.finished = true;
  st.global.distance_m = st.global.route.length_m;
  st.global.next_route = { id: "svydovets-ridge", name: "Svydovets Ridge", length_m: 34000, series_id: "carpathians", series_name: "The Carpathians" };
  const p = await openPanel(page, data(st));
  await page.getByRole("tab", { name: "Places" }).click();
  await expect(page.locator(".series-next .k")).toHaveText("Next in “The Carpathians”");
  await page.getByRole("button", { name: /Walk on: Svydovets Ridge \(34(\.0)? km\)/ }).click();
  expect(await p.sent()).toContainEqual({ command: "walkRoute", id: "svydovets-ridge" });
});
