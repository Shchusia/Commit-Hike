package dev.commithike.actions

import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.components.service
import com.intellij.openapi.project.DumbAware
import dev.commithike.ProjectTrek
import dev.commithike.core.I18n
import dev.commithike.core.Scope

/**
 * Base for all Commit Hike actions: enabled whenever a project is open. Menu
 * texts follow the Commit Hike language (plugin.xml keeps English defaults
 * for search and keymaps).
 */
abstract class TrekAction(private val key: String, private val run: ProjectTrek.() -> Unit) :
    AnAction(),
    DumbAware {
    override fun getActionUpdateThread() = ActionUpdateThread.BGT

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabledAndVisible = e.project != null
        e.presentation.text = I18n.t("action.$key")
    }

    override fun actionPerformed(e: AnActionEvent) {
        e.project?.service<ProjectTrek>()?.run()
    }
}

class ShowTrailAction : TrekAction("showTrail", { showTrail() })
class SetupAction : TrekAction("setup", { setup() })
class ChooseRouteAction : TrekAction("chooseRoute", { chooseRoute(Scope.GLOBAL) })
class ChooseProjectRouteAction : TrekAction("chooseProjectRoute", { chooseRoute(Scope.PROJECT) })
class EnableProjectAction : TrekAction("enableProject", { setProjectEnabled(true) })
class DisableProjectAction : TrekAction("disableProject", { setProjectEnabled(false) })
class VerifyAction : TrekAction("verify", { verify() })
class ImportRouteAction : TrekAction("importRoute", { importRoute() })
class CreateRouteTemplateAction : TrekAction("createRouteTemplate", { createRouteTemplate() })

class SetRestDaysAction : TrekAction("setRestDays", { setRestDays() })

class OpenSettingsAction : TrekAction("openSettings", { openSettings() })

class ExportProgressAction : TrekAction("exportProgress", { exportProgress() })

class ImportProgressAction : TrekAction("importProgress", { importProgress() })
class RemoveRouteAction : TrekAction("removeRoute", { removeRoute() })
class SetHikerIconAction : TrekAction("setHikerIcon", { setHikerIcon() })
class ResetHikerIconAction : TrekAction("resetHikerIcon", { resetHikerIcon() })
class ChangeLanguageAction : TrekAction("changeLanguage", { changeLanguage() })
class ToggleTeamAction : TrekAction("toggleTeam", { toggleTeam() })
class ChangeDifficultyAction : TrekAction("changeDifficulty", { changeDifficulty() })
