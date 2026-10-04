package dev.commithike.ui

import com.intellij.ide.BrowserUtil
import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.application.ModalityState
import com.intellij.openapi.fileChooser.FileChooser
import com.intellij.openapi.fileChooser.FileChooserDescriptorFactory
import com.intellij.openapi.fileChooser.FileChooserFactory
import com.intellij.openapi.fileChooser.FileSaverDescriptor
import com.intellij.openapi.fileEditor.FileEditorManager
import com.intellij.openapi.ide.CopyPasteManager
import com.intellij.openapi.project.Project
import com.intellij.openapi.ui.Messages
import com.intellij.openapi.vfs.LocalFileSystem
import com.intellij.openapi.vfs.VirtualFile
import dev.commithike.core.LocaleInfo
import dev.commithike.core.Route
import dev.commithike.core.RouteChoice
import dev.commithike.core.Scope
import dev.commithike.core.SetupChoice
import dev.commithike.core.TrekUi
import java.awt.datatransfer.StringSelection
import java.io.File

/**
 * [TrekUi] in a JetBrains IDE. The flows run off the UI thread; every dialog
 * hops to the EDT and waits for the answer.
 */
class IdeTrekUi(
    private val project: Project,
    private val notifier: (title: String, text: String, button: String?, action: (() -> Unit)?) -> Unit,
    private val onImportRoute: () -> Unit,
    private val onCreateTemplate: () -> Unit,
) : TrekUi {
    private fun <T> onEdt(block: () -> T): T {
        var result: Result<T>? = null
        ApplicationManager.getApplication().invokeAndWait({ result = runCatching(block) }, ModalityState.defaultModalityState())
        return result!!.getOrThrow()
    }

    override fun chooseFileToOpen(title: String, description: String, ext: String): File? = onEdt {
        val descriptor = FileChooserDescriptorFactory.createSingleFileDescriptor(ext).withTitle(title).withDescription(description)
        FileChooser.chooseFile(descriptor, project, null)?.let { File(it.path) }
    }

    override fun chooseFileOrFolder(title: String, description: String): File? = onEdt {
        val descriptor = FileChooserDescriptorFactory.createSingleFileOrFolderDescriptor().withTitle(title).withDescription(description)
        FileChooser.chooseFile(descriptor, project, null)?.let { File(it.path) }
    }

    override fun chooseFolder(title: String): File? = onEdt {
        FileChooser.chooseFile(FileChooserDescriptorFactory.createSingleFolderDescriptor().withTitle(title), project, null)?.let {
            File(it.path)
        }
    }

    override fun chooseFileToSave(title: String, description: String, ext: String, defaultName: String): File? = onEdt {
        FileChooserFactory.getInstance().createSaveFileDialog(FileSaverDescriptor(title, description, ext), project)
            .save(null as VirtualFile?, defaultName)?.file
    }

    override fun askText(prompt: String, title: String, initial: String): String? = onEdt {
        Messages.showInputDialog(project, prompt, title, null, initial, null)
    }

    override fun confirm(question: String, yes: String, no: String): Boolean = onEdt {
        Messages.showYesNoDialog(project, question, "Commit Hike", yes, no, null) == Messages.YES
    }

    override fun setup(detectedEmail: String): SetupChoice? = onEdt {
        val dialog = SetupDialog(project, detectedEmail)
        if (dialog.showAndGet()) dialog.result().let { SetupChoice(it.emails, it.mode, it.fromHistory, it.difficulty) } else null
    }

    override fun pickRoute(scope: Scope, routes: List<Route>, canRemove: Boolean): RouteChoice? = onEdt {
        val dialog = RouteDialog(project, scope, routes, canRemove, onImport = onImportRoute, onTemplate = onCreateTemplate)
        if (dialog.showAndGet()) dialog.result()?.let { RouteChoice(it.routeId, it.fromHistory) } else null
    }

    override fun pickRouteToRemove(routes: List<Route>): Route? = onEdt {
        val dialog = RemoveRouteDialog(project, routes)
        if (dialog.showAndGet()) dialog.selected else null
    }

    override fun pickLanguage(info: LocaleInfo): String? = onEdt {
        val dialog = LanguageDialog(project, info)
        if (dialog.showAndGet()) dialog.selected else null
    }

    override fun pickRestDays(current: List<Int>): List<Int>? = onEdt {
        val dialog = RestDaysDialog(project, current)
        if (dialog.showAndGet()) dialog.selected else null
    }

    override fun pickDifficulty(current: String): String? = onEdt {
        val dialog = DifficultyDialog(project, current)
        if (dialog.showAndGet()) dialog.selected else null
    }

    override fun notify(text: String, button: String?, title: String?, action: (() -> Unit)?) =
        notifier(title ?: "Commit Hike", text, button, action)

    override fun openInEditor(file: File) {
        onEdt {
            LocalFileSystem.getInstance().refreshAndFindFileByPath(file.path)?.let {
                FileEditorManager.getInstance(project).openFile(it, true)
            }
        }
    }

    override fun copyToClipboard(text: String) = CopyPasteManager.getInstance().setContents(StringSelection(text))

    override fun open(file: File) = BrowserUtil.browse(file.toPath())

    override fun browse(url: String) = BrowserUtil.browse(url)
}
