package transcript

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// rawRecord mirrors one line of a .jsonl transcript. Fields we do not declare
// are ignored by encoding/json, so this covers only what we need.
type rawRecord struct {
	Type        string    `json:"type"`
	UUID        string    `json:"uuid"`
	SessionID   string    `json:"sessionId"`
	Timestamp   time.Time `json:"timestamp"`
	IsMeta      bool      `json:"isMeta"`
	IsSidechain bool      `json:"isSidechain"`

	// Written by Claude Code when a session runs out of context. Stored as a
	// "user" record even though the user did not type it.
	IsCompactSummary bool `json:"isCompactSummary"`
	// Entries shown in the transcript but never sent to the model.
	IsVisibleInTranscriptOnly bool `json:"isVisibleInTranscriptOnly"`

	Message struct {
		Role    string  `json:"role"`
		Content content `json:"content"`
	} `json:"message"`
}

// content is the message.content field, which has two possible shapes:
// a plain string, or an array of typed blocks.
type content struct {
	Text  string
	Tools []Turn
}

// UnmarshalJSON handles both shapes of message.content.
func (c *content) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		c.Text = s
		return nil
	}

	// Tool inputs differ per tool, so json.RawMessage defers decoding them.
	var blocks []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(data, &blocks); err != nil {
		return fmt.Errorf("content is neither a string nor a block array: %w", err)
	}

	var texts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				texts = append(texts, b.Text)
			}
		case "tool_use":
			c.Tools = append(c.Tools, Turn{
				Tool: b.Name,
				Args: Shorten(string(b.Input), 240),
			})
		}
	}
	c.Text = strings.Join(texts, "\n")
	return nil
}

// toRecord converts a raw entry, reporting whether it is one we care about.
func (r rawRecord) toRecord(project string) (Record, bool) {
	if r.IsMeta || r.IsSidechain || r.IsCompactSummary || r.IsVisibleInTranscriptOnly || r.UUID == "" {
		return Record{}, false
	}

	rec := Record{
		UUID:      r.UUID,
		SessionID: r.SessionID,
		Project:   project,
		At:        r.Timestamp,
	}

	switch r.Type {
	case "user":
		text := strings.TrimSpace(r.Message.Content.Text)
		if text == "" || isNoise(text) {
			return Record{}, false
		}
		rec.IsUser = true
		rec.Text = text
		return rec, true

	case "assistant":
		if len(r.Message.Content.Tools) == 0 {
			return Record{}, false
		}
		rec.Tools = r.Message.Content.Tools
		return rec, true
	}

	return Record{}, false
}

// noiseMarkers identify blocks stored as user messages that the user never
// typed. Each of these only ever appears on its own, so dropping the whole
// entry loses nothing.
var noiseMarkers = []string{
	"<system-reminder>",
	"<command-name>",
	"<local-command",
	"<task-notification>",
	"Caveat:",
	"[Request interrupted by user", // also matches "...for tool use]"
}

func isNoise(s string) bool {
	for _, m := range noiseMarkers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}
