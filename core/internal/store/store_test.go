package store

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestConfigRoundTripAndPermissions(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadConfig(); err != ErrNotInitialized {
		t.Fatalf("want ErrNotInitialized, got %v", err)
	}
	in := &Config{Mode: ModeAll, Emails: []string{"me@x.io"}, Locale: "uk"}
	if err := s.SaveConfig(in); err != nil {
		t.Fatal(err)
	}
	out, err := s.LoadConfig()
	if err != nil || out.Locale != "uk" || out.Emails[0] != "me@x.io" {
		t.Fatalf("round trip: %+v %v", out, err)
	}
	for _, f := range []string{"config.json", "key"} {
		fi, _ := os.Stat(filepath.Join(s.Dir, f))
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("%s perms = %v", f, fi.Mode().Perm())
		}
	}
}

func TestIDIsStableAndKeyed(t *testing.T) {
	dir := t.TempDir()
	a, _ := Open(dir)
	b, _ := Open(dir) // reopen: same key
	c, _ := Open(t.TempDir())
	if a.ID("x") != b.ID("x") {
		t.Error("id must be stable for the same key")
	}
	if a.ID("x") == c.ID("x") {
		t.Error("different keys must give different ids")
	}
	if a.ID("ab", "c") == a.ID("a", "bc") {
		t.Error("parts must be separated")
	}
}

func TestLockSerializes(t *testing.T) {
	s, _ := Open(t.TempDir())
	var mu sync.Mutex
	inside := 0
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock, err := s.Lock()
			if err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			inside++
			if inside != 1 {
				t.Error("two holders at once")
			}
			mu.Unlock()
			mu.Lock()
			inside--
			mu.Unlock()
			unlock()
		}()
	}
	wg.Wait()
}
