package dev.commithike.core

import com.google.gson.FieldNamingPolicy
import com.google.gson.GsonBuilder
import com.google.gson.JsonElement

/** What a project's trail panel and status bar show right now. */
data class TrailView(
    val state: String, // loading | not_initialized | ok | error
    val error: String? = null,
    val repo: String? = null,
    val status: Status? = null,
    val flash: String? = null, // "+86 m", shown briefly after a commit
    val team: Team? = null, // teammates, when the user turned them on for this project
    val teamError: String? = null,
)

/** Everything the panel gets with one update, as the panel's protocol names it. */
data class PanelPayload(
    val type: String = "update",
    val state: String,
    val error: String?,
    val repo: String?,
    val locale: String,
    val status: JsonElement?, // the core's own JSON, so every field reaches the panel
    val avatar: String?, // custom hiker PNG as a data URL; null = the panel's default
    val avatarCustom: Boolean,
    val assets: Map<String, RouteAssets>, // pictures for route objects and custom maps, per route id
    val team: Team?,
    val teamError: String?,
    val localeSetting: LocaleInfo?,
    val dev: Boolean, // development build: the trail view may look ahead
    val build: BuildInfo,
    val openView: String?, // a page to open once per openToken, e.g. "settings"
    val openToken: Int,
) {
    /** The JavaScript that hands this to the panel. */
    fun script(): String = "window.commitHike && window.commitHike.update(${gson.toJson(this)});" // Gson escapes <, >, & and quotes

    companion object {
        private val gson = GsonBuilder().setFieldNamingPolicy(FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES).create()

        /** The update for [view]; null while it's still loading. */
        fun of(
            view: TrailView,
            ideLocale: String,
            avatar: Avatar,
            assets: Map<String, RouteAssets>,
            localeSetting: LocaleInfo?,
            dev: Boolean,
            build: BuildInfo,
            openView: String?,
            openToken: Int,
        ): PanelPayload? {
            if (view.state == "loading") return null
            return PanelPayload(
                state = view.state,
                error = view.error,
                repo = view.repo,
                locale = view.status?.locale ?: ideLocale,
                status = view.status?.raw ?: view.status?.let { gson.toJsonTree(it) },
                avatar = avatar.dataUrl,
                avatarCustom = avatar.custom,
                assets = assets,
                team = view.team,
                teamError = view.teamError,
                localeSetting = localeSetting,
                dev = dev,
                build = build,
                openView = openView,
                openToken = openToken,
            )
        }
    }
}
