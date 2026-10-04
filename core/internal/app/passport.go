package app

import (
	"sort"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
	"github.com/Shchusia/commit-hike/core/internal/routes"
	"github.com/Shchusia/commit-hike/core/internal/store"
)

// passport collects a stamp for every stop reached on every journey, current
// and past: the time of the commit that took the journey past it. A route
// walked twice keeps its first stamps.
func (s *Service) passport(cfg *store.Config, st *store.State, eff map[string]float64, chain []string) ([]protocol.Stamp, map[string]int) {
	type trip struct {
		routeID, pid string
		since, until int64
	}
	var trips []trip
	for _, p := range cfg.PastJourneys {
		trips = append(trips, trip{p.RouteID, p.Project, p.Since, p.Until})
	}
	if a := cfg.GlobalJourney; a != nil {
		trips = append(trips, trip{a.RouteID, "", a.Since, 0})
	}
	for pid, a := range cfg.ProjectJourneys {
		trips = append(trips, trip{a.RouteID, pid, a.Since, 0})
	}

	// counted commits, oldest first
	ids := make([]string, 0, len(eff))
	for id := range eff {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool {
		ta, tb := st.Commits[ids[a]].Time, st.Commits[ids[b]].Time
		return ta < tb || (ta == tb && ids[a] < ids[b])
	})

	first := map[string]protocol.Stamp{} // route/waypoint -> earliest stamp
	stops := map[string]int{}
	for _, t := range trips {
		r := s.routes[t.routeID]
		if r == nil {
			continue // a removed route
		}
		stops[r.ID] = len(r.Waypoints)
		wps := append([]routes.Waypoint(nil), r.Waypoints...)
		sort.Slice(wps, func(a, b int) bool { return wps[a].AtM < wps[b].AtM })
		walked, next := 0.0, 0
		for _, id := range ids {
			c := st.Commits[id]
			if c.Time < t.since || (t.until != 0 && c.Time >= t.until) || (t.pid != "" && c.Project != t.pid) {
				continue
			}
			walked += eff[id]
			for next < len(wps) && wps[next].AtM <= walked {
				w := wps[next]
				key := r.ID + "/" + w.ID
				if old, ok := first[key]; !ok || c.Time < old.ReachedAt {
					first[key] = protocol.Stamp{
						RouteID: r.ID, RouteName: r.T(chain, "name"), WaypointID: w.ID,
						Name: r.T(chain, "waypoints."+w.ID+".name"), Kind: w.Kind, ElevationM: w.ElevationM, ReachedAt: c.Time,
					}
				}
				next++
			}
			if next == len(wps) {
				break
			}
		}
	}
	out := make([]protocol.Stamp, 0, len(first))
	for _, st := range first {
		out = append(out, st)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].ReachedAt != out[b].ReachedAt {
			return out[a].ReachedAt < out[b].ReachedAt
		}
		return out[a].RouteID+out[a].WaypointID < out[b].RouteID+out[b].WaypointID
	})
	return out, stops
}
