// Package evidence gathers what happened around a prompt.
//
// A prompt says what the user asked for. The tool calls that follow say how it
// was done. What the user said next says whether it worked — a correction there
// reveals what the first attempt got wrong, which is the single most useful
// input when turning a repeated request into a reusable skill.
package evidence

import (
	"time"

	"github.com/MattK97/skillmine/internal/transcript"
)

const (
	// maxSteps caps tool calls per exchange. Long debugging runs can produce
	// hundreds; only the opening moves are reproducible as a procedure.
	maxSteps = 25

	// maxFollowups caps how many further user turns are collected. Three is
	// enough to catch a correction or a confirmation before the conversation
	// usually drifts to another topic.
	maxFollowups = 3

	maxFollowupText = 600
)

// Item is one prompt together with what followed it.
type Item struct {
	Prompt transcript.Record `json:"prompt"`
	Steps  []transcript.Turn `json:"steps"`

	// Followups holds the user's reactions: corrections, confirmations,
	// or a change of subject.
	Followups []Followup `json:"followups,omitempty"`
}

// Followup is a later user turn and the tool calls it triggered.
type Followup struct {
	Text  string            `json:"text"`
	At    time.Time         `json:"at"`
	Steps []transcript.Turn `json:"steps"`
}

// Collect gathers evidence for the given prompt UUIDs.
//
// It takes UUIDs rather than search results so this package needs no knowledge
// of how the prompts were found.
func Collect(all []transcript.Record, promptUUIDs []string) []Item {
	pos := make(map[string]int, len(all))
	for i, r := range all {
		pos[r.UUID] = i
	}

	items := make([]Item, 0, len(promptUUIDs))
	for _, id := range promptUUIDs {
		i, ok := pos[id]
		if !ok {
			continue
		}
		steps, followups := exchangeAfter(all, i)
		items = append(items, Item{Prompt: all[i], Steps: steps, Followups: followups})
	}
	return items
}

// exchangeAfter walks forward from a prompt, splitting what it finds into tool
// calls made immediately and subsequent user exchanges.
//
// It relies on record order rather than the parentUuid chain: .jsonl files are
// appended sequentially, so order matches time, and the code is far simpler.
// Only the session boundary needs guarding.
func exchangeAfter(all []transcript.Record, start int) ([]transcript.Turn, []Followup) {
	session := all[start].SessionID

	var steps []transcript.Turn
	var followups []Followup

	// Index of the followup currently being filled; -1 means "still on the
	// original prompt". An index rather than a pointer into the slice, because
	// append may reallocate the backing array.
	cur := -1

	for i := start + 1; i < len(all); i++ {
		r := all[i]
		if r.SessionID != session {
			break
		}

		if r.IsUser {
			if len(followups) >= maxFollowups {
				break
			}
			followups = append(followups, Followup{
				Text: transcript.Shorten(r.Text, maxFollowupText),
				At:   r.At,
			})
			cur = len(followups) - 1
			continue
		}

		for _, t := range r.Tools {
			if cur < 0 {
				if len(steps) >= maxSteps {
					break
				}
				steps = append(steps, t)
				continue
			}
			if len(followups[cur].Steps) >= maxSteps {
				break
			}
			followups[cur].Steps = append(followups[cur].Steps, t)
		}
	}
	return steps, followups
}
