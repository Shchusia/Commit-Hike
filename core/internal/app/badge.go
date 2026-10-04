package app

import (
	"fmt"
	"strings"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/render"
)

// Badge draws a README badge for the journey the panel shows: the project's
// own one when it has one, otherwise the global one.
func (s *Service) Badge(repo, lang string) (*protocol.BadgeResult, error) {
	st, err := s.Status(repo, lang)
	if err != nil {
		return nil, err
	}
	j := st.Global
	if st.Project != nil {
		j = st.Project
	}
	if j == nil {
		return nil, fail(protocol.CodeInvalidArgument, "there is no journey to show yet")
	}
	value := badgeDistance(j.DistanceM, st.Locale) + " · " + j.Route.Name
	if j.Finished {
		value = "✓ " + j.Route.Name
	}
	const file = "commit-hike-badge.svg"
	return &protocol.BadgeResult{
		SVG:      render.Badge("🥾 Commit Hike", value),
		FileName: file,
		Markdown: "[![Commit Hike](" + file + ")](https://github.com/Shchusia/Commit-Hike)",
	}, nil
}

// badgeDistance: whole kilometres from 100 km on, one decimal below; the
// unit and the decimal separator follow the language.
func badgeDistance(m float64, lang string) string {
	km := m / 1000
	s := fmt.Sprintf("%.1f", km)
	if km >= 100 {
		s = fmt.Sprintf("%.0f", km)
	}
	if strings.HasPrefix(lang, "uk") {
		return strings.Replace(s, ".", ",", 1) + " км"
	}
	return s + " km"
}
