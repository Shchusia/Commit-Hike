package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/commit-hike/commit-hike/core/internal/protocol"
)

func writePNG(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	img.Set(w/2, h/2, color.NRGBA{A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "icon.png")
	mustWrite(t, p, b.String())
	return p
}

func TestAvatar(t *testing.T) {
	s, _ := setup(t, "")
	a, err := s.Avatar()
	if err != nil || a.Custom || a.DataURL != "" {
		t.Fatalf("default: %+v %v", a, err)
	}
	a, err = s.SetAvatar(writePNG(t, 64, 96))
	if err != nil || !a.Custom || !strings.HasPrefix(a.DataURL, "data:image/png;base64,") {
		t.Fatalf("set: %+v %v", a, err)
	}
	if fi, _ := os.Stat(s.avatarPath()); fi.Mode().Perm() != 0o600 {
		t.Errorf("perms %v", fi.Mode().Perm())
	}

	bad := map[string]string{
		"too big":  writePNG(t, 1024, 64),
		"too tiny": writePNG(t, 4, 4),
		"missing":  filepath.Join(t.TempDir(), "nope.png"),
	}
	jpeg := filepath.Join(t.TempDir(), "x.png")
	mustWrite(t, jpeg, "\xff\xd8\xff\xe0 not a png")
	bad["not a png"] = jpeg
	for name, p := range bad {
		if _, err := s.SetAvatar(p); code(err) != protocol.CodeInvalidImage {
			t.Errorf("%s: want invalid_image, got %v", name, err)
		}
	}
	if a, _ := s.Avatar(); !a.Custom {
		t.Error("a rejected image must not replace the current one")
	}
	if err := s.ResetAvatar(); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetAvatar(); err != nil { // resetting twice is fine
		t.Fatal(err)
	}
	if a, _ := s.Avatar(); a.Custom {
		t.Error("reset should restore the default")
	}
}

func TestJourneyDay(t *testing.T) {
	s, r := setup(t, "demo-trail")
	now := time.Unix(base+50000, 0)
	r.ts = now.Unix() - 4*86400 // first commit 4 days ago -> today is day 5
	r.commit("", "a.go", 1)
	if got := scan(t, s, r.dir, "").Global.Day; got != 5 {
		t.Fatalf("day with history = %d, want 5", got)
	}
	if _, err := s.SetJourney(ScopeGlobal, "", "chornohora-ridge", false, ""); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.Status(r.dir, ""); st.Global.Day != 1 {
		t.Fatalf("a journey starting now is on day 1, got %d", st.Global.Day)
	}
}
