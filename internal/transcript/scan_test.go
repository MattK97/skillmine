package transcript

import (
	"strings"
	"testing"
)

func TestScanFile(t *testing.T) {
	const input = `
{"type":"user","uuid":"u1","sessionId":"s1","timestamp":"2026-07-01T10:00:00Z","message":{"role":"user","content":"reset the database"}}
{"type":"assistant","uuid":"a1","sessionId":"s1","timestamp":"2026-07-01T10:00:05Z","message":{"role":"assistant","content":[{"type":"text","text":"sure"},{"type":"tool_use","name":"Bash","input":{"command":"psql -c 'delete from products'"}}]}}
{"type":"user","uuid":"u2","sessionId":"s1","timestamp":"2026-07-01T10:01:00Z","message":{"role":"user","content":[{"type":"text","text":"thanks"}]}}
`

	got, err := ScanFile(strings.NewReader(input), "demo")
	if err != nil {
		t.Fatalf("ScanFile() unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ScanFile() returned %d records, want 3", len(got))
	}

	if !got[0].IsUser || got[0].Text != "reset the database" {
		t.Errorf("record 0 = %+v", got[0])
	}
	if got[0].Project != "demo" {
		t.Errorf("Project = %q, want %q", got[0].Project, "demo")
	}
	if got[1].IsUser || len(got[1].Tools) != 1 || got[1].Tools[0].Tool != "Bash" {
		t.Errorf("record 1 = %+v", got[1])
	}
	if !strings.Contains(got[1].Tools[0].Args, "delete from products") {
		t.Errorf("Args = %q, want the command text", got[1].Tools[0].Args)
	}
	// content given as a block array must still yield text.
	if !got[2].IsUser || got[2].Text != "thanks" {
		t.Errorf("record 2 = %+v", got[2])
	}
}

func TestScanFileSkips(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"system reminder", `{"type":"user","uuid":"x","message":{"role":"user","content":"<system-reminder>note</system-reminder>"}}`},
		{"slash command", `{"type":"user","uuid":"x","message":{"role":"user","content":"<command-name>/init</command-name>"}}`},
		{"task notification", `{"type":"user","uuid":"x","message":{"role":"user","content":"<task-notification>id</task-notification>"}}`},
		{"interrupted", `{"type":"user","uuid":"x","message":{"role":"user","content":"[Request interrupted by user]"}}`},
		{"interrupted for tool use", `{"type":"user","uuid":"x","message":{"role":"user","content":"[Request interrupted by user for tool use]"}}`},
		{"compact summary", `{"type":"user","uuid":"x","isCompactSummary":true,"message":{"role":"user","content":"This session is being continued"}}`},
		{"meta", `{"type":"user","uuid":"x","isMeta":true,"message":{"role":"user","content":"real text"}}`},
		{"sidechain", `{"type":"user","uuid":"x","isSidechain":true,"message":{"role":"user","content":"real text"}}`},
		{"blank", `{"type":"user","uuid":"x","message":{"role":"user","content":"   "}}`},
		{"tool result only", `{"type":"user","uuid":"x","message":{"role":"user","content":[{"type":"tool_result","content":"ok"}]}}`},
		{"assistant without tools", `{"type":"assistant","uuid":"x","message":{"role":"assistant","content":[{"type":"text","text":"just talking"}]}}`},
		{"malformed json", `{not json at all`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ScanFile(strings.NewReader(tt.line), "p")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 0 {
				t.Errorf("expected the entry to be skipped, got %+v", got)
			}
		})
	}
}

func TestScanFileLongLine(t *testing.T) {
	// A line well past the 64 KB Scanner default must still be read.
	long := strings.Repeat("x", 200_000)
	line := `{"type":"user","uuid":"u1","message":{"role":"user","content":"` + long + `"}}`

	got, err := ScanFile(strings.NewReader(line), "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || len(got[0].Text) != len(long) {
		t.Errorf("long line was truncated or dropped")
	}
}

func TestProjectName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/x/-Users-alice-work", "work"},
		{"/x/-Users-alice-work-api", "work-api"},
		{"/x/-Users-alice-code-projects-demo", "code-projects-demo"},
		{"/x/plain-name", "plain-name"},
	}
	for _, tt := range tests {
		if got := projectName(tt.in); got != tt.want {
			t.Errorf("projectName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestShorten(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"short", 20, "short"},
		{"collapse   inner\nwhitespace", 40, "collapse inner whitespace"},
		{"truncate me here", 8, "truncate…"},
		// Multi-byte characters must not be cut mid-rune.
		{"zażółć gęślą jaźń", 6, "zażółć…"},
	}
	for _, tt := range tests {
		if got := Shorten(tt.in, tt.n); got != tt.want {
			t.Errorf("Shorten(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}
}

func TestPrompts(t *testing.T) {
	recs := []Record{
		{UUID: "1", IsUser: true, Text: "a"},
		{UUID: "2", IsUser: false},
		{UUID: "3", IsUser: true, Text: "b"},
	}
	got := Prompts(recs)
	if len(got) != 2 || got[0].Text != "a" || got[1].Text != "b" {
		t.Errorf("Prompts() = %+v", got)
	}
}
