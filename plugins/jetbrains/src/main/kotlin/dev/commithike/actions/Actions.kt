package dev.commithike.actions

import com.intellij.openapi.actionSystem.ActionUpdateThread
import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.components.service
import com.intellij.openapi.project.DumbAware
import dev.commithike.ProjectTrek
import dev.commithike.core.Scope

/** Base for all Commit Hike actions: enabled whenever a project is open. */
abstract class TrekAction(private val run: ProjectTrek.() -> Unit) :
    AnAction(),
    DumbAware {
    override fun getActionUpdateThread() = ActionUpdateThread.BGT

    override fun update(e: AnActionEvent) {
        e.presentation.isEnabledAndVisible = e.project != null
    }

    override fun actionPerformed(e: AnActionEvent) {
        e.project?.service<ProjectTrek>()?.run()
    }
}

class ShowTrailAction : TrekAction({ showTrail() })
class SetupAction : TrekAction({ setup() })
class ChooseRouteAction : TrekAction({ chooseRoute(Scope.GLOBAL) })
class ChooseProjectRouteAction : TrekAction({ chooseRoute(Scope.PROJECT) })
class EnableProjectAction : TrekAction({ setProjectEnabled(true) })
class DisableProjectAction : TrekAction({ setProjectEnabled(false) })
class VerifyAction : TrekAction({ verify() })
