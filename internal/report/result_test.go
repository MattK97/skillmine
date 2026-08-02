package report

import (
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
	tests := []struct {
		name         string
		hits         []search.Hit
		wantSessions int
		wantDaySpan  int
		wantRepeated bool
	}{
		{
			name: "no hits",
			hits: nil,
		},
		{
			// Three requests in one session on one day is a debugging loop.
			name: "three times in one session",
			hits: []search.Hit{
				hit("s1", "p", "2026-06-01"),
				hit("s1", "p", "2026-06-01"),
				hit("s1", "p", "2026-06-01"),
			},
			wantSessions: 1,
			wantDaySpan:  1,
			wantRepeated: false,
		},
		{
			name: "three times across three sessions",
			hits: []search.Hit{
				hit("s1", "p", "2026-06-01"),
				hit("s2", "p", "2026-06-15"),
				hit("s3", "p", "2026-06-30"),
			},
			wantSessions: 3,
			wantDaySpan:  30,
			wantRepeated: true,
		},
		{
			name: "twice is not enough",
			hits: []search.Hit{
				hit("s1", "p", "2026-06-01"),
				hit("s2", "p", "2026-06-02"),
			},
			wantSessions: 2,
			wantDaySpan:  2,
			wantRepeated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Summarize(tt.hits)

			if got.Hits != len(tt.hits) {
				t.Errorf("Hits = %d, want %d", got.Hits, len(tt.hits))
			}
			if got.Sessions != tt.wantSessions {
				t.Errorf("Sessions = %d, want %d", got.Sessions, tt.wantSessions)
			}
			if got.DaySpan != tt.wantDaySpan {
				t.Errorf("DaySpan = %d, want %d", got.DaySpan, tt.wantDaySpan)
			}
			if got.Repeated != tt.wantRepeated {
				t.Errorf("Repeated = %v, want %v (verdict: %s)", got.Repeated, tt.wantRepeated, got.Verdict)
			}
			if got.Verdict == "" {
				t.Error("Verdict is empty")
			}
		})
	}
}

func TestSummarizeIgnoresInputOrder(t *testing.T) {
	s := Summarize([]search.Hit{
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
