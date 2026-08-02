// Package report formats search results for humans and for language models.
package report

import (
	"fmt"
	"time"

	"github.com/MattK97/skillmine/internal/evidence"
	"github.com/MattK97/skillmine/internal/search"
	"github.com/MattK97/skillmine/internal/transcript"
)

// Thresholds for calling a topic recurring.
//
// Distinct days, not sessions. A Claude Code session keeps its id when resumed,
// so one session routinely spans weeks: measured on a real corpus, three of four
// sessions covered multiple days, which made a session count nearly useless as a
// proxy for separate occasions. Calendar days are what "came up again" means.
const (
	minHits = 3
	minDays = 3
)

// Result is everything the tool has to say about one query.
type Result struct {
	Query    string          `json:"query"`
	Summary  Summary         `json:"summary"`
	Hits     []search.Hit    `json:"hits"`
	Evidence []evidence.Item `json:"evidence,omitempty"`
}

// Summary describes how the matches are spread out.
//
// Deliberately, it does not claim that the matches are the same request. Two
// attempts at measuring that from the text alone failed: mean pairwise
// similarity within the hit set ranked a pure topic search ("flutter", 0.160)
// above a genuine repeated request ("reset database, keep the user", 0.089).
// Telling a subject area apart from a recurring request is a semantic judgement,
// so this type reports facts and hands the judgement to the reader.
type Summary struct {
	Hits     int `json:"hits"`
	Days     int `json:"days"`     // distinct calendar days — the meaningful signal
	Sessions int `json:"sessions"` // informational; sessions can span weeks
	Projects int `json:"projects"`
	DaySpan  int `json:"day_span"`

	First time.Time `json:"first,omitempty"`
	Last  time.Time `json:"last,omitempty"`

	// Recurring states a fact the tool can establish: the query matched on
	// several separate days. It does not mean the matches are one request.
	Recurring bool `json:"recurring"`

	// BroadQuery warns that the query had too few content words to describe a
	// request, so the results are likely a subject area.
	BroadQuery bool `json:"broad_query"`

	Spread   string `json:"spread"`
	Guidance string `json:"guidance"`
}

// Summarize computes the spread of the given hits.
//
// Pass every hit above the score threshold, not a list already truncated for
// display: how many rows are shown must not change the summary.
func Summarize(query string, hits []search.Hit) Summary {
	s := Summary{
		Hits:       len(hits),
		BroadQuery: search.ContentWords(query) < 2,
	}

	if len(hits) == 0 {
		s.Spread = "no matches"
		s.Guidance = "this topic is not in the history; nothing to build a skill from"
		return s
	}

	days := make(map[string]struct{}, len(hits))
	sessions := make(map[string]struct{}, 4)
	projects := make(map[string]struct{}, 4)
	first, last := hits[0].Record.At, hits[0].Record.At

	for _, h := range hits {
		// UTC, matching calendarDays below, so the two never disagree.
		days[h.Record.At.UTC().Format("2006-01-02")] = struct{}{}
		sessions[h.Record.SessionID] = struct{}{}
		projects[h.Record.Project] = struct{}{}
		if h.Record.At.Before(first) {
			first = h.Record.At
		}
		if h.Record.At.After(last) {
			last = h.Record.At
		}
	}

	s.Days = len(days)
	s.Sessions = len(sessions)
	s.Projects = len(projects)
	s.First, s.Last = first, last
	// Span is counted in calendar days, not elapsed hours. Two prompts fifteen
	// hours apart across midnight fall on two days, and reporting a one-day
	// window for them contradicts the day count next to it.
	s.DaySpan = calendarDays(first, last)
	s.Recurring = s.Hits >= minHits && s.Days >= minDays

	s.Spread = fmt.Sprintf("%d %s on %d %s, over a %d-day window",
		s.Hits, plural(s.Hits, "prompt", "prompts"),
		s.Days, plural(s.Days, "day", "separate days"),
		s.DaySpan)

	s.Guidance = guidanceFor(s)
	return s
}

// guidanceFor spells out what the reader still has to decide. The tool measures
// recurrence; whether the matches are one request or one subject is not
// something it can see.
func guidanceFor(s Summary) string {
	switch {
	case s.BroadQuery:
		return "one-word query: these results are a subject area, not one request. " +
			"Re-run with several words taken from the actual request."
	case s.Recurring:
		return "recurring. Read the matched prompts before acting: are they the same " +
			"request phrased differently, or unrelated work that shares a topic? " +
			"Only the first makes a skill."
	case s.Hits < minHits:
		return "too few matches to look like a habit"
	default:
		return "matches cluster into too few separate days to look like a habit; " +
			"repeated attempts within one day are usually one debugging session"
	}
}

// calendarDays counts days inclusively between two instants, ignoring the time
// of day. Both are normalised to UTC first, matching how the day set is keyed.
func calendarDays(first, last time.Time) int {
	a := first.UTC().Truncate(24 * time.Hour)
	b := last.UTC().Truncate(24 * time.Hour)
	return int(b.Sub(a).Hours()/24) + 1
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func shorten(s string, n int) string { return transcript.Shorten(s, n) }
