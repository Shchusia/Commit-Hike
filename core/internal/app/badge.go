package app

import (
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
	value := FormatDistance(j.DistanceM, st.Locale) + " · " + j.Route.Name
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
