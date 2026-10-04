package dev.commithike.ui

import com.intellij.DynamicBundle
import com.intellij.ide.ui.LafManagerListener
import com.intellij.openapi.Disposable
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.service
import com.intellij.openapi.project.DumbAware
import com.intellij.openapi.project.Project
import com.intellij.openapi.util.Disposer
import com.intellij.openapi.util.text.StringUtil
import com.intellij.openapi.wm.CustomStatusBarWidget
import com.intellij.openapi.wm.StatusBarWidget
import com.intellij.openapi.wm.StatusBarWidgetFactory
import com.intellij.openapi.wm.ToolWindow
import com.intellij.openapi.wm.ToolWindowFactory
import com.intellij.ui.JBColor
import com.intellij.ui.components.JBLabel
import com.intellij.ui.content.ContentFactory
import com.intellij.ui.jcef.JBCefApp
import com.intellij.ui.jcef.JBCefBrowser
import com.intellij.ui.jcef.JBCefBrowserBase
import com.intellij.ui.jcef.JBCefJSQuery
import com.intellij.ui.scale.JBUIScale
import com.intellij.util.ui.JBUI
import com.intellij.util.ui.UIUtil
import dev.commithike.CommitHikeApp
import dev.commithike.CommitHikeIcons
import dev.commithike.ProjectTrek
import dev.commithike.TrailListener
import dev.commithike.core.BuildInfo
import dev.commithike.core.I18n
import dev.commithike.core.PanelCommand
import dev.commithike.core.PanelPayload
import dev.commithike.core.Scope
import dev.commithike.core.Status
import java.awt.Color
import java.awt.Cursor
import java.awt.event.MouseAdapter
import java.awt.event.MouseEvent
import java.security.SecureRandom
import java.util.Base64
import javax.swing.JComponent

// ---------------------------------------------------------------- status bar

class TrailWidgetFactory : StatusBarWidgetFactory {
    override fun getId() = TrailWidget.ID
    override fun getDisplayName() = "Commit Hike"
    override fun createWidget(project: Project): StatusBarWidget = TrailWidget(project)
}

/** Shows "8.4 km / 20 km" with the trail icon; "+86 m" for a moment after a commit. */
class TrailWidget(private val project: Project) : CustomStatusBarWidget {
    private val label = JBLabel(CommitHikeIcons.Trail).apply {
        border = JBUI.Borders.empty(0, 6)
        iconTextGap = JBUI.scale(4)
        cursor = Cursor.getPredefinedCursor(Cursor.HAND_CURSOR)
    }

    init {
        project.messageBus.connect(this).subscribe(TrailListener.TOPIC, TrailListener { render() })
        label.addMouseListener(object : MouseAdapter() {
            override fun mouseClicked(e: MouseEvent) = project.service<ProjectTrek>().showTrail()
        })
        render()
    }

    override fun ID() = ID
    override fun getComponent(): JComponent = label

    private fun render() {
        val v = project.service<ProjectTrek>().view
        val s = v.status
        val g = s?.global
        label.text = when {
            v.flash != null -> v.flash
            v.state == "error" -> I18n.t("barError")
            g == null -> "Commit Hike"
            g.finished -> "${I18n.distance(g.distanceM)} ✓"
            else -> "${I18n.distance(g.distanceM)} / ${I18n.distance(g.route.lengthM)}"
        }
        label.toolTipText = tooltip(v.state, v.error, v.repo, s)
    }

    private fun tooltip(state: String, error: String?, repo: String?, s: Status?): String {
        fun e(t: String) = StringUtil.escapeXmlEntities(t)
        val g = s?.global
        val d = I18n::distance
        if (state == "error") return "<html>${I18n.t("barErrorTip")}<br>${e(error ?: I18n.t("unknownError"))}</html>"
        if (s == null || g == null) return I18n.t("barSetup")
        val lines = mutableListOf("<b>${e(g.route.name)}</b>: ${I18n.t("barOf", d(g.distanceM), d(g.route.lengthM))} (${g.percent}%)")
        val next = g.nextWaypoint
        when {
            g.finished -> lines += I18n.t("barCompleted")
            next != null -> lines += I18n.t("barNext", e(next.name), d(g.toNextM ?: 0.0))
        }
        g.elevationM?.let { lines += I18n.t("barAltitude", it.toInt(), g.ascentM.toInt()) }
        lines += I18n.t("barToday", d(s.todayM))
        val p = s.project
        if (s.streakDays() > 0) lines += I18n.t("barStreak", s.streakDays())
        if (repo != null && p != null) {
            lines += I18n.t("barProject", "<b>${e(p.route.name)}</b>", "${d(p.distanceM)} (${p.percent}%)")
        } else if (repo != null && !s.tracked) {
            lines += I18n.t("barNotCounted")
        }
        return "<html>" + lines.joinToString("<br>") + "</html>"
    }

    private fun Status.streakDays() = global?.streakDays ?: 0

    companion object {
        const val ID = "CommitHike"
    }
}

// ---------------------------------------------------------------- tool window

class TrailToolWindowFactory :
    ToolWindowFactory,
    DumbAware {
    override fun createToolWindowContent(project: Project, toolWindow: ToolWindow) {
        // LinkageError: on 2026.2+ the "Web Browser (JCEF)" plugin can be disabled, and
        // then the JCEF classes don't exist at all. Show the explanation instead of failing.
        val jcef = try {
            JBCefApp.isSupported()
        } catch (_: LinkageError) {
            false
        }
        val component: JComponent = if (jcef) {
            TrailBrowser(project, toolWindow.disposable).component
        } else {
            JBLabel("<html>${I18n.t("noJcef")}</html>").apply { border = JBUI.Borders.empty(12) }
        }
        toolWindow.contentManager.addContent(ContentFactory.getInstance().createContent(component, null, false))
    }
}

/**
 * Hosts ui/panel.html (shared with the VS Code extension) in JCEF.
 * Page -> IDE: window.commitHikeHost.send(json), backed by a JBCefJSQuery.
 * IDE -> page: window.commitHike.update(data).
 */
private class TrailBrowser(private val project: Project, parent: Disposable) : Disposable {
    private val browser = JBCefBrowser()
    private val query = JBCefJSQuery.create(browser as JBCefBrowserBase)

    @Volatile private var ready = false

    val component: JComponent get() = browser.component

    init {
        Disposer.register(parent, this)
        Disposer.register(this, browser)
        Disposer.register(this, query)
        query.addHandler { msg ->
            ApplicationManager.getApplication().invokeLater { onMessage(msg) }
            null
        }
        project.messageBus.connect(this).subscribe(TrailListener.TOPIC, TrailListener { push() })
        ApplicationManager.getApplication().messageBus.connect(this)
            .subscribe(LafManagerListener.TOPIC, LafManagerListener { load() }) // theme switched: re-render with new colors
        load()
    }

    private fun load() {
        ready = false
        browser.loadHTML(html())
    }

    private fun html(): String {
        val raw = TrailBrowser::class.java.getResource("/ui/panel.html")?.readText()
            ?: return "<p>panel.html is missing from this build.</p>"
        // Only scripts carrying this page load's nonce may run: route HTML
        // objects, however they were written, can't execute anything.
        val nonce = Base64.getEncoder().encodeToString(ByteArray(18).also { SecureRandom().nextBytes(it) })
        val bridge = "<script nonce=\"$nonce\">window.commitHikeHost={send:function(m){${query.inject("m")}}};</script>"
        // replaceFirst: the page's own script must never be touched by these edits.
        return raw
            .replace("{{CSP}}", "default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-$nonce'; img-src data:")
            .replace("{{NONCE}}", nonce)
            .replaceFirst("<html lang=\"en\">", "<html lang=\"en\" style=\"${themeVars()}\">")
            .replaceFirst("<head>", "<head>$bridge")
            .replaceFirst("<body>", if (JBColor.isBright()) "<body data-theme=\"light\">" else "<body>")
    }

    /** Maps the current IDE theme onto the --jb-* variables panel.html reads. */
    private fun themeVars(): String {
        fun hex(c: Color) = "#%02x%02x%02x".format(c.red, c.green, c.blue)
        val font = UIUtil.getLabelFont()
        val sizePx = font.size2D / JBUIScale.scale(1f) // JCEF applies HiDPI scaling itself
        return listOf(
            "--jb-bg" to hex(UIUtil.getPanelBackground()),
            "--jb-fg" to hex(UIUtil.getLabelForeground()),
            "--jb-muted" to hex(UIUtil.getContextHelpForeground()),
            "--jb-border" to hex(JBColor.border()),
            "--jb-link" to hex(JBUI.CurrentTheme.Link.Foreground.ENABLED),
            "--jb-btn-bg" to hex(JBUI.CurrentTheme.Button.defaultButtonColorStart()),
            "--jb-btn-hover" to hex(JBUI.CurrentTheme.Button.defaultButtonColorStart()),
            "--jb-focus" to hex(JBUI.CurrentTheme.Focus.focusColor()),
            "--jb-font" to "'${font.family.replace("'", "")}', system-ui, sans-serif",
            "--jb-font-size" to "${"%.1f".format(java.util.Locale.ROOT, sizePx)}px",
        ).joinToString(";") { "${it.first}:${it.second}" }
    }

    private fun onMessage(raw: String) {
        val trek = project.service<ProjectTrek>()
        when (val c = PanelCommand.parse(raw) ?: return) {
            PanelCommand.Ready -> {
                ready = true
                push()
            }
            PanelCommand.Setup -> trek.setup()
            PanelCommand.Refresh -> trek.requestRefresh()
            PanelCommand.EnableProject -> trek.setProjectEnabled(true)
            PanelCommand.SetAvatar -> trek.setHikerIcon()
            PanelCommand.ResetAvatar -> trek.resetHikerIcon()
            PanelCommand.ExportProgress -> trek.exportProgress()
            PanelCommand.ImportProgress -> trek.importProgress()
            PanelCommand.CopyDiagnostics -> trek.copyDiagnostics()
            PanelCommand.SaveBadge -> trek.saveBadge()
            PanelCommand.RequestTeam -> trek.requestTeam()
            PanelCommand.ImportRoute -> trek.importRoute()
            PanelCommand.CreateRouteTemplate -> trek.createRouteTemplate()
            PanelCommand.Verify -> trek.verify()
            is PanelCommand.ChooseRoute -> trek.chooseRoute(if (c.project) Scope.PROJECT else Scope.GLOBAL)
            is PanelCommand.SetLocale -> trek.setLocale(c.locale)
            is PanelCommand.SetTeam -> trek.setTeam(c.on)
            is PanelCommand.SetDifficulty -> trek.setDifficulty(c.level)
            is PanelCommand.SetRestDays -> trek.setRestDaysTo(c.days)
            is PanelCommand.SetSettings -> trek.setSettings(c.reduceMotion, c.highContrast, c.notifications)
            is PanelCommand.SavePostcard -> trek.savePostcard(c.fileName, c.png)
            is PanelCommand.PanelError -> service<CommitHikeApp>().noteError(c.message)
            is PanelCommand.WalkRoute -> trek.walkRoute(c.id)
            is PanelCommand.SetTeamGoal -> trek.setTeamGoal(c.route)
            is PanelCommand.OpenUrl -> trek.openShareUrl(c.url)
            is PanelCommand.CopyText -> trek.copyText(c.text)
            is PanelCommand.CopyImage -> trek.copyPostcard(c.png)
        }
    }

    private fun push() {
        if (!ready) return
        val trek = project.service<ProjectTrek>()
        val app = CommitHikeApp.getInstance()
        val payload = PanelPayload.of(
            view = trek.view,
            ideLocale = DynamicBundle.getLocale().toLanguageTag(),
            avatar = app.avatar,
            assets = app.assetsFor(trek.view.status),
            localeSetting = app.localeInfo,
            dev = isDevBuild,
            build = buildInfo,
            openView = trek.openView,
            openToken = trek.openToken,
        ) ?: return
        browser.cefBrowser.executeJavaScript(payload.script(), browser.cefBrowser.url, 0)
    }

    override fun dispose() {}

    private companion object {
        val buildInfo: BuildInfo = BuildInfo.current

        // A dev build (task ... FLAVOR=dev), runIde/runPyCharm, or COMMIT_HIKE_DEV=1 may look ahead.
        val isDevBuild: Boolean =
            buildInfo.flavor == "dev" ||
                System.getProperty("commit-hike.dev") == "true" ||
                System.getenv("COMMIT_HIKE_DEV") == "1"
    }
}
