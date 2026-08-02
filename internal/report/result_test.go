package report

import (
	"strings"
	"testing"
	"time"

	"github.com/MattK97/skillmine/internal/search"
	"github.com/MattK97/skillmine/internal/transcript"
)

func hit(session, project, day string) search.Hit {
	at, err := time.Parse("2006-01-02", day)
	if err != nil {
		panic(err)
	}
	return search.Hit{Record: transcript.Record{SessionID: session, Project: project, At: at}}
}

func TestSummarize(t *testing.T) {
	const query = "reset the test database"

	tests := []struct {
		name          string
		hits          []search.Hit
		wantDays      int
		wantSessions  int
		wantDaySpan   int
		wantRecurring bool
	}{
		{
			name:     "no hits",
			hits:     nil,
			wantDays: 0,
		},
		{
			// Repeated attempts on one day are one debugging session,
			// however many sessions the transcript happens to record.
			name: "four attempts on a single day",
			hits: []search.Hit{
				hit("s1", "p", "2026-06-01"),
				hit("s1", "p", "2026-06-01"),
				hit("s2", "p", "2026-06-01"),
				hit("s2", "p", "2026-06-01"),
			},
			wantDays:      1,
			wantSessions:  2,
			wantDaySpan:   1,
			wantRecurring: false,
		},
		{
			// One long-running session spanning weeks must still count as
			// recurring: session ids survive resumption, days do not.
			name: "one session across three days",
			hits: []search.Hit{
				hit("s1", "p", "2026-06-01"),
				hit("s1", "p", "2026-06-15"),
				hit("s1", "p", "2026-06-30"),
			},
			wantDays:      3,
			wantSessions:  1,
			wantDaySpan:   30,
			wantRecurring: true,
		},
		{
			name: "two days is not enough",
			hits: []search.Hit{
				hit("s1", "p", "2026-06-01"),
				hit("s2", "p", "2026-06-02"),
				hit("s3", "p", "2026-06-02"),
			},
			wantDays:      2,
			wantSessions:  3,
			wantDaySpan:   2,
			wantRecurring: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Summarize(query, tt.hits)

			if got.Hits != len(tt.hits) {
				t.Errorf("Hits = %d, want %d", got.Hits, len(tt.hits))
			}
			if got.Days != tt.wantDays {
				t.Errorf("Days = %d, want %d", got.Days, tt.wantDays)
			}
			if got.Sessions != tt.wantSessions {
				t.Errorf("Sessions = %d, want %d", got.Sessions, tt.wantSessions)
			}
			if got.DaySpan != tt.wantDaySpan {
				t.Errorf("DaySpan = %d, want %d", got.DaySpan, tt.wantDaySpan)
			}
			if got.Recurring != tt.wantRecurring {
				t.Errorf("Recurring = %v, want %v (guidance: %s)", got.Recurring, tt.wantRecurring, got.Guidance)
			}
			if got.Spread == "" || got.Guidance == "" {
				t.Errorf("Spread or Guidance is empty: %+v", got)
			}
		})
	}
}

// TestSummarizeFlagsBroadQuery — a single content word retrieves a subject area,
// which is what produced the misleading verdict this replaced.
func TestSummarizeFlagsBroadQuery(t *testing.T) {
	hits := []search.Hit{
		hit("s1", "p", "2026-06-01"),
		hit("s2", "p", "2026-06-10"),
		hit("s3", "p", "2026-06-20"),
	}

	broad := Summarize("supabase", hits)
	if !broad.BroadQuery {
		t.Error("BroadQuery = false for a one-word query")
	}
	if !strings.Contains(broad.Guidance, "subject area") {
		t.Errorf("guidance should warn about breadth, got: %s", broad.Guidance)
	}

	specific := Summarize("reset database keep user account", hits)
	if specific.BroadQuery {
		t.Error("BroadQuery = true for a multi-word query")
	}
}

// TestSummarizeNeverPromisesASkill — the tool measures recurrence; deciding
// whether that is worth encoding is the reader's job.
func TestSummarizeNeverPromisesASkill(t *testing.T) {
	got := Summarize("reset the test database", []search.Hit{
		hit("s1", "p", "2026-06-01"),
		hit("s2", "p", "2026-06-10"),
		hit("s3", "p", "2026-06-20"),
	})
	if !got.Recurring {
		t.Fatal("expected these hits to count as recurring")
	}
	for _, banned := range []string{"worth", "should", "good candidate"} {
		if strings.Contains(strings.ToLower(got.Guidance), banned) {
			t.Errorf("guidance passes judgement (%q): %s", banned, got.Guidance)
		}
	}
	if !strings.Contains(got.Guidance, "same topic") && !strings.Contains(got.Guidance, "shares a topic") {
		t.Errorf("guidance must ask the reader to check topic vs request, got: %s", got.Guidance)
	}
}

// TestDaySpanNeverBelowDayCount pins an invariant that was briefly violated:
// hits on two calendar days a few hours apart reported a one-day window.
func TestDaySpanNeverBelowDayCount(t *testing.T) {
	at := func(s string) search.Hit {
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return search.Hit{Record: transcript.Record{SessionID: "s", Project: "p", At: ts}}
	}

	tests := []struct {
		name            string
		hits            []search.Hit
		wantDays        int
		wantSpanAtLeast int
	}{
		{
			name:            "fifteen hours apart across midnight",
			hits:            []search.Hit{at("2026-07-27T18:00:00Z"), at("2026-07-28T09:00:00Z")},
			wantDays:        2,
			wantSpanAtLeast: 2,
		},
		{
			name:            "same day, hours apart",
			hits:            []search.Hit{at("2026-07-27T01:00:00Z"), at("2026-07-27T23:00:00Z")},
			wantDays:        1,
			wantSpanAtLeast: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Summarize("some multi word query", tt.hits)
			if got.Days != tt.wantDays {
				t.Errorf("Days = %d, want %d", got.Days, tt.wantDays)
			}
			if got.DaySpan < tt.wantSpanAtLeast {
				t.Errorf("DaySpan = %d, want at least %d", got.DaySpan, tt.wantSpanAtLeast)
			}
			if got.DaySpan < got.Days {
				t.Errorf("DaySpan (%d) below Days (%d): %s", got.DaySpan, got.Days, got.Spread)
			}
		})
	}
}

func TestSummarizeIgnoresInputOrder(t *testing.T) {
	s := Summarize("q1 q2", []search.Hit{
		hit("s1", "p", "2026-06-30"),
		hit("s2", "p", "2026-06-01"),
	})
	if got := s.First.Format("2006-01-02"); got != "2026-06-01" {
		t.Errorf("First = %s, want 2026-06-01", got)
	}
	if got := s.Last.Format("2006-01-02"); got != "2026-06-30" {
		t.Errorf("Last = %s, want 2026-06-30", got)
	}
}
