package dev.commithike

import com.intellij.ide.BrowserUtil
import com.intellij.ide.util.PropertiesComponent
import com.intellij.notification.NotificationAction
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.application.EDT
import com.intellij.openapi.components.Service
import com.intellij.openapi.diagnostic.logger
import com.intellij.openapi.fileChooser.FileChooser
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.fileEditor.FileEditorManager
import com.intellij.openapi.fileEditor.FileEditorManagerEvent
import com.intellij.openapi.fileEditor.FileEditorManagerListener
import com.intellij.openapi.project.Project
import com.intellij.openapi.startup.ProjectActivity
import com.intellij.openapi.ui.Messages
import com.intellij.openapi.util.IconLoader
import com.intellij.openapi.vfs.LocalFileSystem
import com.intellij.openapi.wm.ToolWindowManager
import com.intellij.util.messages.Topic
import dev.commithike.core.CoreException
import dev.commithike.core.I18n
import dev.commithike.core.RatingPrompt
import dev.commithike.core.ScanResult
import dev.commithike.core.Scope
import dev.commithike.core.Status
import dev.commithike.core.Team
import dev.commithike.core.gitGlobalEmail
import dev.commithike.ui.DifficultyDialog
import dev.commithike.ui.LanguageDialog
import dev.commithike.ui.RemoveRouteDialog
import dev.commithike.ui.RouteDialog
import dev.commithike.ui.SetupDialog
import git4idea.repo.GitRepository
import git4idea.repo.GitRepositoryChangeListener
import git4idea.repo.GitRepositoryManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.atomic.AtomicBoolean

private val LOG = logger<ProjectTrek>()

object CommitHikeIcons {
    @JvmField val Trail = IconLoader.getIcon("/icons/commit-hike.svg", CommitHikeIcons::class.java)
}

/** Published on the project bus (on the EDT) whenever [ProjectTrek.view] changes. */
fun interface TrailListener {
    fun trailChanged()

    companion object {
        @Topic.ProjectLevel
        val TOPIC = Topic(TrailListener::class.java, Topic.BroadcastDirection.NONE)
    }
}

/** What the status bar widget and the tool window render. */
data class TrailView(
    val state: String, // loading | not_initialized | ok | error
    val error: String? = null,
    val repo: String? = null,
    val status: Status? = null,
    val flash: String? = null, // "+86 m", shown briefly after a commit
    val team: Team? = null, // teammates, when the user turned them on for this project
    val teamError: String? = null,
)

class CommitHikeStartup : ProjectActivity {
    override suspend fun execute(project: Project) {
        project.getService(ProjectTrek::class.java).start()
    }
}

/**
 * Per project: detects commits through Git4Idea (HEAD changes), asks the core
 * to count them, and keeps the view state. Never writes to the repository.
 */
@Service(Service.Level.PROJECT)
class ProjectTrek(private val project: Project, private val cs: CoroutineScope) {
    @Volatile var view = TrailView("loading")
        private set

    private val app get() = CommitHikeApp.getInstance()
    private val started = AtomicBoolean(false)
    private val heads = ConcurrentHashMap<String, String>()
    private val pending = ConcurrentHashMap<String, Job>()

    @Volatile private var currentRepo: String? = null

    @Volatile private var flash: String? = null
    private var flashJob: Job? = null

    // Teammates are counted from the whole history: cached until the next scan or for five minutes.
    private data class TeamCache(val repo: String, val team: Team?, val error: String?, val at: Long)

    @Volatile private var teamCache: TeamCache? = null
    private val teamLoading = AtomicBoolean(false)

    fun start() {
        if (!started.compareAndSet(false, true)) return
        val bus = project.messageBus.connect(cs)
        bus.subscribe(GitRepository.GIT_REPO_CHANGE, GitRepositoryChangeListener { onRepoChanged(it) })
        bus.subscribe(
            FileEditorManagerListener.FILE_EDITOR_MANAGER,
            object : FileEditorManagerListener {
                override fun selectionChanged(event: FileEditorManagerEvent) {
                    cs.launch { followActiveRepo() }
                }
            },
        )
        cs.launch {
            try {
                app.ensureLoaded()
            } catch (e: Exception) {
                LOG.warn("Commit Hike failed to start", e)
                publish(TrailView("error", error = e.message))
                return@launch
            }
            repositories().forEach { onRepoChanged(it) }
            followActiveRepo(force = true)
            if (!app.initialized) offerSetupOnce()
        }
    }

    fun requestRefresh() {
        cs.launch { refresh() }
    }

    // ---------- commit detection ----------

    private fun repositories(): List<GitRepository> = GitRepositoryManager.getInstance(project).repositories

    private fun onRepoChanged(repo: GitRepository) {
        val root = repo.root.path
        val head = repo.currentRevision ?: ""
        val previous = heads.put(root, head)
        if (previous != head) schedule(root) // first sighting also scans: catches commits made while the IDE was closed
    }

    // git touches several files per commit; coalesce into one scan.
    private fun schedule(root: String) {
        pending.remove(root)?.cancel()
        pending[root] = cs.launch {
            delay(800)
            pending.remove(root)
            scan(root)
        }
    }

    private suspend fun scan(root: String) {
        if (!app.loaded || !app.initialized) return
        // The HEAD seen at the previous scan survives restarts, so a rebase done
        // while the IDE was closed is still recognised as a rewrite.
        val props = PropertiesComponent.getInstance(project)
        val key = "$HEAD_KEY$root"
        val head = heads[root].orEmpty()
        val prev = props.getValue(key)?.takeIf { it != head }
        try {
            celebrate(app.call { it.scan(root, prev) })
            if (head.isNotEmpty()) props.setValue(key, head)
            if (teamCache?.repo == root) teamCache = null // teammates may have moved too
        } catch (e: Exception) {
            LOG.warn("Commit Hike scan failed for $root", e)
        }
        app.refreshAllProjects()
    }

    private suspend fun followActiveRepo(force: Boolean = false) {
        val file = withContext(Dispatchers.EDT) { FileEditorManager.getInstance(project).selectedFiles.firstOrNull() }
        val manager = GitRepositoryManager.getInstance(project)
        val roots = manager.repositories.map { it.root.path }
        val repo = file?.let { manager.getRepositoryForFileQuick(it)?.root?.path }
            ?: currentRepo?.takeIf { it in roots }
            ?: roots.firstOrNull()
        if (force || repo != currentRepo) {
            currentRepo = repo
            refresh()
        }
    }

    // ---------- state ----------

    private suspend fun refresh() {
        if (!app.loaded) return
        val next = if (!app.initialized) {
            TrailView("not_initialized")
        } else {
            try {
                val status = app.call { it.status(currentRepo) }
                app.ensureAssets(status)
                withTeam(TrailView("ok", repo = currentRepo, status = status))
            } catch (e: CoreException) {
                if (e.notInitialized) {
                    app.markInitialized(false)
                    TrailView("not_initialized")
                } else {
                    TrailView("error", error = e.message)
                }
            }
        }
        publish(next)
        val repo = currentRepo
        if (next.status?.team == true && repo != null) loadTeam(repo)
    }

    private fun withTeam(v: TrailView): TrailView {
        val c = teamCache?.takeIf { it.repo == v.repo && v.status?.team == true } ?: return v
        return v.copy(team = c.team, teamError = c.error)
    }

    private fun loadTeam(repo: String, force: Boolean = false) {
        val c = teamCache
        if (!force && c != null && c.repo == repo && System.currentTimeMillis() - c.at < 5 * 60_000) return
        if (!teamLoading.compareAndSet(false, true)) return
        cs.launch {
            try {
                teamCache = try {
                    TeamCache(repo, app.call { it.team(repo) }, null, System.currentTimeMillis())
                } catch (e: CoreException) {
                    TeamCache(repo, null, e.message, System.currentTimeMillis())
                }
            } finally {
                teamLoading.set(false)
            }
            if (repo == currentRepo) publish(withTeam(view))
        }
    }

    /** The panel asks for teammates when the Team tab opens. */
    fun requestTeam() {
        val repo = currentRepo ?: return
        if (view.status?.team == true) loadTeam(repo)
    }

    private suspend fun publish(next: TrailView) {
        view = next.copy(flash = flash)
        withContext(Dispatchers.EDT) {
            if (!project.isDisposed) project.messageBus.syncPublisher(TrailListener.TOPIC).trailChanged()
        }
    }

    private suspend fun celebrate(res: ScanResult) {
        if (res.addedM > 0) {
            flash = "+" + I18n.distance(res.addedM)
            flashJob?.cancel()
            flashJob = cs.launch {
                delay(4000)
                flash = null
                publish(view)
            }
        }
        if (res.rewritten && res.removedCommits > 0) LOG.info(I18n.t("rewritten"))
        if (res.addedM > 0) maybeAskForRating()
        val events = res.events.orEmpty()
        // An encounter on the road is announced when it's the only news of this commit.
        events.lastOrNull { it.type == "danger" }?.danger?.let { d ->
            if (events.size <= 3) notify("⚠", d.text, I18n.t("showTrail")) { showTrail() }
        }
        // Achievements always get a notification of their own.
        for (e in events.filter { it.type == "achievement" }) {
            val a = e.achievement ?: continue
            notify("🏅 ${a.name ?: ""}", a.description ?: "", I18n.t("showTrail")) { showTrail() }
        }
        // Importing history can pass many stops at once: announce only the latest per journey.
        for (scope in Scope.entries) {
            val mine = events.filter { it.journey == scope.cli }
            val journey = if (scope == Scope.GLOBAL) res.global else res.project
            if (journey != null && mine.any { it.type == "finished" }) {
                notify(I18n.t("finished", journey.route.name), I18n.t("finishedText"), I18n.t("chooseTrail")) { chooseRoute(scope) }
            } else {
                val w = mine.lastOrNull { it.type == "waypoint" }?.waypoint ?: continue
                notify(w.name, w.text ?: I18n.t("reached", w.name), I18n.t("showTrail")) { showTrail() }
            }
        }
    }

    // ---------- user flows ----------

    fun showTrail() {
        cs.launch(Dispatchers.EDT) { ToolWindowManager.getInstance(project).getToolWindow(TOOL_WINDOW_ID)?.show() }
    }

    fun setup() = guarded {
        val email = withContext(Dispatchers.IO) { gitGlobalEmail() }
        val result = withContext(Dispatchers.EDT) {
            val dialog = SetupDialog(project, email)
            if (dialog.showAndGet()) dialog.result() else null
        } ?: return@guarded

        app.call { it.init(result.emails, result.mode, result.fromHistory, result.difficulty) }
        app.markInitialized(true)
        app.reloadLocale()
        app.reloadRoutes()
        val repo = currentRepo
        if (result.mode == "selected" && repo != null) {
            val count = withContext(Dispatchers.EDT) {
                Messages.showYesNoDialog(project, I18n.t("countThis"), "Commit Hike", I18n.t("countIt"), I18n.t("notNow"), null) ==
                    Messages.YES
            }
            if (count) app.call { it.setProjectEnabled(repo, true) }
        }
        for (r in repositories()) app.call { it.scan(r.root.path) }
        app.refreshAllProjects()
        showTrail()
    }

    fun chooseRoute(scope: Scope) = guarded {
        if (!app.initialized) {
            setup()
            return@guarded
        }
        val repo = currentRepo
        if (scope == Scope.PROJECT && repo == null) {
            notify("", I18n.t("openRepoForTrail"))
            return@guarded
        }
        val canRemove = scope == Scope.PROJECT && view.status?.project != null
        val choice = withContext(Dispatchers.EDT) {
            val dialog = RouteDialog(
                project,
                scope,
                app.routes.values.sortedBy { it.name },
                canRemove,
                onImport = { importRoute() },
                onTemplate = { createRouteTemplate() },
            )
            if (dialog.showAndGet()) dialog.result() else null
        } ?: return@guarded
        app.call { it.setJourney(scope, choice.routeId, choice.fromHistory, if (scope == Scope.PROJECT) repo else null) }
        refresh()
    }

    fun setProjectEnabled(on: Boolean) = guarded {
        if (!app.initialized) {
            setup()
            return@guarded
        }
        val repo = currentRepo ?: run {
            notify("", I18n.t("openRepoFirst"))
            return@guarded
        }
        if (app.call { it.config() }.mode == "all") {
            notify("", I18n.t("allCounted"))
            return@guarded
        }
        app.call { it.setProjectEnabled(repo, on) }
        if (on) scan(repo) else refresh()
    }

    fun verify() = guarded {
        val repo = currentRepo ?: return@guarded
        if (!app.initialized) return@guarded
        val r = app.call { it.verify(repo) }
        teamCache = null
        app.refreshAllProjects()
        val changes = r.added + r.updated + r.removed
        notify(
            I18n.t("recountTitle"),
            if (changes == 0) I18n.t("recountSame") else I18n.t("recountDone", r.added, r.updated, r.removed),
        )
    }

    // ---------- custom routes ----------

    /** Imports a route pack from a folder or a .zip chosen by the user. */
    fun importRoute() = guarded {
        val file = withContext(Dispatchers.EDT) {
            val descriptor = FileChooserDescriptorFactory.createSingleFileOrFolderDescriptor()
                .withTitle(I18n.t("importTitle"))
                .withDescription(I18n.t("importDesc"))
            FileChooser.chooseFile(descriptor, project, null)
        } ?: return@guarded
        val route = try {
            app.call { it.importRoute(file.path, replace = false) }
        } catch (e: CoreException) {
            if (e.code != "route_exists") throw e
            val replace = withContext(Dispatchers.EDT) {
                Messages.showYesNoDialog(
                    project,
                    I18n.t("replaceQ", e.message ?: ""),
                    "Commit Hike",
                    I18n.t("replace"),
                    I18n.t("cancel"),
                    null,
                ) ==
                    Messages.YES
            }
            if (!replace) return@guarded
            app.call { it.importRoute(file.path, replace = true) }
        }
        app.reloadRoutes()
        notify(
            I18n.t("imported"),
            I18n.t("importedText", route.name, I18n.distance(route.lengthM), route.waypoints.orEmpty().size),
            I18n.t("walkNow"),
        ) { walkRoute(route.id) }
    }

    /** Writes a template route pack the user can edit and then import. */
    fun createRouteTemplate() = guarded {
        val target = withContext(Dispatchers.EDT) {
            val descriptor = FileChooserDescriptorFactory.createSingleFolderDescriptor()
                .withTitle(I18n.t("whereCreate"))
            val folder = FileChooser.chooseFile(descriptor, project, null) ?: return@withContext null
            val id = Messages.showInputDialog(
                project,
                I18n.t("routeIdPrompt"),
                I18n.t("newRoute"),
                null,
                "my-trail",
                null,
            ) ?: return@withContext null
            folder.path to id.trim()
        } ?: return@guarded
        val dir = app.call { it.routeTemplate(target.second, target.first) }
        withContext(Dispatchers.EDT) {
            LocalFileSystem.getInstance().refreshAndFindFileByPath("$dir/route.json")
                ?.let { FileEditorManager.getInstance(project).openFile(it, true) }
        }
        notify(I18n.t("templateCreated"), I18n.t("templateText"), I18n.t("importBtn")) { importRoute() }
    }

    /** Removes one of the user's imported routes. */
    fun removeRoute() = guarded {
        val custom = app.routes.values.filter { !it.builtin }.sortedBy { it.name }
        if (custom.isEmpty()) {
            notify("", I18n.t("noCustom"))
            return@guarded
        }
        val route = withContext(Dispatchers.EDT) {
            val dialog = RemoveRouteDialog(project, custom)
            if (dialog.showAndGet()) dialog.selected else null
        } ?: return@guarded
        app.call { it.removeRoute(route.id) }
        app.reloadRoutes()
        notify("", I18n.t("removed", route.name))
    }

    // ---------- hiker icon ----------

    /** Lets the user pick a PNG (ideally with a transparent background) as the hiker. */
    fun setHikerIcon() = guarded {
        val file = withContext(Dispatchers.EDT) {
            val descriptor = FileChooserDescriptorFactory.createSingleFileDescriptor("png")
                .withTitle(I18n.t("iconTitle"))
                .withDescription(I18n.t("iconDesc"))
            FileChooser.chooseFile(descriptor, project, null)
        } ?: return@guarded
        app.call { it.setAvatar(file.path) }
        app.reloadAvatar()
        app.refreshAllProjects()
    }

    fun resetHikerIcon() = guarded {
        app.call { it.resetAvatar() }
        app.reloadAvatar()
        app.refreshAllProjects()
    }

    // ---------- language & team ----------

    fun changeLanguage() = guarded {
        if (!app.initialized) {
            setup()
            return@guarded
        }
        val info = app.localeInfo ?: app.call { it.locale() }
        val choice = withContext(Dispatchers.EDT) {
            val dialog = LanguageDialog(project, info)
            if (dialog.showAndGet()) dialog.selected else null
        } ?: return@guarded
        setLocale(choice)
    }

    /** "auto" follows the IDE; anything else fixes the language for every IDE. */
    fun setLocale(value: String) = guarded {
        app.reloadLocale(value)
        app.reloadRoutes() // route names come translated
        app.refreshAllProjects()
    }

    fun setTeam(on: Boolean) = guarded {
        val repo = currentRepo
        if (!app.initialized || repo == null) {
            notify("", I18n.t("teamNeedsRepo"))
            return@guarded
        }
        app.call { it.setTeam(repo, on) }
        teamCache = null
        refresh()
        notify("", if (on) I18n.t("teamOn") else I18n.t("teamOff"))
    }

    fun toggleTeam() = setTeam(view.status?.team != true)

    fun changeDifficulty() = guarded {
        if (!app.initialized) {
            setup()
            return@guarded
        }
        val current = app.call { it.difficulty() }.level
        val choice = withContext(Dispatchers.EDT) {
            val dialog = DifficultyDialog(project, current)
            if (dialog.showAndGet()) dialog.selected else null
        } ?: return@guarded
        setDifficulty(choice)
    }

    fun setDifficulty(level: String) = guarded {
        app.call { it.difficulty(level) }
        app.refreshAllProjects()
    }

    private fun walkRoute(id: String) = guarded {
        app.call { it.setJourney(Scope.GLOBAL, id, fromHistory = false) }
        app.refreshAllProjects()
        showTrail()
    }

    /**
     * After a commit that moved the user along: once they've used Commit Hike for
     * a while, ask for a rating (see RatingPrompt for when and how often).
     * Stored app-wide, so several open projects don't ask separately.
     */
    private fun maybeAskForRating() {
        val props = PropertiesComponent.getInstance()
        val now = System.currentTimeMillis()
        var st = RatingPrompt.recordProgress(RatingPrompt.load(props.getValue(RATING_KEY), now), now)
        if (!RatingPrompt.shouldAsk(st, now)) {
            props.setValue(RATING_KEY, RatingPrompt.save(st))
            return
        }
        st = RatingPrompt.asked(st, now) // closing the balloon without an answer counts as "later"
        props.setValue(RATING_KEY, RatingPrompt.save(st))
        val asked = st
        cs.launch {
            delay(4000) // don't pile onto the distance and waypoint notifications
            val answer = { a: String -> props.setValue(RATING_KEY, RatingPrompt.save(RatingPrompt.answered(asked, a))) }
            NotificationGroupManager.getInstance().getNotificationGroup(NOTIFICATION_GROUP)
                .createNotification("Commit Hike", I18n.t("rateAsk"), NotificationType.INFORMATION)
                .addAction(
                    NotificationAction.createSimpleExpiring(I18n.t("rateYes")) {
                        answer("rate")
                        BrowserUtil.browse(RatingPrompt.REVIEWS_URL)
                    },
                )
                .addAction(NotificationAction.createSimpleExpiring(I18n.t("rateLater")) { answer("later") })
                .addAction(NotificationAction.createSimpleExpiring(I18n.t("rateNever")) { answer("never") })
                .notify(project)
        }
    }

    private suspend fun offerSetupOnce() {
        val props = PropertiesComponent.getInstance()
        if (props.getBoolean(SETUP_OFFERED_KEY)) return
        props.setValue(SETUP_OFFERED_KEY, true)
        notify(
            "Commit Hike",
            I18n.t("offer"),
            I18n.t("setUp"),
        ) { setup() }
    }

    // ---------- helpers ----------

    private fun guarded(block: suspend CoroutineScope.() -> Unit): Job = cs.launch {
        try {
            block()
        } catch (e: Exception) {
            if (e is kotlinx.coroutines.CancellationException) throw e
            LOG.warn("Commit Hike action failed", e)
            notify("Commit Hike", e.message ?: e.toString(), type = NotificationType.ERROR)
        }
    }

    private fun notify(
        title: String,
        content: String,
        actionText: String? = null,
        type: NotificationType = NotificationType.INFORMATION,
        action: (() -> Unit)? = null,
    ) {
        val n = NotificationGroupManager.getInstance().getNotificationGroup(NOTIFICATION_GROUP)
            .createNotification(title, content, type)
        if (actionText != null && action != null) n.addAction(NotificationAction.createSimpleExpiring(actionText) { action() })
        n.notify(project)
    }

    companion object {
        const val TOOL_WINDOW_ID = "Commit Hike"
        const val NOTIFICATION_GROUP = "Commit Hike"
        private const val SETUP_OFFERED_KEY = "commit-hike.setupOffered"
        private const val RATING_KEY = "commit-hike.rating"
        private const val HEAD_KEY = "commit-hike.head:"
    }
}
