import org.jetbrains.intellij.platform.gradle.TestFrameworkType

plugins {
    id("java")
    id("org.jetbrains.kotlin.jvm") version "2.0.21"
    id("org.jetbrains.intellij.platform") version "2.5.0"
    id("org.jlleitschuh.gradle.ktlint") version "12.1.2" // `./gradlew ktlintCheck` / `ktlintFormat`
}

group = "dev.commithike"
version = "0.1.0"

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
        intellijIdeaCommunity("2024.3.5")
        bundledPlugin("Git4Idea")
        testFramework(TestFrameworkType.Platform)
    }
    testImplementation("junit:junit:4.13.2")
}

intellijPlatform {
    pluginConfiguration {
        ideaVersion {
            sinceBuild = "243"
            untilBuild = provider { null }
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

sourceSets.main {
    resources.srcDir(generatedResources)
}

tasks.processResources {
    dependsOn(copyShared)
}

// Optional: `./gradlew runPyCharm -PpycharmPath=/path/to/pycharm` starts your
// installed PyCharm with the plugin loaded (a separate sandbox profile).
providers.gradleProperty("pycharmPath").orNull?.let { path ->
    intellijPlatformTesting.runIde.register("runPyCharm") {
        localPath.set(file(path))
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
