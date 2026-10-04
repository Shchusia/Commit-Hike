package dev.commithike.core

import java.net.URI

/**
 * Sharing from the panel: the only web pages the plugin opens for it. The
 * panel builds the URL; this checks it is one of the networks' sharing pages
 * (a Mastodon server's /share for Mastodon), so a message from the panel can
 * never make the IDE open anything else. The same list as the VS Code
 * extension's share.ts.
 */
object ShareLinks {
    private val pages = setOf(
        "https://twitter.com/intent/tweet",
        "https://bsky.app/intent/compose",
        "https://www.threads.net/intent/post",
        "https://www.linkedin.com/sharing/share-offsite/",
        "https://www.facebook.com/sharer/sharer.php",
        "https://t.me/share/url",
        "https://www.reddit.com/submit",
    )
    private val host = Regex("^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$")

    fun allowed(url: String?): Boolean {
        if (url == null || url.length > 4000) return false
        val u = runCatching { URI(url) }.getOrNull() ?: return false
        if (u.scheme != "https" || u.rawUserInfo != null || u.port != -1 || u.rawFragment != null || u.host == null) return false
        if ("https://${u.host}${u.rawPath}" in pages) return true
        // Mastodon: any server, but only its share page with nothing but the text
        val keys = u.rawQuery?.split('&')?.map { it.substringBefore('=') }.orEmpty()
        return u.rawPath == "/share" && host.matches(u.host) && keys == listOf("text")
    }
}
