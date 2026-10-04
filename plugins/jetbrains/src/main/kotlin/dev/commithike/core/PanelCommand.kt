package dev.commithike.core

import com.google.gson.JsonElement
import com.google.gson.JsonObject
import com.google.gson.JsonParser

/**
 * What the trail panel asks the plugin to do. Parsed and checked here, so a
 * message with a missing or mistyped field is dropped instead of throwing in
 * the browser callback.
 */
sealed interface PanelCommand {
    data object Ready : PanelCommand
    data object Setup : PanelCommand
    data object Refresh : PanelCommand
    data object EnableProject : PanelCommand
    data object SetAvatar : PanelCommand
    data object ResetAvatar : PanelCommand
    data object ExportProgress : PanelCommand
    data object ImportProgress : PanelCommand
    data object CopyDiagnostics : PanelCommand
    data object SaveBadge : PanelCommand
    data object RequestTeam : PanelCommand
    data object ImportRoute : PanelCommand
    data object CreateRouteTemplate : PanelCommand
    data object Verify : PanelCommand
    data class ChooseRoute(val project: Boolean) : PanelCommand
    data class SetLocale(val locale: String) : PanelCommand
    data class SetTeam(val on: Boolean) : PanelCommand
    data class SetDifficulty(val level: String) : PanelCommand
    data class SetRestDays(val days: List<Int>) : PanelCommand // 0 = Sunday … 6 = Saturday, checked
    data class SetSettings(val reduceMotion: String?, val highContrast: String?, val notifications: String?) : PanelCommand
    data class SavePostcard(val fileName: String, val png: ByteArray) : PanelCommand {
        override fun equals(other: Any?) = other is SavePostcard && other.fileName == fileName && other.png.contentEquals(png)

        override fun hashCode() = 31 * fileName.hashCode() + png.contentHashCode()
    }
    data class PanelError(val message: String) : PanelCommand

    data class WalkRoute(val id: String) : PanelCommand

    companion object {
        private val levels = setOf("easy", "medium", "hard")
        private val autoOnOff = setOf("auto", "on", "off")
        private val notifications = setOf("all", "milestones", "off")
        private val routeId = Regex("^[a-z0-9]+(-[a-z0-9]+)*$")

        /** The command in [raw] JSON, or null for anything malformed or unknown. */
        fun parse(raw: String): PanelCommand? {
            val msg = runCatching { JsonParser.parseString(raw).asJsonObject }.getOrNull() ?: return null
            return when (msg.string("command")) {
                "ready" -> Ready
                "setup" -> Setup
                "refresh" -> Refresh
                "enableProject" -> EnableProject
                "setAvatar" -> SetAvatar
                "resetAvatar" -> ResetAvatar
                "exportProgress" -> ExportProgress
                "importProgress" -> ImportProgress
                "copyDiagnostics" -> CopyDiagnostics
                "saveBadge" -> SaveBadge
                "requestTeam" -> RequestTeam
                "importRoute" -> ImportRoute
                "createRouteTemplate" -> CreateRouteTemplate
                "verify" -> Verify
                "chooseRoute" -> ChooseRoute(project = msg.string("scope") == "project")
                "setLocale" -> msg.string("locale")?.takeIf { it.isNotBlank() && it.length <= 16 }?.let { SetLocale(it) }
                "setTeam" -> msg.bool("on")?.let { SetTeam(it) }
                "setDifficulty" -> msg.string("level")?.takeIf { it in levels }?.let { SetDifficulty(it) }
                "setRestDays" -> msg.ints("days")?.let { d -> SetRestDays(d.filter { it in 0..6 }.distinct().sorted()) }
                "setSettings" -> SetSettings(
                    msg.string("reduce_motion")?.takeIf { it in autoOnOff },
                    msg.string("high_contrast")?.takeIf { it in autoOnOff },
                    msg.string("notifications")?.takeIf { it in notifications },
                ).takeIf { it.reduceMotion != null || it.highContrast != null || it.notifications != null }
                "savePostcard" -> Files.png(msg.string("data"))?.let {
                    SavePostcard(Files.safeName(msg.string("name"), "commit-hike.png", "png"), it)
                }
                "panelError" -> PanelError("panel: " + msg.string("message").orEmpty().take(500))
                "walkRoute" -> msg.string("id")?.takeIf { routeId.matches(it) }?.let { WalkRoute(it) }
                else -> null
            }
        }

        private fun JsonObject.prim(key: String): JsonElement? = get(key)?.takeIf { it.isJsonPrimitive }

        private fun JsonObject.string(key: String): String? = prim(key)?.takeIf { it.asJsonPrimitive.isString }?.asString

        private fun JsonObject.bool(key: String): Boolean? = prim(key)?.takeIf { it.asJsonPrimitive.isBoolean }?.asBoolean

        private fun JsonObject.ints(key: String): List<Int>? = get(key)?.takeIf { it.isJsonArray }?.asJsonArray
            ?.mapNotNull { e ->
                e.takeIf { it.isJsonPrimitive && it.asJsonPrimitive.isNumber }?.asDouble?.takeIf { it % 1.0 == 0.0 }?.toInt()
            }
    }
}
