package dev.commithike.core

import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.io.File
import java.nio.file.Files
import java.nio.file.Paths
import java.time.LocalDate

/** The user flows end to end: the real core, a scripted user instead of the IDE. */
class TrekFlowsTest {
    /** A user who answers from a script and remembers what they were told. */
    private class ScriptedUser(
        var openFile: File? = null,
        var saveTo: File? = null,
        var confirms: Boolean = true,
        var restDays: List<Int>? = null,
        var difficulty: String? = null,
        var setupChoice: SetupChoice? = null,
        var routeChoice: RouteChoice? = null,
        var folder: File? = null,
        var text: String? = null,
        var language: String? = null,
        var removeFirst: Boolean = true,
    ) : TrekUi {
        val notes = mutableListOf<Pair<String, String?>>()
        val titles = mutableListOf<String?>()
        val actions = mutableListOf<() -> Unit>()
        var asked = 0
        var clipboard: String? = null
        var opened: File? = null
        var edited: File? = null
        var browsed: String? = null
        var image: ByteArray? = null
        var saveDefault: String? = null
        var offeredRoutes: List<Route> = emptyList()

        override fun chooseFileToOpen(title: String, description: String, ext: String) = openFile

        override fun chooseFileOrFolder(title: String, description: String) = openFile

        override fun chooseFolder(title: String) = folder

        override fun chooseFileToSave(title: String, description: String, ext: String, defaultName: String): File? {
            saveDefault = defaultName
            return saveTo
        }

        override fun askText(prompt: String, title: String, initial: String) = text

        override fun confirm(question: String, yes: String, no: String): Boolean {
            asked++
            return confirms
        }

        override fun setup(detectedEmail: String) = setupChoice

        override fun pickRoute(scope: Scope, routes: List<Route>, canRemove: Boolean): RouteChoice? {
            offeredRoutes = routes
            return routeChoice
        }

        override fun pickRouteToRemove(routes: List<Route>) = if (removeFirst) routes.first() else null

        override fun pickLanguage(info: LocaleInfo) = language

        override fun pickRestDays(current: List<Int>) = restDays

        override fun pickDifficulty(current: String) = difficulty

        override fun notify(text: String, button: String?, title: String?, action: (() -> Unit)?) {
            notes += text to button
            titles += title
            action?.let { actions += it }
        }

        override fun openInEditor(file: File) {
            edited = file
        }

        override fun copyToClipboard(text: String) {
            clipboard = text
        }

        override fun open(file: File) {
            opened = file
        }

        override fun browse(url: String) {
            browsed = url
        }

        override fun copyImageToClipboard(png: ByteArray) {
            image = png
        }
    }

    /** The plugin's side: the real core, counted refreshes, flows launched at once. */
    private class Host(val cli: CoreCli, override var currentRepo: String? = null) : TrekHost {
        var refreshed = 0
        var reloaded = 0
        var scanned = 0
        var shown = 0
        var teamForgotten = 0
        var locale: String? = null
        lateinit var flows: TrekFlows
        override val initialized: Boolean get() = runCatching { cli.config() }.isSuccess
        override val routes: Collection<Route> get() = cli.routes()
        override val status: Status? get() = runCatching { cli.status(currentRepo) }.getOrNull()
        override val localeInfo: LocaleInfo? = null

        override fun <T> core(block: (CoreCli) -> T): T = block(cli)

        override fun gitEmail() = "me@x.io"

        override fun markInitialized() {}

        override fun refreshAll() {
            refreshed++
        }

        override fun refreshProject() {
            refreshed++
        }

        override fun scanProject(repo: String) {
            cli.scan(repo)
            scanned++
        }

        override fun scanAllRepositories() {
            currentRepo?.let { scanProject(it) }
        }

        override fun forgetTeam() {
            teamForgotten++
        }

        override fun reloadRoutes() {
            reloaded++
        }

        override fun reloadAvatar() {
            reloaded++
        }

        override fun reloadLocale(set: String?) {
            locale = set
            cli.locale(set)
        }

        override fun showTrail() {
            shown++
        }

        override fun launch(flow: TrekFlows.() -> Unit) = flows.flow()
    }

    private fun flows(host: Host, user: ScriptedUser, build: BuildInfo = BuildInfo("0.5.0", "prod")) = TrekFlows(host, user, build).also {
        host.flows =
            it
    }

    private fun bin() = (
        System.getenv("COMMIT_HIKE_TEST_BINARY")?.let { Paths.get(it) }
            ?: Paths.get("..", "..", "core", "dist", CorePlatform.binaryName())
        )
        .also { if (!Files.exists(it)) fail("Core binary not found at $it. Build it with `task core:dist`") }
        .also { it.toFile().setExecutable(true) }

    private fun freshCli(initialize: Boolean = true): CoreCli {
        val cli = CoreCli(bin(), extraEnv = mapOf("COMMIT_HIKE_HOME" to Files.createTempDirectory("flows-home").toString()))
        if (initialize) cli.init(listOf("me@x.io"), "all", fromHistory = true)
        return cli
    }

    private fun tmp(name: String) = Files.createTempDirectory("flows-out").resolve(name).toFile()

    @Test
    fun daysOffSettingsAndDifficulty() {
        val host = Host(freshCli())
        val user = ScriptedUser(restDays = listOf(6, 0), difficulty = "hard")
        val flows = flows(host, user)
        flows.setRestDays()
        assertEquals(listOf(0, 6), host.cli.restDays().days)
        user.restDays = null // the user cancels: nothing changes
        flows.setRestDays()
        assertEquals(listOf(0, 6), host.cli.restDays().days)
        flows.setRestDaysTo(emptyList())
        assertEquals(emptyList<Int>(), host.cli.restDays().days)
        flows.setSettings(null, "on", "off")
        assertEquals(Settings("auto", "on", "off"), host.cli.status().settings)
        flows.changeDifficulty()
        assertEquals("hard", host.cli.difficulty().level)
        assertEquals(4, host.refreshed)
    }

    @Test
    fun backupsGoOutAndComeBackAskingBeforeReplacing() {
        val first = Host(freshCli())
        first.cli.restDays("sat")
        val file = tmp("backup.json")
        val user = ScriptedUser(saveTo = file, openFile = file)
        flows(first, user).exportProgress(LocalDate.of(2026, 10, 2))
        assertEquals("commit-hike-backup-2026-10-02.json", user.saveDefault)
        assertTrue(file.exists() && user.notes.last().first.contains(file.path))

        // a new computer: nothing to replace, so no question
        val second = Host(freshCli(initialize = false))
        flows(second, user).importProgress()
        assertEquals(0, user.asked)
        assertEquals(listOf(6), second.cli.restDays().days)
        assertEquals(2, second.reloaded) // routes and the hiker icon

        // existing progress: asked first; "no" leaves it alone, "yes" replaces it
        second.cli.restDays("mon")
        user.confirms = false
        flows(second, user).importProgress()
        assertEquals(1, user.asked)
        assertEquals(listOf(1), second.cli.restDays().days)
        user.confirms = true
        flows(second, user).importProgress()
        assertEquals(listOf(6), second.cli.restDays().days)

        // cancelled file choosers do nothing
        val idle = ScriptedUser()
        flows(second, idle).exportProgress()
        flows(second, idle).importProgress()
        assertTrue(idle.notes.isEmpty())
    }

    @Test
    fun aBrokenBackupIsAnErrorNotAQuestion() {
        val bad = tmp("bad.json").apply { writeText("{\"format\":\"nope\"}") }
        val user = ScriptedUser(openFile = bad)
        try {
            flows(Host(freshCli()), user).importProgress()
            fail("a broken backup must fail")
        } catch (e: CoreException) {
            assertEquals("invalid_backup", e.code)
        }
        assertEquals(0, user.asked)
    }

    @Test
    fun badgePostcardAndReport() {
        val host = Host(freshCli())
        val svg = tmp("my-badge.svg")
        val user = ScriptedUser(saveTo = svg)
        val flows = flows(host, user, BuildInfo("0.5.0", "prod", "https://github.com/o/r"))
        flows.saveBadge()
        assertTrue(svg.readText().startsWith("<svg"))
        user.actions.last()()
        assertTrue(user.clipboard!!.contains("(my-badge.svg)")) // the Markdown points at the name the user chose

        val png = tmp("day.png")
        user.saveTo = png
        flows.savePostcard("day.png", byteArrayOf(1, 2, 3))
        assertEquals(3L, png.length())
        user.actions.last()()
        assertEquals(png, user.opened)

        // sharing: the postcard goes to the clipboard, with a note to paste it
        flows.copyPostcard(byteArrayOf(9, 8, 7))
        assertArrayEquals(byteArrayOf(9, 8, 7), user.image)
        assertEquals(I18n.t("postcardCopied"), user.notes.last().first)

        flows.copyDiagnostics("GoLand 2026.2", "Linux", listOf("failed for me@x.io"), null)
        assertTrue(user.clipboard!!.contains("Plugin: 0.5.0 (prod)") && user.clipboard!!.contains("\"initialized\""))
        assertFalse(user.clipboard!!.contains("me@x.io"))
        user.actions.last()()
        assertEquals("https://github.com/o/r/issues", user.browsed)

        val noRepo = ScriptedUser()
        flows(host, noRepo, BuildInfo("1", "prod")).copyDiagnostics("i", "o", emptyList(), null)
        assertEquals(null, noRepo.notes.last().second) // no "Open issues" without a repository
    }

    /** A git repository with one commit by me@x.io. */
    private fun repo(): File {
        val dir = Files.createTempDirectory("flows-repo").toFile()
        fun git(vararg a: String) =
            check(ProcessBuilder(listOf("git", "-C", dir.path) + a).redirectErrorStream(true).start().waitFor() == 0) { a.toList() }
        git("init", "-q")
        File(dir, "a.go").writeText((1..40).joinToString("\n") { "line $it" })
        git("add", "-A")
        git("-c", "user.email=me@x.io", "-c", "user.name=Me", "commit", "-qm", "first")
        return dir
    }

    @Test
    fun setupCountsTheChosenProjectAndShowsTheTrail() {
        val r = repo()
        val host = Host(freshCli(initialize = false), currentRepo = r.path)
        val user = ScriptedUser(setupChoice = SetupChoice(listOf("me@x.io"), "selected", fromHistory = true, difficulty = "easy"))
        flows(host, user).setup()
        assertEquals(1, user.asked) // "count this project?"
        assertEquals("easy", host.cli.difficulty().level)
        assertTrue(host.cli.status(r.path).tracked)
        assertEquals(1, host.shown)
        val nobody = ScriptedUser()
        flows(Host(freshCli(initialize = false)), nobody).setup() // cancelled: nothing happens
        assertTrue(nobody.notes.isEmpty())
    }

    @Test
    fun journeysAndProjects() {
        val r = repo()
        val host = Host(freshCli(), currentRepo = null)
        val user = ScriptedUser(routeChoice = RouteChoice("chornohora-ridge", fromHistory = true))
        flows(host, user).chooseRoute(Scope.PROJECT) // no project open
        assertEquals("", user.titles.last())
        flows(host, user).chooseRoute(Scope.GLOBAL)
        assertTrue(user.offeredRoutes.map { it.name } == user.offeredRoutes.map { it.name }.sorted())
        assertEquals("chornohora-ridge", host.cli.status().global!!.route.id)

        host.currentRepo = r.path
        flows(host, user).setProjectEnabled(true) // every project counts already
        assertTrue(user.notes.last().first.isNotBlank())
        flows(host, user).verify()
        assertEquals(1, host.teamForgotten)
        assertTrue(user.notes.last().first.isNotBlank())

        flows(host, user).setTeam(true)
        assertEquals(2, host.teamForgotten)

        // a team goal from the Team tab: set, recounted, removed
        flows(host, user).setTeamGoal("svydovets-ridge")
        assertEquals(3, host.teamForgotten)
        assertEquals("svydovets-ridge", host.cli.team(r.path).goal!!.routeId)
        assertTrue(host.cli.team(r.path).routes!!.size >= 10)
        flows(host, user).setTeamGoal("")
        assertNull(host.cli.team(r.path).goal)
        host.currentRepo = null
        flows(host, user).setTeam(true) // needs a project
        assertEquals("", user.titles.last())
    }

    @Test
    fun customRoutesFromTemplateToImportToRemoval() {
        val host = Host(freshCli())
        val folder = Files.createTempDirectory("flows-routes").toFile()
        val user = ScriptedUser(folder = folder, text = " my-ridge ")
        flows(host, user).createRouteTemplate()
        val pack = File(folder, "my-ridge")
        assertEquals(File(pack, "route.json").path, user.edited!!.path)

        user.openFile = pack
        user.actions.last()() // "Import" on the notification
        assertTrue(host.cli.routes().any { it.id == "my-ridge" })
        user.actions.last()() // "Walk it now"
        assertEquals("my-ridge", host.cli.status().global!!.route.id)

        user.confirms = false // importing it again asks before replacing; "no" keeps it
        flows(host, user).importRoute()
        assertEquals(1, user.asked)

        user.removeFirst = false
        flows(host, user).removeRoute() // cancelled
        assertTrue(host.cli.routes().any { it.id == "my-ridge" })
        host.cli.setJourney(Scope.GLOBAL, "demo-trail", fromHistory = false) // a route in use can't go
        user.removeFirst = true
        flows(host, user).removeRoute()
        assertFalse(host.cli.routes().any { it.id == "my-ridge" })
        flows(host, user).removeRoute() // nothing left to remove
        assertEquals("", user.titles.last())
    }

    @Test
    fun hikerAndLanguage() {
        val host = Host(freshCli())
        val png = Paths.get("..", "..", "ui", "panel", "hiker-default.png").toFile()
        val user = ScriptedUser(openFile = png, language = "uk")
        flows(host, user).setHikerIcon()
        assertTrue(host.cli.avatar().custom)
        flows(host, user).resetHikerIcon()
        assertFalse(host.cli.avatar().custom)
        flows(host, user).changeLanguage()
        assertEquals("uk", host.locale)
        assertEquals("uk", host.cli.locale().locale)
        user.language = null
        flows(host, user).changeLanguage() // cancelled
        assertEquals("uk", host.locale)
    }
}
