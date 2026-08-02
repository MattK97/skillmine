// Package report formats search results for humans and for language models.
package report

import (
	"time"

	"github.com/MattK97/skillmine/internal/evidence"
	"github.com/MattK97/skillmine/internal/search"
	"github.com/MattK97/skillmine/internal/transcript"
)

// Thresholds for calling a topic repeated.
//
// Spread across sessions matters more than raw count: three requests inside one
// session is a debugging loop, three across three sessions is a habit.
const (
	minHits     = 3
	minSessions = 2
)

// Result is everything the tool has to say about one query.
type Result struct {
	Query    string          `json:"query"`
	Summary  Summary         `json:"summary"`
	Hits     []search.Hit    `json:"hits"`
	Evidence []evidence.Item `json:"evidence,omitempty"`
}

// Summary is the spread of the hits, plus a ready-made verdict so a model
// consuming the JSON does not have to derive one.
type Summary struct {
	Hits     int `json:"hits"`
	Sessions int `json:"sessions"`
	Projects int `json:"projects"`
	DaySpan  int `json:"day_span"`

	First time.Time `json:"first,omitempty"`
	Last  time.Time `json:"last,omitempty"`

	Repeated bool   `json:"repeated"`
	Verdict  string `json:"verdict"`
}

// Summarize computes the spread of the given hits.
//
// Pass every hit above the score threshold, not a list already truncated for
// display: how many rows are shown must not change the verdict.
func Summarize(hits []search.Hit) Summary {
	s := Summary{Hits: len(hits)}
	if len(hits) == 0 {
		s.Verdict = "no matches — this topic is not in the history"
		return s
	}

	sessions := make(map[string]struct{}, len(hits))
	projects := make(map[string]struct{}, 4)
	first, last := hits[0].Record.At, hits[0].Record.At

	for _, h := range hits {
		sessions[h.Record.SessionID] = struct{}{}
		projects[h.Record.Project] = struct{}{}
		if h.Record.At.Before(first) {
			first = h.Record.At
		}
		if h.Record.At.After(last) {
			last = h.Record.At
		}
	}

	s.Sessions = len(sessions)
	s.Projects = len(projects)
	s.First, s.Last = first, last
	s.DaySpan = int(last.Sub(first).Hours()/24) + 1
	s.Repeated = s.Hits >= minHits && s.Sessions >= minSessions

	if s.Repeated {
		s.Verdict = "repeated — worth turning into a skill"
	} else {
		s.Verdict = "not repeated often enough to justify a skill"
	}
	return s
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func shorten(s string, n int) string { return transcript.Shorten(s, n) }
