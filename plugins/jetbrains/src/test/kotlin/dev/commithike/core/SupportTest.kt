package dev.commithike.core

import com.google.gson.JsonParser
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class SupportTest {
    @Test
    fun fileNames() {
        assertEquals("passwd.png", Files.safeName("../../etc/passwd", "x.png", "png"))
        assertEquals("c-d.png", Files.safeName("a b/c:d.png", "x.png", "png"))
        assertEquals("commit-hike.png", Files.safeName(null, "commit-hike.png", "png"))
        assertEquals("commit-hike.png", Files.safeName("...", "commit-hike.png", "png"))
        assertEquals("Day.PNG", Files.safeName("Day.PNG", "x.png", "png"))
        assertNull(Files.png("data:image/png;base64,!!!"))
        assertNull(Files.png(null))
    }

    @Test
    fun restDays() {
        assertEquals("sun,sat", RestDays.arg(listOf(6, 0, 6, 9)))
        assertEquals("none", RestDays.arg(emptyList()))
    }

    @Test
    fun errorLogKeepsTheLastOnes() {
        var t = 0
        val log = ErrorLog(keep = 3) { "t${t++}" }
        repeat(5) { log.note("e$it") }
        assertEquals(listOf("t2 e2", "t3 e3", "t4 e4"), log.recent())
        log.note("x".repeat(1000))
        assertEquals(600, log.recent().last().length)
    }

    @Test
    fun reportIsCompleteAndScrubbed() {
        val r = Report.build(
            "1.0 (prod)",
            "GoLand 2026.2",
            "Linux 6.1 amd64",
            "{\"commits\": 3}",
            listOf("failed in /home/denis/x for me@x.io"),
            "/home/denis",
        )
        assertTrue(r.contains("Plugin: 1.0 (prod)") && r.contains("IDE: GoLand 2026.2") && r.contains("\"commits\": 3"))
        assertTrue(r.contains("- failed in ~/x for <email>"))
        assertFalse(r.contains("denis") || r.contains("me@x.io"))
        assertTrue(Report.build("1", "i", "o", "{}", emptyList(), null).endsWith("- none"))
    }

    @Test
    fun panelPayload() {
        val status = Status(locale = "uk")
        val raw = JsonParser.parseString("""{"locale":"uk","new_field":{"x":1}}""")
        val view = TrailView("ok", repo = "/r", status = status.copy(raw = raw))
        val p = PanelPayload.of(
            view, "en-US", Avatar(true, "data:image/png;base64,AA"), emptyMap(), null, dev = false,
            BuildInfo("0.5.0", "prod"), openView = "settings", openToken = 2,
        )!!
        assertEquals("uk", p.locale)
        val json = JsonParser.parseString(
            p.script().removePrefix("window.commitHike && window.commitHike.update(").removeSuffix(");"),
        ).asJsonObject
        assertEquals(1, json.getAsJsonObject("status").getAsJsonObject("new_field").get("x").asInt) // the core's JSON passes untouched
        assertEquals(true, json.get("avatar_custom").asBoolean)
        assertEquals("settings", json.get("open_view").asString)
        assertNull(PanelPayload.of(TrailView("loading"), "en", Avatar(), emptyMap(), null, false, BuildInfo("1", "prod"), null, 0))
        // the IDE language when the core hasn't said yet; "</script>" can't end the script early
        val err = PanelPayload.of(
            TrailView(
                "error",
                error = "</script><b>",
            ),
            "en-US", Avatar(), emptyMap(), null, false, BuildInfo("1", "prod"), null, 0,
        )!!
        assertEquals("en-US", err.locale)
        assertFalse(err.script().contains("</script>"))
    }
}
