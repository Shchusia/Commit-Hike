package app

import (
	"strings"
	"testing"
)

func TestPromptLine(t *testing.T) {
	s, r := setup(t, "chornohora-ridge")
	if got, err := s.Prompt(PromptOptions{Repo: r.dir, Lang: "uk", Icon: "🥾"}); err != nil || got != "🥾 0 м · Заросляк" {
		t.Fatalf("before any commit: %q %v", got, err)
	}
	r.commit("", "a.go", 3) // 50 m in tests (1 base point = 1 m)
	// Without --scan the prompt shows what the last scan counted: nothing yet.
	if got, _ := s.Prompt(PromptOptions{Repo: r.dir, Lang: "en", Icon: "🥾"}); got != "🥾 0 m · Zaroslyak" {
		t.Fatalf("no scan: %q", got)
	}
	got, err := s.Prompt(PromptOptions{Repo: r.dir, Lang: "en", Icon: "🥾", Scan: true})
	if err != nil || got != "🥾 50 m · Zaroslyak" {
		t.Fatalf("scan after a commit: %q %v", got, err)
	}
	// The same HEAD again: no scan (a commit made meanwhile isn't counted yet,
	// because the prompt only looks at HEAD).
	heads, _ := s.st.ReadFile(promptHeadsFile)
	if !strings.Contains(string(heads), ":") {
		t.Fatalf("the HEAD cache wasn't written: %s", heads)
	}
	if got, _ := s.Prompt(PromptOptions{Repo: r.dir, Lang: "de", Icon: ""}); got != "50 m · Saroslak" {
		t.Fatalf("no icon: %q", got)
	}
}

func TestPromptOutsideARepository(t *testing.T) {
	s, _ := setup(t, "demo-trail")
	got, err := s.Prompt(PromptOptions{Repo: t.TempDir(), Lang: "en", Icon: "🥾", Scan: true})
	if err != nil || !strings.HasPrefix(got, "🥾 0 m · ") {
		t.Fatalf("outside a repository the main journey shows: %q %v", got, err)
	}
}

func TestLangFromEnv(t *testing.T) {
	for _, c := range []struct{ all, msgs, lang, want string }{
		{"", "", "uk_UA.UTF-8", "uk"},
		{"de_DE@euro", "", "en_US.UTF-8", "de"},
		{"", "pl_PL", "", "pl"},
		{"C", "", "uk_UA.UTF-8", ""},
		{"", "", "", ""},
		{"", "", "_", ""},
	} {
		t.Setenv("LC_ALL", c.all)
		t.Setenv("LC_MESSAGES", c.msgs)
		t.Setenv("LANG", c.lang)
		if got := LangFromEnv(); got != c.want {
			t.Errorf("LC_ALL=%q LC_MESSAGES=%q LANG=%q: %q, want %q", c.all, c.msgs, c.lang, got, c.want)
		}
	}
}
