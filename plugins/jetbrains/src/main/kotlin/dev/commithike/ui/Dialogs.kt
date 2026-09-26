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
import dev.commithike.core.Route
import dev.commithike.core.Scope
import dev.commithike.core.formatDistance
import javax.swing.JComponent

/** First-run setup: all three choices on one screen, the JetBrains way. */
class SetupDialog(project: Project, detectedEmail: String) : DialogWrapper(project) {
    data class Result(val mode: String, val fromHistory: Boolean, val emails: List<String>)

    private var mode = "all"
    private var start = "history"
    private var emails = detectedEmail

    init {
        title = "Set Up Commit Hike"
        setOKButtonText("Start Journey")
        init()
    }

    override fun createCenterPanel(): JComponent = panel {
        row {
            text(
                "Every commit you make moves you along a hiking trail. " +
                    "Commit Hike never writes to your projects, and everything stays on this computer.",
                maxLineLength = 60,
            )
        }
        buttonsGroup("Which projects should count?") {
            row { radioButton("All my projects", "all") }
            row { radioButton("Only projects I choose", "selected") }
        }.bind(::mode)
        buttonsGroup("Where does your journey start?") {
            row { radioButton("Include commits I already made", "history") }
            row { radioButton("Start from today", "today") }
        }.bind(::start)
        row("Your author emails:") {
            textField()
                .bindText(::emails)
                .columns(COLUMNS_LARGE)
                .comment("Commits with these emails count as yours. Separate several with commas.")
                .validationOnApply {
                    if (it.text.split(",").none { e -> e.contains("@") }) error("Enter at least one email address.") else null
                }
        }
    }

    fun result() = Result(mode, start == "history", emails.split(",").map { it.trim() }.filter { it.isNotEmpty() })
}

/** Picks a trail for all projects or for the current one. */
class RouteDialog(project: Project, scope: Scope, routes: List<Route>, canRemove: Boolean) : DialogWrapper(project) {
    data class Result(val routeId: String, val fromHistory: Boolean)

    private data class Choice(val id: String, val label: String, val detail: String)

    private val choices = routes.map { Choice(it.id, it.name, formatDistance(it.lengthM)) } +
        if (canRemove) listOf(Choice(NONE, "No trail for this project", "")) else emptyList()
    private var selected: Choice? = choices.firstOrNull()
    private var start = "history"

    init {
        title = if (scope == Scope.GLOBAL) "Trail for All Projects" else "Trail for This Project"
        setOKButtonText("Choose")
        init()
    }

    override fun createCenterPanel(): JComponent = panel {
        row("Trail:") {
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
        buttonsGroup("Where does this journey start?") {
            row { radioButton("Include commits I already made", "history") }
            row { radioButton("Start from now", "now") }
        }.bind(::start)
    }

    fun result(): Result? = selected?.let { Result(it.id, start == "history") }

    companion object {
        const val NONE = "none"
    }
}
