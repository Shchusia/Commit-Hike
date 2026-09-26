package dev.commithike.ui

import com.google.gson.FieldNamingPolicy
import com.google.gson.GsonBuilder
import com.google.gson.JsonParser
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
import dev.commithike.core.Scope
import dev.commithike.core.Status
import dev.commithike.core.formatDistance
import java.awt.Color
import java.awt.Cursor
import java.awt.event.MouseAdapter
import java.awt.event.MouseEvent
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
            v.state == "error" -> "Commit Hike: error"
            g == null -> "Commit Hike"
            g.finished -> "${formatDistance(g.distanceM)} ✓"
            else -> "${formatDistance(g.distanceM)} / ${formatDistance(g.route.lengthM)}"
        }
        label.toolTipText = tooltip(v.state, v.error, v.repo, s)
    }

    private fun tooltip(state: String, error: String?, repo: String?, s: Status?): String {
        fun e(t: String) = StringUtil.escapeXmlEntities(t)
        val g = s?.global
        if (state == "error") return "<html>Commit Hike couldn't read your progress:<br>${e(error ?: "unknown error")}</html>"
        if (s == null || g == null) return "Set up Commit Hike to turn your commits into a journey."
        val lines =
            mutableListOf("<b>${e(g.route.name)}</b>: ${formatDistance(g.distanceM)} of ${formatDistance(g.route.lengthM)} (${g.percent}%)")
        val next = g.nextWaypoint
        when {
            g.finished -> lines += "Trail completed."
            next != null -> lines += "Next stop: ${e(next.name)}, in ${formatDistance(g.toNextM ?: 0.0)}"
        }
        lines += "Today: ${formatDistance(s.todayM)}"
        val p = s.project
        if (s.streakDays() > 0) lines += "Streak: ${s.streakDays()} days"
        if (repo != null && p != null) {
            lines += "This project: <b>${e(p.route.name)}</b>, ${formatDistance(p.distanceM)} (${p.percent}%)"
        } else if (repo != null && !s.tracked) {
            lines += "This project isn't counted."
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
        val component: JComponent = if (JBCefApp.isSupported()) {
            TrailBrowser(project, toolWindow.disposable).component
        } else {
            JBLabel(
                "<html>The trail view needs the IDE's embedded browser (JCEF), which isn't available here. " +
                    "Your progress is still shown in the status bar.</html>",
            ).apply { border = JBUI.Borders.empty(12) }
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
    private val gson = GsonBuilder().setFieldNamingPolicy(FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES).create()

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
        val bridge = "<script>window.commitHikeHost={send:function(m){${query.inject("m")}}};</script>"
        return raw
            .replace("{{CSP}}", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src data:")
            .replace("{{NONCE}}", "")
            .replace("<html lang=\"en\">", "<html lang=\"en\" style=\"${themeVars()}\">")
            .replace("<head>", "<head>$bridge")
            .replace("<body>", if (JBColor.isBright()) "<body data-theme=\"light\">" else "<body>")
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
        val msg = try {
            JsonParser.parseString(raw).asJsonObject
        } catch (e: Exception) {
            return
        }
        val trek = project.service<ProjectTrek>()
        when (msg.get("command")?.asString) {
            "ready" -> {
                ready = true
                push()
            }
            "setup" -> trek.setup()
            "refresh" -> trek.requestRefresh()
            "enableProject" -> trek.setProjectEnabled(true)
            "setAvatar" -> trek.setHikerIcon()
            "resetAvatar" -> trek.resetHikerIcon()
            "chooseRoute" -> trek.chooseRoute(if (msg.get("scope")?.asString == "project") Scope.PROJECT else Scope.GLOBAL)
        }
    }

    private data class PanelData(
        val type: String = "update",
        val state: String,
        val error: String?,
        val repo: String?,
        val locale: String,
        val status: Status?,
        val avatar: String?, // custom hiker PNG as a data URL; null = the panel's default
        val avatarCustom: Boolean,
    )

    private fun push() {
        if (!ready) return
        val v = project.service<ProjectTrek>().view
        if (v.state == "loading") return
        val locale = v.status?.locale ?: DynamicBundle.getLocale().toLanguageTag()
        val avatar = CommitHikeApp.getInstance().avatar
        val data = PanelData(
            state = v.state,
            error = v.error,
            repo = v.repo,
            locale = locale,
            status = v.status,
            avatar = avatar.dataUrl,
            avatarCustom = avatar.custom,
        )
        // Gson escapes <, >, & and quotes, so its output is a safe JS literal.
        val js = "window.commitHike && window.commitHike.update(${gson.toJson(data)});"
        browser.cefBrowser.executeJavaScript(js, browser.cefBrowser.url, 0)
    }

    override fun dispose() {}
}
