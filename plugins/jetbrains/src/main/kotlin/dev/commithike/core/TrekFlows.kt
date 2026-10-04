package dev.commithike.core

import java.io.File
import java.time.LocalDate

/** The answers of the setup dialog. */
data class SetupChoice(val emails: List<String>, val mode: String, val fromHistory: Boolean, val difficulty: String)

/** A trail picked for a journey ("none" ends a project's own journey). */
data class RouteChoice(val routeId: String, val fromHistory: Boolean)

/** What the user flows need from the IDE: dialogs, files, the clipboard, notifications. */
interface TrekUi {
    fun chooseFileToOpen(title: String, description: String, ext: String): File?

    fun chooseFileOrFolder(title: String, description: String): File?

    fun chooseFolder(title: String): File?

    fun chooseFileToSave(title: String, description: String, ext: String, defaultName: String): File?

    fun askText(prompt: String, title: String, initial: String): String?

    fun confirm(question: String, yes: String, no: String = I18n.t("cancel")): Boolean

    fun setup(detectedEmail: String): SetupChoice?

    fun pickRoute(scope: Scope, routes: List<Route>, canRemove: Boolean): RouteChoice?

    fun pickRouteToRemove(routes: List<Route>): Route?

    fun pickLanguage(info: LocaleInfo): String?

    fun pickRestDays(current: List<Int>): List<Int>?

    fun pickDifficulty(current: String): String?

    /** [title]: null = "Commit Hike", "" = none. */
    fun notify(text: String, button: String? = null, title: String? = null, action: (() -> Unit)? = null)

    fun openInEditor(file: File)

    fun copyToClipboard(text: String)

    fun open(file: File)

    fun browse(url: String)

    /** Puts a PNG picture into the system clipboard. */
    fun copyImageToClipboard(png: ByteArray)
}

/** What the user flows need from the plugin: the core, its state, and refreshing what changed. */
interface TrekHost {
    val currentRepo: String?
    val initialized: Boolean
    val routes: Collection<Route>
    val status: Status?
    val localeInfo: LocaleInfo?

    fun <T> core(block: (CoreCli) -> T): T

    fun gitEmail(): String

    fun markInitialized()

    fun refreshAll()

    fun refreshProject()

    fun scanProject(repo: String)

    fun scanAllRepositories()

    fun forgetTeam()

    fun reloadRoutes()

    fun reloadAvatar()

    fun reloadLocale(set: String?)

    fun reloadRoutesAndAvatar() {
        reloadRoutes()
        reloadAvatar()
    }

    fun showTrail()

    /** Runs another flow off the UI thread, e.g. from a notification's button. */
    fun launch(flow: TrekFlows.() -> Unit)
}

/**
 * The user flows behind the settings, backups, badges and postcards: ask, call
 * the core, refresh, tell. Kept apart from the IDE (see [TrekUi]) so they're
 * tested end to end with the real core and a scripted user.
 */
class TrekFlows(private val host: TrekHost, private val ui: TrekUi, private val build: BuildInfo = BuildInfo.current) {
    // ---------- setup and journeys ----------

    fun setup() {
        val choice = ui.setup(host.gitEmail()) ?: return
        host.core { it.init(choice.emails, choice.mode, choice.fromHistory, choice.difficulty) }
        host.markInitialized()
        host.reloadLocale(null)
        host.reloadRoutes()
        val repo = host.currentRepo
        if (choice.mode == "selected" && repo != null && ui.confirm(I18n.t("countThis"), I18n.t("countIt"), I18n.t("notNow"))) {
            host.core { it.setProjectEnabled(repo, true) }
        }
        host.scanAllRepositories()
        host.refreshAll()
        host.showTrail()
    }

    fun chooseRoute(scope: Scope) {
        val repo = host.currentRepo
        if (scope == Scope.PROJECT && repo == null) {
            ui.notify(I18n.t("openRepoForTrail"), title = "")
            return
        }
        val canRemove = scope == Scope.PROJECT && host.status?.project != null
        val choice = ui.pickRoute(scope, host.routes.sortedBy { it.name }, canRemove) ?: return
        host.core { it.setJourney(scope, choice.routeId, choice.fromHistory, if (scope == Scope.PROJECT) repo else null) }
        host.refreshProject()
    }

    fun walkRoute(id: String) {
        host.core { it.setJourney(Scope.GLOBAL, id, fromHistory = false) }
        host.refreshAll()
        host.showTrail()
    }

    fun setProjectEnabled(on: Boolean) {
        val repo = host.currentRepo ?: return ui.notify(I18n.t("openRepoFirst"), title = "")
        if (host.core { it.config() }.mode == "all") return ui.notify(I18n.t("allCounted"), title = "")
        host.core { it.setProjectEnabled(repo, on) }
        if (on) host.scanProject(repo) else host.refreshProject()
    }

    /** Recounts this project from its git history. */
    fun verify() {
        val repo = host.currentRepo ?: return
        if (!host.initialized) return
        val r = host.core { it.verify(repo) }
        host.forgetTeam()
        host.refreshAll()
        val changes = r.added + r.updated + r.removed
        ui.notify(
            if (changes ==
                0
            ) {
                I18n.t("recountSame")
            } else {
                I18n.t("recountDone", r.added, r.updated, r.removed)
            },
            title = I18n.t("recountTitle"),
        )
    }

    // ---------- custom routes ----------

    /** Imports a route pack from a folder or a .zip; replacing one asks first. */
    fun importRoute() {
        val file = ui.chooseFileOrFolder(I18n.t("importTitle"), I18n.t("importDesc")) ?: return
        val route = try {
            host.core { it.importRoute(file.path, replace = false) }
        } catch (e: CoreException) {
            if (e.code != "route_exists") throw e
            if (!ui.confirm(I18n.t("replaceQ", e.message ?: ""), I18n.t("replace"))) return
            host.core { it.importRoute(file.path, replace = true) }
        }
        host.reloadRoutes()
        ui.notify(
            I18n.t("importedText", route.name, I18n.distance(route.lengthM), route.waypoints.orEmpty().size),
            I18n.t("walkNow"),
            I18n.t("imported"),
        ) { host.launch { walkRoute(route.id) } }
    }

    /** Writes a template route pack the user can edit and then import. */
    fun createRouteTemplate() {
        val folder = ui.chooseFolder(I18n.t("whereCreate")) ?: return
        val id = ui.askText(I18n.t("routeIdPrompt"), I18n.t("newRoute"), "my-trail")?.trim()?.takeIf { it.isNotEmpty() } ?: return
        val dir = host.core { it.routeTemplate(id, folder.path) }
        ui.openInEditor(File(dir, "route.json"))
        ui.notify(I18n.t("templateText"), I18n.t("importBtn"), I18n.t("templateCreated")) { host.launch { importRoute() } }
    }

    /** Removes one of the user's imported routes. */
    fun removeRoute() {
        val custom = host.routes.filter { !it.builtin }.sortedBy { it.name }
        if (custom.isEmpty()) return ui.notify(I18n.t("noCustom"), title = "")
        val route = ui.pickRouteToRemove(custom) ?: return
        host.core { it.removeRoute(route.id) }
        host.reloadRoutes()
        ui.notify(I18n.t("removed", route.name), title = "")
    }

    // ---------- hiker, language, team ----------

    /** A PNG, ideally with a transparent background, as the hiker. */
    fun setHikerIcon() {
        val file = ui.chooseFileToOpen(I18n.t("iconTitle"), I18n.t("iconDesc"), "png") ?: return
        host.core { it.setAvatar(file.path) }
        host.reloadAvatar()
        host.refreshAll()
    }

    fun resetHikerIcon() {
        host.core { it.resetAvatar() }
        host.reloadAvatar()
        host.refreshAll()
    }

    fun changeLanguage() {
        val info = host.localeInfo ?: host.core { it.locale() }
        setLocale(ui.pickLanguage(info) ?: return)
    }

    /** "auto" follows the IDE; anything else fixes the language for every IDE. */
    fun setLocale(value: String) {
        host.reloadLocale(value)
        host.reloadRoutes() // route names come translated
        host.refreshAll()
    }

    fun setTeam(on: Boolean) {
        val repo = host.currentRepo
        if (!host.initialized || repo == null) return ui.notify(I18n.t("teamNeedsRepo"), title = "")
        host.core { it.setTeam(repo, on) }
        host.forgetTeam()
        host.refreshProject()
        ui.notify(if (on) I18n.t("teamOn") else I18n.t("teamOff"), title = "")
    }

    /** The team goal from the Team tab: [route] is checked (see PanelCommand), "" removes it. */
    fun setTeamGoal(route: String) {
        val repo = host.currentRepo ?: return
        if (!host.initialized) return
        host.core { it.setTeamGoal(repo, route) }
        host.forgetTeam()
        host.refreshProject()
    }

    // ---------- days off, settings, difficulty ----------

    fun setRestDays() {
        val current = host.core { it.restDays() }.days
        setRestDaysTo(ui.pickRestDays(current) ?: return)
    }

    /** Days off from the settings page: 0 = Sunday … 6 = Saturday. */
    fun setRestDaysTo(days: List<Int>) {
        host.core { it.restDays(RestDays.arg(days)) }
        host.refreshAll()
    }

    fun setSettings(reduceMotion: String?, highContrast: String?, notifications: String?) {
        host.core { it.settings(reduceMotion, highContrast, notifications) }
        host.refreshAll()
    }

    fun changeDifficulty() {
        val current = host.core { it.difficulty() }.level
        setDifficulty(ui.pickDifficulty(current) ?: return)
    }

    fun setDifficulty(level: String) {
        host.core { it.difficulty(level) }
        host.refreshAll()
    }

    fun exportProgress(today: LocalDate = LocalDate.now()) {
        val target =
            ui.chooseFileToSave(I18n.t("backupSaveTitle"), I18n.t("backupSaveDesc"), "json", "commit-hike-backup-$today.json") ?: return
        val res = host.core { it.exportBackup(target.path) }
        ui.notify(I18n.t("backupSaved", res.path))
    }

    /** Restoring over existing progress asks first. */
    fun importProgress() {
        val file = ui.chooseFileToOpen(I18n.t("backupOpenTitle"), I18n.t("backupOpenDesc"), "json") ?: return
        val res = try {
            host.core { it.importBackup(file.path, replace = false) }
        } catch (e: CoreException) {
            if (e.code != "data_exists") throw e
            if (!ui.confirm(I18n.t("backupReplaceQ"), I18n.t("replace"))) return
            host.core { it.importBackup(file.path, replace = true) }
        }
        host.reloadRoutesAndAvatar()
        host.refreshAll()
        ui.notify(I18n.t("backupRestored", res.commits))
    }

    fun saveBadge() {
        val badge = host.core { it.badge(host.currentRepo) }
        val target = ui.chooseFileToSave(I18n.t("badgeSaveTitle"), I18n.t("badgeSaveDesc"), "svg", badge.fileName) ?: return
        target.writeText(badge.svg)
        val markdown = badge.markdown.replace(badge.fileName, target.name)
        ui.notify(I18n.t("badgeSaved", target.path), I18n.t("copyMarkdown")) { ui.copyToClipboard(markdown) }
    }

    /** [png] is already checked (a real PNG), [fileName] already safe: see [PanelCommand]. */
    fun savePostcard(fileName: String, png: ByteArray) {
        val target = ui.chooseFileToSave(I18n.t("postcardSaveTitle"), I18n.t("postcardSaveDesc"), "png", fileName) ?: return
        target.writeBytes(png)
        ui.notify(I18n.t("postcardSaved", target.path), I18n.t("openBtn")) { ui.open(target) }
    }

    /** Sharing: the postcard into the clipboard, ready to paste into a post. [png] is already checked. */
    fun copyPostcard(png: ByteArray) {
        ui.copyImageToClipboard(png)
        ui.notify(I18n.t("postcardCopied"))
    }

    /** "Copy a report for the developer": versions, settings and recent errors, nothing personal. */
    fun copyDiagnostics(ide: String, os: String, errors: List<String>, home: String?) {
        val core = try {
            host.core { it.diagnosticsJson() }
        } catch (e: CoreException) {
            "{ \"error\": \"${e.message}\" }"
        }
        ui.copyToClipboard(Report.build("${build.version} (${build.flavor}), JetBrains plugin", ide, os, core, errors, home))
        val issues = build.repo.takeIf { it.isNotBlank() }?.let { "$it/issues" }
        if (issues !=
            null
        ) {
            ui.notify(I18n.t("reportCopied"), I18n.t("openIssues")) { ui.browse(issues) }
        } else {
            ui.notify(I18n.t("reportCopied"))
        }
    }
}
