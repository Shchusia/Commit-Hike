package dev.commithike.core

import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import java.util.Base64

class PanelCommandTest {
    private fun parse(json: String) = PanelCommand.parse(json)

    private val pngBytes = byteArrayOf(0x89.toByte(), 'P'.code.toByte(), 'N'.code.toByte(), 'G'.code.toByte(), 13, 10, 26, 10, 0, 0)
    private val png = "data:image/png;base64," + Base64.getEncoder().encodeToString(pngBytes)

    @Test
    fun simpleCommands() {
        val simple = mapOf(
            "ready" to PanelCommand.Ready, "setup" to PanelCommand.Setup, "refresh" to PanelCommand.Refresh,
            "enableProject" to PanelCommand.EnableProject, "setAvatar" to PanelCommand.SetAvatar, "resetAvatar" to PanelCommand.ResetAvatar,
            "exportProgress" to PanelCommand.ExportProgress, "importProgress" to PanelCommand.ImportProgress,
            "copyDiagnostics" to PanelCommand.CopyDiagnostics, "saveBadge" to PanelCommand.SaveBadge,
            "requestTeam" to PanelCommand.RequestTeam, "importRoute" to PanelCommand.ImportRoute,
            "verify" to PanelCommand.Verify,
        )
        for ((name, cmd) in simple) assertEquals(name, cmd, parse("""{"command":"$name"}"""))
    }

    @Test
    fun commandsWithArguments() {
        assertEquals(PanelCommand.ChooseRoute(project = true), parse("""{"command":"chooseRoute","scope":"project"}"""))
        assertEquals(PanelCommand.ChooseRoute(project = false), parse("""{"command":"chooseRoute"}"""))
        assertEquals(PanelCommand.SetLocale("uk"), parse("""{"command":"setLocale","locale":"uk"}"""))
        assertEquals(PanelCommand.SetTeam(true), parse("""{"command":"setTeam","on":true}"""))
        assertEquals(PanelCommand.SetDifficulty("hard"), parse("""{"command":"setDifficulty","level":"hard"}"""))
        assertEquals(PanelCommand.SetRestDays(listOf(0, 6)), parse("""{"command":"setRestDays","days":[6,0,6,9,-1,2.5,"x"]}"""))
        assertEquals(
            PanelCommand.SetSettings("on", null, "milestones"),
            parse("""{"command":"setSettings","reduce_motion":"on","high_contrast":"maybe","notifications":"milestones"}"""),
        )
        assertEquals(
            PanelCommand.SetSettings(null, null, null, festive = "off", ambient = null),
            parse("""{"command":"setSettings","festive":"off","ambient":"sometimes"}"""),
        )
        assertEquals(PanelCommand.PanelError("panel: boom"), parse("""{"command":"panelError","message":"boom"}"""))
        assertEquals(PanelCommand.WalkRoute("svydovets-ridge"), parse("""{"command":"walkRoute","id":"svydovets-ridge"}"""))
        assertNull(parse("""{"command":"walkRoute","id":"../../etc"}"""))
    }

    @Test
    fun postcardsAreRealPngsWithSafeNames() {
        val c = parse("""{"command":"savePostcard","name":"../../etc/day-7.png","data":"$png"}""") as PanelCommand.SavePostcard
        assertEquals("day-7.png", c.fileName)
        assertArrayEquals(pngBytes, c.png)
        assertNull(parse("""{"command":"savePostcard","name":"x.png","data":"data:text/html;base64,PHNjcmlwdD4="}"""))
        assertNull(parse("""{"command":"savePostcard","name":"x.png","data":"data:image/png;base64,PHNjcmlwdD4="}""")) // not a PNG inside
    }

    @Test
    fun malformedOrMistypedMessagesAreDropped() {
        for (bad in listOf(
            "not json", "[]", "{}", """{"command":42}""", """{"command":"launchRockets"}""",
            """{"command":"setLocale","locale":["uk"]}""", """{"command":"setLocale","locale":""}""",
            """{"command":"setTeam","on":"yes"}""", """{"command":"setDifficulty","level":"nightmare"}""",
            """{"command":"setRestDays","days":"sat"}""", """{"command":"setSettings","notifications":"loud"}""",
        )) {
            assertNull(bad, parse(bad))
        }
    }

    @Test
    fun sharing() {
        val x = "https://twitter.com/intent/tweet?text=hi&url=https%3A%2F%2Fgithub.com"
        assertEquals(PanelCommand.OpenUrl(x), parse("""{"command":"openUrl","url":"$x"}"""))
        val masto = "https://social.example.org/share?text=hi"
        assertEquals(PanelCommand.OpenUrl(masto), parse("""{"command":"openUrl","url":"$masto"}"""))
        for (bad in listOf(
            "http://twitter.com/intent/tweet?text=hi", "https://twitter.com/settings", "https://evil.example/intent/tweet",
            "https://mastodon.social/share?text=hi&next=x", "https://localhost/share?text=hi", "https://u:p@bsky.app/intent/compose?text=x",
            "https://bsky.app:8443/intent/compose?text=x", "file:///etc/passwd", "javascript:alert(1)",
        )) {
            // refused, and logged rather than dropped silently
            assertTrue(parse("""{"command":"openUrl","url":"$bad"}""") is PanelCommand.PanelError)
        }
        assertNull(parse("""{"command":"openUrl","url":42}"""))
        assertEquals(PanelCommand.CopyText("🥾 hi"), parse("""{"command":"copyText","text":"🥾 hi"}"""))
        assertNull(parse("""{"command":"copyText","text":""}"""))
        assertEquals(PanelCommand.CopyImage(pngBytes), parse("""{"command":"copyImage","data":"$png"}"""))
        assertNull(parse("""{"command":"copyImage","data":"data:image/svg+xml;base64,PHN2Zz4="}"""))
    }

    @Test
    fun teamGoal() {
        assertEquals(PanelCommand.SetTeamGoal("camino-frances"), parse("""{"command":"setTeamGoal","route":"camino-frances"}"""))
        assertEquals(PanelCommand.SetTeamGoal(""), parse("""{"command":"setTeamGoal","route":""}"""))
        assertNull(parse("""{"command":"setTeamGoal","route":"../etc"}"""))
        assertNull(parse("""{"command":"setTeamGoal"}"""))
    }
}
