package app

import (
	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/score"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

// Scoring reports which rules count the commits and, with history "on" or
// "off", moves the start of the volume rules. "on" applies them to the whole
// history at the real pace (commits from before the pace keep their old
// distance), "off" back to the moment this install moved to them. A repo is
// then re-read from git, so its records from before 1.2.0 store their exact
// lines instead of the ones recovered from their points.
func (s *Service) Scoring(history, repo string) (*protocol.Scoring, error) {
	switch history {
	case "", "on", "off":
	default:
		return nil, fail(protocol.CodeInvalidArgument, "history must be on or off")
	}
	out, err := s.setScoring(history)
	if err != nil || repo == "" {
		return out, err
	}
	if out.Verified, err = s.Verify(repo); err != nil {
		return nil, err
	}
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	st, err := s.st.LoadState()
	if err != nil {
		return nil, err
	}
	out.TotalM = s.status(cfg, st, "", "").TotalM
	return out, nil
}

func (s *Service) setScoring(history string) (*protocol.Scoring, error) {
	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	cfg, err := s.st.LoadConfig()
	if err != nil {
		return nil, err
	}
	st, err := s.st.LoadState()
	if err != nil {
		return nil, err
	}
	changed := s.migratePace(cfg)
	changed = s.migrateScoring(cfg) || changed
	previous := s.status(cfg, st, "", "").TotalM
	start := historyStart(cfg)
	switch history {
	case "on":
		if *cfg.ScoringFrom != start {
			cfg.ScoringFrom, changed = &start, true
		}
	case "off":
		if cfg.ScoringUpgradedAt == 0 {
			return nil, fail(protocol.CodeInvalidArgument, "this install started with the volume rules: they count the whole history")
		}
		if *cfg.ScoringFrom != cfg.ScoringUpgradedAt {
			at := cfg.ScoringUpgradedAt
			cfg.ScoringFrom, changed = &at, true
		}
	}
	if changed {
		if err := s.st.SaveConfig(cfg); err != nil {
			return nil, err
		}
	}
	return &protocol.Scoring{
		From: *cfg.ScoringFrom, History: *cfg.ScoringFrom <= start, UpgradedAt: cfg.ScoringUpgradedAt,
		TotalM: score.Round1(s.status(cfg, st, "", "").TotalM), PreviousM: score.Round1(previous),
	}, nil
}

// historyStart is where the volume rules start when they count the whole
// history: where the real pace starts, since commits from before it keep the
// distance they had (1 base point = 1 m).
func historyStart(cfg *store.Config) int64 {
	if cfg.PaceFrom == nil {
		return 0
	}
	return *cfg.PaceFrom
}
