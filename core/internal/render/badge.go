package render

import (
	"fmt"
	"hash/fnv"
	"html"
	"strings"
	"unicode"
)

// Badge draws a README badge in the shields.io style: a dark label on the
// left, the value on the trail's yellow on the right. Text is measured with
// an approximation of Verdana 11px, which GitHub renders badges with.
func Badge(label, value string) string {
	const h, pad = 20, 6.0
	lw, vw := textWidth(label)+2*pad, textWidth(value)+2*pad
	w := lw + vw
	esc := html.EscapeString
	// ids unique per badge, so several badges inlined in one page don't share a clip path
	sum := fnv.New32a()
	_, _ = sum.Write([]byte(label + "\x00" + value))
	id := fmt.Sprintf("ch%x", sum.Sum32())
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%d" role="img" aria-label="%s: %s">`+
		`<title>%s: %s</title>`+
		`<linearGradient id="s" x2="0" y2="100%%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>`+
		`<clipPath id="r"><rect width="%.0f" height="%d" rx="3" fill="#fff"/></clipPath>`+
		`<g clip-path="url(#r)"><rect width="%.0f" height="%d" fill="#2f3337"/><rect x="%.0f" width="%.0f" height="%d" fill="#e8b53d"/><rect width="%.0f" height="%d" fill="url(#s)"/></g>`+
		`<g text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">`+
		// textLength makes the text exactly as wide as measured, whatever font the viewer has
		`<text x="%.1f" y="14" fill="#f3efe4" textLength="%.1f" lengthAdjust="spacingAndGlyphs">%s</text>`+
		`<text x="%.1f" y="14" fill="#1b1f22" textLength="%.1f" lengthAdjust="spacingAndGlyphs">%s</text></g></svg>`,
		w, h, esc(label), esc(value), esc(label), esc(value),
		w, h, lw, h, lw, vw, h, w, h,
		lw/2, lw-2*pad, esc(label), lw+vw/2, vw-2*pad, esc(value))
	return strings.NewReplacer(`id="s"`, `id="`+id+`s"`, `url(#s)`, `url(#`+id+`s)`, `id="r"`, `id="`+id+`r"`, `url(#r)`, `url(#`+id+`r)`).Replace(svg)
}

// textWidth approximates the width of s in Verdana 11px.
func textWidth(s string) float64 {
	w := 0.0
	for _, r := range s {
		switch {
		case r > 0x2000 && !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '·' && r != '✓':
			w += 14 // emoji and symbols
		case strings.ContainsRune("iljI.,:;'!|", r):
			w += 3.5
		case strings.ContainsRune("mwMW", r):
			w += 10.5
		case r == ' ' || r == '·':
			w += 4
		case unicode.IsDigit(r):
			w += 7
		case unicode.Is(unicode.Cyrillic, r) && unicode.IsUpper(r):
			w += 8.6
		case unicode.Is(unicode.Cyrillic, r):
			w += 7.4
		case unicode.IsUpper(r):
			w += 7.8
		default:
			w += 6.8
		}
	}
	return w
}
