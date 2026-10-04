// Sharing from the panel: the only web pages the plugin opens for it.
// The panel builds the URL; this checks it is one of the networks' sharing
// pages (a Mastodon server's /share for Mastodon), so a message from the
// panel can never make the editor open anything else.

const PAGES = new Set([
  "https://twitter.com/intent/tweet",
  "https://bsky.app/intent/compose",
  "https://www.threads.net/intent/post",
  "https://www.linkedin.com/sharing/share-offsite/",
  "https://www.facebook.com/sharer/sharer.php",
  "https://t.me/share/url",
  "https://www.reddit.com/submit",
]);
const HOST = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$/;

/** Whether [url] is a sharing page the panel may open. */
export function isShareUrl(url: unknown): url is string {
  if (typeof url !== "string" || url.length > 4000) return false;
  let u: URL;
  try {
    u = new URL(url);
  } catch {
    return false;
  }
  if (u.protocol !== "https:" || u.username || u.password || u.port || u.hash) return false;
  if (PAGES.has(u.origin + u.pathname)) return true;
  // Mastodon: any server, but only its share page with nothing but the text
  return u.pathname === "/share" && HOST.test(u.hostname) && [...u.searchParams.keys()].join() === "text";
}

/** The PNG bytes of a data URL from the panel, or null. */
export function pngBytes(dataUrl: unknown): Buffer | null {
  const m = /^data:image\/png;base64,([A-Za-z0-9+/=]+)$/.exec(typeof dataUrl === "string" && dataUrl.length < 20_000_000 ? dataUrl : "");
  return m ? Buffer.from(m[1], "base64") : null;
}
