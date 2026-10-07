// The routes website (commit-hike.dev): its address, the pages the
// panel may open there, and the panel's search turned into core arguments.

/** The site; COMMIT_HIKE_SITE overrides it (the core reads the same variable). */
export function siteUrl(env: NodeJS.ProcessEnv = process.env): string {
  const v = (env.COMMIT_HIKE_SITE ?? "").trim().replace(/\/+$/, "");
  return /^https?:\/\/[^/\s]+$/.test(v) ? v : "https://commit-hike.dev";
}

/** Whether [url] is a page of the routes site (opened from a route card). */
export function isSitePage(url: unknown, base = siteUrl()): url is string {
  if (typeof url !== "string" || url.length > 2000) return false;
  try {
    const u = new URL(url);
    return u.origin === new URL(base).origin && !u.username && !u.password && /^\/(routes\/[a-z0-9-]+|u\/\d+)?$/.test(u.pathname);
  } catch {
    return false;
  }
}

const ONE_OF = {
  kind: ["real", "story"],
  length: ["s", "m", "l", "xl"],
  sort: ["top", "rating", "rating_asc", "popular", "downloads_asc", "new", "short", "long"],
} as const;

/** The panel's search as arguments for `commit-hike site routes`; anything odd is dropped. */
export function siteSearchArgs(query: unknown): string[] {
  const q = (query && typeof query === "object" ? query : {}) as Record<string, unknown>;
  const args = ["site", "routes"];
  if (typeof q.q === "string" && q.q.trim()) args.push("--q", q.q.trim().slice(0, 100));
  for (const key of Object.keys(ONE_OF) as (keyof typeof ONE_OF)[]) {
    const v = q[key];
    if (typeof v === "string" && (ONE_OF[key] as readonly string[]).includes(v)) args.push(`--${key}`, v);
  }
  const page = Number(q.page);
  if (Number.isInteger(page) && page > 1 && page < 10_000) args.push("--page", String(page));
  return args;
}

/** A route id the panel asks to install. */
export function isRouteId(id: unknown): id is string {
  return typeof id === "string" && id.length <= 80 && /^[a-z0-9]+(-[a-z0-9]+)*$/.test(id);
}
