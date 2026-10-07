package app

import (
	"slices"
	"strings"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

var weekdayNames = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// RestDays shows or changes the weekdays off. set is a comma-separated list
// of day names (mon … sun) or "none"; "" only reads.
func (s *Service) RestDays(set string) (*protocol.RestDaysInfo, error) {
	if set != "" {
		days, err := parseRestDays(set)
		if err != nil {
			return nil, err
		}
		unlock, err := s.lock()
		if err != nil {
			return nil, err
		}
		defer unlock()
		cfg, err := s.st.LoadConfig()
		if err != nil {
			return nil, err
		}
		cfg.RestDays = days
		if err := s.st.SaveConfig(cfg); err != nil {
			return nil, err
		}
	}
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	return &protocol.RestDaysInfo{Days: append([]int{}, cfg.RestDays...)}, nil
}

func parseRestDays(set string) ([]int, error) {
	if strings.TrimSpace(set) == "none" {
		return nil, nil
	}
	var days []int
	for _, part := range strings.Split(set, ",") {
		name := strings.ToLower(strings.TrimSpace(part))
		i := slices.Index(weekdayNames, name)
		if i < 0 && len(name) > 3 {
			i = slices.Index(weekdayNames, name[:3]) // "saturday" works too
		}
		if i < 0 {
			return nil, fail(protocol.CodeInvalidArgument, "unknown day %q: use mon, tue, wed, thu, fri, sat, sun or none", part)
		}
		if !slices.Contains(days, i) {
			days = append(days, i)
		}
	}
	if len(days) == 7 {
		return nil, fail(protocol.CodeInvalidArgument, "at least one day must be a working day")
	}
	slices.Sort(days)
	return days, nil
}

// SettingsChange lists the settings to change; empty fields stay as they are.
type SettingsChange struct {
	ReduceMotion, HighContrast, Notifications, Festive, Ambient string
}

// Settings shows or changes the panel and notification choices.
func (s *Service) Settings(ch SettingsChange) (*protocol.Settings, error) {
	check := func(name, v string, allowed ...string) error {
		if v != "" && !slices.Contains(allowed, v) {
			return fail(protocol.CodeInvalidArgument, "%s must be one of %s", name, strings.Join(allowed, ", "))
		}
		return nil
	}
	for _, err := range []error{
		check("reduce-motion", ch.ReduceMotion, "auto", "on", "off"),
		check("high-contrast", ch.HighContrast, "auto", "on", "off"),
		check("notifications", ch.Notifications, "all", "milestones", "off"),
		check("festive", ch.Festive, "on", "off"),
		check("ambient", ch.Ambient, "on", "off"),
	} {
		if err != nil {
			return nil, err
		}
	}
	if ch != (SettingsChange{}) {
		unlock, err := s.lock()
		if err != nil {
			return nil, err
		}
		defer unlock()
		cfg, err := s.st.LoadConfig()
		if err != nil {
			return nil, err
		}
		set := func(dst *string, v, def string) {
			if v == def {
				*dst = "" // the default isn't stored
			} else if v != "" {
				*dst = v
			}
		}
		set(&cfg.Prefs.ReduceMotion, ch.ReduceMotion, "auto")
		set(&cfg.Prefs.HighContrast, ch.HighContrast, "auto")
		set(&cfg.Prefs.Notifications, ch.Notifications, "all")
		set(&cfg.Prefs.Festive, ch.Festive, "on")
		set(&cfg.Prefs.Ambient, ch.Ambient, "on")
		if err := s.st.SaveConfig(cfg); err != nil {
			return nil, err
		}
	}
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	out := settingsOf(cfg)
	return &out, nil
}

func settingsOf(cfg *store.Config) protocol.Settings {
	or := func(v, def string) string {
		if v == "" {
			return def
		}
		return v
	}
	return protocol.Settings{
		ReduceMotion:  or(cfg.Prefs.ReduceMotion, "auto"),
		HighContrast:  or(cfg.Prefs.HighContrast, "auto"),
		Notifications: or(cfg.Prefs.Notifications, "all"),
		Festive:       or(cfg.Prefs.Festive, "on"),
		Ambient:       or(cfg.Prefs.Ambient, "on"),
	}
}
