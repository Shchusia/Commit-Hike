package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"strings"

	"github.com/Shchusia/commit-hike/core/internal/gitlog"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

// promptHeadsFile remembers the HEAD each work tree had at the last prompt,
// so `prompt --scan` only scans after a commit, not on every prompt.
const promptHeadsFile = "prompt-heads.json"

// PromptOptions tune Prompt.
type PromptOptions struct {
	Repo string // a path inside a git repository; "" or outside a repository shows the main journey
	Lang string
	Icon string // shown before the distance
	// Scan first when HEAD moved since the last prompt for this work tree.
	// It lets people who don't use an IDE count commits from their shell.
	Scan bool
}

// Prompt is one short line for a shell prompt or an editor's status line,
// e.g. "🥾 16,0 км · Озеро Несамовите": the distance and where the walker is
// on the journey the panel would show (the project's own, else the main one).
// It returns "" when there is nothing to show yet.
func (s *Service) Prompt(o PromptOptions) (string, error) {
	if o.Scan && o.Repo != "" {
		if err := s.scanIfMoved(o.Repo, o.Lang); err != nil {
			return "", err
		}
	}
	st, err := s.Status(o.Repo, o.Lang)
	if err != nil {
		return "", err
	}
	j := st.Global
	if st.Project != nil {
		j = st.Project
	}
	if j == nil {
		return "", nil
	}
	text := o.Icon + " "
	if o.Icon == "" {
		text = ""
	}
	switch {
	case j.Finished:
		return text + "✓ " + j.Route.Name, nil
	case j.LastWaypoint != nil:
		return text + FormatDistance(j.DistanceM, st.Locale) + " · " + j.LastWaypoint.Name, nil
	}
	return text + FormatDistance(j.DistanceM, st.Locale) + " · " + j.Route.Name, nil
}

// scanIfMoved scans the repository containing dir when its HEAD differs from
// the one seen at the previous prompt. The cache is per work tree (keyed by a
// hash of its path, like the rest of the state) and lives in the data folder.
func (s *Service) scanIfMoved(dir, lang string) error {
	top, err := gitlog.TopLevel(dir)
	if err != nil {
		return nil //nolint:nilerr // not a repository: nothing to scan
	}
	head := gitlog.Head(top)
	if head == "" {
		return nil // an empty repository
	}
	heads := map[string]string{}
	if data, err := s.st.ReadFile(promptHeadsFile); err == nil {
		_ = json.Unmarshal(data, &heads) // a broken cache only costs one extra scan
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	key := s.st.ID("prompt", top)
	prev, seen := heads[key]
	if seen && prev == head {
		return nil
	}
	if _, err := s.ScanWith(top, lang, ScanOptions{PrevHead: prev}); err != nil {
		if errors.Is(err, store.ErrNotInitialized) {
			return nil
		}
		return err
	}
	heads[key] = head
	data, err := json.Marshal(heads)
	if err != nil {
		return err
	}
	return s.st.WriteFile(promptHeadsFile, data)
}

// FormatDistance writes a distance the way the panel does: whole meters
// below 1 km, one decimal below 100 km, whole kilometres from there, with the
// language's decimal separator and units.
func FormatDistance(m float64, lang string) string {
	m = math.Max(0, m)
	base := strings.ToLower(strings.SplitN(strings.ReplaceAll(lang, "_", "-"), "-", 2)[0])
	unitM, unitKM, comma := "m", "km", false
	switch base {
	case "uk":
		unitM, unitKM, comma = "м", "км", true
	case "pl", "de", "es":
		comma = true
	}
	if m < 1000 {
		return fmt.Sprintf("%.0f %s", math.Round(m), unitM)
	}
	km := m / 1000
	if km >= 100 {
		return fmt.Sprintf("%.0f %s", math.Round(km), unitKM)
	}
	s := fmt.Sprintf("%.1f", km)
	if comma {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s + " " + unitKM
}

// LangFromEnv picks a language from the POSIX locale variables (LC_ALL,
// LC_MESSAGES, LANG), e.g. "uk_UA.UTF-8" → "uk". "" when none is set.
func LangFromEnv() string {
	for _, v := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		val := os.Getenv(v)
		if val == "" {
			continue
		}
		parts := strings.FieldsFunc(val, func(r rune) bool { return r == '_' || r == '.' || r == '@' || r == '-' })
		if len(parts) == 0 {
			continue
		}
		code := strings.ToLower(parts[0])
		if code == "c" || code == "posix" {
			return ""
		}
		return code
	}
	return ""
}
