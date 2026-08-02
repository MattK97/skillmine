package evidence

import (
	"testing"

	"github.com/MattK97/skillmine/internal/transcript"
)

func TestCollect(t *testing.T) {
	all := []transcript.Record{
		{UUID: "u1", SessionID: "s1", IsUser: true, Text: "reset the database"},
		{UUID: "a1", SessionID: "s1", Tools: []transcript.Turn{{Tool: "Bash", Args: "psql"}}},
		{UUID: "a2", SessionID: "s1", Tools: []transcript.Turn{{Tool: "Read", Args: "schema.sql"}}},
		{UUID: "u2", SessionID: "s1", IsUser: true, Text: "thanks"},
		{UUID: "a3", SessionID: "s1", Tools: []transcript.Turn{{Tool: "Write"}}},
	}

	got := Collect(all, []string{"u1"})
	if len(got) != 1 {
		t.Fatalf("Collect() returned %d items, want 1", len(got))
	}
	// Steps stop at the next user turn.
	if len(got[0].Steps) != 2 {
		t.Fatalf("Steps = %+v, want 2", got[0].Steps)
	}
	if got[0].Steps[0].Tool != "Bash" || got[0].Steps[1].Tool != "Read" {
		t.Errorf("Steps = %+v", got[0].Steps)
	}
}

// TestCollectCapturesUserReactions is the core behaviour of this package:
// a correction tells you what the first attempt missed.
func TestCollectCapturesUserReactions(t *testing.T) {
	all := []transcript.Record{
		{UUID: "u1", SessionID: "s1", IsUser: true, Text: "reset the database"},
		{UUID: "a1", SessionID: "s1", Tools: []transcript.Turn{{Tool: "Bash", Args: "delete from products"}}},
		{UUID: "u2", SessionID: "s1", IsUser: true, Text: "no, keep the vehicles table too"},
		{UUID: "a2", SessionID: "s1", Tools: []transcript.Turn{{Tool: "Bash", Args: "insert into vehicles"}}},
		{UUID: "u3", SessionID: "s1", IsUser: true, Text: "perfect, works"},
	}

	item := Collect(all, []string{"u1"})[0]

	if len(item.Steps) != 1 || item.Steps[0].Args != "delete from products" {
		t.Errorf("Steps = %+v", item.Steps)
	}
	if len(item.Followups) != 2 {
		t.Fatalf("Followups = %d, want 2", len(item.Followups))
	}
	if item.Followups[0].Text != "no, keep the vehicles table too" {
		t.Errorf("Followups[0].Text = %q", item.Followups[0].Text)
	}
	// Tools after a correction belong to that correction, not to the prompt.
	if len(item.Followups[0].Steps) != 1 || item.Followups[0].Steps[0].Args != "insert into vehicles" {
		t.Errorf("Followups[0].Steps = %+v", item.Followups[0].Steps)
	}
	if len(item.Followups[1].Steps) != 0 {
		t.Errorf("Followups[1].Steps = %+v, want none", item.Followups[1].Steps)
	}
}

// TestCollectStopsAtSessionBoundary — adjacency in the corpus does not imply
// the same conversation.
func TestCollectStopsAtSessionBoundary(t *testing.T) {
	all := []transcript.Record{
		{UUID: "u1", SessionID: "s1", IsUser: true, Text: "something"},
		{UUID: "a1", SessionID: "s2", Tools: []transcript.Turn{{Tool: "Bash"}}},
	}
	if got := Collect(all, []string{"u1"}); len(got[0].Steps) != 0 {
		t.Errorf("Steps = %+v, want none: different session", got[0].Steps)
	}
}

func TestCollectPromptAtEnd(t *testing.T) {
	all := []transcript.Record{{UUID: "u1", SessionID: "s1", IsUser: true, Text: "last"}}
	got := Collect(all, []string{"u1"})
	if len(got) != 1 || len(got[0].Steps) != 0 {
		t.Errorf("Collect() = %+v", got)
	}
}

func TestCollectUnknownUUID(t *testing.T) {
	if got := Collect(nil, []string{"missing"}); len(got) != 0 {
		t.Errorf("Collect() = %+v, want empty", got)
	}
}

func TestCollectCapsSteps(t *testing.T) {
	all := []transcript.Record{{UUID: "u1", SessionID: "s1", IsUser: true, Text: "start"}}
	for i := 0; i < maxSteps+10; i++ {
		all = append(all, transcript.Record{SessionID: "s1", Tools: []transcript.Turn{{Tool: "Bash"}}})
	}
	if got := Collect(all, []string{"u1"}); len(got[0].Steps) != maxSteps {
		t.Errorf("Steps = %d, want capped at %d", len(got[0].Steps), maxSteps)
	}
}

func TestCollectCapsFollowups(t *testing.T) {
	all := []transcript.Record{{UUID: "u1", SessionID: "s1", IsUser: true, Text: "start"}}
	for i := 0; i < maxFollowups+5; i++ {
		all = append(all, transcript.Record{SessionID: "s1", IsUser: true, Text: "more"})
	}
	if got := Collect(all, []string{"u1"}); len(got[0].Followups) != maxFollowups {
		t.Errorf("Followups = %d, want capped at %d", len(got[0].Followups), maxFollowups)
	}
}
