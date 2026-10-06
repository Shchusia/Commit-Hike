package dev.commithike.core

import com.google.gson.JsonParser
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class SiteTest {
    @Test
    fun theSitesAddress() {
        assertEquals(Site.DEFAULT, Site.url(emptyMap()))
        assertEquals("http://127.0.0.1:9000", Site.url(mapOf("COMMIT_HIKE_SITE" to "http://127.0.0.1:9000/")))
        assertEquals(Site.DEFAULT, Site.url(mapOf("COMMIT_HIKE_SITE" to "javascript:alert(1)")))
    }

    @Test
    fun onlyRouteAndAuthorPagesOpen() {
        val base = Site.DEFAULT
        for (ok in listOf("$base/routes/svydovets-ridge", "$base/u/12", "$base/")) assertTrue(Site.isPage(ok, base))
        for (bad in listOf(
            "https://evil.example/routes/x",
            "$base/admin",
            "$base/routes/../admin",
            "https://u:p@commit-hike.dev/routes/x",
        )) {
            assertFalse(Site.isPage(bad, base))
        }
    }

    @Test
    fun thePanelsSearchBecomesCoreArguments() {
        val q = JsonParser.parseString("""{"q":" lakes ","kind":"story","length":"m","sort":"rating","page":3}""").asJsonObject
        assertEquals(
            listOf("site", "routes", "--q", "lakes", "--kind", "story", "--length", "m", "--sort", "rating", "--page", "3"),
            Site.searchArgs(q),
        )
        val odd = JsonParser.parseString("""{"q":"","kind":"--evil","sort":["x"],"page":-1}""").asJsonObject
        assertEquals(listOf("site", "routes"), Site.searchArgs(odd))
    }

    @Test
    fun panelCommands() {
        fun parse(s: String) = PanelCommand.parse(s)
        assertEquals(PanelCommand.SiteSearch(listOf("site", "routes", "--q", "x")), parse("""{"command":"siteSearch","query":{"q":"x"}}"""))
        assertEquals(
            PanelCommand.SiteInstall("lake-walk", true),
            parse("""{"command":"siteInstall","id":"lake-walk","replaceLocal":true}"""),
        )
        assertEquals(
            PanelCommand.SiteInstall("lake-walk", false),
            parse("""{"command":"siteInstall","id":"lake-walk","replaceLocal":"yes"}"""),
        )
        assertNull(parse("""{"command":"siteInstall","id":"../etc"}"""))
        assertEquals(
            PanelCommand.OpenUrl("${Site.url()}/routes/lake-walk"),
            parse("""{"command":"openUrl","url":"${Site.url()}/routes/lake-walk"}"""),
        )
        assertNull(parse("""{"command":"openUrl","url":"${Site.url()}/admin"}"""))
    }
}
