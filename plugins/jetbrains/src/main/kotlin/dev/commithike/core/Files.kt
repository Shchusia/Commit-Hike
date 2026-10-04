package dev.commithike.core

import java.io.File
import java.util.Base64

/** Checks for what the panel hands over to be saved. */
object Files {
    private const val PNG_PREFIX = "data:image/png;base64,"
    private const val MAX_DATA_URL = 20_000_000
    private val PNG_MAGIC = byteArrayOf(0x89.toByte(), 'P'.code.toByte(), 'N'.code.toByte(), 'G'.code.toByte())

    /** The bytes of a PNG data URL, or null unless it really is a PNG of a sane size. */
    fun png(dataUrl: String?): ByteArray? {
        if (dataUrl == null || !dataUrl.startsWith(PNG_PREFIX) || dataUrl.length > MAX_DATA_URL) return null
        val bytes = runCatching { Base64.getDecoder().decode(dataUrl.substring(PNG_PREFIX.length)) }.getOrNull() ?: return null
        return bytes.takeIf { it.size > 8 && it.copyOfRange(0, 4).contentEquals(PNG_MAGIC) }
    }

    /** Just a file name (no folders), with only safe characters and the given extension. */
    fun safeName(name: String?, fallback: String, ext: String): String {
        val base = File(name ?: "").name.replace(Regex("[^\\w.-]"), "-").trim('.', '-')
        val n = base.ifEmpty { fallback }
        return if (n.endsWith(".$ext", ignoreCase = true)) n else "$n.$ext"
    }
}
