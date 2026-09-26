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
        assertEquals(50.0, scan.addedM, 0.001)
        assertEquals(50.0, scan.status().global!!.distanceM, 0.001)
        assertEquals("Old Bridge", scan.global!!.nextWaypoint!!.name)
        // 50 m: below the 100 m "first steps" threshold, so nothing is unlocked yet
        assertTrue(scan.events.orEmpty().isEmpty())
        assertEquals(5, scan.global!!.achievements!!.size)
        assertTrue(scan.global!!.achievements!!.all { it.unlockedAt == 0L })

        cli.lang = "uk"
        val st = cli.status(repo.path)
        assertEquals("uk", st.locale)
        assertEquals("Демо-стежка", st.global!!.route.name)

        val routes = cli.routes()
        val ridge = routes.first { it.id == "chornohora-ridge" }
        assertTrue(ridge.waypoints!!.isNotEmpty())
        val proj = cli.setJourney(Scope.PROJECT, ridge.id, fromHistory = true, repo = repo.path)
        assertEquals("Чорногірський хребет", proj.project!!.route.name)
        assertEquals(50.0, proj.project!!.distanceM, 0.001)

        try {
            cli.setJourney(Scope.GLOBAL, "atlantis", fromHistory = false)
            fail("expected unknown_route")
        } catch (e: CoreException) {
            assertEquals("unknown_route", e.code)
        }

        val notRepo = cli.scan(Files.createTempDirectory("ch-plain").toString())
        assertFalse(notRepo.tracked)

        assertEquals(0, cli.verify(repo.path).removed)
        assertEquals(listOf("me@x.io"), cli.config().emails)
    }

    private fun coreBinary(): Path {
        System.getenv("COMMIT_HIKE_TEST_BINARY")?.let { return Paths.get(it) }
        val p = Paths.get("..", "..", "core", "dist", CorePlatform.binaryName())
        if (!Files.exists(p)) fail("Core binary not found at $p. Build it with `task core:dist`")
        p.toFile().setExecutable(true)
        return p
    }

    private fun git(dir: File, vararg args: String) {
        val pb = ProcessBuilder(listOf("git", "-C", dir.path) + args).redirectErrorStream(true)
        pb.environment().putAll(
            mapOf(
                "GIT_AUTHOR_NAME" to "T",
                "GIT_AUTHOR_EMAIL" to "me@x.io",
                "GIT_COMMITTER_NAME" to "T",
                "GIT_COMMITTER_EMAIL" to "me@x.io",
            ),
        )
        val p = pb.start()
        val out = p.inputStream.readBytes().toString(Charsets.UTF_8)
        check(p.waitFor() == 0) { "git ${args.toList()}: $out" }
    }
}
