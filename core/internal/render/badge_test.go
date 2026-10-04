package render

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestBadgeIsValidSVGWithEscapedText(t *testing.T) {
	svg := Badge("🥾 Commit Hike", `342 km · <Chornohora> & "Ridge"`)
	if err := xml.Unmarshal([]byte(svg), new(struct{})); err != nil {
		t.Fatalf("not valid XML: %v\n%s", err, svg)
	}
	if !strings.Contains(svg, "&lt;Chornohora&gt; &amp;") || strings.Contains(svg, "<Chornohora>") {
		t.Fatalf("text must be escaped: %s", svg)
	}
	if !strings.Contains(svg, `role="img"`) || !strings.Contains(svg, "<title>") {
		t.Fatal("badges need a role and a title for screen readers")
	}
}

func TestLongerTextMakesAWiderBadge(t *testing.T) {
	if textWidth("Chornohora Ridge") <= textWidth("Demo") || textWidth("MW") <= textWidth("il") {
		t.Fatal("widths must follow the text")
	}
}

func TestBadgesHaveTheirOwnIDs(t *testing.T) {
	a, b := Badge("x", "1 km"), Badge("x", "2 km")
	if strings.Contains(a, `id="r"`) || strings.Contains(a, `url(#s)`) {
		t.Fatal("ids must be unique per badge")
	}
	idOf := func(svg string) string { i := strings.Index(svg, `clipPath id="`); return svg[i : i+24] }
	if idOf(a) == idOf(b) {
		t.Fatal("different badges need different ids")
	}
}
