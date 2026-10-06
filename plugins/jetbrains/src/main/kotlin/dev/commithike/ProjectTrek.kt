package dev.commithike

import com.intellij.ide.BrowserUtil
import com.intellij.ide.util.PropertiesComponent
import com.intellij.notification.NotificationAction
import com.intellij.notification.NotificationGroupManager
import com.intellij.notification.NotificationType
import com.intellij.openapi.application.EDT
import com.intellij.openapi.components.Service
import com.intellij.openapi.diagnostic.logger
import com.intellij.openapi.fileEditor.FileEditorManager
import com.intellij.openapi.fileEditor.FileEditorManagerEvent
import com.intellij.openapi.fileEditor.FileEditorManagerListener
import com.intellij.openapi.ide.CopyPasteManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.startup.ProjectActivity
import com.intellij.openapi.util.IconLoader
import com.intellij.openapi.util.SystemInfo
import com.intellij.openapi.wm.ToolWindowManager
import com.intellij.util.messages.Topic
import dev.commithike.core.Celebration
import dev.commithike.core.CoreCli
import dev.commithike.core.CoreException
import dev.commithike.core.I18n
import dev.commithike.core.LocaleInfo
import dev.commithike.core.RatingPrompt
import dev.commithike.core.Route
import dev.commithike.core.ScanResult
import dev.commithike.core.Scope
import dev.commithike.core.Status
import dev.commithike.core.Team
import dev.commithike.core.TrailView
import dev.commithike.core.TrekFlows
import dev.commithike.core.TrekHost
import dev.commithike.core.gitGlobalEmail
import dev.commithike.ui.IdeTrekUi
import git4idea.repo.GitRepository
import git4idea.repo.GitRepositoryChangeListener
import git4idea.repo.GitRepositoryManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
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

    // A page the panel should open (once per token), e.g. the settings page.
    @Volatile var openView: String? = null

    @Volatile var openToken = 0
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
        // all | milestones (stops, achievements, the finish) | off: chosen on the settings page
        val plan = Celebration.plan(res, view.status?.settings?.notifications ?: "all")
        plan.flash?.let { text ->
            flash = text
            flashJob?.cancel()
            flashJob = cs.launch {
                delay(4000)
                flash = null
                publish(view)
            }
        }
        if (plan.rewritten) LOG.info(I18n.t("rewritten"))
        if (plan.askForRating) maybeAskForRating()
        for (n in plan.notes) {
            notify(n.title, n.text, n.button) {
                when (n.action) {
                    Celebration.Action.SHOW_TRAIL -> showTrail()
                    Celebration.Action.CHOOSE_GLOBAL_ROUTE -> chooseRoute(Scope.GLOBAL)
                    Celebration.Action.CHOOSE_PROJECT_ROUTE -> chooseRoute(Scope.PROJECT)
                    Celebration.Action.WALK_ROUTE -> n.routeId?.let { walkRoute(it) }
                }
            }
        }
    }

    // ---------- user flows ----------

    fun showTrail() {
        cs.launch(Dispatchers.EDT) { ToolWindowManager.getInstance(project).getToolWindow(TOOL_WINDOW_ID)?.show() }
    }

    fun toggleTeam() = setTeam(view.status?.team != true)

    /** Saves an SVG badge with the trail, for a README such as a GitHub profile. */
    // ---------- settings, backups, badge, postcard, report: see TrekFlows ----------

    private val flows: TrekFlows by lazy {
        TrekFlows(
            host = object : TrekHost {
                override val currentRepo: String? get() = this@ProjectTrek.currentRepo
                override val initialized: Boolean get() = app.initialized
                override val routes: Collection<Route> get() = app.routes.values
                override val status: Status? get() = view.status
                override val localeInfo: LocaleInfo? get() = app.localeInfo

                override fun <T> core(block: (CoreCli) -> T): T = runBlocking { app.call(block) }

                override fun gitEmail() = gitGlobalEmail()

                override fun markInitialized() = app.markInitialized(true)

                override fun refreshAll() = app.refreshAllProjects()

                override fun refreshProject() = runBlocking { refresh() }

                override fun scanProject(repo: String) = runBlocking { scan(repo) }

                override fun scanAllRepositories() {
                    for (r in repositories()) core { it.scan(r.root.path) }
                }

                override fun forgetTeam() {
                    teamCache = null
                }

                override fun reloadRoutes() = runBlocking { app.reloadRoutes() }

                override fun reloadAvatar() = runBlocking { app.reloadAvatar() }

                override fun reloadLocale(set: String?) = runBlocking { app.reloadLocale(set) }

                override fun showTrail() = this@ProjectTrek.showTrail()

                override fun launch(flow: TrekFlows.() -> Unit) {
                    runFlow(block = flow)
                }
            },
            ui = IdeTrekUi(
                project,
                notifier = { title, text, button, action -> notify(title, text, button, action = action) },
                onImportRoute = { importRoute() },
                onFindRoutes = { openSite() },
                onCreateTemplate = { createRouteTemplate() },
            ),
        )
    }

    /** Runs a flow off the UI thread (its dialogs go to the EDT themselves); [needsSetup]: set up first. */
    private fun runFlow(needsSetup: Boolean = false, block: TrekFlows.() -> Unit): Job = guarded {
        if (needsSetup && !app.initialized) {
            setup()
            return@guarded
        }
        withContext(Dispatchers.IO) { flows.block() }
    }

    fun saveBadge(): Job = runFlow { saveBadge() }

    fun savePostcard(fileName: String, png: ByteArray): Job = runFlow { savePostcard(fileName, png) }

    fun copyPostcard(png: ByteArray): Job = runFlow { copyPostcard(png) }

    /** [url] is already checked by ShareLinks (see PanelCommand). */
    fun openShareUrl(url: String) = BrowserUtil.browse(url)

    fun copyText(text: String) = CopyPasteManager.getInstance().setContents(java.awt.datatransfer.StringSelection(text))

    fun copyDiagnostics(): Job = runFlow {
        val info = com.intellij.openapi.application.ApplicationInfo.getInstance()
        copyDiagnostics(
            ide = "${info.fullApplicationName} (${info.build.asString()})",
            os = "${SystemInfo.OS_NAME} ${SystemInfo.OS_VERSION} ${SystemInfo.OS_ARCH}",
            errors = app.recentErrors(),
            home = System.getProperty("user.home"),
        )
    }

    fun setRestDays(): Job = runFlow(needsSetup = true) { setRestDays() }

    /** Days off from the settings page: 0 = Sunday … 6 = Saturday. */
    fun setRestDaysTo(days: List<Int>): Job = runFlow { setRestDaysTo(days) }

    fun setSettings(reduceMotion: String?, highContrast: String?, notifications: String?): Job = runFlow {
        setSettings(reduceMotion, highContrast, notifications)
    }

    fun exportProgress(): Job = runFlow(needsSetup = true) { exportProgress() }

    fun importProgress(): Job = runFlow { importProgress() }

    fun changeDifficulty(): Job = runFlow(needsSetup = true) { changeDifficulty() }

    fun setDifficulty(level: String): Job = runFlow { setDifficulty(level) }

    fun setup(): Job = runFlow { setup() }

    fun chooseRoute(scope: Scope): Job = runFlow(needsSetup = true) { chooseRoute(scope) }

    fun setProjectEnabled(on: Boolean): Job = runFlow(needsSetup = true) { setProjectEnabled(on) }

    fun verify(): Job = runFlow { verify() }

    fun importRoute(): Job = runFlow { importRoute() }

    fun createRouteTemplate(): Job = runFlow { createRouteTemplate() }

    fun removeRoute(): Job = runFlow { removeRoute() }

    fun setHikerIcon(): Job = runFlow { setHikerIcon() }

    fun resetHikerIcon(): Job = runFlow { resetHikerIcon() }

    fun changeLanguage(): Job = runFlow(needsSetup = true) { changeLanguage() }

    fun setLocale(value: String): Job = runFlow { setLocale(value) }

    fun setTeam(on: Boolean): Job = runFlow { setTeam(on) }

    fun setTeamGoal(route: String): Job = runFlow { setTeamGoal(route) }

    /** Walks on to [id], usually the next route of a series. */
    fun walkRoute(id: String): Job = runFlow { walkRoute(id) }

    /** The panel's site view: run the flow off the UI thread and hand its answer to [reply]. */
    fun siteSearch(args: List<String>, reply: (String) -> Unit): Job = guarded { reply(flows.siteSearch(args)) }

    fun siteInstall(id: String, replaceLocal: Boolean, reply: (String) -> Unit): Job = guarded {
        reply(flows.siteInstall(id, replaceLocal))
    }

    /** Tools → Commit Hike → Routes from the Site…: the tool window on its site page. */
    fun openSite() {
        openView = "site"
        openToken++
        showTrail()
        cs.launch { publish(view) }
    }

    /** Tools → Commit Hike → Settings…: show the tool window on its settings page. */
    fun openSettings() {
        openView = "settings"
        openToken++
        showTrail()
        cs.launch { publish(view) }
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
            app.noteError(e.message ?: e.toString())
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
