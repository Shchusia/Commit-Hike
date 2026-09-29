package dev.commithike.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.io.File
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths

class CoreCliTest {

    @Test
    fun binaryNamesMatchGoTargets() {
        assertEquals("commit-hike-windows-amd64.exe", CorePlatform.binaryName("Windows 11", "amd64"))
        assertEquals("commit-hike-darwin-arm64", CorePlatform.binaryName("Mac OS X", "aarch64"))
        assertEquals("commit-hike-darwin-amd64", CorePlatform.binaryName("Mac OS X", "x86_64"))
        assertEquals("commit-hike-linux-arm64", CorePlatform.binaryName("Linux", "arm64"))
        try {
            CorePlatform.binaryName("SunOS", "sparc")
            fail("expected an unsupported-platform error")
        } catch (e: CoreException) {
            assertTrue(e.message!!.contains("doesn't support"))
        }
    }

    @Test
    fun formatsDistances() {
        assertEquals("42 m", formatDistance(42.4))
        assertEquals("8.4 km", formatDistance(8400.0))
        assertEquals("250 km", formatDistance(250_000.0))
    }

    @Test
    fun talksToTheRealCore() {
        val bin = coreBinary()
        val home = Files.createTempDirectory("ct-home")
        val repo = Files.createTempDirectory("ct-repo").toFile()
        git(repo, "init", "-q")
        File(repo, "a.kt").writeText("x\ny\nz\n")
        git(repo, "add", "-A")
        git(repo, "commit", "-qm", "c")

        val cli = CoreCli(bin, extraEnv = mapOf("COMMIT_HIKE_HOME" to home.toString()))

        try {
            cli.status()
            fail("expected not_initialized")
        } catch (e: CoreException) {
            assertTrue(e.notInitialized)
        }

        cli.init(listOf("me@x.io"), "all", fromHistory = true)
        val scan = cli.scan(repo.path)
        assertEquals(1, scan.newCommits)
        assertEquals(650.0, scan.addedM, 0.001) // 3 lines = 50 base points × 13 (medium)
        assertEquals(650.0, scan.status().global!!.distanceM, 0.001)
        assertEquals("medium", scan.difficulty)
        assertEquals("Old Bridge", scan.global!!.nextWaypoint!!.name)
        // 650 m: past the 100 m "first steps" threshold and nothing else
        assertEquals(listOf("first-steps"), scan.events.orEmpty().mapNotNull { it.achievement?.id })
        assertEquals(5, scan.global!!.achievements!!.size)
        assertEquals(1, scan.global!!.achievements!!.count { it.unlockedAt != 0L })

        cli.lang = "uk"
        val st = cli.status(repo.path)
        assertEquals("uk", st.locale)
        assertEquals("Демо-стежка", st.global!!.route.name)

        val routes = cli.routes()
        val ridge = routes.first { it.id == "chornohora-ridge" }
        assertTrue(ridge.waypoints!!.isNotEmpty())
        val proj = cli.setJourney(Scope.PROJECT, ridge.id, fromHistory = true, repo = repo.path)
        assertEquals("Чорногірський хребет", proj.project!!.route.name)
        assertEquals(650.0, proj.project!!.distanceM, 0.001)

        try {
            cli.setJourney(Scope.GLOBAL, "atlantis", fromHistory = false)
            fail("expected unknown_route")
        } catch (e: CoreException) {
            assertEquals("unknown_route", e.code)
        }

        val notRepo = cli.scan(Files.createTempDirectory("ch-plain").toString())
        assertFalse(notRepo.tracked)

        // The panel gets the core's JSON untouched: the GPS track and stop coordinates
        // must reach it even though the widget doesn't need them.
        val raw = cli.status(repo.path).raw!!.asJsonObject
        val rawRoute = raw.getAsJsonObject("project").getAsJsonObject("route")
        assertTrue(rawRoute.getAsJsonArray("track").size() > 10)
        assertTrue(rawRoute.getAsJsonArray("waypoints").any { it.asJsonObject.has("lat") })

        // Map data and daily stats survive the round trip through the Kotlin models.
        val hoverla = proj.project!!.route.waypoints!!.first { it.id == "hoverla" }
        assertEquals("peak", hoverla.kind)
        assertEquals(2061.0, hoverla.elevationM, 0.001)
        assertTrue(proj.project!!.route.biomes!!.isNotEmpty())
        assertEquals(14, proj.project!!.daily!!.size)
        // Day 1 is the day of the first commit (UTC). Test commits are dated a few
        // hours back, so early in the UTC morning that is already yesterday.
        val firstCommitDay = gitOut(repo, "log", "--reverse", "--format=%at").lines().first().trim().toLong() / 86_400
        val today = System.currentTimeMillis() / 1000 / 86_400
        assertEquals((today - firstCommitDay + 1).toInt(), proj.project!!.day)

        // Custom routes: template -> import -> remove.
        val work = Files.createTempDirectory("ch-routes").toString()
        val dir = cli.routeTemplate("my-trail", work)
        val imported = cli.importRoute(dir, replace = false)
        assertEquals("Моя стежка", imported.name)
        assertFalse(imported.builtin)
        try {
            cli.importRoute(dir, replace = false)
            fail("expected route_exists")
        } catch (e: CoreException) {
            assertEquals("route_exists", e.code)
        }
        cli.removeRoute("my-trail")

        // Hiker icon: the panel's default figure is itself a valid custom icon.
        assertFalse(cli.avatar().custom)
        val icon = Paths.get("..", "..", "ui", "panel", "hiker-default.png").toString()
        val av = cli.setAvatar(icon)
        assertTrue(av.custom && av.dataUrl!!.startsWith("data:image/png;base64,"))
        try {
            cli.setAvatar("$dir/route.json")
            fail("expected invalid_image")
        } catch (e: CoreException) {
            assertEquals("invalid_image", e.code)
        }
        cli.resetAvatar()
        assertFalse(cli.avatar().custom)
        assertTrue(cli.routes().none { it.id == "my-trail" })

        assertEquals(0, cli.verify(repo.path).removed)
        assertEquals(listOf("me@x.io"), cli.config().emails)

        // Elevation, facts and objects reach the Kotlin models.
        val ridgeRoute = proj.project!!.route
        assertEquals(2061.0, ridgeRoute.maxElevationM, 0.001)
        assertTrue(ridgeRoute.facts!!.isNotEmpty() && ridgeRoute.objects!!.isNotEmpty())
        assertEquals(1358.0, proj.project!!.elevationM!!, 5.0)

        // A rewrite (reset) is recounted when the previous HEAD is passed.
        val head = gitOut(repo, "rev-parse", "HEAD")
        File(repo, "b.kt").writeText("1\n2\n3\n")
        git(repo, "add", "-A")
        git(repo, "commit", "-qm", "d")
        cli.scan(repo.path)
        val after = gitOut(repo, "rev-parse", "HEAD")
        git(repo, "reset", "-q", "--hard", head)
        val rewritten = cli.scan(repo.path, prevHead = after)
        assertTrue(rewritten.rewritten)
        assertEquals(1, rewritten.removedCommits)

        // Team, language and route pictures.
        cli.setTeam(repo.path, true)
        assertTrue(cli.status(repo.path).team)
        val team = cli.team(repo.path)
        assertEquals(1, team.members!!.size)
        assertTrue(team.members!!.first().me)
        assertEquals("uk", cli.locale("uk").effective)
        assertEquals("", cli.locale("auto").locale)
        val assets = cli.routeAssets("seven-lighthouses")
        assertTrue(assets.images!!["assets/parchment-map.svg"]!!.startsWith("data:image/svg+xml;base64,"))
        assertTrue(assets.html!!.containsKey("assets/aurora.html"))

        // Difficulty, tunnels and encounters.
        assertEquals("easy", cli.difficulty("easy").level)
        assertTrue(cli.difficulty().typicalDayM!!.getValue("easy") > cli.difficulty().typicalDayM!!.getValue("hard"))
        // Tunnels and encounters, from a fixture pack rather than shipped content.
        val tunnel = cli.importRoute(
            Paths.get("..", "..", "core", "internal", "app", "testdata", "routes", "tunnel-test").toString(),
            replace = true,
        )
        assertEquals(2, tunnel.underground!!.size)
    }

    @Test
    fun translationsAreComplete() {
        assertEquals(emptyMap<String, Set<String>>(), I18n.missing())
        I18n.setLanguage("uk-UA")
        assertEquals("uk", I18n.lang)
        assertEquals("8,4 км", I18n.distance(8400.0))
        assertEquals("Сьогодні: 5 м", I18n.t("barToday", I18n.distance(5.0)))
        I18n.setLanguage("de")
        assertEquals("en", I18n.lang)
    }

    private fun coreBinary(): Path {
        System.getenv("COMMIT_HIKE_TEST_BINARY")?.let { return Paths.get(it) }
        val p = Paths.get("..", "..", "core", "dist", CorePlatform.binaryName())
        if (!Files.exists(p)) fail("Core binary not found at $p. Build it with `task core:dist`")
        p.toFile().setExecutable(true)
        return p
    }

    private fun gitOut(dir: File, vararg args: String): String {
        val p = ProcessBuilder(listOf("git", "-C", dir.path) + args).start()
        val out = p.inputStream.readBytes().toString(Charsets.UTF_8).trim()
        check(p.waitFor() == 0) { "git ${args.toList()}" }
        return out
    }

    // Every git call gets its own minute: commits made within one second
    // share a dedup key (author email + time) and would count as one.
    private var clock = System.currentTimeMillis() / 1000 - 6 * 3600

    private fun git(dir: File, vararg args: String) {
        val pb = ProcessBuilder(listOf("git", "-C", dir.path) + args).redirectErrorStream(true)
        clock += 60
        pb.environment().putAll(
            mapOf(
                "GIT_AUTHOR_NAME" to "T",
                "GIT_AUTHOR_EMAIL" to "me@x.io",
                "GIT_COMMITTER_NAME" to "T",
                "GIT_COMMITTER_EMAIL" to "me@x.io",
                "GIT_AUTHOR_DATE" to "@$clock +0000",
                "GIT_COMMITTER_DATE" to "@$clock +0000",
            ),
        )
        val p = pb.start()
        val out = p.inputStream.readBytes().toString(Charsets.UTF_8)
        check(p.waitFor() == 0) { "git ${args.toList()}: $out" }
    }
}
