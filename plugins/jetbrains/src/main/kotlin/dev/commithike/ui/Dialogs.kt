package dev.commithike.ui

import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.DialogWrapper
import com.intellij.ui.SimpleListCellRenderer
import com.intellij.ui.dsl.builder.COLUMNS_LARGE
import com.intellij.ui.dsl.builder.bind
import com.intellij.ui.dsl.builder.bindItem
import com.intellij.ui.dsl.builder.bindText
import com.intellij.ui.dsl.builder.columns
import com.intellij.ui.dsl.builder.panel
import dev.commithike.core.I18n
import dev.commithike.core.LocaleInfo
import dev.commithike.core.Route
import dev.commithike.core.Scope
import javax.swing.JComponent

/** First-run setup: all three choices on one screen, the JetBrains way. */
class SetupDialog(project: Project, detectedEmail: String) : DialogWrapper(project) {
    data class Result(val mode: String, val fromHistory: Boolean, val emails: List<String>, val difficulty: String)

    private var mode = "all"
    private var start = "history"
    private var level = "medium"
    private var emails = detectedEmail

    init {
        title = I18n.t("setupTitle")
        setOKButtonText(I18n.t("setupStart"))
        init()
    }

    override fun createCenterPanel(): JComponent = panel {
        row {
            text(I18n.t("setupIntro"), maxLineLength = 60)
        }
        buttonsGroup(I18n.t("modeQuestion")) {
            row { radioButton(I18n.t("modeAll"), "all") }
            row { radioButton(I18n.t("modeSelected"), "selected") }
        }.bind(::mode)
        buttonsGroup(I18n.t("histQuestion")) {
            row { radioButton(I18n.t("histYes"), "history") }
            row { radioButton(I18n.t("histToday"), "today") }
        }.bind(::start)
        buttonsGroup(I18n.t("diffQuestion")) {
            for (l in listOf("easy", "medium", "hard")) row { radioButton(I18n.levelLabel(l), l) }
        }.bind(::level)
        row(I18n.t("emails")) {
            textField()
                .bindText(::emails)
                .columns(COLUMNS_LARGE)
                .comment(I18n.t("emailsComment"))
                .validationOnApply {
                    if (it.text.split(",").none { e -> e.contains("@") }) error(I18n.t("emailsInvalid")) else null
                }
        }
    }

    fun result() = Result(mode, start == "history", emails.split(",").map { it.trim() }.filter { it.isNotEmpty() }, level)
}

/** Picks a trail for all projects or for the current one. */
class RouteDialog(
    project: Project,
    scope: Scope,
    routes: List<Route>,
    canRemove: Boolean,
    private val onImport: () -> Unit,
    private val onTemplate: () -> Unit,
) : DialogWrapper(project) {
    data class Result(val routeId: String, val fromHistory: Boolean)

    private data class Choice(val id: String, val label: String, val detail: String)

    private val choices = routes.map { Choice(it.id, it.name, I18n.distance(it.lengthM)) } +
        if (canRemove) listOf(Choice(NONE, I18n.t("noProjectTrail"), "")) else emptyList()
    private var selected: Choice? = choices.firstOrNull()
    private var start = "history"

    init {
        title = if (scope == Scope.GLOBAL) I18n.t("trailAll") else I18n.t("trailProject")
        setOKButtonText(I18n.t("choose"))
        init()
    }

    override fun createCenterPanel(): JComponent = panel {
        row(I18n.t("trail")) {
            comboBox(
                choices,
                SimpleListCellRenderer.create<Choice?> { label, value, _ ->
                    label.text = if (value == null) {
                        ""
                    } else if (value.detail.isEmpty()) {
                        value.label
                    } else {
                        "${value.label}  (${value.detail})"
                    }
                },
            ).bindItem(::selected)
        }
        buttonsGroup(I18n.t("journeyStart")) {
            row { radioButton(I18n.t("histYes"), "history") }
            row { radioButton(I18n.t("histNow"), "now") }
        }.bind(::start)
        separator()
        row {
            link(I18n.t("importLink")) {
                close(CANCEL_EXIT_CODE)
                onImport()
            }
            link(I18n.t("templateLink")) {
                close(CANCEL_EXIT_CODE)
                onTemplate()
            }
        }
    }

    fun result(): Result? = selected?.let { Result(it.id, start == "history") }

    companion object {
        const val NONE = "none"
    }
}

/** Picks one of the user's imported routes to remove. */
class RemoveRouteDialog(project: Project, private val routes: List<Route>) : DialogWrapper(project) {
    var selected: Route? = routes.firstOrNull()

    init {
        title = I18n.t("removeTitle")
        setOKButtonText(I18n.t("remove"))
        init()
    }

    override fun createCenterPanel(): JComponent = panel {
        row(I18n.t("route")) {
            comboBox(
                routes,
                SimpleListCellRenderer.create<Route?> { label, value, _ ->
                    label.text = value?.let { "${it.name}  (${I18n.distance(it.lengthM)})" } ?: ""
                },
            ).bindItem(::selected)
        }
        row { comment(I18n.t("removeHint")) }
    }
}

/** Language for route texts, the trail view and plugin messages; "auto" follows the IDE. */
class LanguageDialog(project: Project, info: LocaleInfo) : DialogWrapper(project) {
    private data class Choice(val value: String, val label: String)

    private val choices = listOf(Choice("auto", I18n.t("langAuto"))) +
        (listOf("en", "uk") + info.available.orEmpty()).distinct().map { Choice(it, I18n.languageNames[it] ?: it) }
    private var chosen: Choice? = choices.firstOrNull { it.value == info.locale.ifEmpty { "auto" } } ?: choices.first()

    val selected: String? get() = chosen?.value

    init {
        title = I18n.t("langTitle")
        setOKButtonText(I18n.t("ok"))
        init()
    }

    override fun createCenterPanel(): JComponent = panel {
        row(I18n.t("langLabel")) {
            comboBox(choices, SimpleListCellRenderer.create<Choice?> { label, value, _ -> label.text = value?.label ?: "" })
                .bindItem(::chosen)
        }
        row { comment(I18n.t("langComment")) }
    }
}

/** How fast commits move you; a change applies from now on. */
class DifficultyDialog(project: Project, current: String) : DialogWrapper(project) {
    private var level = current

    val selected: String get() = level

    init {
        title = I18n.t("diffTitle")
        setOKButtonText(I18n.t("ok"))
        init()
    }

    override fun createCenterPanel(): JComponent = panel {
        buttonsGroup(I18n.t("diffLabel")) {
            for (l in listOf("easy", "medium", "hard")) row { radioButton(I18n.levelLabel(l), l) }
        }.bind(::level)
        row { comment(I18n.t("diffComment")) }
    }
}
