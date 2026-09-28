import kotlinx.kover.gradle.plugin.dsl.GroupingEntityType
import org.jetbrains.intellij.platform.gradle.IntelliJPlatformType
import org.jetbrains.intellij.platform.gradle.TestFrameworkType

plugins {
    id("java")
    id("org.jetbrains.kotlin.jvm") version "2.0.21"
    id("org.jetbrains.intellij.platform") version "2.5.0"
    id("org.jlleitschuh.gradle.ktlint") version "12.1.2" // `./gradlew ktlintCheck` / `ktlintFormat`
    id("org.jetbrains.kotlinx.kover") version "0.9.1" // coverage: `./gradlew koverLog koverHtmlReport`
}

group = "dev.commithike"
// One version for the whole project: the VERSION file in the repository root.
version = rootDir.resolve("../../VERSION").readText().trim()

// dev: the trail view may look ahead along the route; prod: only back (what
// the marketplaces get). `./gradlew buildPlugin -PcommitHikeFlavor=dev`
val flavor = providers.gradleProperty("commitHikeFlavor").getOrElse("prod")
check(flavor == "dev" || flavor == "prod") { "commitHikeFlavor must be dev or prod, got $flavor" }

// The IDE the plugin is compiled against; also the baseline for verification.
val platformVersion = "2024.3.5"

kotlin {
    jvmToolchain(21)
}

repositories {
    mavenCentral()
    intellijPlatform {
        defaultRepositories()
    }
}

dependencies {
    intellijPlatform {
        // Compiled against the common platform, so it runs in PyCharm, IDEA, GoLand, WebStorm...
        intellijIdeaCommunity(platformVersion)
        bundledPlugin("Git4Idea")
        testFramework(TestFrameworkType.Platform)
    }
    testImplementation("junit:junit:4.13.2")
}

intellijPlatform {
    pluginConfiguration {
        changeNotes.set(provider { changeNotesFor(project.version.toString()) })
        ideaVersion {
            sinceBuild = "243"
            untilBuild = provider { null }
        }
    }
    // Marketplace publishing: `task jetbrains:publish` (secrets come from the environment,
    // never from files in the repository). A version like 0.3.0-beta goes to the beta channel.
    publishing {
        token.set(providers.environmentVariable("JETBRAINS_TOKEN"))
        channels.set(listOf(project.version.toString().substringAfter('-', "").substringBefore('.').ifEmpty { "default" }))
    }
    // Plugin signing: without it the IDE warns users when they install the plugin.
    // See docs/publishing.md for creating the certificate.
    signing {
        certificateChain.set(providers.environmentVariable("CERTIFICATE_CHAIN"))
        privateKey.set(providers.environmentVariable("PRIVATE_KEY"))
        password.set(providers.environmentVariable("PRIVATE_KEY_PASSWORD"))
    }
    pluginVerification {
        ides {
            // The IDE we compile against (already downloaded for the build).
            ide(IntelliJPlatformType.IntellijIdeaCommunity, platformVersion)
            // Your installed IDE, no download: -PpycharmPath=/snap/pycharm-professional/current
            providers.gradleProperty("pycharmPath").orNull?.let { local(it) }
            // Every recent release (downloads several IDEs, for CI): -PverifyRecommended
            if (providers.gradleProperty("verifyRecommended").isPresent) recommended()
        }
    }
}

// The shared trail panel and the core binaries go into the plugin jar as resources.
val coreDist = rootDir.resolve("../../core/dist")
val generatedResources = layout.buildDirectory.dir("generated/commit-hike")

val copyShared by tasks.registering(Copy::class) {
    doFirst {
        check(coreDist.listFiles()?.any { it.name.startsWith("commit-hike-") } == true) {
            "Core binaries not found in $coreDist. Run `task core:dist` in the repository root first (needs Go)."
        }
    }
    from(rootDir.resolve("../../ui/panel/panel.html")) { into("ui") }
    from(coreDist) {
        include("commit-hike-*")
        into("bin")
    }
    into(generatedResources)
}

// Version and flavor, readable by the plugin at runtime.
val buildInfoDir = layout.buildDirectory.dir("generated/build-info")
val writeBuildInfo by tasks.registering {
    val out = buildInfoDir.map { it.file("commit-hike-build.properties") }
    val text = "version=$version\nflavor=$flavor\n"
    inputs.property("text", text)
    outputs.file(out)
    doLast { out.get().asFile.apply { parentFile.mkdirs() }.writeText(text) }
}

sourceSets.main {
    resources.srcDir(generatedResources)
    resources.srcDir(buildInfoDir)
}

tasks.processResources {
    dependsOn(copyShared, writeBuildInfo)
}

/** The CHANGELOG.md section for a version (or Unreleased) as simple HTML for the Marketplace. */
fun changeNotesFor(v: String): String {
    val lines = rootDir.resolve("../../CHANGELOG.md").readLines()
    fun section(title: String): List<String>? {
        val start = lines.indexOfFirst { it.startsWith("## [$title]") }
        if (start < 0) return null
        val end = lines.drop(start + 1).indexOfFirst { it.startsWith("## ") }.let { if (it < 0) lines.size else start + 1 + it }
        return lines.subList(start + 1, end)
    }
    val body = section(v) ?: section("Unreleased") ?: return "See CHANGELOG.md"
    fun esc(t: String) = t.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
        .replace(Regex("`([^`]+)`"), "<code>$1</code>").replace(Regex("\\*\\*([^*]+)\\*\\*"), "<b>$1</b>")
    val html = StringBuilder()
    var inList = false
    for (raw in body) {
        val l = raw.trim()
        if (l.startsWith("- ")) {
            if (!inList) html.append("<ul>").also { inList = true }
            html.append("<li>").append(esc(l.removePrefix("- "))).append("</li>")
            continue
        }
        if (inList) html.append("</ul>").also { inList = false }
        if (l.startsWith("### ")) {
            html.append("<h3>").append(esc(l.removePrefix("### "))).append("</h3>")
        } else if (l.isNotEmpty()) {
            html.append("<p>").append(esc(l)).append("</p>")
        }
    }
    if (inList) html.append("</ul>")
    return html.toString().ifEmpty { "See CHANGELOG.md" }
}

// `./gradlew runIde` is a development build: the trail view may look ahead.
tasks.named<org.jetbrains.intellij.platform.gradle.tasks.RunIdeTask>("runIde") {
    jvmArgumentProviders += CommandLineArgumentProvider { listOf("-Dcommit-hike.dev=true") }
}

// Optional: `./gradlew runPyCharm -PpycharmPath=/path/to/pycharm` starts your
// installed PyCharm with the plugin loaded (a separate sandbox profile).
providers.gradleProperty("pycharmPath").orNull?.let { path ->
    intellijPlatformTesting.runIde.register("runPyCharm") {
        localPath.set(file(path))
        task { jvmArgumentProviders += CommandLineArgumentProvider { listOf("-Dcommit-hike.dev=true") } }
    }
}

ktlint {
    version.set("1.5.0")
    // Set explicitly: ktlint under Gradle doesn't pick up .editorconfig files
    // outside this Gradle project (the repository root one).
    additionalEditorconfig.set(
        mapOf(
            "ktlint_code_style" to "intellij_idea",
            "max_line_length" to "140",
        ),
    )
}

kover {
    reports {
        total {
            // koverLog prints coverage per class, so the gaps are visible in the console
            log {
                groupBy.set(GroupingEntityType.CLASS)
            }
        }
    }
}
