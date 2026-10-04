import * as assert from "node:assert/strict";
import { test } from "node:test";
import { isShareUrl, pngBytes } from "../share";

void test("the sharing pages of the networks are allowed", () => {
  for (const url of [
    "https://twitter.com/intent/tweet?text=hi&url=https%3A%2F%2Fgithub.com",
    "https://bsky.app/intent/compose?text=hi",
    "https://www.threads.net/intent/post?text=hi",
    "https://www.linkedin.com/sharing/share-offsite/?url=https%3A%2F%2Fgithub.com",
    "https://www.facebook.com/sharer/sharer.php?u=https%3A%2F%2Fgithub.com",
    "https://t.me/share/url?url=x&text=hi",
    "https://www.reddit.com/submit?url=x&title=hi",
    "https://mastodon.social/share?text=hi",
    "https://social.example.org/share?text=hi",
  ]) assert.ok(isShareUrl(url), url);
});

void test("anything else is refused", () => {
  for (const url of [
    "http://twitter.com/intent/tweet?text=hi", // not https
    "https://twitter.com/settings",
    "https://evil.example/intent/tweet?text=hi",
    "https://mastodon.social/share?text=hi&redirect=x", // more than the text
    "https://mastodon.social/admin?text=hi",
    "https://user:pw@bsky.app/intent/compose?text=hi",
    "https://bsky.app:8443/intent/compose?text=hi",
    "https://localhost/share?text=hi", // not a public host name
    "file:///etc/passwd", "javascript:alert(1)", "", 42, null,
    "https://bsky.app/intent/compose?text=" + "x".repeat(5000),
  ]) assert.equal(isShareUrl(url), false, String(url).slice(0, 80));
});

void test("pngBytes accepts only PNG data URLs", () => {
  assert.deepEqual(pngBytes("data:image/png;base64,iVBORw0K"), Buffer.from("iVBORw0K", "base64"));
  assert.equal(pngBytes("data:image/svg+xml;base64,PHN2Zz4="), null);
  assert.equal(pngBytes(undefined), null);
});
