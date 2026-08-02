// Package transcript reads Claude Code conversation history from .jsonl files.
package transcript

import (
	"strings"
	"time"
)

// Record is a single transcript entry: either a user prompt or an assistant
// response. Text is set only for user prompts, Tools only for assistant ones.
type Record struct {
	UUID      string    `json:"uuid"`
	SessionID string    `json:"session_id"`
	Project   string    `json:"project"`
	At        time.Time `json:"at"`
	IsUser    bool      `json:"is_user"`

	Text  string `json:"text,omitempty"`
	Tools []Turn `json:"tools,omitempty"`
}

// Turn is one tool invocation made by the assistant.
type Turn struct {
	Tool string `json:"tool"`
	Args string `json:"args,omitempty"` // truncated preview
}

// Prompts returns only the user prompts, preserving order.
func Prompts(recs []Record) []Record {
	out := make([]Record, 0, len(recs)/4)
	for _, r := range recs {
		if r.IsUser {
			out = append(out, r)
		}
	}
	return out
}

// Shorten collapses whitespace and truncates to n characters.
// It counts runes, not bytes, so multi-byte characters stay intact.
func Shorten(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
